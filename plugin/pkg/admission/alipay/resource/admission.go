package resource

import (
	"fmt"
	"io"
	"strconv"

	"github.com/golang/glog"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"

	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
	sigmabe "k8s.io/kubernetes/plugin/pkg/admission/alipay/resourcemutationbe"
)

const (
	// PluginName is the name for current plugin, it should be unique among all plugins
	PluginName = "AlipayResourceAdmission"
)

var (
	_ admission.ValidationInterface                           = &AlipayResourceAdmission{}
	_ kubeapiserveradmission.WantsInternalKubeInformerFactory = &AlipayResourceAdmission{}
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

	nsLister corelisters.NamespaceLister
}

func newAlipayResourceAdmission() *AlipayResourceAdmission {
	return &AlipayResourceAdmission{
		Handler: admission.NewHandler(admission.Create),
	}
}

func (a *AlipayResourceAdmission) SetInternalKubeInformerFactory(f internalversion.SharedInformerFactory) {
	a.nsLister = f.Core().InternalVersion().Namespaces().Lister()
	a.SetReadyFunc(f.Core().InternalVersion().Namespaces().Informer().HasSynced)
}

func (a *AlipayResourceAdmission) ValidateInitialization() error {
	if a.nsLister == nil {
		return fmt.Errorf("missing namespaceLister")
	}
	return nil
}

// Validate checks if user provided proper resource.
// Right now, we expect :
// 1. cpu.request >=0, cpu.limit > 0
// 2. memory.request = memory.limit > 0
func (a *AlipayResourceAdmission) Validate(attr admission.Attributes) (err error) {
	if shouldIgnore(attr) {
		return nil
	}
	if !a.WaitForReady(attr.GetContext()) {
		return admission.NewForbidden(attr, fmt.Errorf("not yet ready to handle request"))
	}

	pod, ok := attr.GetObject().(*core.Pod)
	if !ok {
		return admission.NewForbidden(attr, fmt.Errorf("unexpected resource"))
	}

	ns, err := a.nsLister.Get(attr.GetNamespace())
	if err != nil {
		return admission.NewForbidden(attr, err)
	}
	if shouldSkipValidation(ns) {
		return nil
	}

	if err = validatePodResource(pod); err != nil {
		return admission.NewForbidden(attr, err)
	}
	return nil
}

func shouldSkipValidation(ns *core.Namespace) bool {
	if ns.Annotations == nil {
		return false
	}
	v, _ := strconv.ParseBool(ns.Annotations[alipaysigmak8sapi.SkipResourceAdmission])
	return v
}

func validatePodResource(pod *core.Pod) error {
	for _, c := range pod.Spec.Containers {
		// expect cpu.limit is greater than zero
		// ignore sigma best effort container here.
		if c.Resources.Limits.Cpu().IsZero() &&
			!sigmabe.IsSigmaBestEffortPod(pod) {
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

		// pod should have ephemeral storage resource, and request equal to limit.
		if c.Resources.Limits.StorageEphemeral().IsZero() {
			return fmt.Errorf("container %s should have ephemeral storage limit", c.Name)
		}

		if c.Resources.Requests.StorageEphemeral().IsZero() {
			return fmt.Errorf("container %s should have ephemeral storage request", c.Name)
		}

		if c.Resources.Limits.StorageEphemeral().Cmp(*c.Resources.Requests.StorageEphemeral()) != 0 {
			return fmt.Errorf("container %s ephemeral storage limit %s should equal to ephemeral storage request %s", c.Name, c.Resources.Limits.Memory(), c.Resources.Requests.Memory())
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
