/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/network/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// PredictedInterfaceLister helps list PredictedInterfaces.
type PredictedInterfaceLister interface {
	// List lists all PredictedInterfaces in the indexer.
	List(selector labels.Selector) (ret []*v1.PredictedInterface, err error)
	// Get retrieves the PredictedInterface from the index for a given name.
	Get(name string) (*v1.PredictedInterface, error)
	PredictedInterfaceListerExpansion
}

// predictedInterfaceLister implements the PredictedInterfaceLister interface.
type predictedInterfaceLister struct {
	indexer cache.Indexer
}

// NewPredictedInterfaceLister returns a new PredictedInterfaceLister.
func NewPredictedInterfaceLister(indexer cache.Indexer) PredictedInterfaceLister {
	return &predictedInterfaceLister{indexer: indexer}
}

// List lists all PredictedInterfaces in the indexer.
func (s *predictedInterfaceLister) List(selector labels.Selector) (ret []*v1.PredictedInterface, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.PredictedInterface))
	})
	return ret, err
}

// Get retrieves the PredictedInterface from the index for a given name.
func (s *predictedInterfaceLister) Get(name string) (*v1.PredictedInterface, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("predictedinterface"), name)
	}
	return obj.(*v1.PredictedInterface), nil
}
