/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/machine/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// OpsTypeLister helps list OpsTypes.
type OpsTypeLister interface {
	// List lists all OpsTypes in the indexer.
	List(selector labels.Selector) (ret []*v1.OpsType, err error)
	// Get retrieves the OpsType from the index for a given name.
	Get(name string) (*v1.OpsType, error)
	OpsTypeListerExpansion
}

// opsTypeLister implements the OpsTypeLister interface.
type opsTypeLister struct {
	indexer cache.Indexer
}

// NewOpsTypeLister returns a new OpsTypeLister.
func NewOpsTypeLister(indexer cache.Indexer) OpsTypeLister {
	return &opsTypeLister{indexer: indexer}
}

// List lists all OpsTypes in the indexer.
func (s *opsTypeLister) List(selector labels.Selector) (ret []*v1.OpsType, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.OpsType))
	})
	return ret, err
}

// Get retrieves the OpsType from the index for a given name.
func (s *opsTypeLister) Get(name string) (*v1.OpsType, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("opstype"), name)
	}
	return obj.(*v1.OpsType), nil
}
