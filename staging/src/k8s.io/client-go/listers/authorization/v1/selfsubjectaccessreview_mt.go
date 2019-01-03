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
	v1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// SelfSubjectAccessReviewLister helps list SelfSubjectAccessReviews.
type SelfSubjectAccessReviewLister interface {
	// List lists all SelfSubjectAccessReviews in the indexer.
	List(selector labels.Selector) (ret []*v1.SelfSubjectAccessReview, err error)
	// Get retrieves the SelfSubjectAccessReview from the index for a given name.
	Get(name string) (*v1.SelfSubjectAccessReview, error)
	SelfSubjectAccessReviewListerExpansion
}

type selfSubjectAccessReviewLister struct {
	indexer cache.Indexer
	tenant  multitenancy.TenantInfo
}

func (s *selfSubjectAccessReviewLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return &selfSubjectAccessReviewLister{
		indexer: s.indexer,
		tenant:  tenant,
	}
}

// NewSelfSubjectAccessReviewLister returns a new SelfSubjectAccessReviewLister.
func NewSelfSubjectAccessReviewLister(indexer cache.Indexer) SelfSubjectAccessReviewLister {
	return &selfSubjectAccessReviewLister{indexer: indexer}
}

// List lists all SelfSubjectAccessReviews in the indexer.
func (s *selfSubjectAccessReviewLister) List(selector labels.Selector) (ret []*v1.SelfSubjectAccessReview, err error) {
	if s.tenant != nil {
		err = multitenancycache.ListAllWithTenant(s.indexer, selector, s.tenant, func(m interface{}) {
			ret = append(ret, m.(*v1.SelfSubjectAccessReview))
		})
	} else {
		err = cache.ListAll(s.indexer, selector, func(m interface{}) {
			ret = append(ret, m.(*v1.SelfSubjectAccessReview))
		})
	}
	return ret, err
}

// Get retrieves the SelfSubjectAccessReview from the index for a given name.
func (s *selfSubjectAccessReviewLister) Get(name string) (*v1.SelfSubjectAccessReview, error) {
	if s.tenant == nil {
		// fail hard so that we don't allow any cluster-scoped get w/o tenant
		debug.PrintStack()
		return nil, fmt.Errorf("cannot get selfsubjectaccessreview w/o specifying tenant")
	}

	obj, exists, err := s.indexer.GetByKey(multitenancyutil.TransformTenantInfoToJointString(s.tenant, "/") + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("selfsubjectaccessreview"), name)
	}
	return obj.(*v1.SelfSubjectAccessReview), nil
}
