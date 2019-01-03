// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	scheme "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// BucketBindingsGetter has a method to return a BucketBindingInterface.
// A group's client should implement this interface.
type BucketBindingsGetter interface {
	BucketBindings() BucketBindingInterface
}

// BucketBindingInterface has methods to work with BucketBinding resources.
type BucketBindingInterface interface {
	Create(*v1alpha1.BucketBinding) (*v1alpha1.BucketBinding, error)
	Update(*v1alpha1.BucketBinding) (*v1alpha1.BucketBinding, error)
	UpdateStatus(*v1alpha1.BucketBinding) (*v1alpha1.BucketBinding, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.BucketBinding, error)
	List(opts v1.ListOptions) (*v1alpha1.BucketBindingList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.BucketBinding, err error)
	BucketBindingExpansion
}

// bucketBindings implements BucketBindingInterface
type bucketBindings struct {
	client rest.Interface
}

// newBucketBindings returns a BucketBindings
func newBucketBindings(c *ClusterV1alpha1Client) *bucketBindings {
	return &bucketBindings{
		client: c.RESTClient(),
	}
}

// Get takes name of the bucketBinding, and returns the corresponding bucketBinding object, and an error if there is any.
func (c *bucketBindings) Get(name string, options v1.GetOptions) (result *v1alpha1.BucketBinding, err error) {
	result = &v1alpha1.BucketBinding{}
	err = c.client.Get().
		Resource("bucketbindings").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of BucketBindings that match those selectors.
func (c *bucketBindings) List(opts v1.ListOptions) (result *v1alpha1.BucketBindingList, err error) {
	result = &v1alpha1.BucketBindingList{}
	err = c.client.Get().
		Resource("bucketbindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested bucketBindings.
func (c *bucketBindings) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("bucketbindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a bucketBinding and creates it.  Returns the server's representation of the bucketBinding, and an error, if there is any.
func (c *bucketBindings) Create(bucketBinding *v1alpha1.BucketBinding) (result *v1alpha1.BucketBinding, err error) {
	result = &v1alpha1.BucketBinding{}
	err = c.client.Post().
		Resource("bucketbindings").
		Body(bucketBinding).
		Do().
		Into(result)
	return
}

// Update takes the representation of a bucketBinding and updates it. Returns the server's representation of the bucketBinding, and an error, if there is any.
func (c *bucketBindings) Update(bucketBinding *v1alpha1.BucketBinding) (result *v1alpha1.BucketBinding, err error) {
	result = &v1alpha1.BucketBinding{}
	err = c.client.Put().
		Resource("bucketbindings").
		Name(bucketBinding.Name).
		Body(bucketBinding).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *bucketBindings) UpdateStatus(bucketBinding *v1alpha1.BucketBinding) (result *v1alpha1.BucketBinding, err error) {
	result = &v1alpha1.BucketBinding{}
	err = c.client.Put().
		Resource("bucketbindings").
		Name(bucketBinding.Name).
		SubResource("status").
		Body(bucketBinding).
		Do().
		Into(result)
	return
}

// Delete takes name of the bucketBinding and deletes it. Returns an error if one occurs.
func (c *bucketBindings) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("bucketbindings").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *bucketBindings) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("bucketbindings").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched bucketBinding.
func (c *bucketBindings) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.BucketBinding, err error) {
	result = &v1alpha1.BucketBinding{}
	err = c.client.Patch(pt).
		Resource("bucketbindings").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
