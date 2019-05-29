/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	time "time"

	kubernetes "gitlab.alipay-inc.com/sigma/clientset/kubernetes"
	v1 "gitlab.alipay-inc.com/sigma/clientset/listers/machine/v1"
	machinev1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/machine/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
	clientgokubernetes "k8s.io/client-go/kubernetes"
	cache "k8s.io/client-go/tools/cache"
)

// MachinePackageVersionInformer provides access to a shared informer and lister for
// MachinePackageVersions.
type MachinePackageVersionInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.MachinePackageVersionLister
}

type machinePackageVersionInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewMachinePackageVersionInformer constructs a new informer for MachinePackageVersion type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewMachinePackageVersionInformer(client kubernetes.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredMachinePackageVersionInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredMachinePackageVersionInformer constructs a new informer for MachinePackageVersion type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredMachinePackageVersionInformer(client kubernetes.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MachineV1().MachinePackageVersions().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MachineV1().MachinePackageVersions().Watch(options)
			},
		},
		&machinev1.MachinePackageVersion{},
		resyncPeriod,
		indexers,
	)
}

func (f *machinePackageVersionInformer) defaultInformer(client clientgokubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredMachinePackageVersionInformer(client.(kubernetes.Interface), resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *machinePackageVersionInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&machinev1.MachinePackageVersion{}, f.defaultInformer)
}

func (f *machinePackageVersionInformer) Lister() v1.MachinePackageVersionLister {
	return v1.NewMachinePackageVersionLister(f.Informer().GetIndexer())
}
