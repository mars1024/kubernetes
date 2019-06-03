/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	time "time"

	kubernetes "gitlab.alipay-inc.com/sigma/clientset/kubernetes"
	v1 "gitlab.alipay-inc.com/sigma/clientset/listers/schedulingextensions/v1"
	schedulingextensionsv1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/schedulingextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
	clientgokubernetes "k8s.io/client-go/kubernetes"
	cache "k8s.io/client-go/tools/cache"
)

// PodGroupInformer provides access to a shared informer and lister for
// PodGroups.
type PodGroupInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.PodGroupLister
}

type podGroupInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewPodGroupInformer constructs a new informer for PodGroup type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewPodGroupInformer(client kubernetes.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredPodGroupInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredPodGroupInformer constructs a new informer for PodGroup type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredPodGroupInformer(client kubernetes.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.SchedulingextensionsV1().PodGroups(namespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.SchedulingextensionsV1().PodGroups(namespace).Watch(options)
			},
		},
		&schedulingextensionsv1.PodGroup{},
		resyncPeriod,
		indexers,
	)
}

func (f *podGroupInformer) defaultInformer(client clientgokubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredPodGroupInformer(client.(kubernetes.Interface), f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *podGroupInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&schedulingextensionsv1.PodGroup{}, f.defaultInformer)
}

func (f *podGroupInformer) Lister() v1.PodGroupLister {
	return v1.NewPodGroupLister(f.Informer().GetIndexer())
}
