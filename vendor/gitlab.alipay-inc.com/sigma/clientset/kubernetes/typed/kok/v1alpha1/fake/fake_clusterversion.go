/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/kok/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeClusterVersions implements ClusterVersionInterface
type FakeClusterVersions struct {
	Fake *FakeKokV1alpha1
}

var clusterversionsResource = schema.GroupVersionResource{Group: "kok.sigma.alipay.com", Version: "v1alpha1", Resource: "clusterversions"}

var clusterversionsKind = schema.GroupVersionKind{Group: "kok.sigma.alipay.com", Version: "v1alpha1", Kind: "ClusterVersion"}

// Get takes name of the clusterVersion, and returns the corresponding clusterVersion object, and an error if there is any.
func (c *FakeClusterVersions) Get(name string, options v1.GetOptions) (result *v1alpha1.ClusterVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(clusterversionsResource, name), &v1alpha1.ClusterVersion{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterVersion), err
}

// List takes label and field selectors, and returns the list of ClusterVersions that match those selectors.
func (c *FakeClusterVersions) List(opts v1.ListOptions) (result *v1alpha1.ClusterVersionList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(clusterversionsResource, clusterversionsKind, opts), &v1alpha1.ClusterVersionList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ClusterVersionList{ListMeta: obj.(*v1alpha1.ClusterVersionList).ListMeta}
	for _, item := range obj.(*v1alpha1.ClusterVersionList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusterVersions.
func (c *FakeClusterVersions) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(clusterversionsResource, opts))
}

// Create takes the representation of a clusterVersion and creates it.  Returns the server's representation of the clusterVersion, and an error, if there is any.
func (c *FakeClusterVersions) Create(clusterVersion *v1alpha1.ClusterVersion) (result *v1alpha1.ClusterVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(clusterversionsResource, clusterVersion), &v1alpha1.ClusterVersion{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterVersion), err
}

// Update takes the representation of a clusterVersion and updates it. Returns the server's representation of the clusterVersion, and an error, if there is any.
func (c *FakeClusterVersions) Update(clusterVersion *v1alpha1.ClusterVersion) (result *v1alpha1.ClusterVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(clusterversionsResource, clusterVersion), &v1alpha1.ClusterVersion{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterVersion), err
}

// Delete takes name of the clusterVersion and deletes it. Returns an error if one occurs.
func (c *FakeClusterVersions) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(clusterversionsResource, name), &v1alpha1.ClusterVersion{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusterVersions) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(clusterversionsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.ClusterVersionList{})
	return err
}

// Patch applies the patch and returns the patched clusterVersion.
func (c *FakeClusterVersions) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clusterversionsResource, name, data, subresources...), &v1alpha1.ClusterVersion{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterVersion), err
}
