/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	machinev1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/machine/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMachinePackages implements MachinePackageInterface
type FakeMachinePackages struct {
	Fake *FakeMachineV1
}

var machinepackagesResource = schema.GroupVersionResource{Group: "machine.sigma.alipay.com", Version: "v1", Resource: "machinepackages"}

var machinepackagesKind = schema.GroupVersionKind{Group: "machine.sigma.alipay.com", Version: "v1", Kind: "MachinePackage"}

// Get takes name of the machinePackage, and returns the corresponding machinePackage object, and an error if there is any.
func (c *FakeMachinePackages) Get(name string, options v1.GetOptions) (result *machinev1.MachinePackage, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(machinepackagesResource, name), &machinev1.MachinePackage{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.MachinePackage), err
}

// List takes label and field selectors, and returns the list of MachinePackages that match those selectors.
func (c *FakeMachinePackages) List(opts v1.ListOptions) (result *machinev1.MachinePackageList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(machinepackagesResource, machinepackagesKind, opts), &machinev1.MachinePackageList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &machinev1.MachinePackageList{ListMeta: obj.(*machinev1.MachinePackageList).ListMeta}
	for _, item := range obj.(*machinev1.MachinePackageList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested machinePackages.
func (c *FakeMachinePackages) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(machinepackagesResource, opts))
}

// Create takes the representation of a machinePackage and creates it.  Returns the server's representation of the machinePackage, and an error, if there is any.
func (c *FakeMachinePackages) Create(machinePackage *machinev1.MachinePackage) (result *machinev1.MachinePackage, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(machinepackagesResource, machinePackage), &machinev1.MachinePackage{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.MachinePackage), err
}

// Update takes the representation of a machinePackage and updates it. Returns the server's representation of the machinePackage, and an error, if there is any.
func (c *FakeMachinePackages) Update(machinePackage *machinev1.MachinePackage) (result *machinev1.MachinePackage, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(machinepackagesResource, machinePackage), &machinev1.MachinePackage{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.MachinePackage), err
}

// Delete takes name of the machinePackage and deletes it. Returns an error if one occurs.
func (c *FakeMachinePackages) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(machinepackagesResource, name), &machinev1.MachinePackage{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMachinePackages) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(machinepackagesResource, listOptions)

	_, err := c.Fake.Invokes(action, &machinev1.MachinePackageList{})
	return err
}

// Patch applies the patch and returns the patched machinePackage.
func (c *FakeMachinePackages) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *machinev1.MachinePackage, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(machinepackagesResource, name, data, subresources...), &machinev1.MachinePackage{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.MachinePackage), err
}
