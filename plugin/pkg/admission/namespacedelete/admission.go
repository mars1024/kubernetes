package namespacedelete

import (
	"errors"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
)

const (
	// PluginName indicates name of admission plugin.
	PluginName = "AuditDelete"
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
	client internalclientset.Interface
	// podCounterHandler is used for testing purpose: they are set to fake
	// functions when testing
	podCounterHandler func(string) (int, error)
}

var _ admission.ValidationInterface = &auditDeletePlugin{}
var _ = kubeapiserveradmission.WantsInternalKubeClientSet(&auditDeletePlugin{})

// NewPlugin creates a new auditDelete admission plugin.
func newPlugin() *auditDeletePlugin {
	return &auditDeletePlugin{
		Handler: admission.NewHandler(admission.Delete),
	}
}

// ValidateInitialization implements the InitializationValidator interface.
func (p *auditDeletePlugin) ValidateInitialization() error {
	if p.client == nil {
		return fmt.Errorf("%s requires a client", PluginName)
	}
	p.podCounterHandler = p.podCounter
	return nil
}

// SetInternalKubeClientSet implements the WantsInternalKubeClientSet interface.
func (p *auditDeletePlugin) SetInternalKubeClientSet(client internalclientset.Interface) {
	p.client = client
}

// Validate makes an admission decision based on the request attributes
func (p *auditDeletePlugin) Validate(a admission.Attributes) error {
	if a == nil {
		return nil
	}
	if a.GetOperation() != admission.Delete {
		return nil
	}
	if a.GetKind().Kind != "Namespace" {
		return nil
	}
	namespace := a.GetNamespace()
	if namespace == "" {
		return nil
	}
	return p.validateNamespaceDeletion(namespace)
}

func (p *auditDeletePlugin) podCounter(namespace string) (int, error) {
	list, err := p.client.Core().Pods(namespace).List(v1.ListOptions{})
	if err != nil {
		return 0, err
	}
	return len(list.Items), nil
}

// validateNamespaceDeletion returns an error if the namespace contains any workload resources
func (p *auditDeletePlugin) validateNamespaceDeletion(namespace string) (err error) {

	counters := []struct {
		kind    string
		counter func(namespace string) (int, error)
	}{
		{"pods", p.podCounterHandler},
	}

	var errList []error
	var nonEmptyList []string

	for _, c := range counters {
		num, err := c.counter(namespace)
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
		return errors.New(errStr)
	}
	return nil
}
