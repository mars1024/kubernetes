/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	scheme "gitlab.alipay-inc.com/sigma/clientset/kubernetes/scheme"
	v1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/network/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// SecurityGroupsGetter has a method to return a SecurityGroupInterface.
// A group's client should implement this interface.
type SecurityGroupsGetter interface {
	SecurityGroups() SecurityGroupInterface
}

// SecurityGroupInterface has methods to work with SecurityGroup resources.
type SecurityGroupInterface interface {
	Create(*v1.SecurityGroup) (*v1.SecurityGroup, error)
	Update(*v1.SecurityGroup) (*v1.SecurityGroup, error)
	UpdateStatus(*v1.SecurityGroup) (*v1.SecurityGroup, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.SecurityGroup, error)
	List(opts metav1.ListOptions) (*v1.SecurityGroupList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.SecurityGroup, err error)
	SecurityGroupExpansion
}

// securityGroups implements SecurityGroupInterface
type securityGroups struct {
	client rest.Interface
}

// newSecurityGroups returns a SecurityGroups
func newSecurityGroups(c *NetworkV1Client) *securityGroups {
	return &securityGroups{
		client: c.RESTClient(),
	}
}

// Get takes name of the securityGroup, and returns the corresponding securityGroup object, and an error if there is any.
func (c *securityGroups) Get(name string, options metav1.GetOptions) (result *v1.SecurityGroup, err error) {
	result = &v1.SecurityGroup{}
	err = c.client.Get().
		Resource("securitygroups").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of SecurityGroups that match those selectors.
func (c *securityGroups) List(opts metav1.ListOptions) (result *v1.SecurityGroupList, err error) {
	result = &v1.SecurityGroupList{}
	err = c.client.Get().
		Resource("securitygroups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested securityGroups.
func (c *securityGroups) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("securitygroups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a securityGroup and creates it.  Returns the server's representation of the securityGroup, and an error, if there is any.
func (c *securityGroups) Create(securityGroup *v1.SecurityGroup) (result *v1.SecurityGroup, err error) {
	result = &v1.SecurityGroup{}
	err = c.client.Post().
		Resource("securitygroups").
		Body(securityGroup).
		Do().
		Into(result)
	return
}

// Update takes the representation of a securityGroup and updates it. Returns the server's representation of the securityGroup, and an error, if there is any.
func (c *securityGroups) Update(securityGroup *v1.SecurityGroup) (result *v1.SecurityGroup, err error) {
	result = &v1.SecurityGroup{}
	err = c.client.Put().
		Resource("securitygroups").
		Name(securityGroup.Name).
		Body(securityGroup).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *securityGroups) UpdateStatus(securityGroup *v1.SecurityGroup) (result *v1.SecurityGroup, err error) {
	result = &v1.SecurityGroup{}
	err = c.client.Put().
		Resource("securitygroups").
		Name(securityGroup.Name).
		SubResource("status").
		Body(securityGroup).
		Do().
		Into(result)
	return
}

// Delete takes name of the securityGroup and deletes it. Returns an error if one occurs.
func (c *securityGroups) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("securitygroups").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *securityGroups) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return c.client.Delete().
		Resource("securitygroups").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched securityGroup.
func (c *securityGroups) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.SecurityGroup, err error) {
	result = &v1.SecurityGroup{}
	err = c.client.Patch(pt).
		Resource("securitygroups").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
