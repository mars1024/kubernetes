// +build !multitenancy

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/apps/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// CafeInPlaceDeploymentLister helps list CafeInPlaceDeployments.
type CafeInPlaceDeploymentLister interface {
	// List lists all CafeInPlaceDeployments in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.CafeInPlaceDeployment, err error)
	// CafeInPlaceDeployments returns an object that can list and get CafeInPlaceDeployments.
	CafeInPlaceDeployments(namespace string) CafeInPlaceDeploymentNamespaceLister
	CafeInPlaceDeploymentListerExpansion
}

// cafeInPlaceDeploymentLister implements the CafeInPlaceDeploymentLister interface.
type cafeInPlaceDeploymentLister struct {
	indexer cache.Indexer
}

// NewCafeInPlaceDeploymentLister returns a new CafeInPlaceDeploymentLister.
func NewCafeInPlaceDeploymentLister(indexer cache.Indexer) CafeInPlaceDeploymentLister {
	return &cafeInPlaceDeploymentLister{indexer: indexer}
}

// List lists all CafeInPlaceDeployments in the indexer.
func (s *cafeInPlaceDeploymentLister) List(selector labels.Selector) (ret []*v1alpha1.CafeInPlaceDeployment, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.CafeInPlaceDeployment))
	})
	return ret, err
}

// CafeInPlaceDeployments returns an object that can list and get CafeInPlaceDeployments.
func (s *cafeInPlaceDeploymentLister) CafeInPlaceDeployments(namespace string) CafeInPlaceDeploymentNamespaceLister {
	return cafeInPlaceDeploymentNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// CafeInPlaceDeploymentNamespaceLister helps list and get CafeInPlaceDeployments.
type CafeInPlaceDeploymentNamespaceLister interface {
	// List lists all CafeInPlaceDeployments in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.CafeInPlaceDeployment, err error)
	// Get retrieves the CafeInPlaceDeployment from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.CafeInPlaceDeployment, error)
	CafeInPlaceDeploymentNamespaceListerExpansion
}

// cafeInPlaceDeploymentNamespaceLister implements the CafeInPlaceDeploymentNamespaceLister
// interface.
type cafeInPlaceDeploymentNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all CafeInPlaceDeployments in the indexer for a given namespace.
func (s cafeInPlaceDeploymentNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.CafeInPlaceDeployment, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.CafeInPlaceDeployment))
	})
	return ret, err
}

// Get retrieves the CafeInPlaceDeployment from the indexer for a given namespace and name.
func (s cafeInPlaceDeploymentNamespaceLister) Get(name string) (*v1alpha1.CafeInPlaceDeployment, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("cafeinplacedeployment"), name)
	}
	return obj.(*v1alpha1.CafeInPlaceDeployment), nil
}
