/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	storageextensionsv1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/storageextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeNodeDiskStorageClasses implements NodeDiskStorageClassInterface
type FakeNodeDiskStorageClasses struct {
	Fake *FakeStorageextensionsV1
}

var nodediskstorageclassesResource = schema.GroupVersionResource{Group: "storageextensions.sigma.alipay.com", Version: "v1", Resource: "nodediskstorageclasses"}

var nodediskstorageclassesKind = schema.GroupVersionKind{Group: "storageextensions.sigma.alipay.com", Version: "v1", Kind: "NodeDiskStorageClass"}

// Get takes name of the nodeDiskStorageClass, and returns the corresponding nodeDiskStorageClass object, and an error if there is any.
func (c *FakeNodeDiskStorageClasses) Get(name string, options v1.GetOptions) (result *storageextensionsv1.NodeDiskStorageClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(nodediskstorageclassesResource, name), &storageextensionsv1.NodeDiskStorageClass{})
	if obj == nil {
		return nil, err
	}
	return obj.(*storageextensionsv1.NodeDiskStorageClass), err
}

// List takes label and field selectors, and returns the list of NodeDiskStorageClasses that match those selectors.
func (c *FakeNodeDiskStorageClasses) List(opts v1.ListOptions) (result *storageextensionsv1.NodeDiskStorageClassList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(nodediskstorageclassesResource, nodediskstorageclassesKind, opts), &storageextensionsv1.NodeDiskStorageClassList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &storageextensionsv1.NodeDiskStorageClassList{ListMeta: obj.(*storageextensionsv1.NodeDiskStorageClassList).ListMeta}
	for _, item := range obj.(*storageextensionsv1.NodeDiskStorageClassList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested nodeDiskStorageClasses.
func (c *FakeNodeDiskStorageClasses) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(nodediskstorageclassesResource, opts))
}

// Create takes the representation of a nodeDiskStorageClass and creates it.  Returns the server's representation of the nodeDiskStorageClass, and an error, if there is any.
func (c *FakeNodeDiskStorageClasses) Create(nodeDiskStorageClass *storageextensionsv1.NodeDiskStorageClass) (result *storageextensionsv1.NodeDiskStorageClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(nodediskstorageclassesResource, nodeDiskStorageClass), &storageextensionsv1.NodeDiskStorageClass{})
	if obj == nil {
		return nil, err
	}
	return obj.(*storageextensionsv1.NodeDiskStorageClass), err
}

// Update takes the representation of a nodeDiskStorageClass and updates it. Returns the server's representation of the nodeDiskStorageClass, and an error, if there is any.
func (c *FakeNodeDiskStorageClasses) Update(nodeDiskStorageClass *storageextensionsv1.NodeDiskStorageClass) (result *storageextensionsv1.NodeDiskStorageClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(nodediskstorageclassesResource, nodeDiskStorageClass), &storageextensionsv1.NodeDiskStorageClass{})
	if obj == nil {
		return nil, err
	}
	return obj.(*storageextensionsv1.NodeDiskStorageClass), err
}

// Delete takes name of the nodeDiskStorageClass and deletes it. Returns an error if one occurs.
func (c *FakeNodeDiskStorageClasses) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(nodediskstorageclassesResource, name), &storageextensionsv1.NodeDiskStorageClass{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeNodeDiskStorageClasses) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(nodediskstorageclassesResource, listOptions)

	_, err := c.Fake.Invokes(action, &storageextensionsv1.NodeDiskStorageClassList{})
	return err
}

// Patch applies the patch and returns the patched nodeDiskStorageClass.
func (c *FakeNodeDiskStorageClasses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *storageextensionsv1.NodeDiskStorageClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(nodediskstorageclassesResource, name, data, subresources...), &storageextensionsv1.NodeDiskStorageClass{})
	if obj == nil {
		return nil, err
	}
	return obj.(*storageextensionsv1.NodeDiskStorageClass), err
}
