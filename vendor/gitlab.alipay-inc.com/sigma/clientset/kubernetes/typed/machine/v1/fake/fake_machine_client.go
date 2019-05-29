/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1 "gitlab.alipay-inc.com/sigma/clientset/kubernetes/typed/machine/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeMachineV1 struct {
	*testing.Fake
}

func (c *FakeMachineV1) Machines() v1.MachineInterface {
	return &FakeMachines{c}
}

func (c *FakeMachineV1) MachineOpses() v1.MachineOpsInterface {
	return &FakeMachineOpses{c}
}

func (c *FakeMachineV1) MachinePackages() v1.MachinePackageInterface {
	return &FakeMachinePackages{c}
}

func (c *FakeMachineV1) MachinePackageVersions() v1.MachinePackageVersionInterface {
	return &FakeMachinePackageVersions{c}
}

func (c *FakeMachineV1) OpsTypes() v1.OpsTypeInterface {
	return &FakeOpsTypes{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeMachineV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
