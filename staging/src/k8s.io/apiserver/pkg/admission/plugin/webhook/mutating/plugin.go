/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mutating

import (
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/configuration"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/generic"
	"k8s.io/apiserver/pkg/util/feature"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancyconfiguration "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/admission/webhook/configuration"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
)

const (
	// Name of admission plug-in
	PluginName = "MutatingAdmissionWebhook"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(configFile io.Reader) (admission.Interface, error) {
		plugin, err := NewMutatingWebhook(configFile)
		if err != nil {
			return nil, err
		}

		return plugin, nil
	})
}

// Plugin is an implementation of admission.Interface.
type Plugin struct {
	*generic.Webhook

	scheme         *runtime.Scheme
	jsonSerializer *json.Serializer
}

var _ admission.MutationInterface = &Plugin{}

// NewMutatingWebhook returns a generic admission webhook plugin.
func NewMutatingWebhook(configFile io.Reader) (*Plugin, error) {
	handler := admission.NewHandler(admission.Connect, admission.Create, admission.Delete, admission.Update)
	p := &Plugin{}
	var err error
	if !feature.DefaultFeatureGate.Enabled(multitenancy.FeatureName) {
		p.Webhook, err = generic.NewMutatingWebhookWithObjectSelectorProxy(generic.NewWebhook, handler, configFile, configuration.NewMutatingWebhookConfigurationManager, newMutatingDispatcher(p))
	} else {
		p.Webhook, err = generic.NewMutatingWebhookWithObjectSelectorProxy(generic.NewWebhook, handler, configFile, multitenancyconfiguration.NewMutatingWebhookConfigurationManager, newMutatingDispatcher(p))
	}
	if err != nil {
		return nil, err
	}

	return p, nil
}

// SetScheme sets a serializer(NegotiatedSerializer) which is derived from the scheme
func (a *Plugin) SetScheme(scheme *runtime.Scheme) {
	a.Webhook.SetScheme(scheme)
	if scheme != nil {
		a.scheme = scheme
		a.jsonSerializer = json.NewSerializer(json.DefaultMetaFactory, scheme, scheme, false)
	}
}

// ValidateInitialization implements the InitializationValidator interface.
func (a *Plugin) ValidateInitialization() error {
	if err := a.Webhook.ValidateInitialization(); err != nil {
		return err
	}
	if a.scheme == nil {
		return fmt.Errorf("scheme is not properly setup")
	}
	if a.jsonSerializer == nil {
		return fmt.Errorf("jsonSerializer is not properly setup")
	}
	return nil
}

// Admit makes an admission decision based on the request attributes.
func (a *Plugin) Admit(attr admission.Attributes) error {
	if feature.DefaultFeatureGate.Enabled(multitenancy.FeatureName) {
		tenant, err := multitenancyutil.TransformTenantInfoFromUser(attr.GetUserInfo())
		if err != nil {
			return err
		}
		var aInterface interface{} = a
		a = aInterface.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(*Plugin)
	}
	return a.Webhook.Dispatch(attr)
}
