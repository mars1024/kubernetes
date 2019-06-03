/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	monitoringv1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/monitoring/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeNotificationChannels implements NotificationChannelInterface
type FakeNotificationChannels struct {
	Fake *FakeMonitoringV1
}

var notificationchannelsResource = schema.GroupVersionResource{Group: "monitoring.sigma.alipay.com", Version: "v1", Resource: "notificationchannels"}

var notificationchannelsKind = schema.GroupVersionKind{Group: "monitoring.sigma.alipay.com", Version: "v1", Kind: "NotificationChannel"}

// Get takes name of the notificationChannel, and returns the corresponding notificationChannel object, and an error if there is any.
func (c *FakeNotificationChannels) Get(name string, options v1.GetOptions) (result *monitoringv1.NotificationChannel, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(notificationchannelsResource, name), &monitoringv1.NotificationChannel{})
	if obj == nil {
		return nil, err
	}
	return obj.(*monitoringv1.NotificationChannel), err
}

// List takes label and field selectors, and returns the list of NotificationChannels that match those selectors.
func (c *FakeNotificationChannels) List(opts v1.ListOptions) (result *monitoringv1.NotificationChannelList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(notificationchannelsResource, notificationchannelsKind, opts), &monitoringv1.NotificationChannelList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &monitoringv1.NotificationChannelList{ListMeta: obj.(*monitoringv1.NotificationChannelList).ListMeta}
	for _, item := range obj.(*monitoringv1.NotificationChannelList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested notificationChannels.
func (c *FakeNotificationChannels) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(notificationchannelsResource, opts))
}

// Create takes the representation of a notificationChannel and creates it.  Returns the server's representation of the notificationChannel, and an error, if there is any.
func (c *FakeNotificationChannels) Create(notificationChannel *monitoringv1.NotificationChannel) (result *monitoringv1.NotificationChannel, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(notificationchannelsResource, notificationChannel), &monitoringv1.NotificationChannel{})
	if obj == nil {
		return nil, err
	}
	return obj.(*monitoringv1.NotificationChannel), err
}

// Update takes the representation of a notificationChannel and updates it. Returns the server's representation of the notificationChannel, and an error, if there is any.
func (c *FakeNotificationChannels) Update(notificationChannel *monitoringv1.NotificationChannel) (result *monitoringv1.NotificationChannel, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(notificationchannelsResource, notificationChannel), &monitoringv1.NotificationChannel{})
	if obj == nil {
		return nil, err
	}
	return obj.(*monitoringv1.NotificationChannel), err
}

// Delete takes name of the notificationChannel and deletes it. Returns an error if one occurs.
func (c *FakeNotificationChannels) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(notificationchannelsResource, name), &monitoringv1.NotificationChannel{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeNotificationChannels) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(notificationchannelsResource, listOptions)

	_, err := c.Fake.Invokes(action, &monitoringv1.NotificationChannelList{})
	return err
}

// Patch applies the patch and returns the patched notificationChannel.
func (c *FakeNotificationChannels) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *monitoringv1.NotificationChannel, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(notificationchannelsResource, name, data, subresources...), &monitoringv1.NotificationChannel{})
	if obj == nil {
		return nil, err
	}
	return obj.(*monitoringv1.NotificationChannel), err
}
