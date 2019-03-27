package v1beta1

import (
	v1beta1 "gitlab.alibaba-inc.com/sigma/sigma-k8s-extensions/pkg/apis/apps/v1beta1"
	scheme "gitlab.alibaba-inc.com/sigma/sigma-k8s-extensions/pkg/client/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// InPlaceSetsGetter has a method to return a InPlaceSetInterface.
// A group's client should implement this interface.
type InPlaceSetsGetter interface {
	InPlaceSets(namespace string) InPlaceSetInterface
}

// InPlaceSetInterface has methods to work with InPlaceSet resources.
type InPlaceSetInterface interface {
	Create(*v1beta1.InPlaceSet) (*v1beta1.InPlaceSet, error)
	Update(*v1beta1.InPlaceSet) (*v1beta1.InPlaceSet, error)
	UpdateStatus(*v1beta1.InPlaceSet) (*v1beta1.InPlaceSet, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1beta1.InPlaceSet, error)
	List(opts v1.ListOptions) (*v1beta1.InPlaceSetList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.InPlaceSet, err error)
}

// inPlaceSets implements InPlaceSetInterface
type inPlaceSets struct {
	client rest.Interface
	ns     string
}

// newInPlaceSets returns a InPlaceSets
func newInPlaceSets(c *AppsV1beta1Client, namespace string) *inPlaceSets {
	return &inPlaceSets{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the inPlaceSet, and returns the corresponding inPlaceSet object, and an error if there is any.
func (c *inPlaceSets) Get(name string, options v1.GetOptions) (result *v1beta1.InPlaceSet, err error) {
	result = &v1beta1.InPlaceSet{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("inplacesets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of InPlaceSets that match those selectors.
func (c *inPlaceSets) List(opts v1.ListOptions) (result *v1beta1.InPlaceSetList, err error) {
	result = &v1beta1.InPlaceSetList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("inplacesets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested inPlaceSets.
func (c *inPlaceSets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("inplacesets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a inPlaceSet and creates it.  Returns the server's representation of the inPlaceSet, and an error, if there is any.
func (c *inPlaceSets) Create(inPlaceSet *v1beta1.InPlaceSet) (result *v1beta1.InPlaceSet, err error) {
	result = &v1beta1.InPlaceSet{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("inplacesets").
		Body(inPlaceSet).
		Do().
		Into(result)
	return
}

// Update takes the representation of a inPlaceSet and updates it. Returns the server's representation of the inPlaceSet, and an error, if there is any.
func (c *inPlaceSets) Update(inPlaceSet *v1beta1.InPlaceSet) (result *v1beta1.InPlaceSet, err error) {
	result = &v1beta1.InPlaceSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("inplacesets").
		Name(inPlaceSet.Name).
		Body(inPlaceSet).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *inPlaceSets) UpdateStatus(inPlaceSet *v1beta1.InPlaceSet) (result *v1beta1.InPlaceSet, err error) {
	result = &v1beta1.InPlaceSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("inplacesets").
		Name(inPlaceSet.Name).
		SubResource("status").
		Body(inPlaceSet).
		Do().
		Into(result)
	return
}

// Delete takes name of the inPlaceSet and deletes it. Returns an error if one occurs.
func (c *inPlaceSets) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("inplacesets").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *inPlaceSets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("inplacesets").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched inPlaceSet.
func (c *inPlaceSets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.InPlaceSet, err error) {
	result = &v1beta1.InPlaceSet{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("inplacesets").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
