package resource

import (
	"fmt"
	"io"

	"github.com/golang/glog"
	"k8s.io/apiserver/pkg/admission"

	"k8s.io/kubernetes/pkg/apis/core"
)

const (
	// PluginName is the name for current plugin, it should be unique among all plugins
	PluginName = "AlipayResourceAdmission"
)

var (
	_ admission.ValidationInterface = &AlipayResourceAdmission{}
)

// Register is used to register current plugin to APIServer
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return newAlipayResourceAdmission(), nil
	})
}

// AlipayResourceAdmission is the main struct to validate pod resource setting
type AlipayResourceAdmission struct {
	*admission.Handler
}

func newAlipayResourceAdmission() *AlipayResourceAdmission {
	return &AlipayResourceAdmission{
		Handler: admission.NewHandler(admission.Create),
	}
}

// Validate checks if user provided proper resource.
// Right now, we expect :
// 1. cpu.request >=0, cpu.limit > 0
// 2. memory.request = memory.limit > 0
func (a *AlipayResourceAdmission) Validate(attr admission.Attributes) (err error) {
	if shouldIgnore(attr) {
		return nil
	}

	pod, ok := attr.GetObject().(*core.Pod)
	if !ok {
		return admission.NewForbidden(attr, fmt.Errorf("unexpected resource"))
	}

	if err = validatePodResource(pod); err != nil {
		return admission.NewForbidden(attr, err)
	}
	return nil
}

func validatePodResource(pod *core.Pod) error {
	for _, c := range pod.Spec.Containers {
		// expect cpu.limit is greater than zero
		if c.Resources.Limits.Cpu().IsZero() {
			return fmt.Errorf("container %s cpu limit should greater than 0", c.Name)
		}

		// memory request should equal to limit, and greater than zero
		if c.Resources.Limits.Memory().IsZero() {
			return fmt.Errorf("container %s memory limit should greater than 0", c.Name)
		}

		if c.Resources.Requests.Memory().IsZero() {
			return fmt.Errorf("container %s memory request should greater than 0", c.Name)
		}

		if c.Resources.Limits.Memory().Cmp(*c.Resources.Requests.Memory()) != 0 {
			return fmt.Errorf("container %s memory limit %s should equal to memory request %s", c.Name, c.Resources.Limits.Memory(), c.Resources.Requests.Memory())
		}
	}
	return nil
}

func shouldIgnore(a admission.Attributes) bool {
	resource := a.GetResource().GroupResource()
	if resource != core.Resource("pods") {
		return true
	}

	if a.GetSubresource() != "" {
		return true
	}

	_, ok := a.GetObject().(*core.Pod)
	if !ok {
		glog.Errorf("expected pod but got %s", a.GetKind().Kind)
		return true
	}

	if a.GetOperation() != admission.Create {
		// only admit created pod
		return true
	}
	return false
}
