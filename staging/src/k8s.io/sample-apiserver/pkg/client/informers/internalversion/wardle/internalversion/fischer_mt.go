// +build multitenancy

/*
Copyright The Kubernetes Authors.

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

// Code generated by informer-gen. DO NOT EDIT.

package internalversion

import (
	time "time"

	multitenancycache "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/cache"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	wardle "k8s.io/sample-apiserver/pkg/apis/wardle"
	clientsetinternalversion "k8s.io/sample-apiserver/pkg/client/clientset/internalversion"
	internalinterfaces "k8s.io/sample-apiserver/pkg/client/informers/internalversion/internalinterfaces"
	internalversion "k8s.io/sample-apiserver/pkg/client/listers/wardle/internalversion"
)

// FischerInformer provides access to a shared informer and lister for
// Fischers.
type FischerInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() internalversion.FischerLister
}

type fischerInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewFischerInformer constructs a new informer for Fischer type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFischerInformer(client clientsetinternalversion.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredFischerInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredFischerInformer constructs a new informer for Fischer type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredFischerInformer(client clientsetinternalversion.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Wardle().Fischers().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Wardle().Fischers().Watch(options)
			},
		},
		&wardle.Fischer{},
		resyncPeriod,
		indexers,
	)
}

func (f *fischerInformer) defaultInformer(client clientsetinternalversion.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredFischerInformer(client, resyncPeriod, cache.Indexers{
		multitenancycache.TenantIndex:          multitenancycache.MetaTenantIndexFunc,
		multitenancycache.TenantNamespaceIndex: multitenancycache.MetaTenantNamespaceIndexFunc,
	}, f.tweakListOptions)
}

func (f *fischerInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&wardle.Fischer{}, f.defaultInformer)
}

func (f *fischerInformer) Lister() internalversion.FischerLister {
	return internalversion.NewFischerLister(f.Informer().GetIndexer())
}
