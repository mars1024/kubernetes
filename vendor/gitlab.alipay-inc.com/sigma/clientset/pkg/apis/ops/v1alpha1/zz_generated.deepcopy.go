// +build !ignore_autogenerated

/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BatchPublish) DeepCopyInto(out *BatchPublish) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BatchPublish.
func (in *BatchPublish) DeepCopy() *BatchPublish {
	if in == nil {
		return nil
	}
	out := new(BatchPublish)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BetaPublishVersion) DeepCopyInto(out *BetaPublishVersion) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BetaPublishVersion.
func (in *BetaPublishVersion) DeepCopy() *BetaPublishVersion {
	if in == nil {
		return nil
	}
	out := new(BetaPublishVersion)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterInfo) DeepCopyInto(out *ClusterInfo) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterInfo.
func (in *ClusterInfo) DeepCopy() *ClusterInfo {
	if in == nil {
		return nil
	}
	out := new(ClusterInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterMachinePackageVersion) DeepCopyInto(out *ClusterMachinePackageVersion) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterMachinePackageVersion.
func (in *ClusterMachinePackageVersion) DeepCopy() *ClusterMachinePackageVersion {
	if in == nil {
		return nil
	}
	out := new(ClusterMachinePackageVersion)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterMachinePackageVersion) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterMachinePackageVersionList) DeepCopyInto(out *ClusterMachinePackageVersionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterMachinePackageVersion, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterMachinePackageVersionList.
func (in *ClusterMachinePackageVersionList) DeepCopy() *ClusterMachinePackageVersionList {
	if in == nil {
		return nil
	}
	out := new(ClusterMachinePackageVersionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterMachinePackageVersionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterMachinePackageVersionSpec) DeepCopyInto(out *ClusterMachinePackageVersionSpec) {
	*out = *in
	if in.Packages != nil {
		in, out := &in.Packages, &out.Packages
		*out = make([]PackageConfig, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterMachinePackageVersionSpec.
func (in *ClusterMachinePackageVersionSpec) DeepCopy() *ClusterMachinePackageVersionSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterMachinePackageVersionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Machine) DeepCopyInto(out *Machine) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Machine.
func (in *Machine) DeepCopy() *Machine {
	if in == nil {
		return nil
	}
	out := new(Machine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Machine) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineDriverOptions) DeepCopyInto(out *MachineDriverOptions) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineDriverOptions.
func (in *MachineDriverOptions) DeepCopy() *MachineDriverOptions {
	if in == nil {
		return nil
	}
	out := new(MachineDriverOptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineList) DeepCopyInto(out *MachineList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Machine, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineList.
func (in *MachineList) DeepCopy() *MachineList {
	if in == nil {
		return nil
	}
	out := new(MachineList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachineList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePackageBetaPublish) DeepCopyInto(out *MachinePackageBetaPublish) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePackageBetaPublish.
func (in *MachinePackageBetaPublish) DeepCopy() *MachinePackageBetaPublish {
	if in == nil {
		return nil
	}
	out := new(MachinePackageBetaPublish)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachinePackageBetaPublish) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePackageBetaPublishList) DeepCopyInto(out *MachinePackageBetaPublishList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MachinePackageBetaPublish, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePackageBetaPublishList.
func (in *MachinePackageBetaPublishList) DeepCopy() *MachinePackageBetaPublishList {
	if in == nil {
		return nil
	}
	out := new(MachinePackageBetaPublishList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachinePackageBetaPublishList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePackageBetaPublishSpec) DeepCopyInto(out *MachinePackageBetaPublishSpec) {
	*out = *in
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.RandomPick != nil {
		in, out := &in.RandomPick, &out.RandomPick
		*out = new(intstr.IntOrString)
		**out = **in
	}
	if in.Machines != nil {
		in, out := &in.Machines, &out.Machines
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Packages != nil {
		in, out := &in.Packages, &out.Packages
		*out = make([]PackageConfig, len(*in))
		copy(*out, *in)
	}
	in.Strategy.DeepCopyInto(&out.Strategy)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePackageBetaPublishSpec.
func (in *MachinePackageBetaPublishSpec) DeepCopy() *MachinePackageBetaPublishSpec {
	if in == nil {
		return nil
	}
	out := new(MachinePackageBetaPublishSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePackageBetaPublishStatus) DeepCopyInto(out *MachinePackageBetaPublishStatus) {
	*out = *in
	if in.Machines != nil {
		in, out := &in.Machines, &out.Machines
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Upgrading != nil {
		in, out := &in.Upgrading, &out.Upgrading
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Succeeded != nil {
		in, out := &in.Succeeded, &out.Succeeded
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Failed != nil {
		in, out := &in.Failed, &out.Failed
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.RandomPick != nil {
		in, out := &in.RandomPick, &out.RandomPick
		*out = new(intstr.IntOrString)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePackageBetaPublishStatus.
func (in *MachinePackageBetaPublishStatus) DeepCopy() *MachinePackageBetaPublishStatus {
	if in == nil {
		return nil
	}
	out := new(MachinePackageBetaPublishStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePackageBetaPublishStrategy) DeepCopyInto(out *MachinePackageBetaPublishStrategy) {
	*out = *in
	if in.BatchPublish != nil {
		in, out := &in.BatchPublish, &out.BatchPublish
		*out = new(BatchPublish)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePackageBetaPublishStrategy.
func (in *MachinePackageBetaPublishStrategy) DeepCopy() *MachinePackageBetaPublishStrategy {
	if in == nil {
		return nil
	}
	out := new(MachinePackageBetaPublishStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePackageVersion) DeepCopyInto(out *MachinePackageVersion) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePackageVersion.
func (in *MachinePackageVersion) DeepCopy() *MachinePackageVersion {
	if in == nil {
		return nil
	}
	out := new(MachinePackageVersion)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachinePackageVersion) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePackageVersionList) DeepCopyInto(out *MachinePackageVersionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MachinePackageVersion, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePackageVersionList.
func (in *MachinePackageVersionList) DeepCopy() *MachinePackageVersionList {
	if in == nil {
		return nil
	}
	out := new(MachinePackageVersionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachinePackageVersionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePackageVersionSpec) DeepCopyInto(out *MachinePackageVersionSpec) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePackageVersionSpec.
func (in *MachinePackageVersionSpec) DeepCopy() *MachinePackageVersionSpec {
	if in == nil {
		return nil
	}
	out := new(MachinePackageVersionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineSpec) DeepCopyInto(out *MachineSpec) {
	*out = *in
	if in.PackageCustomConfigs != nil {
		in, out := &in.PackageCustomConfigs, &out.PackageCustomConfigs
		*out = make([]PackageCustomConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.MachineDriverOptions.DeepCopyInto(&out.MachineDriverOptions)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineSpec.
func (in *MachineSpec) DeepCopy() *MachineSpec {
	if in == nil {
		return nil
	}
	out := new(MachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineStatus) DeepCopyInto(out *MachineStatus) {
	*out = *in
	if in.Packages != nil {
		in, out := &in.Packages, &out.Packages
		*out = make([]PackageConfig, len(*in))
		copy(*out, *in)
	}
	if in.BetaPublishVersions != nil {
		in, out := &in.BetaPublishVersions, &out.BetaPublishVersions
		*out = make([]BetaPublishVersion, len(*in))
		copy(*out, *in)
	}
	if in.PackageConditions != nil {
		in, out := &in.PackageConditions, &out.PackageConditions
		*out = make([]PackageCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.ClusterInfo = in.ClusterInfo
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineStatus.
func (in *MachineStatus) DeepCopy() *MachineStatus {
	if in == nil {
		return nil
	}
	out := new(MachineStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PackageCondition) DeepCopyInto(out *PackageCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PackageCondition.
func (in *PackageCondition) DeepCopy() *PackageCondition {
	if in == nil {
		return nil
	}
	out := new(PackageCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PackageConfig) DeepCopyInto(out *PackageConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PackageConfig.
func (in *PackageConfig) DeepCopy() *PackageConfig {
	if in == nil {
		return nil
	}
	out := new(PackageConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PackageCustomConfig) DeepCopyInto(out *PackageCustomConfig) {
	*out = *in
	if in.CustomConfig != nil {
		in, out := &in.CustomConfig, &out.CustomConfig
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PackageCustomConfig.
func (in *PackageCustomConfig) DeepCopy() *PackageCustomConfig {
	if in == nil {
		return nil
	}
	out := new(PackageCustomConfig)
	in.DeepCopyInto(out)
	return out
}
