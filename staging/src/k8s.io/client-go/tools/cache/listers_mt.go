//+build multitenancy

/*
Copyright 2014 The Kubernetes Authors.

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

package cache

import (
	"fmt"
	"runtime/debug"

	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// AppendFunc is used to add a matching item to whatever list the caller is using
type AppendFunc func(interface{})

func ListAll(store Store, selector labels.Selector, appendFn AppendFunc) error {
	for _, m := range store.List() {
		metadata, err := meta.Accessor(m)
		if err != nil {
			return err
		}
		if selector.Matches(labels.Set(metadata.GetLabels())) {
			appendFn(m)
		}
	}
	return nil
}

func ListAllByNamespace(indexer Indexer, namespace string, selector labels.Selector, appendFn AppendFunc) error {
	if namespace == metav1.NamespaceAll {
		for _, m := range indexer.List() {
			metadata, err := meta.Accessor(m)
			if err != nil {
				return err
			}
			if selector.Matches(labels.Set(metadata.GetLabels())) {
				appendFn(m)
			}
		}
		return nil
	}

	items, err := indexer.Index(NamespaceIndex, &metav1.ObjectMeta{Namespace: namespace})
	if err != nil {
		// Ignore error; do slow search without index.
		glog.Warningf("can not retrieve list of objects using index : %v", err)
		for _, m := range indexer.List() {
			metadata, err := meta.Accessor(m)
			if err != nil {
				return err
			}
			if metadata.GetNamespace() == namespace && selector.Matches(labels.Set(metadata.GetLabels())) {
				appendFn(m)
			}

		}
		return nil
	}
	for _, m := range items {
		metadata, err := meta.Accessor(m)
		if err != nil {
			return err
		}
		if selector.Matches(labels.Set(metadata.GetLabels())) {
			appendFn(m)
		}
	}

	return nil
}

// GenericLister is a lister skin on a generic Indexer
type GenericLister interface {
	// List will return all objects across namespaces
	List(selector labels.Selector) (ret []runtime.Object, err error)
	// Get will attempt to retrieve assuming that name==key
	Get(name string) (runtime.Object, error)
	// ByNamespace will give you a GenericNamespaceLister for one namespace
	ByNamespace(namespace string) GenericNamespaceLister
}

// GenericNamespaceLister is a lister skin on a generic Indexer
type GenericNamespaceLister interface {
	// List will return all objects in this namespace
	List(selector labels.Selector) (ret []runtime.Object, err error)
	// Get will attempt to retrieve by namespace and name
	Get(name string) (runtime.Object, error)
}

func NewGenericLister(indexer Indexer, resource schema.GroupResource) GenericLister {
	return &genericLister{indexer: indexer, resource: resource}
}

type genericLister struct {
	indexer  Indexer
	resource schema.GroupResource
	tenant   multitenancy.TenantInfo
}

func (s *genericLister) List(selector labels.Selector) (ret []runtime.Object, err error) {
	if s.tenant != nil {
		err = internalListAllWithTenant(s.indexer, selector, s.tenant, func(m interface{}) {
			ret = append(ret, m.(runtime.Object))
		})
	} else {
		err = ListAll(s.indexer, selector, func(m interface{}) {
			ret = append(ret, m.(runtime.Object))
		})
	}
	return ret, err
}

func (s *genericLister) ByNamespace(namespace string) GenericNamespaceLister {
	return &genericNamespaceLister{indexer: s.indexer, namespace: namespace, resource: s.resource}
}

func (s *genericLister) Get(name string) (runtime.Object, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(s.resource, name)
	}
	return obj.(runtime.Object), nil
}

type genericNamespaceLister struct {
	indexer   Indexer
	namespace string
	resource  schema.GroupResource
	tenant    multitenancy.TenantInfo
}

func (s *genericNamespaceLister) List(selector labels.Selector) (ret []runtime.Object, err error) {
	if s.tenant == nil {
		debug.PrintStack()
		// fail hard so that we don't allow any namespaced lister w/o tenant
		return nil, fmt.Errorf("cannot namespaced list resources w/o specifying tenant")
	}
	err = internalListAllByNamespaceWithTenant(s.indexer, s.namespace, selector, s.tenant, func(m interface{}) {
		ret = append(ret, m.(runtime.Object))
	})
	return ret, err
}

func (s *genericNamespaceLister) Get(name string) (runtime.Object, error) {
	if s.tenant == nil {
		// fail hard so that we don't allow any cluster-scoped get w/o tenant
		debug.PrintStack()
		return nil, fmt.Errorf("cannot get initializerconfiguration w/o specifying tenant")
	}

	obj, exists, err := s.indexer.GetByKey(multitenancyutil.TransformTenantInfoToJointString(s.tenant, "/") + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(s.resource, name)
	}
	return obj.(runtime.Object), nil
}

// multitenancy
func (s *genericLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *s
	copied.tenant = tenant
	return &copied
}

func (s *genericNamespaceLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *s
	copied.tenant = tenant
	return &copied
}

// HACK(zuoxiu.jm): copied the following internal type/function for resolving the cyclic dependencies
const (
	internalTenantIndex          string = "tenant"
	internalTenantNamespaceIndex string = "tenant_namespace"
)

func internalListAllWithTenant(index Indexer, selector labels.Selector, tenant multitenancy.TenantInfo, appendFunc AppendFunc) error {
	items, err := index.Index(internalTenantIndex, &metav1.ObjectMeta{Annotations: multitenancyutil.TransformTenantInfoToAnnotations(tenant)})
	if err != nil {
		return err
	}
	for _, m := range items {
		metadata, err := meta.Accessor(m)
		if err != nil {
			return err
		}
		if selector.Matches(labels.Set(metadata.GetLabels())) {
			appendFunc(m)
		}
	}
	return nil
}

func internalListAllByNamespaceWithTenant(index Indexer, namespace string, selector labels.Selector, tenant multitenancy.TenantInfo, appendFunc AppendFunc) error {
	items, err := index.Index(internalTenantNamespaceIndex, &metav1.ObjectMeta{
		Namespace:   namespace,
		Annotations: multitenancyutil.TransformTenantInfoToAnnotations(tenant),
	})
	if err != nil {
		return err
	}
	for _, m := range items {
		metadata, err := meta.Accessor(m)
		if err != nil {
			return err
		}
		if selector.Matches(labels.Set(metadata.GetLabels())) {
			appendFunc(m)
		}
	}
	return nil
}
