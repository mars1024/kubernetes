package v1beta1

import (
	v1beta1 "gitlab.alibaba-inc.com/sigma/sigma-k8s-extensions/pkg/apis/apps/v1beta1"
	scheme "gitlab.alibaba-inc.com/sigma/sigma-k8s-extensions/pkg/client/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CapacityPreviewsGetter has a method to return a CapacityPreviewInterface.
// A group's client should implement this interface.
type CapacityPreviewsGetter interface {
	CapacityPreviews(namespace string) CapacityPreviewInterface
}

// CapacityPreviewInterface has methods to work with CapacityPreview resources.
type CapacityPreviewInterface interface {
	Create(*v1beta1.CapacityPreview) (*v1beta1.CapacityPreview, error)
	Update(*v1beta1.CapacityPreview) (*v1beta1.CapacityPreview, error)
	UpdateStatus(*v1beta1.CapacityPreview) (*v1beta1.CapacityPreview, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1beta1.CapacityPreview, error)
	List(opts v1.ListOptions) (*v1beta1.CapacityPreviewList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.CapacityPreview, err error)
}

// capacityPreviews implements CapacityPreviewInterface
type capacityPreviews struct {
	client rest.Interface
	ns     string
}

// newCapacityPreviews returns a CapacityPreviews
func newCapacityPreviews(c *AppsV1beta1Client, namespace string) *capacityPreviews {
	return &capacityPreviews{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the capacityPreview, and returns the corresponding capacityPreview object, and an error if there is any.
func (c *capacityPreviews) Get(name string, options v1.GetOptions) (result *v1beta1.CapacityPreview, err error) {
	result = &v1beta1.CapacityPreview{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("capacitypreviews").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of CapacityPreviews that match those selectors.
func (c *capacityPreviews) List(opts v1.ListOptions) (result *v1beta1.CapacityPreviewList, err error) {
	result = &v1beta1.CapacityPreviewList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("capacitypreviews").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested capacityPreviews.
func (c *capacityPreviews) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("capacitypreviews").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a capacityPreview and creates it.  Returns the server's representation of the capacityPreview, and an error, if there is any.
func (c *capacityPreviews) Create(capacityPreview *v1beta1.CapacityPreview) (result *v1beta1.CapacityPreview, err error) {
	result = &v1beta1.CapacityPreview{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("capacitypreviews").
		Body(capacityPreview).
		Do().
		Into(result)
	return
}

// Update takes the representation of a capacityPreview and updates it. Returns the server's representation of the capacityPreview, and an error, if there is any.
func (c *capacityPreviews) Update(capacityPreview *v1beta1.CapacityPreview) (result *v1beta1.CapacityPreview, err error) {
	result = &v1beta1.CapacityPreview{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("capacitypreviews").
		Name(capacityPreview.Name).
		Body(capacityPreview).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *capacityPreviews) UpdateStatus(capacityPreview *v1beta1.CapacityPreview) (result *v1beta1.CapacityPreview, err error) {
	result = &v1beta1.CapacityPreview{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("capacitypreviews").
		Name(capacityPreview.Name).
		SubResource("status").
		Body(capacityPreview).
		Do().
		Into(result)
	return
}

// Delete takes name of the capacityPreview and deletes it. Returns an error if one occurs.
func (c *capacityPreviews) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("capacitypreviews").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *capacityPreviews) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("capacitypreviews").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched capacityPreview.
func (c *capacityPreviews) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.CapacityPreview, err error) {
	result = &v1beta1.CapacityPreview{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("capacitypreviews").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
