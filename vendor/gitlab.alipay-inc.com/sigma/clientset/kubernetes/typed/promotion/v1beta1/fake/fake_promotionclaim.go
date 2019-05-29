/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1beta1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/promotion/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePromotionClaims implements PromotionClaimInterface
type FakePromotionClaims struct {
	Fake *FakePromotionV1beta1
}

var promotionclaimsResource = schema.GroupVersionResource{Group: "promotion.sigma.alipay.com", Version: "v1beta1", Resource: "promotionclaims"}

var promotionclaimsKind = schema.GroupVersionKind{Group: "promotion.sigma.alipay.com", Version: "v1beta1", Kind: "PromotionClaim"}

// Get takes name of the promotionClaim, and returns the corresponding promotionClaim object, and an error if there is any.
func (c *FakePromotionClaims) Get(name string, options v1.GetOptions) (result *v1beta1.PromotionClaim, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(promotionclaimsResource, name), &v1beta1.PromotionClaim{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.PromotionClaim), err
}

// List takes label and field selectors, and returns the list of PromotionClaims that match those selectors.
func (c *FakePromotionClaims) List(opts v1.ListOptions) (result *v1beta1.PromotionClaimList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(promotionclaimsResource, promotionclaimsKind, opts), &v1beta1.PromotionClaimList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.PromotionClaimList{ListMeta: obj.(*v1beta1.PromotionClaimList).ListMeta}
	for _, item := range obj.(*v1beta1.PromotionClaimList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested promotionClaims.
func (c *FakePromotionClaims) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(promotionclaimsResource, opts))
}

// Create takes the representation of a promotionClaim and creates it.  Returns the server's representation of the promotionClaim, and an error, if there is any.
func (c *FakePromotionClaims) Create(promotionClaim *v1beta1.PromotionClaim) (result *v1beta1.PromotionClaim, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(promotionclaimsResource, promotionClaim), &v1beta1.PromotionClaim{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.PromotionClaim), err
}

// Update takes the representation of a promotionClaim and updates it. Returns the server's representation of the promotionClaim, and an error, if there is any.
func (c *FakePromotionClaims) Update(promotionClaim *v1beta1.PromotionClaim) (result *v1beta1.PromotionClaim, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(promotionclaimsResource, promotionClaim), &v1beta1.PromotionClaim{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.PromotionClaim), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePromotionClaims) UpdateStatus(promotionClaim *v1beta1.PromotionClaim) (*v1beta1.PromotionClaim, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(promotionclaimsResource, "status", promotionClaim), &v1beta1.PromotionClaim{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.PromotionClaim), err
}

// Delete takes name of the promotionClaim and deletes it. Returns an error if one occurs.
func (c *FakePromotionClaims) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(promotionclaimsResource, name), &v1beta1.PromotionClaim{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePromotionClaims) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(promotionclaimsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1beta1.PromotionClaimList{})
	return err
}

// Patch applies the patch and returns the patched promotionClaim.
func (c *FakePromotionClaims) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.PromotionClaim, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(promotionclaimsResource, name, data, subresources...), &v1beta1.PromotionClaim{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.PromotionClaim), err
}
