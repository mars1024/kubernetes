/*
Copyright 2018 The Kubernetes Authors.

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

// Package v1alpha1 contains definitions of Vertical Pod Autoscaler related objects.
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceProfileList is a list of ResourceProfile objects.
type ResourceProfileList struct {
	metav1.TypeMeta `json:",inline"`
	// metadata is the standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// items is the list of resource profile objects.
	Items []ResourceProfile `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceProfile is the configuration for a pod resource profile,
// which automatically manages pod resources based on historical and real time resource utilization.
type ResourceProfile struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the behavior of the resource profile.
	// More info: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status
	Spec ResourceProfileSpec `json:"spec" protobuf:"bytes,2,name=spec"`
}

// ResourceProfileSpec is the specification of the behavior of the resource profile
type ResourceProfileSpec struct {
	// A label query that determines the set of pods controlled by the resource profile
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	Selector *metav1.LabelSelector `json:"selector" protobuf:"bytes,1,name=selector"`

	// Describes whether pod resource update should be enabled.
	AutoPilot bool `json:"autoPilot,omitempty" protobuf:"bytes,2,opt,name=autoPilot"`

	// The most recently computed amount of resources recommended by the
	// autoscaler for the controlled pods.
	RecommendedResource *RecommendedPodResources `json:"recommendedResource,omitempty" protobuf:"bytes,3,opt,name=recommendedResource"`
}

// RecommendedPodResources is the recommendation of resources computed by
// autoscaler. It contains a recommendation for each container in the pod.
type RecommendedPodResources struct {
	// Resources recommended by the autoscaler for each container.
	// +optional
	ContainerRecommendations []RecommendedContainerResources `json:"containers,omitempty" protobuf:"bytes,1,rep,name=containers"`
}

// RecommendedContainerResources is the recommendation of resources computed by
// autoscaler for a specific container. Respects the container resource policy
// if present in the spec. In particular the recommendation is not produced for
// containers with `ContainerScalingMode` set to 'Off'.
type RecommendedContainerResources struct {
	// Name of the container.
	ContainerName string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Recommended amount of resources.
	Target corev1.ResourceList `json:"target" protobuf:"bytes,2,rep,name=target,casttype=ResourceList,castkey=ResourceName"`
}
