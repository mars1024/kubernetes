/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	scheme "gitlab.alipay-inc.com/sigma/clientset/kubernetes/scheme"
	v1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// NotificationReceiversGetter has a method to return a NotificationReceiverInterface.
// A group's client should implement this interface.
type NotificationReceiversGetter interface {
	NotificationReceivers() NotificationReceiverInterface
}

// NotificationReceiverInterface has methods to work with NotificationReceiver resources.
type NotificationReceiverInterface interface {
	Create(*v1.NotificationReceiver) (*v1.NotificationReceiver, error)
	Update(*v1.NotificationReceiver) (*v1.NotificationReceiver, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.NotificationReceiver, error)
	List(opts metav1.ListOptions) (*v1.NotificationReceiverList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.NotificationReceiver, err error)
	NotificationReceiverExpansion
}

// notificationReceivers implements NotificationReceiverInterface
type notificationReceivers struct {
	client rest.Interface
}

// newNotificationReceivers returns a NotificationReceivers
func newNotificationReceivers(c *MonitoringV1Client) *notificationReceivers {
	return &notificationReceivers{
		client: c.RESTClient(),
	}
}

// Get takes name of the notificationReceiver, and returns the corresponding notificationReceiver object, and an error if there is any.
func (c *notificationReceivers) Get(name string, options metav1.GetOptions) (result *v1.NotificationReceiver, err error) {
	result = &v1.NotificationReceiver{}
	err = c.client.Get().
		Resource("notificationreceivers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of NotificationReceivers that match those selectors.
func (c *notificationReceivers) List(opts metav1.ListOptions) (result *v1.NotificationReceiverList, err error) {
	result = &v1.NotificationReceiverList{}
	err = c.client.Get().
		Resource("notificationreceivers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested notificationReceivers.
func (c *notificationReceivers) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("notificationreceivers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a notificationReceiver and creates it.  Returns the server's representation of the notificationReceiver, and an error, if there is any.
func (c *notificationReceivers) Create(notificationReceiver *v1.NotificationReceiver) (result *v1.NotificationReceiver, err error) {
	result = &v1.NotificationReceiver{}
	err = c.client.Post().
		Resource("notificationreceivers").
		Body(notificationReceiver).
		Do().
		Into(result)
	return
}

// Update takes the representation of a notificationReceiver and updates it. Returns the server's representation of the notificationReceiver, and an error, if there is any.
func (c *notificationReceivers) Update(notificationReceiver *v1.NotificationReceiver) (result *v1.NotificationReceiver, err error) {
	result = &v1.NotificationReceiver{}
	err = c.client.Put().
		Resource("notificationreceivers").
		Name(notificationReceiver.Name).
		Body(notificationReceiver).
		Do().
		Into(result)
	return
}

// Delete takes name of the notificationReceiver and deletes it. Returns an error if one occurs.
func (c *notificationReceivers) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("notificationreceivers").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *notificationReceivers) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return c.client.Delete().
		Resource("notificationreceivers").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched notificationReceiver.
func (c *notificationReceivers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.NotificationReceiver, err error) {
	result = &v1.NotificationReceiver{}
	err = c.client.Patch(pt).
		Resource("notificationreceivers").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
