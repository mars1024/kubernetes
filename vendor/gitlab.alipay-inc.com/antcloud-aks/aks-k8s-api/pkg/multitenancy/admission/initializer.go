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
	"k8s.io/apiserver/pkg/admission"
	informers "gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/client/informers_generated/externalversions"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/client/clientset_generated/clientset"
	"k8s.io/apiserver/pkg/server/storage"
)

// WantsCafeExtensionKubeClientSet defines a function which sets CafeK8SExtension ClientSet for admission plugins that need it
type WantsCafeExtensionKubeClientSet interface {
	SetCafeExtensionKubeClientSet(clientset.Interface)
	admission.InitializationValidator
}

// WantsCafeExtensionKubeInformerFactory defines a function which sets CafeK8SExtension InformerFactory for admission plugins that need it
type WantsCafeExtensionKubeInformerFactory interface {
	SetCafeExtensionKubeInformerFactory(informers.SharedInformerFactory)
	admission.InitializationValidator
}

type WantsStorageFactory interface {
	SetStorageFactory(factory storage.StorageFactory)
	admission.InitializationValidator
}

// PluginInitializer is used for initialization of the Kubernetes specific admission plugins.
type PluginInitializer struct {
	storageFactory storage.StorageFactory
	client         clientset.Interface
	informers      informers.SharedInformerFactory
}

var _ admission.PluginInitializer = &PluginInitializer{}

func NewPluginInitializer(
	client clientset.Interface,
	sharedInformers informers.SharedInformerFactory,
	storageFactory storage.StorageFactory,
) *PluginInitializer {
	return &PluginInitializer{
		client:         client,
		informers:      sharedInformers,
		storageFactory: storageFactory,
	}
}

// Initialize checks the initialization interfaces implemented by each plugin
// and provide the appropriate initialization data
func (i *PluginInitializer) Initialize(plugin admission.Interface) {
	if wants, ok := plugin.(WantsCafeExtensionKubeClientSet); ok {
		wants.SetCafeExtensionKubeClientSet(i.client)
	}

	if wants, ok := plugin.(WantsCafeExtensionKubeInformerFactory); ok {
		wants.SetCafeExtensionKubeInformerFactory(i.informers)
	}

	if wants, ok := plugin.(WantsStorageFactory); ok {
		wants.SetStorageFactory(i.storageFactory)
	}
}
