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
	"k8s.io/api/core/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// ClusterResourceQuota
// +k8s:openapi-gen=true
// +resource:path=clusterresourcequotas,strategy=ClusterResourceQuotaStrategy
type ClusterResourceQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired quota
	Spec ClusterResourceQuotaSpec `json:"spec,omitempty"`

	// Status defines the actual enforced quota and its current usage
	Status ClusterResourceQuotaStatus `json:"status,omitempty"`
}

// ClusterResourceQuotaSpec defines the desired quota restrictions
type ClusterResourceQuotaSpec struct {
	// Selector is the selector used to match namespaces.
	Selector ClusterResourceQuotaSelector `json:"selector"`
	// Quota defines the desired quota
	Quota v1.ResourceQuotaSpec `json:"quota"`
}

// ClusterResourceQuotaSelector is used to select namespaces.
type ClusterResourceQuotaSelector struct {
	AnnotationSelector map[string]string `json:"annotationSelector,omitempty"`
}

// ClusterResourceQuotaStatus defines the actual enforced quota and its current usage
type ClusterResourceQuotaStatus struct {
	// Total defines the actual enforced quota and its current usage across all namespaces
	Total v1.ResourceQuotaStatus `json:"total"`

	// Namespaces slices the usage by namespace.  This division allows for quick resolution of
	// deletion reconciliation inside of a single project without requiring a recalculation
	// across all projects.  This map can be used to pull the deltas for a given project.
	Namespaces []NamespaceResourceQuotaStatus `json:"namespaces"`
}

// NamespaceResourceQuotaStatus contains details for the current status of this namespace
type NamespaceResourceQuotaStatus struct {
	Name   string                 `json:"name"`
	Status v1.ResourceQuotaStatus `json:"status"`
}

// DefaultingFunction sets default ClusterResourceQuota field values
func (ClusterResourceQuotaSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*ClusterResourceQuota)
	// set default field values here
	log.Printf("Defaulting fields for ClusterResourceQuota %s\n", obj.Name)
}
