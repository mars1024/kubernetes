/*
Copyright 2019 The Alipay.com Inc Authors.

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

package clusterquotamapping

import (
	"fmt"
	"time"

	"k8s.io/client-go/util/workqueue"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	kapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1informers "k8s.io/client-go/informers/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"

	listers "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/listers_generated/cluster/v1alpha1"
	cafeinformers "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/informers_generated/externalversions/cluster/v1alpha1"
	clusterapi "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	clustercache "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/cache"
	quotahelper "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/quota"

	"github.com/golang/glog"
)

func NewClusterResourceQuotaMappingController(namespaceInformer corev1informers.NamespaceInformer, quotaInformer cafeinformers.ClusterResourceQuotaInformer) *ClusterResourceQuotaMappingController {
	c := newClusterResourceQuotaMappingController(namespaceInformer.Informer(), quotaInformer)
	c.namespaceLister = v1NamespaceLister{lister: namespaceInformer.Lister()}
	return c
}

type namespaceLister interface {
	Each(label labels.Selector, fn func(metav1.Object) bool) error
	Get(name string) (metav1.Object, error)
}

type v1NamespaceLister struct {
	lister corev1listers.NamespaceLister
}

func (l v1NamespaceLister) Each(label labels.Selector, fn func(metav1.Object) bool) error {
	results, err := l.lister.List(label)
	if err != nil {
		return err
	}
	for i := range results {
		if !fn(results[i]) {
			return nil
		}
	}
	return nil
}
func (l v1NamespaceLister) Get(name string) (metav1.Object, error) {
	return l.lister.Get(name)
}

func newClusterResourceQuotaMappingController(namespaceInformer cache.SharedIndexInformer, quotaInformer cafeinformers.ClusterResourceQuotaInformer) *ClusterResourceQuotaMappingController {
	c := &ClusterResourceQuotaMappingController{
		namespaceQueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "controller_clusterquotamappingcontroller_namespaces"),
		quotaQueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "controller_clusterquotamappingcontroller_clusterquotas"),
		clusterQuotaMapper: NewClusterQuotaMapper(),
	}
	namespaceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addNamespace,
		UpdateFunc: c.updateNamespace,
		DeleteFunc: c.deleteNamespace,
	})
	c.namespacesSynced = namespaceInformer.HasSynced

	quotaInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addQuota,
		UpdateFunc: c.updateQuota,
		DeleteFunc: c.deleteQuota,
	})
	c.quotaLister = quotaInformer.Lister()
	c.quotasSynced = quotaInformer.Informer().HasSynced

	return c
}

type ClusterResourceQuotaMappingController struct {
	namespaceQueue   workqueue.RateLimitingInterface
	namespaceLister  namespaceLister
	namespacesSynced func() bool

	quotaQueue   workqueue.RateLimitingInterface
	quotaLister  listers.ClusterResourceQuotaLister
	quotasSynced func() bool

	clusterQuotaMapper *clusterQuotaMapper
}

func (c *ClusterResourceQuotaMappingController) GetClusterQuotaMapper() ClusterQuotaMapper {
	return c.clusterQuotaMapper
}

func (c *ClusterResourceQuotaMappingController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.namespaceQueue.ShutDown()
	defer c.quotaQueue.ShutDown()

	glog.Infof("Starting ClusterQuotaMappingController controller")
	defer glog.Infof("Shutting down ClusterQuotaMappingController controller")

	if !cache.WaitForCacheSync(stopCh, c.namespacesSynced, c.quotasSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	glog.Infof("Starting workers for quota mapping controller workers")
	for i := 0; i < workers; i++ {
		go wait.Until(c.namespaceWorker, time.Second, stopCh)
		go wait.Until(c.quotaWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *ClusterResourceQuotaMappingController) syncQuota(quota *clusterapi.ClusterResourceQuota) error {
	matcherFunc, err := quotahelper.GetObjectMatcher(quota.Spec.Selector)
	if err != nil {
		return err
	}

	if err := c.namespaceLister.Each(labels.Everything(), func(obj metav1.Object) bool {
		// attempt to set the mapping. The quotas never collide with each other (same quota is never processed twice in parallel)
		// so this means that the project we have is out of date, pull a more recent copy from the cache and retest
		for {
			matches, err := matcherFunc(obj)
			if err != nil {
				utilruntime.HandleError(err)
				break
			}
			if matches {
				success, quotaMatches, _ := c.clusterQuotaMapper.setMapping(quota, obj)
				if success {
					break
				}

				// if the quota is mismatched, then someone has updated the quota or has deleted the entry entirely.
				// if we've been updated, we'll be rekicked, if we've been deleted we should stop.  Either way, this
				// execution is finished
				if !quotaMatches {
					return false
				}
				fullName := getNamespaceFullName(obj)
				newer, err := c.namespaceLister.Get(fullName)
				if kapierrors.IsNotFound(err) {
					// if the namespace is gone, then the deleteNamespace path will be called, just continue
					break
				}
				if err != nil {
					utilruntime.HandleError(err)
					break
				}
				obj = newer
			}
		}
		return true
	}); err != nil {
		return err
	}

	c.clusterQuotaMapper.completeQuota(quota)
	return nil
}

func (c *ClusterResourceQuotaMappingController) syncNamespace(namespace metav1.Object) error {
	allQuotas, err := c.quotaLister.List(labels.Everything())
	if err != nil {
		return err
	}

	for i := range allQuotas {
		quota := allQuotas[i]

		for {
			matcherFunc, err := quotahelper.GetObjectMatcher(quota.Spec.Selector)
			if err != nil {
				utilruntime.HandleError(err)
				break
			}

			// attempt to set the mapping. The namespaces never collide with each other (same namespace is never processed twice in parallel)
			// so this means that the quota we have is out of date, pull a more recent copy from the cache and retest
			matches, err := matcherFunc(namespace)
			if err != nil {
				utilruntime.HandleError(err)
				break
			}
			if matches {
				success, _, namespaceMatches := c.clusterQuotaMapper.setMapping(quota, namespace)
				if success {
					return nil
				}

				// if the namespace is mismatched, then someone has updated the namespace or has deleted the entry entirely.
				// if we've been updated, we'll be rekicked, if we've been deleted we should stop.  Either way, this
				// execution is finished
				if !namespaceMatches {
					return nil
				}

				quota, err = c.quotaLister.Get(quota.Name)
				if kapierrors.IsNotFound(err) {
					// if the quota is gone, then the deleteQuota path will be called, just continue
					break
				}
				if err != nil {
					utilruntime.HandleError(err)
					break
				}
			}
		}
	}

	c.clusterQuotaMapper.completeNamespace(namespace)
	return nil
}

func (c *ClusterResourceQuotaMappingController) namespaceWork() bool {
	key, quit := c.namespaceQueue.Get()
	if quit {
		return true
	}
	defer c.namespaceQueue.Done(key)

	namespace, err := c.namespaceLister.Get(key.(string))
	if kapierrors.IsNotFound(err) {
		c.namespaceQueue.Forget(key)
		return false
	}
	if err != nil {
		utilruntime.HandleError(err)
		return false
	}

	err = c.syncNamespace(namespace)
	outOfRetries := c.namespaceQueue.NumRequeues(key) > 5
	switch {
	case err != nil && outOfRetries:
		utilruntime.HandleError(err)
		c.namespaceQueue.Forget(key)

	case err != nil && !outOfRetries:
		c.namespaceQueue.AddRateLimited(key)

	default:
		c.namespaceQueue.Forget(key)
	}

	return false
}

func (c *ClusterResourceQuotaMappingController) namespaceWorker() {
	for {
		if quit := c.namespaceWork(); quit {
			return
		}
	}
}

func (c *ClusterResourceQuotaMappingController) addNamespace(cur interface{}) {
	c.enqueueNamespace(cur)
}

func (c *ClusterResourceQuotaMappingController) updateNamespace(old, cur interface{}) {
	c.enqueueNamespace(cur)
}

func (c *ClusterResourceQuotaMappingController) deleteNamespace(obj interface{}) {
	var namespaceToDelete *v1.Namespace
	switch ns := obj.(type) {
	case cache.DeletedFinalStateUnknown:
		switch nested := ns.Obj.(type) {
		case *v1.Namespace:
			namespaceToDelete = nested
		default:
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a Namespace %T", ns.Obj))
			return
		}
	case *v1.Namespace:
		namespaceToDelete = obj.(*v1.Namespace)
	default:
		utilruntime.HandleError(fmt.Errorf("not a Namespace %v", obj))
		return
	}
	fullName := getNamespaceFullName(namespaceToDelete)
	c.clusterQuotaMapper.removeNamespace(fullName)
}

func (c *ClusterResourceQuotaMappingController) enqueueNamespace(obj interface{}) {
	ns, ok := obj.(*v1.Namespace)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("not a Quota %v", obj))
		return
	}
	if !c.clusterQuotaMapper.requireNamespace(ns) {
		return
	}

	tenantWrappedKeyFunc := clustercache.MultiTenancyKeyFuncWrapper(cache.DeletionHandlingMetaNamespaceKeyFunc)
	key, err := tenantWrappedKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.namespaceQueue.Add(key)
}

func (c *ClusterResourceQuotaMappingController) quotaWork() bool {
	key, quit := c.quotaQueue.Get()
	if quit {
		return true
	}
	defer c.quotaQueue.Done(key)

	quota, err := c.quotaLister.Get(key.(string))
	if err != nil {
		if kapierrors.IsNotFound(err) {
			c.quotaQueue.Forget(key)
			return false
		}
		utilruntime.HandleError(err)
		return false
	}

	err = c.syncQuota(quota)
	outOfRetries := c.quotaQueue.NumRequeues(key) > 5
	switch {
	case err != nil && outOfRetries:
		utilruntime.HandleError(err)
		c.quotaQueue.Forget(key)

	case err != nil && !outOfRetries:
		c.quotaQueue.AddRateLimited(key)

	default:
		c.quotaQueue.Forget(key)
	}

	return false
}

func (c *ClusterResourceQuotaMappingController) quotaWorker() {
	for {
		if quit := c.quotaWork(); quit {
			return
		}
	}
}

func (c *ClusterResourceQuotaMappingController) addQuota(cur interface{}) {
	c.enqueueQuota(cur)
}

func (c *ClusterResourceQuotaMappingController) updateQuota(old, cur interface{}) {
	c.enqueueQuota(cur)
}

func (c *ClusterResourceQuotaMappingController) deleteQuota(obj interface{}) {
	quota, ok1 := obj.(*clusterapi.ClusterResourceQuota)
	if !ok1 {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %v", obj))
			return
		}
		quota, ok = tombstone.Obj.(*clusterapi.ClusterResourceQuota)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a Quota %v", obj))
			return
		}
	}

	c.clusterQuotaMapper.removeQuota(quota.Name)
}

func (c *ClusterResourceQuotaMappingController) enqueueQuota(obj interface{}) {
	quota, ok := obj.(*clusterapi.ClusterResourceQuota)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("not a Quota %v", obj))
		return
	}
	if !c.clusterQuotaMapper.requireQuota(quota) {
		return
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.quotaQueue.Add(key)
}

func getNamespaceFullName(ns metav1.Object) string {
	tenantInfo, err := clustercache.TransformTenantInfoFromAnnotations(ns.GetAnnotations())
	if err == nil {
		return clustercache.TransformTenantInfoToJointString(tenantInfo, "/") + "/" + ns.GetName()
	}
	return ns.GetName()
}
