/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	time "time"

	kubernetes "gitlab.alipay-inc.com/sigma/clientset/kubernetes"
	v1 "gitlab.alipay-inc.com/sigma/clientset/listers/network/v1"
	networkv1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/network/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
	clientgokubernetes "k8s.io/client-go/kubernetes"
	cache "k8s.io/client-go/tools/cache"
)

// SecurityGroupInformer provides access to a shared informer and lister for
// SecurityGroups.
type SecurityGroupInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.SecurityGroupLister
}

type securityGroupInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewSecurityGroupInformer constructs a new informer for SecurityGroup type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewSecurityGroupInformer(client kubernetes.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredSecurityGroupInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredSecurityGroupInformer constructs a new informer for SecurityGroup type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredSecurityGroupInformer(client kubernetes.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NetworkV1().SecurityGroups().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NetworkV1().SecurityGroups().Watch(options)
			},
		},
		&networkv1.SecurityGroup{},
		resyncPeriod,
		indexers,
	)
}

func (f *securityGroupInformer) defaultInformer(client clientgokubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredSecurityGroupInformer(client.(kubernetes.Interface), resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *securityGroupInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&networkv1.SecurityGroup{}, f.defaultInformer)
}

func (f *securityGroupInformer) Lister() v1.SecurityGroupLister {
	return v1.NewSecurityGroupLister(f.Informer().GetIndexer())
}
