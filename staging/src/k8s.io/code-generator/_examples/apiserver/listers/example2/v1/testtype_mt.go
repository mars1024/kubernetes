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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1 "k8s.io/code-generator/_examples/apiserver/apis/example2/v1"
)

// TestTypeLister helps list TestTypes.
type TestTypeLister interface {
	// List lists all TestTypes in the indexer.
	List(selector labels.Selector) (ret []*v1.TestType, err error)
	// TestTypes returns an object that can list and get TestTypes.
	TestTypes(namespace string) TestTypeNamespaceLister
	TestTypeListerExpansion
}

type testTypeLister struct {
	indexer cache.Indexer
	tenant  multitenancy.TenantInfo
}

func (s *testTypeLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return &testTypeLister{
		indexer: s.indexer,
		tenant:  tenant,
	}
}

// NewTestTypeLister returns a new TestTypeLister.
func NewTestTypeLister(indexer cache.Indexer) TestTypeLister {
	return &testTypeLister{indexer: indexer}
}

// List lists all TestTypes in the indexer.
func (s *testTypeLister) List(selector labels.Selector) (ret []*v1.TestType, err error) {
	if s.tenant != nil {
		err = multitenancycache.ListAllWithTenant(s.indexer, selector, s.tenant, func(m interface{}) {
			ret = append(ret, m.(*v1.TestType))
		})
	} else {
		err = cache.ListAll(s.indexer, selector, func(m interface{}) {
			ret = append(ret, m.(*v1.TestType))
		})
	}
	return ret, err
}

// TestTypes returns an object that can list and get TestTypes.
func (s *testTypeLister) TestTypes(namespace string) TestTypeNamespaceLister {
	return testTypeNamespaceLister{indexer: s.indexer, namespace: namespace, tenant: s.tenant}
}

// TestTypeNamespaceLister helps list and get TestTypes.
type TestTypeNamespaceLister interface {
	// List lists all TestTypes in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1.TestType, err error)
	// Get retrieves the TestType from the indexer for a given namespace and name.
	Get(name string) (*v1.TestType, error)
	TestTypeNamespaceListerExpansion
}

// testTypeNamespaceLister implements the TestTypeNamespaceLister
// interface.
type testTypeNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
	tenant    multitenancy.TenantInfo
}

// List lists all TestTypes in the indexer for a given namespace.
func (s testTypeNamespaceLister) List(selector labels.Selector) (ret []*v1.TestType, err error) {
	if s.tenant == nil {
		debug.PrintStack()
		// fail hard so that we don't allow any namespaced lister w/o tenant
		return nil, fmt.Errorf("cannot namespaced list resources w/o specifying tenant")
	}
	err = multitenancycache.ListAllByNamespaceWithTenant(s.indexer, s.namespace, selector, s.tenant, func(m interface{}) {
		ret = append(ret, m.(*v1.TestType))
	})

	return ret, err
}

// Get retrieves the TestType from the indexer for a given namespace and name.
func (s testTypeNamespaceLister) Get(name string) (*v1.TestType, error) {
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
		return nil, errors.NewNotFound(v1.Resource("testtype"), name)
	}
	return obj.(*v1.TestType), nil
}
