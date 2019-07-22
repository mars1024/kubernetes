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

package validating

import (
	"io"

	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/configuration"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/generic"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"k8s.io/apiserver/pkg/util/feature"
	multitenancyconfiguration "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/admission/webhook/configuration"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
)

const (
	// Name of admission plug-in
	PluginName = "ValidatingAdmissionWebhook"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(configFile io.Reader) (admission.Interface, error) {
		plugin, err := NewValidatingAdmissionWebhook(configFile)
		if err != nil {
			return nil, err
		}

		return plugin, nil
	})
}

// Plugin is an implementation of admission.Interface.
type Plugin struct {
	*generic.Webhook
}

var _ admission.ValidationInterface = &Plugin{}

// NewValidatingAdmissionWebhook returns a generic admission webhook plugin.
func NewValidatingAdmissionWebhook(configFile io.Reader) (*Plugin, error) {
	handler := admission.NewHandler(admission.Connect, admission.Create, admission.Delete, admission.Update)
	var err error
	var webhook *generic.Webhook
	if !feature.DefaultFeatureGate.Enabled(multitenancy.FeatureName) {
		webhook, err = generic.NewWebhookWithObjectSelectorProxy(generic.NewWebhook, handler, configFile, configuration.NewMutatingWebhookConfigurationManager, newValidatingDispatcher)
	} else {
		webhook, err = generic.NewWebhookWithObjectSelectorProxy(generic.NewWebhook, handler, configFile, multitenancyconfiguration.NewMutatingWebhookConfigurationManager, newValidatingDispatcher)
	}
	if err != nil {
		return nil, err
	}
	return &Plugin{webhook}, nil
}

// Validate makes an admission decision based on the request attributes.
func (a *Plugin) Validate(attr admission.Attributes) error {
	if feature.DefaultFeatureGate.Enabled(multitenancy.FeatureName) {
		tenant, err := multitenancyutil.TransformTenantInfoFromUser(attr.GetUserInfo())
		if err != nil {
			return err
		}
		var aInterface interface{} = a.Webhook
		a.Webhook = aInterface.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(*generic.Webhook)
	}
	return a.Webhook.Dispatch(attr)
}
