/*
Copyright 2017 The Kubernetes Authors.

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

package api

const (
	// When create, start or stop pod, use this annotation to represent desired state
	AnnotationContainerStateSpec = "pod.beta1.sigma.ali/container-state-spec"

	// When update or upgrade pod, store the old pod spec in this annotation
	// makes rollback controller can do rollback when operate failed
	AnnotationPodLastSpec = "pod.beta1.sigma.ali/last-spec"

	// Update result should be stored in this annotation
	AnnotationPodUpdateStatus = "pod.beta1.sigma.ali/update-status"

	// PodInplaceUpdateState is used to store inplace update state.
	// The state should be one of "created"/"accepted"/"failed".
	// https://yuque.antfin-inc.com/sys/sigma3.x/inplace-update-design-doc
	AnnotationPodInplaceUpdateState = "pod.beta1.sigma.ali/inplace-update-state"

	AnnotationLocalInfo = "node.beta1.sigma.ali/local-info"

	AnnotationPodAllocSpec = "pod.beta1.sigma.ali/alloc-spec"

	AnnotationPodNetworkStats = "pod.beta1.sigma.ali/network-status"

	AnnotationPodNetworkStatsHistory = "pod.beta1.sigma.ali/network-status-history"

	// numeric number of network priority
	// http://docs.alibaba-inc.com/pages/viewpage.action?pageId=479572415
	AnnotationNetPriority = "pod.beta1.sigma.ali/net-priority"

	// AnnotationPodSpecHash is a pod spec hash string provided by user
	AnnotationPodSpecHash = "pod.beta1.sigma.ali/pod-spec-hash"

	AnnotationPodRegisterNamingState = "pod.beta1.sigma.ali/naming-register-state"

	// AnnotationAutopilot is the prefix of autopilot service in node annotation
	AnnotationAutopilot = "node.beta1.sigma.ali/autopilot"

	// AnnotationDanglingPods records the dangling pods
	// Please refer to: https://lark.alipay.com/sys/sigma3.x/iqymrh
	AnnotationDanglingPods = "node.beta1.sigma.ali/dangling-pods"

	// AnnotationRebuildContainerInfo is container info which from sigma 2.0 container
	AnnotationRebuildContainerInfo = "pod.beta1.sigma.ali/rebuild-container-info"

	// AnnotationPodHostNameTemplate is pod hostname template which used to generate hostname.
	AnnotationPodHostNameTemplate = "pod.beta1.sigma.ali/hostname-template"

	// AnnotationNodeCPUSharePool is annotation key of the cpu share pool of Node API
	AnnotationNodeCPUSharePool = "node.beta1.sigma.ali/cpu-sharepool"

	// AnnotationContainerExtraConfig is annotation key of container's config defined by user
	AnnotationContainerExtraConfig = "pod.beta1.sigma.ali/container-extra-config"
)
