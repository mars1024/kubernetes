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
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkType describes a cluster's network type.
// Only one of the following network type may be specified.
type NetworkType string

const (
	VPCNetwork     NetworkType = "VPC"
	ClassicNetwork NetworkType = "CLASSIC"
	VLANNetwork    NetworkType = "VLAN"
)

const (
	ClusterCoreDNSConditionType             MinionClusterConditionType = "CoreDNS"
	ClusterBasicMasterResourceConditionType MinionClusterConditionType = "MasterResources"
	ClusterBasicRBACPoliciesConditionType   MinionClusterConditionType = "RBACPolicies"
)

type MinionClusterPhase string

// These are the valid phases of a cluster.
const (
	// ClusterInitializing means the cluster is available for use in the system
	ClusterInitializing MinionClusterPhase = "Initializing"
	// ClusterActive means the cluster is available for use in the system
	ClusterActive MinionClusterPhase = "Active"
	// ClusterTerminating means the cluster is undergoing graceful termination
	ClusterTerminating MinionClusterPhase = "Terminating"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MinionCluster
// +k8s:openapi-gen=true
// +resource:path=minionclusters,strategy=MinionClusterStrategy
type MinionCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MinionClusterSpec   `json:"spec,omitempty"`
	Status MinionClusterStatus `json:"status,omitempty"`
}

// MinionClusterSpec defines the desired state of MinionCluster
type MinionClusterSpec struct {
	// Networking holds configuration for the networking topology of the cluster.
	// +optional
	Networking *Networking `json:"networking,omitempty" protobuf:"bytes,1,opt,name=networking"`

	// SecurityProfile holds configuration for the security configuration of the cluster.
	// +optional
	SecurityProfile SecurityProfile `json:"securityProfile,omitempty" protobuf:"bytes,5,opt,name=securityProfile"`
}

// Networking contains elements describing minion cluster's networking configuration.
type Networking struct {
	// Required. Network type of this minion cluster.
	// One of VPC, CLASSIC, VLAN
	// Default to CLASSIC
	NetworkType NetworkType `json:"networkType,omitempty" protobuf:"bytes,1,opt,name=networkType,casttype=NetworkType"`

	// Required: A CIDR notation IP range from which to assign service cluster IPs. This must not
	// overlap with any IP ranges assigned to nodes for pods.
	ServiceClusterIPRange string `json:"serviceClusterIPRange,omitempty" protobuf:"bytes,2,opt,name=serviceClusterIPRange"`

	// Required: A port range to reserve for services with NodePort visibility.
	// Example: {base: 30000, size: 2767}. Inclusive at both ends of the range.
	ServiceNodePortRange PortRange `json:"serviceNodePortRange,omitempty" protobuf:"bytes,3,opt,name=serviceNodePortRange"`

	// PodIPRange defines IP ranges assigned to nodes for pods. This must not overlap with service
	// cluster IPs and this IP range should be zone scope.
	// +optional
	PodIPRange map[string]string `json:"podIPRange,omitempty" protobuf:"bytes,4,rep,name=podIPRange"`

	// DNS holds configuration for DNS.
	// +optional
	DNS *DNS `json:"dns,omitempty" protobuf:"bytes,5,opt,name=dns"`

	// MasterEndpointIP is an endpoint by which clients accesses kubernetes master.
	MasterEndpointIP string `json:"masterEndpointIP,omitempty" protobuf:"bytes,6,opt,name=masterEndpointIP"`
}

// PortRange represents a range of TCP/UDP ports. To represent a single port,
// set Size to 1.
type PortRange struct {
	Base int32 `json:"base,omitempty" protobuf:"varint,1,name=base"`
	Size int32 `json:"size,omitempty" protobuf:"varint,2,name=size"`
}

// DNS contains elements describing DNS configuration
type DNS struct {
	// Local provides configuration knobs for configuring the local dns.
	Local *LocalDNS `json:"local,omitempty" protobuf:"bytes,1,name=local"`

	// External describes how to connect to an external dns.
	// Local and External should be mutually exclusive
	External *ExternalDNS `json:"external,omitempty" protobuf:"bytes,2,name=external"`
}

// LocalDNS describes that minion cluster operator should run an dns locally
type LocalDNS struct {
	// ClusterIP specifies the cluster ip to use for DNS service. This must within
	// service cluster ip range.
	ClusterIP string `json:"clusterIP,omitempty" protobuf:"bytes,1,opt,name=clusterIP"`

	// Image specifies which container image to use for running dns.
	Image string `json:"image" protobuf:"bytes,2,opt,name=image"`
}

// ExternalDNS describes and external DNS service
type ExternalDNS struct {
}

// SecurityProfile contains elements describing how to configure cluster security.
type SecurityProfile struct {
}

// MinionClusterStatus is information about the current status of a MinionCluster
type MinionClusterStatus struct {
	// Phase is the current lifecycle phase of the cluster.
	// +optional
	Phase MinionClusterPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=MinionClusterPhase"`

	// ObservedGeneration reflects the generation of the most recently observed minion cluster.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,2,opt,name=observedGeneration"`

	// Represents the latest available observations of a minion cluster's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []MinionClusterCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,3,rep,name=conditions"`
}

type MinionClusterConditionType string

const (
	MinionClusterSetupDNSFailure MinionClusterConditionType = "SetupDNSFailure"
)

// MinionClusterCondition describes the state of a minion cluster at a certain point.
type MinionClusterCondition struct {
	// Type of replication controller condition.
	Type MinionClusterConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=MinionClusterConditionType"`

	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=k8s.io/api/core/v1.ConditionStatus"`

	// The last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`

	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

// DefaultingFunction sets default MinionCluster field values
func (MinionClusterSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*MinionCluster)
	// set default field values here
	log.Printf("Defaulting fields for MinionCluster %s\n", obj.Name)
	SetDefaults_MinionCluster(obj)
}
