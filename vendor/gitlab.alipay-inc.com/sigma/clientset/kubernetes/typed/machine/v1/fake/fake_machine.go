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

// FakeMachines implements MachineInterface
type FakeMachines struct {
	Fake *FakeMachineV1
}

var machinesResource = schema.GroupVersionResource{Group: "machine.sigma.alipay.com", Version: "v1", Resource: "machines"}

var machinesKind = schema.GroupVersionKind{Group: "machine.sigma.alipay.com", Version: "v1", Kind: "Machine"}

// Get takes name of the machine, and returns the corresponding machine object, and an error if there is any.
func (c *FakeMachines) Get(name string, options v1.GetOptions) (result *machinev1.Machine, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(machinesResource, name), &machinev1.Machine{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.Machine), err
}

// List takes label and field selectors, and returns the list of Machines that match those selectors.
func (c *FakeMachines) List(opts v1.ListOptions) (result *machinev1.MachineList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(machinesResource, machinesKind, opts), &machinev1.MachineList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &machinev1.MachineList{ListMeta: obj.(*machinev1.MachineList).ListMeta}
	for _, item := range obj.(*machinev1.MachineList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested machines.
func (c *FakeMachines) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(machinesResource, opts))
}

// Create takes the representation of a machine and creates it.  Returns the server's representation of the machine, and an error, if there is any.
func (c *FakeMachines) Create(machine *machinev1.Machine) (result *machinev1.Machine, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(machinesResource, machine), &machinev1.Machine{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.Machine), err
}

// Update takes the representation of a machine and updates it. Returns the server's representation of the machine, and an error, if there is any.
func (c *FakeMachines) Update(machine *machinev1.Machine) (result *machinev1.Machine, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(machinesResource, machine), &machinev1.Machine{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.Machine), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMachines) UpdateStatus(machine *machinev1.Machine) (*machinev1.Machine, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(machinesResource, "status", machine), &machinev1.Machine{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.Machine), err
}

// Delete takes name of the machine and deletes it. Returns an error if one occurs.
func (c *FakeMachines) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(machinesResource, name), &machinev1.Machine{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMachines) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(machinesResource, listOptions)

	_, err := c.Fake.Invokes(action, &machinev1.MachineList{})
	return err
}

// Patch applies the patch and returns the patched machine.
func (c *FakeMachines) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *machinev1.Machine, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(machinesResource, name, data, subresources...), &machinev1.Machine{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.Machine), err
}
