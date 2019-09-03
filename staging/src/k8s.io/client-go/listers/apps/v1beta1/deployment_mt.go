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

package v1beta1

import (
	"fmt"
	"runtime/debug"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancycache "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/cache"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	v1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// DeploymentLister helps list Deployments.
type DeploymentLister interface {
	// List lists all Deployments in the indexer.
	List(selector labels.Selector) (ret []*v1beta1.Deployment, err error)
	// Deployments returns an object that can list and get Deployments.
	Deployments(namespace string) DeploymentNamespaceLister
	DeploymentListerExpansion
}

type deploymentLister struct {
	indexer cache.Indexer
	tenant  multitenancy.TenantInfo
}

func (s *deploymentLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return &deploymentLister{
		indexer: s.indexer,
		tenant:  tenant,
	}
}

// NewDeploymentLister returns a new DeploymentLister.
func NewDeploymentLister(indexer cache.Indexer) DeploymentLister {
	return &deploymentLister{indexer: indexer}
}

// List lists all Deployments in the indexer.
func (s *deploymentLister) List(selector labels.Selector) (ret []*v1beta1.Deployment, err error) {
	if s.tenant != nil {
		err = multitenancycache.ListAllWithTenant(s.indexer, selector, s.tenant, func(m interface{}) {
			ret = append(ret, m.(*v1beta1.Deployment))
		})
	} else {
		err = cache.ListAll(s.indexer, selector, func(m interface{}) {
			ret = append(ret, m.(*v1beta1.Deployment))
		})
	}
	return ret, err
}

// Deployments returns an object that can list and get Deployments.
func (s *deploymentLister) Deployments(namespace string) DeploymentNamespaceLister {
	return deploymentNamespaceLister{indexer: s.indexer, namespace: namespace, tenant: s.tenant}
}

// DeploymentNamespaceLister helps list and get Deployments.
type DeploymentNamespaceLister interface {
	// List lists all Deployments in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1beta1.Deployment, err error)
	// Get retrieves the Deployment from the indexer for a given namespace and name.
	Get(name string) (*v1beta1.Deployment, error)
	DeploymentNamespaceListerExpansion
}

// deploymentNamespaceLister implements the DeploymentNamespaceLister
// interface.
type deploymentNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
	tenant    multitenancy.TenantInfo
}

// List lists all Deployments in the indexer for a given namespace.
func (s deploymentNamespaceLister) List(selector labels.Selector) (ret []*v1beta1.Deployment, err error) {
	if s.tenant == nil {
		debug.PrintStack()
		// fail hard so that we don't allow any namespaced lister w/o tenant
		return nil, fmt.Errorf("cannot namespaced list resources w/o specifying tenant")
	}
	err = multitenancycache.ListAllByNamespaceWithTenant(s.indexer, s.namespace, selector, s.tenant, func(m interface{}) {
		ret = append(ret, m.(*v1beta1.Deployment))
	})

	return ret, err
}

// Get retrieves the Deployment from the indexer for a given namespace and name.
func (s deploymentNamespaceLister) Get(name string) (*v1beta1.Deployment, error) {
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
		return nil, errors.NewNotFound(v1beta1.Resource("deployment"), name)
	}
	return obj.(*v1beta1.Deployment), nil
}
