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

package v1

import (
	"fmt"
	"runtime/debug"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancycache "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/cache"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	v1 "k8s.io/apiextensions-apiserver/examples/client-go/pkg/apis/cr/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ExampleLister helps list Examples.
type ExampleLister interface {
	// List lists all Examples in the indexer.
	List(selector labels.Selector) (ret []*v1.Example, err error)
	// Examples returns an object that can list and get Examples.
	Examples(namespace string) ExampleNamespaceLister
	ExampleListerExpansion
}

type exampleLister struct {
	indexer cache.Indexer
	tenant  multitenancy.TenantInfo
}

func (s *exampleLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return &exampleLister{
		indexer: s.indexer,
		tenant:  tenant,
	}
}

// NewExampleLister returns a new ExampleLister.
func NewExampleLister(indexer cache.Indexer) ExampleLister {
	return &exampleLister{indexer: indexer}
}

// List lists all Examples in the indexer.
func (s *exampleLister) List(selector labels.Selector) (ret []*v1.Example, err error) {
	if s.tenant != nil {
		err = multitenancycache.ListAllWithTenant(s.indexer, selector, s.tenant, func(m interface{}) {
			ret = append(ret, m.(*v1.Example))
		})
	} else {
		err = cache.ListAll(s.indexer, selector, func(m interface{}) {
			ret = append(ret, m.(*v1.Example))
		})
	}
	return ret, err
}

// Examples returns an object that can list and get Examples.
func (s *exampleLister) Examples(namespace string) ExampleNamespaceLister {
	return exampleNamespaceLister{indexer: s.indexer, namespace: namespace, tenant: s.tenant}
}

// ExampleNamespaceLister helps list and get Examples.
type ExampleNamespaceLister interface {
	// List lists all Examples in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1.Example, err error)
	// Get retrieves the Example from the indexer for a given namespace and name.
	Get(name string) (*v1.Example, error)
	ExampleNamespaceListerExpansion
}

// exampleNamespaceLister implements the ExampleNamespaceLister
// interface.
type exampleNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
	tenant    multitenancy.TenantInfo
}

// List lists all Examples in the indexer for a given namespace.
func (s exampleNamespaceLister) List(selector labels.Selector) (ret []*v1.Example, err error) {
	if s.tenant == nil {
		debug.PrintStack()
		// fail hard so that we don't allow any namespaced lister w/o tenant
		return nil, fmt.Errorf("cannot namespaced list resources w/o specifying tenant")
	}
	err = multitenancycache.ListAllByNamespaceWithTenant(s.indexer, s.namespace, selector, s.tenant, func(m interface{}) {
		ret = append(ret, m.(*v1.Example))
	})

	return ret, err
}

// Get retrieves the Example from the indexer for a given namespace and name.
func (s exampleNamespaceLister) Get(name string) (*v1.Example, error) {
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
		return nil, errors.NewNotFound(v1.Resource("example"), name)
	}
	return obj.(*v1.Example), nil
}
