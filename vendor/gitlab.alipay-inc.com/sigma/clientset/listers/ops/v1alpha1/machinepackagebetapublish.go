/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/ops/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// MachinePackageBetaPublishLister helps list MachinePackageBetaPublishes.
type MachinePackageBetaPublishLister interface {
	// List lists all MachinePackageBetaPublishes in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.MachinePackageBetaPublish, err error)
	// Get retrieves the MachinePackageBetaPublish from the index for a given name.
	Get(name string) (*v1alpha1.MachinePackageBetaPublish, error)
	MachinePackageBetaPublishListerExpansion
}

// machinePackageBetaPublishLister implements the MachinePackageBetaPublishLister interface.
type machinePackageBetaPublishLister struct {
	indexer cache.Indexer
}

// NewMachinePackageBetaPublishLister returns a new MachinePackageBetaPublishLister.
func NewMachinePackageBetaPublishLister(indexer cache.Indexer) MachinePackageBetaPublishLister {
	return &machinePackageBetaPublishLister{indexer: indexer}
}

// List lists all MachinePackageBetaPublishes in the indexer.
func (s *machinePackageBetaPublishLister) List(selector labels.Selector) (ret []*v1alpha1.MachinePackageBetaPublish, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MachinePackageBetaPublish))
	})
	return ret, err
}

// Get retrieves the MachinePackageBetaPublish from the index for a given name.
func (s *machinePackageBetaPublishLister) Get(name string) (*v1alpha1.MachinePackageBetaPublish, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("machinepackagebetapublish"), name)
	}
	return obj.(*v1alpha1.MachinePackageBetaPublish), nil
}
