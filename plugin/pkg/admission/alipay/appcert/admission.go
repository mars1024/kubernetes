package appcert

import (
	"io"

	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
)

const PluginName = "AlipayAppCert"

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewAlipayAppCert(), nil
	})
}

// AlipayAppCert is an implementation of admission.Interface.
type AlipayAppCert struct {
	*admission.Handler
}

func NewAlipayAppCert() *AlipayAppCert {
	return &AlipayAppCert{Handler: admission.NewHandler(admission.Create)}
}

func (c *AlipayAppCert) Admit(a admission.Attributes) (err error) {
	return nil
}

// internal util functions
func fetchAppIdentity(pod *core.Pod) (appLocalKey string, err error) {

}
