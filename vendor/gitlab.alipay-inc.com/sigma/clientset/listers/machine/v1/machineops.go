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

// MachineOpsLister helps list MachineOpses.
type MachineOpsLister interface {
	// List lists all MachineOpses in the indexer.
	List(selector labels.Selector) (ret []*v1.MachineOps, err error)
	// Get retrieves the MachineOps from the index for a given name.
	Get(name string) (*v1.MachineOps, error)
	MachineOpsListerExpansion
}

// machineOpsLister implements the MachineOpsLister interface.
type machineOpsLister struct {
	indexer cache.Indexer
}

// NewMachineOpsLister returns a new MachineOpsLister.
func NewMachineOpsLister(indexer cache.Indexer) MachineOpsLister {
	return &machineOpsLister{indexer: indexer}
}

// List lists all MachineOpses in the indexer.
func (s *machineOpsLister) List(selector labels.Selector) (ret []*v1.MachineOps, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.MachineOps))
	})
	return ret, err
}

// Get retrieves the MachineOps from the index for a given name.
func (s *machineOpsLister) Get(name string) (*v1.MachineOps, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("machineops"), name)
	}
	return obj.(*v1.MachineOps), nil
}
