package auditdelete

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/apps"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	settingslisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	statefulsetcontroller "k8s.io/kubernetes/pkg/controller/statefulset"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
)

const (
	// PluginName indicates name of admission plugin.
	PluginName = "AuditDelete"

	enableCascadingDeletion = "sigma.ali/enable-cascading-deletion"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return newPlugin(), nil
	})
}

// auditDeletePlugin is an implementation of admission.Interface.
type auditDeletePlugin struct {
	*admission.Handler
	podLister    settingslisters.PodLister
	// podCounterHandler is used for testing purpose: they are set to fake
	// functions when testing
	podCounterHandler func(string, metav1.Object) (int, error)
}

var _ admission.ValidationInterface = &auditDeletePlugin{}
var _ = kubeapiserveradmission.WantsInternalKubeInformerFactory(&auditDeletePlugin{})

// NewPlugin creates a new auditDelete admission plugin.
func newPlugin() *auditDeletePlugin {
	return &auditDeletePlugin{
		Handler: admission.NewHandler(admission.Delete),
	}
}

// ValidateInitialization implements the InitializationValidator interface.
func (p *auditDeletePlugin) ValidateInitialization() error {
	if p.podLister == nil {
		return fmt.Errorf("%s requires a podLister", PluginName)
	}
	p.podCounterHandler = p.podCounter
	return nil
}

func (p *auditDeletePlugin) SetInternalKubeInformerFactory(f informers.SharedInformerFactory) {
	podInformer := f.Core().InternalVersion().Pods()
	p.podLister = podInformer.Lister()
	p.SetReadyFunc(podInformer.Informer().HasSynced)
}

// Validate makes an admission decision based on the request attributes
func (p *auditDeletePlugin) Validate(a admission.Attributes) error {
	if a == nil {
		return nil
	}
	if a.GetOperation() != admission.Delete || len(a.GetSubresource()) != 0 || strings.HasPrefix(a.GetNamespace(), "e2e-tests") {
		return nil
	}

	objectMeta, err := getObjectMeta(a.GetObject())
	if err != nil {
		return err
	} else if objectMeta.GetAnnotations()[enableCascadingDeletion] == "true" {
		return nil
	}

	if a.GetKind().Kind == "Namespace" {
		namespace := a.GetNamespace()
		if namespace == "" {
			return nil
		}
		return p.validateNamespaceDeletion(namespace)
	}

	if a.GetKind().Kind == "InPlaceSet" {
		return p.validateWorkloadDeletion(objectMeta)
	} else if a.GetKind().Kind == "StatefulSet" {
		set, ok := a.GetObject().(*apps.StatefulSet)
		if !ok {
			return errors.NewBadRequest("Resource was marked with kind StatefulSet but was unable to be converted by AuditDelete")
		}
		if set.Labels[statefulsetcontroller.ModeForStatefulSet] == "sigma" {
			return p.validateWorkloadDeletion(objectMeta)
		}
	}

	return nil
}

func (p *auditDeletePlugin) podCounter(namespace string, expectedOwner metav1.Object) (int, error) {
	list, err := p.podLister.Pods(namespace).List(labels.Everything())
	if err != nil {
		return 0, err
	}
	count := 0
	for _, p := range list {
		if p.DeletionTimestamp != nil {
			continue
		}
		if expectedOwner != nil {
			owner := metav1.GetControllerOf(p)
			if owner == nil || owner.UID != expectedOwner.GetUID() {
				continue
			}
		}
		count++
	}
	return count, nil
}

// validateNamespaceDeletion returns an error if the namespace contains any workload resources
func (p *auditDeletePlugin) validateNamespaceDeletion(namespace string) (err error) {

	counters := []struct {
		kind    string
		counter func(string, metav1.Object) (int, error)
	}{
		{"pods", p.podCounterHandler},
	}

	var errList []error
	var nonEmptyList []string

	for _, c := range counters {
		num, err := c.counter(namespace, nil)
		if err != nil {
			errList = append(errList, fmt.Errorf("error listing %s, %v", c.kind, err))
			continue
		}
		if num > 0 {
			nonEmptyList = append(nonEmptyList, fmt.Sprintf("%s(%d)", c.kind, num))
		}
	}

	errStr := ""
	if len(nonEmptyList) > 0 {
		errStr += fmt.Sprintf("The namespace %s you are trying to remove contains one or more of these resources: %v. Please delete them and try again.", namespace, nonEmptyList)
	}
	if len(errList) > 0 {
		errStr += fmt.Sprintf("The following error(s) occurred while validating the DELETE operation on the namespace %s: %v.", namespace, errList)
	}
	if errStr != "" {
		return fmt.Errorf(errStr)
	}
	return nil
}

// validateNamespaceDeletion returns an error if the workload contains any pods
func (p *auditDeletePlugin) validateWorkloadDeletion(obj metav1.Object) (err error) {
	podNum, err := p.podCounterHandler(obj.GetNamespace(), obj)
	if err != nil {
		return fmt.Errorf("error listing pods for %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
	}

	if podNum > 0 {
		return fmt.Errorf("forbid to delete %s/%s for existing %d pods", obj.GetNamespace(), obj.GetName(), podNum)
	}
	return nil
}

func getObjectMeta(obj runtime.Object) (*metav1.ObjectMeta, error) {
	var objMeta metav1.ObjectMeta
	var foundObjectMeta bool
	e := reflect.ValueOf(obj).Elem()
	for i := 0; i < e.NumField() && !foundObjectMeta; i++ {
		if e.Type().Field(i).Type == reflect.TypeOf(metav1.ObjectMeta{}) {
			objMeta = e.Field(i).Interface().(metav1.ObjectMeta)
			foundObjectMeta = true
		}
	}

	if !foundObjectMeta {
		return nil, fmt.Errorf("not found ObjectMeta in %#v", obj)
	}
	return &objMeta, nil
}
