// +build !ignore_autogenerated

/*
Copyright 2018 Sigma.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by main. DO NOT EDIT.

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CapacityPreview) DeepCopyInto(out *CapacityPreview) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CapacityPreview.
func (in *CapacityPreview) DeepCopy() *CapacityPreview {
	if in == nil {
		return nil
	}
	out := new(CapacityPreview)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CapacityPreview) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CapacityPreviewList) DeepCopyInto(out *CapacityPreviewList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CapacityPreview, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CapacityPreviewList.
func (in *CapacityPreviewList) DeepCopy() *CapacityPreviewList {
	if in == nil {
		return nil
	}
	out := new(CapacityPreviewList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CapacityPreviewList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CapacityPreviewSpec) DeepCopyInto(out *CapacityPreviewSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CapacityPreviewSpec.
func (in *CapacityPreviewSpec) DeepCopy() *CapacityPreviewSpec {
	if in == nil {
		return nil
	}
	out := new(CapacityPreviewSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CapacityPreviewStatus) DeepCopyInto(out *CapacityPreviewStatus) {
	*out = *in
	if in.FailedReasons != nil {
		in, out := &in.FailedReasons, &out.FailedReasons
		*out = make([]*FailedReason, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(FailedReason)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.PreviewAllocatedItems != nil {
		in, out := &in.PreviewAllocatedItems, &out.PreviewAllocatedItems
		*out = make([]*PreviewAllocatedItem, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(PreviewAllocatedItem)
				**out = **in
			}
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CapacityPreviewStatus.
func (in *CapacityPreviewStatus) DeepCopy() *CapacityPreviewStatus {
	if in == nil {
		return nil
	}
	out := new(CapacityPreviewStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FailedReason) DeepCopyInto(out *FailedReason) {
	*out = *in
	if in.NodeNames != nil {
		in, out := &in.NodeNames, &out.NodeNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FailedReason.
func (in *FailedReason) DeepCopy() *FailedReason {
	if in == nil {
		return nil
	}
	out := new(FailedReason)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InPlaceSet) DeepCopyInto(out *InPlaceSet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InPlaceSet.
func (in *InPlaceSet) DeepCopy() *InPlaceSet {
	if in == nil {
		return nil
	}
	out := new(InPlaceSet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InPlaceSet) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InPlaceSetCondition) DeepCopyInto(out *InPlaceSetCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InPlaceSetCondition.
func (in *InPlaceSetCondition) DeepCopy() *InPlaceSetCondition {
	if in == nil {
		return nil
	}
	out := new(InPlaceSetCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InPlaceSetList) DeepCopyInto(out *InPlaceSetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]InPlaceSet, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InPlaceSetList.
func (in *InPlaceSetList) DeepCopy() *InPlaceSetList {
	if in == nil {
		return nil
	}
	out := new(InPlaceSetList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InPlaceSetList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InPlaceSetSpec) DeepCopyInto(out *InPlaceSetSpec) {
	*out = *in
	if in.PodsToDelete != nil {
		in, out := &in.PodsToDelete, &out.PodsToDelete
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	in.Template.DeepCopyInto(&out.Template)
	if in.VolumeClaimTemplates != nil {
		in, out := &in.VolumeClaimTemplates, &out.VolumeClaimTemplates
		*out = make([]corev1.PersistentVolumeClaim, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.UpgradeStrategy.DeepCopyInto(&out.UpgradeStrategy)
	if in.RevisionHistoryLimit != nil {
		in, out := &in.RevisionHistoryLimit, &out.RevisionHistoryLimit
		*out = new(int32)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InPlaceSetSpec.
func (in *InPlaceSetSpec) DeepCopy() *InPlaceSetSpec {
	if in == nil {
		return nil
	}
	out := new(InPlaceSetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InPlaceSetStatus) DeepCopyInto(out *InPlaceSetStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]InPlaceSetCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InPlaceSetStatus.
func (in *InPlaceSetStatus) DeepCopy() *InPlaceSetStatus {
	if in == nil {
		return nil
	}
	out := new(InPlaceSetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InPlaceUpgradeStrategy) DeepCopyInto(out *InPlaceUpgradeStrategy) {
	*out = *in
	if in.MaxUnavailable != nil {
		in, out := &in.MaxUnavailable, &out.MaxUnavailable
		*out = new(int32)
		**out = **in
	}
	if in.Partition != nil {
		in, out := &in.Partition, &out.Partition
		*out = new(int32)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InPlaceUpgradeStrategy.
func (in *InPlaceUpgradeStrategy) DeepCopy() *InPlaceUpgradeStrategy {
	if in == nil {
		return nil
	}
	out := new(InPlaceUpgradeStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PreviewAllocatedItem) DeepCopyInto(out *PreviewAllocatedItem) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PreviewAllocatedItem.
func (in *PreviewAllocatedItem) DeepCopy() *PreviewAllocatedItem {
	if in == nil {
		return nil
	}
	out := new(PreviewAllocatedItem)
	in.DeepCopyInto(out)
	return out
}
