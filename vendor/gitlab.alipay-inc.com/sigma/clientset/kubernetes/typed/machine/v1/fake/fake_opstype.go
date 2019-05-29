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

// FakeOpsTypes implements OpsTypeInterface
type FakeOpsTypes struct {
	Fake *FakeMachineV1
}

var opstypesResource = schema.GroupVersionResource{Group: "machine.sigma.alipay.com", Version: "v1", Resource: "opstypes"}

var opstypesKind = schema.GroupVersionKind{Group: "machine.sigma.alipay.com", Version: "v1", Kind: "OpsType"}

// Get takes name of the opsType, and returns the corresponding opsType object, and an error if there is any.
func (c *FakeOpsTypes) Get(name string, options v1.GetOptions) (result *machinev1.OpsType, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(opstypesResource, name), &machinev1.OpsType{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.OpsType), err
}

// List takes label and field selectors, and returns the list of OpsTypes that match those selectors.
func (c *FakeOpsTypes) List(opts v1.ListOptions) (result *machinev1.OpsTypeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(opstypesResource, opstypesKind, opts), &machinev1.OpsTypeList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &machinev1.OpsTypeList{ListMeta: obj.(*machinev1.OpsTypeList).ListMeta}
	for _, item := range obj.(*machinev1.OpsTypeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested opsTypes.
func (c *FakeOpsTypes) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(opstypesResource, opts))
}

// Create takes the representation of a opsType and creates it.  Returns the server's representation of the opsType, and an error, if there is any.
func (c *FakeOpsTypes) Create(opsType *machinev1.OpsType) (result *machinev1.OpsType, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(opstypesResource, opsType), &machinev1.OpsType{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.OpsType), err
}

// Update takes the representation of a opsType and updates it. Returns the server's representation of the opsType, and an error, if there is any.
func (c *FakeOpsTypes) Update(opsType *machinev1.OpsType) (result *machinev1.OpsType, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(opstypesResource, opsType), &machinev1.OpsType{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.OpsType), err
}

// Delete takes name of the opsType and deletes it. Returns an error if one occurs.
func (c *FakeOpsTypes) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(opstypesResource, name), &machinev1.OpsType{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeOpsTypes) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(opstypesResource, listOptions)

	_, err := c.Fake.Invokes(action, &machinev1.OpsTypeList{})
	return err
}

// Patch applies the patch and returns the patched opsType.
func (c *FakeOpsTypes) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *machinev1.OpsType, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(opstypesResource, name, data, subresources...), &machinev1.OpsType{})
	if obj == nil {
		return nil, err
	}
	return obj.(*machinev1.OpsType), err
}
