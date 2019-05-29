// +build !ignore_autogenerated

/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RecommendedContainerResources) DeepCopyInto(out *RecommendedContainerResources) {
	*out = *in
	if in.Target != nil {
		in, out := &in.Target, &out.Target
		*out = make(v1.ResourceList, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RecommendedContainerResources.
func (in *RecommendedContainerResources) DeepCopy() *RecommendedContainerResources {
	if in == nil {
		return nil
	}
	out := new(RecommendedContainerResources)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RecommendedPodResources) DeepCopyInto(out *RecommendedPodResources) {
	*out = *in
	if in.ContainerRecommendations != nil {
		in, out := &in.ContainerRecommendations, &out.ContainerRecommendations
		*out = make([]RecommendedContainerResources, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RecommendedPodResources.
func (in *RecommendedPodResources) DeepCopy() *RecommendedPodResources {
	if in == nil {
		return nil
	}
	out := new(RecommendedPodResources)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceProfile) DeepCopyInto(out *ResourceProfile) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceProfile.
func (in *ResourceProfile) DeepCopy() *ResourceProfile {
	if in == nil {
		return nil
	}
	out := new(ResourceProfile)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ResourceProfile) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceProfileList) DeepCopyInto(out *ResourceProfileList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ResourceProfile, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceProfileList.
func (in *ResourceProfileList) DeepCopy() *ResourceProfileList {
	if in == nil {
		return nil
	}
	out := new(ResourceProfileList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ResourceProfileList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceProfileSpec) DeepCopyInto(out *ResourceProfileSpec) {
	*out = *in
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.RecommendedResource != nil {
		in, out := &in.RecommendedResource, &out.RecommendedResource
		*out = new(RecommendedPodResources)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceProfileSpec.
func (in *ResourceProfileSpec) DeepCopy() *ResourceProfileSpec {
	if in == nil {
		return nil
	}
	out := new(ResourceProfileSpec)
	in.DeepCopyInto(out)
	return out
}
