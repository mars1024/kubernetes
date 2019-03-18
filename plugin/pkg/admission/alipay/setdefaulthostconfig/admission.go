package resourcemutationqos

import (
	"fmt"
	"io"

	log "github.com/golang/glog"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/plugin/pkg/admission/alipay/setdefault"
	"k8s.io/kubernetes/plugin/pkg/admission/alipay/util"
)

const (
	// PluginName is the name for current plugin, it should be unique among all plugins.
	PluginName = "AlipaySetDefaultHostConfig"
)

var (
	_ admission.MutationInterface = &AlipaySetDefaultHostConfig{}
)

// Register is used to register current plugin to APIServer.
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName,
		func(config io.Reader) (admission.Interface, error) {
			return newAlipaySetDefaultHostConfig(), nil
		})
}

// AlipaySetDefaultHostConfig is the main struct to mutate pod resource setting.
type AlipaySetDefaultHostConfig struct {
	*admission.Handler
}

func newAlipaySetDefaultHostConfig() *AlipaySetDefaultHostConfig {
	return &AlipaySetDefaultHostConfig{
		Handler: admission.NewHandler(admission.Create),
	}
}

// Admit set default host config for online pod.
// This should cooperate with AlipayResourceMutationBestEffort.
func (a *AlipaySetDefaultHostConfig) Admit(attr admission.Attributes) (err error) {
	if shouldIgnore(attr) {
		return nil
	}

	pod, ok := attr.GetObject().(*core.Pod)
	if !ok {
		return admission.NewForbidden(attr, fmt.Errorf("unexpected resource"))
	}

	if err = setDefaultHostConfig(pod); err != nil {
		return admission.NewForbidden(attr, err)
	}
	return nil
}

func setDefaultHostConfig(pod *core.Pod) error {
	allocSpec, err := util.PodAllocSpec(pod)
	if err != nil {
		return err
	}

	if allocSpec == nil {
		if err = setdefault.SetDefaultHostConfig(pod); err != nil {
			return fmt.Errorf("failed to set default alloc spec")
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
		log.Errorf("expected pod but got %s", a.GetKind().Kind)
		return true
	}

	if a.GetOperation() != admission.Create {
		// only admit created pod.
		return true
	}

	return false
}
