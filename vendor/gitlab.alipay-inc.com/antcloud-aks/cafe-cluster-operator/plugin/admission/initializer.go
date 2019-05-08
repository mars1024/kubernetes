/*
Copyright 2019 The Kubernetes Authors.

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

package admission

import (
	"k8s.io/client-go/rest"
	"k8s.io/apiserver/pkg/admission"

	informers "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/informers_generated/externalversions"
)

// WantsRESTClientConfig gives access to a RESTClientConfig.  It's useful for doing unusual things with transports.
type WantsRESTClientConfig interface {
	SetRESTClientConfig(rest.Config)
	admission.InitializationValidator
}

// WantsCafeClusterOperatorKubeInformerFactory defines a function which sets CafeCusterOperator InformerFactory for admission plugins that need it
type WantsCafeClusterOperatorKubeInformerFactory interface {
	SetCafeClusterOperatorKubeInformerFactory(informers.SharedInformerFactory)
	admission.InitializationValidator
}

// PluginInitializer is used for initialization of the Kubernetes specific admission plugins.
type PluginInitializer struct {
	restClientConfig rest.Config
	informers        informers.SharedInformerFactory
}

var _ admission.PluginInitializer = &PluginInitializer{}

func NewPluginInitializer(
	restClientConfig rest.Config,
	sharedInformers informers.SharedInformerFactory,
) *PluginInitializer {
	return &PluginInitializer{
		restClientConfig: restClientConfig,
		informers:        sharedInformers,
	}
}

// Initialize checks the initialization interfaces implemented by each plugin
// and provide the appropriate initialization data
func (i *PluginInitializer) Initialize(plugin admission.Interface) {
	if wants, ok := plugin.(WantsRESTClientConfig); ok {
		wants.SetRESTClientConfig(i.restClientConfig)
	}

	if wants, ok := plugin.(WantsCafeClusterOperatorKubeInformerFactory); ok {
		wants.SetCafeClusterOperatorKubeInformerFactory(i.informers)
	}
}
