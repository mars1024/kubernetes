/*
Copyright 2019 The Alipay Authors.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A scope selector represents the AND of the selectors represented
// by the scope selector requirements.
type ScopeSelector struct {
	// A list of scope selector requirements by
	// labels of the k8s native resource quotas.
	// +optional
	MatchExpressions []ScopeSelectorRequirement `json:"matchExpressions,omitempty"`
}

// A scope selector requirement is a selector that contains values, a key,
// and an operator that relates the key and values.
type ScopeSelectorRequirement struct {
	// The name of the scope that the selector applies to.
	Key string `json:"key"`
	// Represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists, DoesNotExist.
	Operator ScopeSelectorOperator `json:"operator"`
	// An array of string values. If the operator is In or NotIn,
	// the values array must be non-empty. If the operator is Exists or DoesNotExist,
	// the values array must be empty.
	// This array is replaced during a strategic merge patch.
	// +optional
	Values []string `json:"values,omitempty"`
}

// A scope selector operator is the set of operators that can be used in
// a scope selector requirement.
type ScopeSelectorOperator string

const (
	ScopeSelectorOpIn           ScopeSelectorOperator = "In"
	ScopeSelectorOpNotIn        ScopeSelectorOperator = "NotIn"
	ScopeSelectorOpExists       ScopeSelectorOperator = "Exists"
	ScopeSelectorOpDoesNotExist ScopeSelectorOperator = "DoesNotExist"
)

// ClusterResourceQuotaSpec defines the desired state of ClusterResourceQuota
type ClusterResourceQuotaSpec struct {
	// Hard is the set of desired hard limits for each named resource.
	// More info: https://kubernetes.io/docs/concepts/policy/resource-quotas/
	// +optional
	Hard corev1.ResourceList `json:"hard,omitempty"`
	// SubQuotasShared here will force the sub-quotas share the desired hard limits.
	// Defaults to false.
	// +optional
	SubQuotasShared bool `json:"subQuotasShared,omitempty"`
	// ScopeSelector is a collection of filters that must match
	// each sub quotas tracked by a ClusterResourceQuota,
	// it is expressed using ScopeSelectorOperator in combination with possible values.
	ScopeSelector *ScopeSelector `json:"scopeSelector,omitempty"`
}

// ClusterResourceQuotaStatus defines the observed state of ClusterResourceQuota
type ClusterResourceQuotaStatus struct {
	// Hard is the set of enforced hard limits for each named resource.
	// More info: https://kubernetes.io/docs/concepts/policy/resource-quotas/
	// +optional
	Hard corev1.ResourceList `json:"hard,omitempty"`
	// Used is the current observed total usage of the resource of all sub-quotas.
	// +optional
	Used corev1.ResourceList `json:"used,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterResourceQuota is the Schema for the clusterresourcequota API
// +k8s:openapi-gen=true
type ClusterResourceQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterResourceQuotaSpec   `json:"spec,omitempty"`
	Status ClusterResourceQuotaStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterResourceQuotaList contains a list of ClusterResourceQuota
type ClusterResourceQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterResourceQuota `json:"items"`
}
