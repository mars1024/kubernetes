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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// A ResourceQuotaScope defines a filter that must match each object tracked by a quota
type ResourceQuotaScope string

const (
	// Match all pod objects where spec.activeDeadlineSeconds
	ResourceQuotaScopeTerminating ResourceQuotaScope = "Terminating"
	// Match all pod objects where !spec.activeDeadlineSeconds
	ResourceQuotaScopeNotTerminating ResourceQuotaScope = "NotTerminating"
	// Match all pod objects that have best effort quality of service
	ResourceQuotaScopeBestEffort ResourceQuotaScope = "BestEffort"
	// Match all pod objects that do not have best effort quality of service
	ResourceQuotaScopeNotBestEffort ResourceQuotaScope = "NotBestEffort"
	// Match all pod objects that have priority class mentioned
	ResourceQuotaScopePriorityClass ResourceQuotaScope = "PriorityClass"
)

// ResourceList is a set of (resource name, quantity) pairs.
type ResourceList map[string]resource.Quantity

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// ClusterResourceQuota
// +k8s:openapi-gen=true
// +resource:path=clusterresourcequota,strategy=ClusterResourceQuotaStrategy
type ClusterResourceQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterResourceQuotaSpec   `json:"spec,omitempty"`
	Status ClusterResourceQuotaStatus `json:"status,omitempty"`
}

// ClusterResourceQuotaSpec defines the desired quota restrictions
type ClusterResourceQuotaSpec struct {
	// hard is the set of desired hard limits for each named resource.
	// More info: https://kubernetes.io/docs/concepts/policy/resource-quotas/
	// +optional
	Hard map[string]resource.Quantity `json:"hard,omitempty" protobuf:"bytes,1,rep,name=hard"`
	// A collection of filters that must match each object tracked by a quota.
	// If not specified, the quota matches all objects.
	// +optional
	Scopes []ResourceQuotaScope `json:"scopes,omitempty" protobuf:"bytes,2,rep,name=scopes,casttype=ResourceQuotaScope"`
	// scopeSelector is also a collection of filters like scopes that must match each object tracked by a quota
	// but expressed using ScopeSelectorOperator in combination with possible values.
	// For a resource to match, both scopes AND scopeSelector (if specified in spec), must be matched.
	// +optional
	ScopeSelector *ScopeSelector `json:"scopeSelector,omitempty" protobuf:"bytes,3,opt,name=scopeSelector"`
}

// A scope selector represents the AND of the selectors represented
// by the scoped-resource selector requirements.
type ScopeSelector struct {
	// A list of scope selector requirements by scope of the resources.
	// +optional
	MatchExpressions []ScopedResourceSelectorRequirement `json:"matchExpressions,omitempty" protobuf:"bytes,1,rep,name=matchExpressions"`
}

// A scoped-resource selector requirement is a selector that contains values, a scope name, and an operator
// that relates the scope name and values.
type ScopedResourceSelectorRequirement struct {
	// The name of the scope that the selector applies to.
	ScopeName ResourceQuotaScope `json:"scopeName" protobuf:"bytes,1,opt,name=scopeName"`
	// Represents a scope's relationship to a set of values.
	// Valid operators are In, NotIn, Exists, DoesNotExist.
	Operator ScopeSelectorOperator `json:"operator" protobuf:"bytes,2,opt,name=operator,casttype=ScopedResourceSelectorOperator"`
	// An array of string values. If the operator is In or NotIn,
	// the values array must be non-empty. If the operator is Exists or DoesNotExist,
	// the values array must be empty.
	// This array is replaced during a strategic merge patch.
	// +optional
	Values []string `json:"values,omitempty" protobuf:"bytes,3,rep,name=values"`
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

// ClusterResourceQuotaStatus defines the actual enforced quota and its current usage
type ClusterResourceQuotaStatus struct {
	// Total defines the actual enforced quota and its current usage across all namespaces
	Total ResourceQuotaStatus

	// Namespaces slices the usage by namespace.  This division allows for quick resolution of
	// deletion reconciliation inside of a single project without requiring a recalculation
	// across all projects.  This map can be used to pull the deltas for a given project.
	Namespaces ResourceQuotasStatusByNamespace
}

// ResourceQuotaStatus defines the enforced hard limits and observed use.
type ResourceQuotaStatus struct {
	// Hard is the set of enforced hard limits for each named resource.
	// More info: https://kubernetes.io/docs/concepts/policy/resource-quotas/
	// +optional
	Hard map[string]resource.Quantity `json:"hard,omitempty" protobuf:"bytes,1,rep,name=hard"`
	// Used is the current observed total usage of the resource in the namespace.
	// +optional
	Used map[string]resource.Quantity `json:"used,omitempty" protobuf:"bytes,2,rep,name=used"`
}

// ResourceQuotasStatusByNamespace provides type correct methods
type ResourceQuotasStatusByNamespace struct {
	//orderedMap orderedMap
}

// DefaultingFunction sets default ClusterResourceQuota field values
func (ClusterResourceQuotaSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*ClusterResourceQuota)
	// set default field values here
	log.Printf("Defaulting fields for ClusterResourceQuota %s\n", obj.Name)
}
