/*
Copyright 2018 The Alipay.com Inc Authors.

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

package v1alpha1

import (
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	InPlaceSetEventTypeGetPodSucc = "SuccessfulGetPod"
	InPlaceSetEventTypeGetPodFail = "FailedGetPod"

	InPlaceSetEventTypeCreatePodSucc = "SuccessfulCreatePod"
	InPlaceSetEventTypeCreatePodFail = "FailedCreatePod"

	InPlaceSetEventTypeDeletePodSucc = "SuccessfulDeletePod"
	InPlaceSetEventTypeDeletePodFail = "FailedDeletePod"

	InPlaceSetEventTypeUpdate     = "Update"
	InPlaceSetEventTypeUpdateSucc = "SuccessfulUpdate"
	InPlaceSetEventTypeUpdateFail = "FailedUpdate"
)

type InPlaceSetConditionType string

const (
	InPlaceSetReplicaFailure    InPlaceSetConditionType = "ReplicaFailure"
	InPlaceSetUpgradeFailure    InPlaceSetConditionType = "UpgradeFailure"
	InPlaceSetPodUpgradeFailure InPlaceSetConditionType = "InPlaceSetPodUpgradeFailure"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InPlaceSet
// +k8s:openapi-gen=true
// +resource:path=inplacesets,strategy=InPlaceSetStrategy
type InPlaceSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InPlaceSetSpec   `json:"spec,omitempty"`
	Status InPlaceSetStatus `json:"status,omitempty"`
}

// InPlaceSetSpec defines the desired state of InPlaceSet
type InPlaceSetSpec struct {
	// Replicas is the desired number of replicas of the given Template.
	// These are replicas in the sense that they are instantiations of the
	// same Template, but individual replicas also have a consistent identity.
	// If unspecified, defaults to 0.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Selector is a label query over pods that should match the replica count.
	// It must match the pod template's labels.
	Selector metav1.LabelSelector `json:"selector,omitempty"`

	// Template is the object that describes the pod that will be created if
	// insufficient replicas are detected. Each pod stamped out by the InPlaceSet
	// will fulfill this Template, but have a unique identity from the rest
	// of the InPlaceSet.
	Template corev1.PodTemplateSpec `json:"template,omitempty"`

	// UpgradeStrategy indicates the InPlaceSetUpdateStrategy that will be
	// employed to update Pods in the InPlaceSet when a revision is made to
	// Template.
	Strategy UpgradeStrategy `json:"strategy,omitempty"`

	// Minimum number of seconds for which a newly created pod should be ready
	// without any of its container crashing, for it to be considered available.
	// Defaults to 0 (pod will be considered available as soon as it is ready)
	// +optional
	MinReadySeconds int32 `json:"minReadySeconds,omitempty"`
}

type UpgradeStrategy struct {
	// Indicates that the deployment is paused and will not be processed by the
	// deployment controller.
	// +optional
	Pause bool `json:"pause,omitempty"`

	// Indicates the number of the pods in current version for which a upgrading
	// should be paused.
	// Defaults to 0 (all of pods will be upgraded)
	// +optional
	Partition int32 `json:"partition,omitempty"`
}

// InPlaceSetStatus defines the observed state of InPlaceSet
type InPlaceSetStatus struct {
	// ObservedGeneration is the most recent generation observed for this InPlaceSet. It corresponds to the
	// InPlaceSet's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// the number of scheduled replicas for the inPlaceSet
	// +optional
	ScheduledReplicas int32 `json:"scheduledReplicas,omitempty"`

	// The number of available replicas (ready for at least minReadySeconds) for this replica set.
	// +optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// Replicas is the most recently oberved number of replicas.
	Replicas int32 `json:"replicas,omitempty"`

	// The number of pods in current version
	UpdatedReplicas int32 `json:"updatedReplicas,omitempty"`

	// The number of ready current revision replicas for this InPlaceSet.
	// A pod is updated ready means all of its container has bean updated by sigma.
	// +optional
	UpdatedReadyReplicas int32 `json:"updatedReadyReplicas,omitempty"`

	// The number of available current revision replicas for this InPlaceSet.
	// A pod is updated available means the pod is ready for current revision and accessible
	// +optional
	UpdatedAvailableReplicas int32 `json:"updatedAvailableReplicas,omitempty"`

	// The number of pods that have labels matching the labels of the pod template of the InPlaceSet.
	// +optional
	FullyLabeledReplicas int32 `json:"fullyLabeledReplicas,omitempty"`

	// Represents the latest available observations of a InPlaceSet's current state.
	// +optional
	Conditions []InPlaceSetCondition `json:"conditions,omitempty"`
}

type InPlaceSetCondition struct {
	// Type of in place set condition.
	Type InPlaceSetConditionType `json:"type,omitempty"`

	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status,omitempty"`

	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"last_transition_time,omitempty"`

	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`

	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// DefaultingFunction sets default InPlaceSet field values
func (InPlaceSetSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*InPlaceSet)
	// Set default field values here
	DefaultingInPlaceSet(obj)
	log.Printf("Defaulting fields for InPlaceSet %s\n", obj.Name)
}

func DefaultingInPlaceSet(ips *InPlaceSet) {
	SetDefaults_PodSpec(&ips.Spec.Template.Spec)
}
