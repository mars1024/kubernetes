package auditdelete

import (
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/apps"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	settingslisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
)

const (
	// PluginName indicates name of admission plugin.
	PluginName = "AuditDelete"

	modeForStatefulSet = "statefulset.beta1.sigma.ali/mode"
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
	podCounterHandler func(string, admission.Attributes) (int, error)
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
	if a.GetOperation() != admission.Delete || len(a.GetSubresource()) != 0 {
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
		return p.validateWorkloadDeletion(a)
	} else if a.GetKind().Kind == "StatefulSet" {
		set, ok := a.GetObject().(*apps.StatefulSet)
		if !ok {
			return errors.NewBadRequest("Resource was marked with kind StatefulSet but was unable to be converted by AuditDelete")
		}
		if set.Labels[modeForStatefulSet] == "sigma" {
			return p.validateWorkloadDeletion(a)
		}
	}

	return nil
}

func (p *auditDeletePlugin) podCounter(namespace string, expectedOwner admission.Attributes) (int, error) {
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
			if owner == nil || owner.Name != expectedOwner.GetName() || owner.Kind != expectedOwner.GetKind().Kind {
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
		counter func(string, admission.Attributes) (int, error)
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
func (p *auditDeletePlugin) validateWorkloadDeletion(a admission.Attributes) (err error) {
	podNum, err := p.podCounterHandler(a.GetNamespace(), a)
	if err != nil {
		return fmt.Errorf("error listing pods for %s %s/%s: %v", a.GetKind().Kind, a.GetNamespace(), a.GetName(), err)
	}

	if podNum > 0 {
		return fmt.Errorf("forbid to delete %s %s/%s for existing %d pods", a.GetKind().Kind, a.GetNamespace(), a.GetName(), podNum)
	}
	return nil
}
