/*
Copyright 2019 The Alipay Authors.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Bundle is the minimum set of pods
type Bundle struct {
	// Minimum number of
	MinNumber int32 `json:"minNumber"`
	// Label selector for pods. Existing ReplicaSets whose pods are
	// selected by this will be the ones affected by this podgroup.
	// It must match the pod template's labels.
	Selector *metav1.LabelSelector `json:"selector"`
	// The bundle strategy to use to schedule pods.
	// +optional
	Strategy *BundleStrategy `json:"strategy,omitempty"`
}

// OneNodeStrategy contains strategies for one node.
type OneNodeStrategy struct {
	// If specified, the podGroup's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// If specified, the podGroup's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// The Quality of Service (QOS) classification assigned to the podGroup based on sigma qos class definition
	// See SigmaQOSClass type for available QOS classes
	// +optional
	QOSClass string `json:"qosClass,omitempty"`
	// If specified, indicates the podGroup's priority. "system-node-critical" and
	// "system-cluster-critical" are two special keywords which indicate the
	// highest priorities with the former being the highest priority. Any other
	// name must be defined by creating a PriorityClass object with that name.
	// If not specified, the podGroup priority will be default or zero if there is no
	// default.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
	// The priority value. Various system components use this field to find the
	// priority of the podGroup. When Priority Admission Controller is enabled, it
	// prevents users from setting this field. The admission controller populates
	// this field from PriorityClassName.
	// The higher the value, the higher the priority.
	// +optional
	Priority *int32 `json:"priority,omitempty"`
}

// BundleStrategy describes how to schedule a bundle if defined.
type BundleStrategy struct {
	// Schedule behavior of a bundle. "true" means all the pods should be scheduled to one node.
	ShouldScheduleToOneNode bool `json:"shouldScheduleToOneNode"`
	// OneNode contains strategies for one node. Works only if ShouldScheduleToOneNode is true.
	OneNode *OneNodeStrategy `json:"oneNode"`
}

// PodGroupSpec defines the desired state of PodGroup
type PodGroupSpec struct {
	// Bundles of the podGroup
	Bundles []Bundle `json:"bundles"`
}

// PodGroupConditionType is a valid value for PodGroupCondition.Type
type PodGroupConditionType string

// These are valid conditions of podGroup.
const (
	// PodGroupScheduled represents status of the scheduling process for this podGroup.
	PodGroupScheduled PodGroupConditionType = "Scheduled"
	// PodGroupReady means the podGroup is able to service requests and should be added to the
	// load balancing pools of all matching services.
	PodGroupReady PodGroupConditionType = "Ready"
	// PodGroupLegal means the spec of podGroup is legal.
	PodGroupLegal PodGroupConditionType = "Legal"
)

type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in the condition.
// "ConditionFalse" means a resource is not in the condition. "ConditionUnknown" means kubernetes
// can't decide if a resource is in the condition or not. In the future, we could add other
// intermediate conditions, e.g. ConditionDegraded.
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

// PodGroupCondition contains details for the current condition of this podGroup.
type PodGroupCondition struct {
	// Type is the type of the condition.
	// Currently only Ready.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-conditions
	Type PodGroupConditionType `json:"type"`
	// Status is the status of the condition.
	// Can be True, False, Unknown.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-conditions
	Status ConditionStatus `json:"status"`
	// Last time we probed the condition.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// BundleStatus describes the status of a bundle.
type BundleStatus struct {
	// Total number of pods which belong to this bundle.
	TotalPods int32 `json:"totalPods"`
	// Total number of pods which are scheduled.
	ScheduledPods int32 `json:"scheduledPods"`
	// Total number of pods which failed to schedule.
	SchedulingFailedPods int32 `json:"schedulingFailedPods"`
	// Total number of pods which are running.
	RunningPods int32 `json:"runningPods"`
	// Total number of pods which are completed. This is only for jobs.
	CompletedPods int32 `json:"completedPods"`
}

// PodGroupStatus defines the observed state of PodGroup
type PodGroupStatus struct {
	// Current service state of podGroup.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []PodGroupCondition `json:"conditions,omitempty"`
	// Current service state of bundles.
	Bundles []BundleStatus `json:"bundles,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodGroup is the Schema for the podgroups API
// +k8s:openapi-gen=true
type PodGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodGroupSpec   `json:"spec,omitempty"`
	Status PodGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodGroupList contains a list of PodGroup
type PodGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodGroup `json:"items"`
}
