/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	scheme "gitlab.alipay-inc.com/sigma/clientset/kubernetes/scheme"
	v1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/cluster/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ClusterVersionsGetter has a method to return a ClusterVersionInterface.
// A group's client should implement this interface.
type ClusterVersionsGetter interface {
	ClusterVersions() ClusterVersionInterface
}

// ClusterVersionInterface has methods to work with ClusterVersion resources.
type ClusterVersionInterface interface {
	Create(*v1.ClusterVersion) (*v1.ClusterVersion, error)
	Update(*v1.ClusterVersion) (*v1.ClusterVersion, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.ClusterVersion, error)
	List(opts metav1.ListOptions) (*v1.ClusterVersionList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.ClusterVersion, err error)
	ClusterVersionExpansion
}

// clusterVersions implements ClusterVersionInterface
type clusterVersions struct {
	client rest.Interface
}

// newClusterVersions returns a ClusterVersions
func newClusterVersions(c *ClusterV1Client) *clusterVersions {
	return &clusterVersions{
		client: c.RESTClient(),
	}
}

// Get takes name of the clusterVersion, and returns the corresponding clusterVersion object, and an error if there is any.
func (c *clusterVersions) Get(name string, options metav1.GetOptions) (result *v1.ClusterVersion, err error) {
	result = &v1.ClusterVersion{}
	err = c.client.Get().
		Resource("clusterversions").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterVersions that match those selectors.
func (c *clusterVersions) List(opts metav1.ListOptions) (result *v1.ClusterVersionList, err error) {
	result = &v1.ClusterVersionList{}
	err = c.client.Get().
		Resource("clusterversions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusterVersions.
func (c *clusterVersions) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("clusterversions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a clusterVersion and creates it.  Returns the server's representation of the clusterVersion, and an error, if there is any.
func (c *clusterVersions) Create(clusterVersion *v1.ClusterVersion) (result *v1.ClusterVersion, err error) {
	result = &v1.ClusterVersion{}
	err = c.client.Post().
		Resource("clusterversions").
		Body(clusterVersion).
		Do().
		Into(result)
	return
}

// Update takes the representation of a clusterVersion and updates it. Returns the server's representation of the clusterVersion, and an error, if there is any.
func (c *clusterVersions) Update(clusterVersion *v1.ClusterVersion) (result *v1.ClusterVersion, err error) {
	result = &v1.ClusterVersion{}
	err = c.client.Put().
		Resource("clusterversions").
		Name(clusterVersion.Name).
		Body(clusterVersion).
		Do().
		Into(result)
	return
}

// Delete takes name of the clusterVersion and deletes it. Returns an error if one occurs.
func (c *clusterVersions) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("clusterversions").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusterVersions) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return c.client.Delete().
		Resource("clusterversions").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched clusterVersion.
func (c *clusterVersions) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.ClusterVersion, err error) {
	result = &v1.ClusterVersion{}
	err = c.client.Patch(pt).
		Resource("clusterversions").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
