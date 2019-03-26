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

package v1beta1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CapacityPreviewSpec defines the desired state of resource requirement spec
type CapacityPreviewSpec struct {
	// Template is the object that describes the pod that will be created.
	Template v1.PodTemplateSpec `json:"template"`
	// Number of pods to create.
	Replicas *int32 `json:"replicas,omitempty"`
}

type CapacityPreviewStatus struct {
	// indicate the preivew's status
	Phase PreviewPhase `json:"phase"`

	// if AvailableReplicas < Replicas, show the scheduler error.
	FailedReasons []*FailedReason `json:"failedReasons,omitempty"`

	// Number of pods can be create
	AvailableReplicas int32

	// Key:NodeName, Value: how many pod can be allocate on the node.
	PreviewAllocatedItems []*PreviewAllocatedItem `json:"previewAllocatedItems,omitempty"`
}

type FailedReason struct {
	ErrorCode string
	NodeNames []string // the nodeName of the same ErrorCode
}

type PreviewAllocatedItem struct {
	NodeName      string
	AllocateCount int32
}

type PreviewPhase string

const (
	PreviewPhasePending   PreviewPhase = "Pending" // wait for process
	PreviewPhaseCompleted PreviewPhase = "Completed"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CapacityPreview is the Schema for the capacitypreviews API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type CapacityPreview struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CapacityPreviewSpec   `json:"spec,omitempty"`
	Status CapacityPreviewStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CapacityPreviewList contains a list of CapacityPreview
type CapacityPreviewList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CapacityPreview `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CapacityPreview{}, &CapacityPreviewList{})
}
