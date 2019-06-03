// +build !ignore_autogenerated

/*
Copyright 2019 The Alipay Authors.
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Cluster) DeepCopyInto(out *Cluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Cluster.
func (in *Cluster) DeepCopy() *Cluster {
	if in == nil {
		return nil
	}
	out := new(Cluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Cluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterComponentConfig) DeepCopyInto(out *ClusterComponentConfig) {
	*out = *in
	if in.Master != nil {
		in, out := &in.Master, &out.Master
		*out = make([]ComponentConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Custom != nil {
		in, out := &in.Custom, &out.Custom
		*out = make([]ComponentConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterComponentConfig.
func (in *ClusterComponentConfig) DeepCopy() *ClusterComponentConfig {
	if in == nil {
		return nil
	}
	out := new(ClusterComponentConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterCondition) DeepCopyInto(out *ClusterCondition) {
	*out = *in
	in.LastProbeTime.DeepCopyInto(&out.LastProbeTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterCondition.
func (in *ClusterCondition) DeepCopy() *ClusterCondition {
	if in == nil {
		return nil
	}
	out := new(ClusterCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterConfig) DeepCopyInto(out *ClusterConfig) {
	*out = *in
	if in.ApiserverDomains != nil {
		in, out := &in.ApiserverDomains, &out.ApiserverDomains
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ExtensionApiserverDomains != nil {
		in, out := &in.ExtensionApiserverDomains, &out.ExtensionApiserverDomains
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterConfig.
func (in *ClusterConfig) DeepCopy() *ClusterConfig {
	if in == nil {
		return nil
	}
	out := new(ClusterConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterList) DeepCopyInto(out *ClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Cluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterList.
func (in *ClusterList) DeepCopy() *ClusterList {
	if in == nil {
		return nil
	}
	out := new(ClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterSpec) DeepCopyInto(out *ClusterSpec) {
	*out = *in
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	in.Components.DeepCopyInto(&out.Components)
	in.ETCD.DeepCopyInto(&out.ETCD)
	out.PKI = in.PKI
	in.ClusterConfig.DeepCopyInto(&out.ClusterConfig)
	if in.ExternalUsers != nil {
		in, out := &in.ExternalUsers, &out.ExternalUsers
		*out = make([]ExternalUser, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Notify != nil {
		in, out := &in.Notify, &out.Notify
		*out = make([]NotifyConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.Log = in.Log
	if in.ServerSSLs != nil {
		in, out := &in.ServerSSLs, &out.ServerSSLs
		*out = make([]ServerSSL, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterSpec.
func (in *ClusterSpec) DeepCopy() *ClusterSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterStatus) DeepCopyInto(out *ClusterStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]ClusterCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ComponentVersionHistory != nil {
		in, out := &in.ComponentVersionHistory, &out.ComponentVersionHistory
		*out = make([]ClusterVersionHistory, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterStatus.
func (in *ClusterStatus) DeepCopy() *ClusterStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterVersion) DeepCopyInto(out *ClusterVersion) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterVersion.
func (in *ClusterVersion) DeepCopy() *ClusterVersion {
	if in == nil {
		return nil
	}
	out := new(ClusterVersion)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterVersion) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterVersionComponent) DeepCopyInto(out *ClusterVersionComponent) {
	*out = *in
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]ComponentVersionArg, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Templates != nil {
		in, out := &in.Templates, &out.Templates
		*out = make([]ComponentVersionTemplate, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterVersionComponent.
func (in *ClusterVersionComponent) DeepCopy() *ClusterVersionComponent {
	if in == nil {
		return nil
	}
	out := new(ClusterVersionComponent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterVersionHistory) DeepCopyInto(out *ClusterVersionHistory) {
	*out = *in
	in.Components.DeepCopyInto(&out.Components)
	in.AppliedAt.DeepCopyInto(&out.AppliedAt)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterVersionHistory.
func (in *ClusterVersionHistory) DeepCopy() *ClusterVersionHistory {
	if in == nil {
		return nil
	}
	out := new(ClusterVersionHistory)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterVersionList) DeepCopyInto(out *ClusterVersionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterVersion, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterVersionList.
func (in *ClusterVersionList) DeepCopy() *ClusterVersionList {
	if in == nil {
		return nil
	}
	out := new(ClusterVersionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterVersionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterVersionSpec) DeepCopyInto(out *ClusterVersionSpec) {
	*out = *in
	if in.Master != nil {
		in, out := &in.Master, &out.Master
		*out = make([]ClusterVersionComponent, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Custom != nil {
		in, out := &in.Custom, &out.Custom
		*out = make([]ClusterVersionComponent, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterVersionSpec.
func (in *ClusterVersionSpec) DeepCopy() *ClusterVersionSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterVersionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentConfig) DeepCopyInto(out *ComponentConfig) {
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentConfig.
func (in *ComponentConfig) DeepCopy() *ComponentConfig {
	if in == nil {
		return nil
	}
	out := new(ComponentConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentVersionArg) DeepCopyInto(out *ComponentVersionArg) {
	*out = *in
	if in.DefaultValue != nil {
		in, out := &in.DefaultValue, &out.DefaultValue
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentVersionArg.
func (in *ComponentVersionArg) DeepCopy() *ComponentVersionArg {
	if in == nil {
		return nil
	}
	out := new(ComponentVersionArg)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentVersionTemplate) DeepCopyInto(out *ComponentVersionTemplate) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentVersionTemplate.
func (in *ComponentVersionTemplate) DeepCopy() *ComponentVersionTemplate {
	if in == nil {
		return nil
	}
	out := new(ComponentVersionTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DNSRRService) DeepCopyInto(out *DNSRRService) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DNSRRService.
func (in *DNSRRService) DeepCopy() *DNSRRService {
	if in == nil {
		return nil
	}
	out := new(DNSRRService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ETCDConfig) DeepCopyInto(out *ETCDConfig) {
	*out = *in
	in.DataVolume.DeepCopyInto(&out.DataVolume)
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = new(ServiceConfig)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ETCDConfig.
func (in *ETCDConfig) DeepCopy() *ETCDConfig {
	if in == nil {
		return nil
	}
	out := new(ETCDConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ETCDDataVolume) DeepCopyInto(out *ETCDDataVolume) {
	*out = *in
	if in.ClaimTemplate != nil {
		in, out := &in.ClaimTemplate, &out.ClaimTemplate
		*out = new(v1.PersistentVolumeClaimSpec)
		(*in).DeepCopyInto(*out)
	}
	in.Source.DeepCopyInto(&out.Source)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ETCDDataVolume.
func (in *ETCDDataVolume) DeepCopy() *ETCDDataVolume {
	if in == nil {
		return nil
	}
	out := new(ETCDDataVolume)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalUser) DeepCopyInto(out *ExternalUser) {
	*out = *in
	if in.Groups != nil {
		in, out := &in.Groups, &out.Groups
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalUser.
func (in *ExternalUser) DeepCopy() *ExternalUser {
	if in == nil {
		return nil
	}
	out := new(ExternalUser)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HeadlessService) DeepCopyInto(out *HeadlessService) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HeadlessService.
func (in *HeadlessService) DeepCopy() *HeadlessService {
	if in == nil {
		return nil
	}
	out := new(HeadlessService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogConfig) DeepCopyInto(out *LogConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogConfig.
func (in *LogConfig) DeepCopy() *LogConfig {
	if in == nil {
		return nil
	}
	out := new(LogConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotifyConfig) DeepCopyInto(out *NotifyConfig) {
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotifyConfig.
func (in *NotifyConfig) DeepCopy() *NotifyConfig {
	if in == nil {
		return nil
	}
	out := new(NotifyConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PKIConfig) DeepCopyInto(out *PKIConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PKIConfig.
func (in *PKIConfig) DeepCopy() *PKIConfig {
	if in == nil {
		return nil
	}
	out := new(PKIConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServerSSL) DeepCopyInto(out *ServerSSL) {
	*out = *in
	if in.Hosts != nil {
		in, out := &in.Hosts, &out.Hosts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServerSSL.
func (in *ServerSSL) DeepCopy() *ServerSSL {
	if in == nil {
		return nil
	}
	out := new(ServerSSL)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceConfig) DeepCopyInto(out *ServiceConfig) {
	*out = *in
	if in.Headless != nil {
		in, out := &in.Headless, &out.Headless
		*out = new(HeadlessService)
		**out = **in
	}
	if in.DNSRR != nil {
		in, out := &in.DNSRR, &out.DNSRR
		*out = new(DNSRRService)
		**out = **in
	}
	if in.VIPServer != nil {
		in, out := &in.VIPServer, &out.VIPServer
		*out = new(VIPServerService)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceConfig.
func (in *ServiceConfig) DeepCopy() *ServiceConfig {
	if in == nil {
		return nil
	}
	out := new(ServiceConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VIPServerService) DeepCopyInto(out *VIPServerService) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VIPServerService.
func (in *VIPServerService) DeepCopy() *VIPServerService {
	if in == nil {
		return nil
	}
	out := new(VIPServerService)
	in.DeepCopyInto(out)
	return out
}
