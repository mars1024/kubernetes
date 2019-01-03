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

// Code generated by lister-gen. DO NOT EDIT.

package internalversion

import (
	"fmt"
	"runtime/debug"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancycache "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/cache"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	autoscaling "k8s.io/kubernetes/pkg/apis/autoscaling"
)

// HorizontalPodAutoscalerLister helps list HorizontalPodAutoscalers.
type HorizontalPodAutoscalerLister interface {
	// List lists all HorizontalPodAutoscalers in the indexer.
	List(selector labels.Selector) (ret []*autoscaling.HorizontalPodAutoscaler, err error)
	// HorizontalPodAutoscalers returns an object that can list and get HorizontalPodAutoscalers.
	HorizontalPodAutoscalers(namespace string) HorizontalPodAutoscalerNamespaceLister
	HorizontalPodAutoscalerListerExpansion
}

type horizontalPodAutoscalerLister struct {
	indexer cache.Indexer
	tenant  multitenancy.TenantInfo
}

func (s *horizontalPodAutoscalerLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return &horizontalPodAutoscalerLister{
		indexer: s.indexer,
		tenant:  tenant,
	}
}

// NewHorizontalPodAutoscalerLister returns a new HorizontalPodAutoscalerLister.
func NewHorizontalPodAutoscalerLister(indexer cache.Indexer) HorizontalPodAutoscalerLister {
	return &horizontalPodAutoscalerLister{indexer: indexer}
}

// List lists all HorizontalPodAutoscalers in the indexer.
func (s *horizontalPodAutoscalerLister) List(selector labels.Selector) (ret []*autoscaling.HorizontalPodAutoscaler, err error) {
	if s.tenant != nil {
		err = multitenancycache.ListAllWithTenant(s.indexer, selector, s.tenant, func(m interface{}) {
			ret = append(ret, m.(*autoscaling.HorizontalPodAutoscaler))
		})
	} else {
		err = cache.ListAll(s.indexer, selector, func(m interface{}) {
			ret = append(ret, m.(*autoscaling.HorizontalPodAutoscaler))
		})
	}
	return ret, err
}

// HorizontalPodAutoscalers returns an object that can list and get HorizontalPodAutoscalers.
func (s *horizontalPodAutoscalerLister) HorizontalPodAutoscalers(namespace string) HorizontalPodAutoscalerNamespaceLister {
	return horizontalPodAutoscalerNamespaceLister{indexer: s.indexer, namespace: namespace, tenant: s.tenant}
}

// HorizontalPodAutoscalerNamespaceLister helps list and get HorizontalPodAutoscalers.
type HorizontalPodAutoscalerNamespaceLister interface {
	// List lists all HorizontalPodAutoscalers in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*autoscaling.HorizontalPodAutoscaler, err error)
	// Get retrieves the HorizontalPodAutoscaler from the indexer for a given namespace and name.
	Get(name string) (*autoscaling.HorizontalPodAutoscaler, error)
	HorizontalPodAutoscalerNamespaceListerExpansion
}

// horizontalPodAutoscalerNamespaceLister implements the HorizontalPodAutoscalerNamespaceLister
// interface.
type horizontalPodAutoscalerNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
	tenant    multitenancy.TenantInfo
}

// List lists all HorizontalPodAutoscalers in the indexer for a given namespace.
func (s horizontalPodAutoscalerNamespaceLister) List(selector labels.Selector) (ret []*autoscaling.HorizontalPodAutoscaler, err error) {
	if s.tenant == nil {
		debug.PrintStack()
		// fail hard so that we don't allow any namespaced lister w/o tenant
		return nil, fmt.Errorf("cannot namespaced list resources w/o specifying tenant")
	}
	err = multitenancycache.ListAllByNamespaceWithTenant(s.indexer, s.namespace, selector, s.tenant, func(m interface{}) {
		ret = append(ret, m.(*autoscaling.HorizontalPodAutoscaler))
	})

	return ret, err
}

// Get retrieves the HorizontalPodAutoscaler from the indexer for a given namespace and name.
func (s horizontalPodAutoscalerNamespaceLister) Get(name string) (*autoscaling.HorizontalPodAutoscaler, error) {
	if s.tenant == nil {
		debug.PrintStack()
		// fail hard so that we don't allow any namespaced lister w/o tenant
		return nil, fmt.Errorf("cannot namespaced get resources w/o specifying tenant")
	}
	obj, exists, err := s.indexer.GetByKey(multitenancyutil.TransformTenantInfoToJointString(s.tenant, "/") + "/" + s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(autoscaling.Resource("horizontalpodautoscaler"), name)
	}
	return obj.(*autoscaling.HorizontalPodAutoscaler), nil
}
