/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	scheme "gitlab.alipay-inc.com/sigma/clientset/kubernetes/scheme"
	v1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/storageextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// NodeDiskStorageClassesGetter has a method to return a NodeDiskStorageClassInterface.
// A group's client should implement this interface.
type NodeDiskStorageClassesGetter interface {
	NodeDiskStorageClasses() NodeDiskStorageClassInterface
}

// NodeDiskStorageClassInterface has methods to work with NodeDiskStorageClass resources.
type NodeDiskStorageClassInterface interface {
	Create(*v1.NodeDiskStorageClass) (*v1.NodeDiskStorageClass, error)
	Update(*v1.NodeDiskStorageClass) (*v1.NodeDiskStorageClass, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.NodeDiskStorageClass, error)
	List(opts metav1.ListOptions) (*v1.NodeDiskStorageClassList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.NodeDiskStorageClass, err error)
	NodeDiskStorageClassExpansion
}

// nodeDiskStorageClasses implements NodeDiskStorageClassInterface
type nodeDiskStorageClasses struct {
	client rest.Interface
}

// newNodeDiskStorageClasses returns a NodeDiskStorageClasses
func newNodeDiskStorageClasses(c *StorageextensionsV1Client) *nodeDiskStorageClasses {
	return &nodeDiskStorageClasses{
		client: c.RESTClient(),
	}
}

// Get takes name of the nodeDiskStorageClass, and returns the corresponding nodeDiskStorageClass object, and an error if there is any.
func (c *nodeDiskStorageClasses) Get(name string, options metav1.GetOptions) (result *v1.NodeDiskStorageClass, err error) {
	result = &v1.NodeDiskStorageClass{}
	err = c.client.Get().
		Resource("nodediskstorageclasses").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of NodeDiskStorageClasses that match those selectors.
func (c *nodeDiskStorageClasses) List(opts metav1.ListOptions) (result *v1.NodeDiskStorageClassList, err error) {
	result = &v1.NodeDiskStorageClassList{}
	err = c.client.Get().
		Resource("nodediskstorageclasses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested nodeDiskStorageClasses.
func (c *nodeDiskStorageClasses) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("nodediskstorageclasses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a nodeDiskStorageClass and creates it.  Returns the server's representation of the nodeDiskStorageClass, and an error, if there is any.
func (c *nodeDiskStorageClasses) Create(nodeDiskStorageClass *v1.NodeDiskStorageClass) (result *v1.NodeDiskStorageClass, err error) {
	result = &v1.NodeDiskStorageClass{}
	err = c.client.Post().
		Resource("nodediskstorageclasses").
		Body(nodeDiskStorageClass).
		Do().
		Into(result)
	return
}

// Update takes the representation of a nodeDiskStorageClass and updates it. Returns the server's representation of the nodeDiskStorageClass, and an error, if there is any.
func (c *nodeDiskStorageClasses) Update(nodeDiskStorageClass *v1.NodeDiskStorageClass) (result *v1.NodeDiskStorageClass, err error) {
	result = &v1.NodeDiskStorageClass{}
	err = c.client.Put().
		Resource("nodediskstorageclasses").
		Name(nodeDiskStorageClass.Name).
		Body(nodeDiskStorageClass).
		Do().
		Into(result)
	return
}

// Delete takes name of the nodeDiskStorageClass and deletes it. Returns an error if one occurs.
func (c *nodeDiskStorageClasses) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("nodediskstorageclasses").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *nodeDiskStorageClasses) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return c.client.Delete().
		Resource("nodediskstorageclasses").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched nodeDiskStorageClass.
func (c *nodeDiskStorageClasses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.NodeDiskStorageClass, err error) {
	result = &v1.NodeDiskStorageClass{}
	err = c.client.Patch(pt).
		Resource("nodediskstorageclasses").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
