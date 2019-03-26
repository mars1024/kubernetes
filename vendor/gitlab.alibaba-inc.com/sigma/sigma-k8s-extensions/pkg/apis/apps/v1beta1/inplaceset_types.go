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

// InPlaceSetSpec defines the desired state of InPlaceSet
type InPlaceSetSpec struct {
	// Indicates that the InPlaceSet is paused.
	// +optional
	Paused bool `json:"paused,omitempty"`

	// list of pods to delete, value is the name of pod
	// +optional
	// +patchStrategy=merge
	PodsToDelete []string `json:"podsToDelete,omitempty" patchStrategy:"merge"`

	// replicas is the desired number of replicas of the given Template.
	// These are replicas in the sense that they are instantiations of the
	// same Template, but individual replicas also have a consistent identity.
	// If unspecified, defaults to 0.
	// TODO: Consider a rename of this field.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// selector is a label query over pods that should match the replica count.
	// It must match the pod template's labels.
	Selector *metav1.LabelSelector `json:"selector"`

	// template is the object that describes the pod that will be created if
	// insufficient replicas are detected. Each pod stamped out by the InPlaceSet
	// will fulfill this Template, but have a unique identity from the rest
	// of the InPlaceSet.
	Template v1.PodTemplateSpec `json:"template"`

	// volumeClaimTemplates is a list of claims that pods are allowed to reference.
	// The InPlaceSet controller is responsible for mapping network identities to
	// claims in a way that maintains the identity of a pod. Every claim in
	// this list must have at least one matching (by name) volumeMount in one
	// container in the template. A claim in this list takes precedence over
	// any volumes in the template, with the same name.
	// TODO: Define the behavior if a claim already exists with the same name.
	// +optional
	VolumeClaimTemplates []v1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`

	// serviceName is the name of the service that governs this InPlaceSet.
	// This service must exist before the InPlaceSet, and is responsible for
	// the network identity of the set. Pods get DNS/hostnames that follow the
	// pattern: pod-specific-string.serviceName.default.svc.cluster.local
	// where "pod-specific-string" is managed by the InPlaceSet controller.
	// +optional
	ServiceName string `json:"serviceName,omitempty"`

	// ScalePolicyType controls how pods are created during initial scale up,
	// when replacing pods on nodes, or when scaling down. The default policy is
	// `OrderedReady`, where pods are created in increasing order (pod-0, then
	// pod-1, etc) and the controller will wait until each pod is ready before
	// continuing. When scaling down, the pods are removed in the opposite order.
	// The alternative policy is `Parallel` which will create pods in parallel
	// to match the desired scale without waiting, and on scale down will delete
	// all pods at once.
	// +optional
	ScalePolicy ScalePolicyType `json:"scalePolicy,omitempty"`

	// upgradeStrategy indicates the InPlaceSetUpdateStrategy that will be
	// employed to update Pods in the InPlaceSet when a revision is made to
	// Template.
	UpgradeStrategy InPlaceUpgradeStrategy `json:"upgradeStrategy,omitempty"`

	// revisionHistoryLimit is the maximum number of revisions that will
	// be maintained in the InPlaceSet revision history. The revision history
	// consists of all revisions not represented by a currently applied
	// InPlaceSetSpec version. The default value is 10.
	RevisionHistoryLimit *int32 `json:"revisionHistoryLimit,omitempty"`

	// Minimum number of seconds for which a newly created pod should be ready
	// without any of its container crashing, for it to be considered available.
	// Defaults to 0 (pod will be considered available as soon as it is ready)
	// +optional
	MinReadySeconds int32 `json:"minReadySeconds,omitempty"`
}

// InPlaceSet policy for scale up and down.
type ScalePolicyType string

const (
	// InPlaceSet can only delete pods specified to be deleted in podControl
	ScaleDeleteOnly = "SpecifiedDeleteOnly"
	// InPlaceSet can always scale up and down, in order to keep status.replica equal to spec.replica
	ScaleAlways = "Always"
)

type InPlaceUpgradeStrategy struct {
	// Type indicates the type of the InPlaceUpgradeStrategy.
	// Default is Never.
	// +optional
	StrategyType InPlaceUpgradeStrategyType `json:"strategyType,omitempty"`
	// Max unavailable pod count for final-state upgrade.
	// +optional
	MaxUnavailable *int32 `json:"maxUnavailable,omitempty"`
	// Partition indicates the number at which the InPlaceSet should be
	// partitioned for publishing.
	// Default value is 0.
	// +optional
	Partition *int32 `json:"partition,omitempty"`
}

type InPlaceUpgradeStrategyType string

const (
	UpgradeNone = "None"
	// InPlaceSet will not update pods image
	UpgradeAfterUpgrade = "AfterUpgrade"
	// InPlaceSet will update pods image in batch
	UpgradeAlways = "Always"
)

// InPlaceSetStatus defines the observed state of InPlaceSet
type InPlaceSetStatus struct {
	// observedGeneration is the most recent generation observed for this InPlaceSet. It corresponds to the
	// InPlaceSet's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Replicas is the most recently oberved number of replicas.
	Replicas int32 `json:"replicas"`

	UpdatedReplicas int32 `json:"updatedReplicas,omitempty"`

	// The number of pods that have labels matching the labels of the pod template of the inplaceset.
	// +optional
	FullyLabeledReplicas int32 `json:"fullyLabeledReplicas,omitempty"`

	// The number of ready replicas for this replica set.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// The number of available replicas (ready for at least minReadySeconds) for this replica set.
	// +optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// Represents the latest available observations of a inplaceset's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []InPlaceSetCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

type InPlaceSetConditionType string

// These are valid conditions of a inplaceset.
const (
	// InPlaceSetReplicaFailure is added in a replica set when one of its pods fails to be created
	// due to insufficient quota, limit ranges, pod security policy, node selectors, etc. or deleted
	// due to kubelet being down or finalizers are failing.
	InPlaceSetReplicaFailure InPlaceSetConditionType = "ReplicaFailure"

	InPlaceSetUpgradeFailure InPlaceSetConditionType = "UpgradeFailure"
)

// InPlaceSetCondition describes the state of a InPlaceSet at a certain point.
type InPlaceSetCondition struct {
	// Type of InPlaceSet condition.
	Type InPlaceSetConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InPlaceSet is the Schema for the inplacesets API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type InPlaceSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InPlaceSetSpec   `json:"spec,omitempty"`
	Status InPlaceSetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InPlaceSetList contains a list of InPlaceSet
type InPlaceSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InPlaceSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InPlaceSet{}, &InPlaceSetList{})
}
