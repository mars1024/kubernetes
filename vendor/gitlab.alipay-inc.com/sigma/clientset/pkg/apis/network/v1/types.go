/*
Copyright 2019 The Alipay Authors.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InterfaceSpec defines the desired state of Interface
type InterfaceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	NodeName              string                `json:"nodeName,omitempty"`
	VSwitchSelector       *metav1.LabelSelector `json:"vSwitchSelector"`
	SecurityGroupSelector *metav1.LabelSelector `json:"securityGroupSelector"`
	ReserveExpireTime     *metav1.Time          `json:"releaseTime,omitempty"`
	AutoDetachWithoutPod  bool                  `json:"autoDetachWithoutPod,omitempty"`

	StaticPrivateIPAddress string `json:"staticPrivateIPAddress,omitempty"`

	// options
	Tags          []InterfaceTag `json:"tags,omitempty"`
	InterfaceName string         `json:"interfaceName,omitempty"`
}

type InterfaceID string

type InterfaceTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// InterfaceStatus defines the observed state of Interface
type InterfaceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	InterfaceID      InterfaceID `json:"interfaceID,omitempty"`
	Status           string      `json:"status,omitempty"`
	AutoDetachStatus string      `json:"autoDetachStatus,omitempty"`
	VSwitchID        VSwitchID   `json:"vSwitchID,omitempty"`
	PrivateIPAddress string      `json:"privateIPAddress,omitempty"`
	PrefixLength     int32       `json:"networkPrefixLength,omitempty"`
	MacAddress       string      `json:"macAddress,omitempty"`
	Gateway          string      `json:"gateway,omitempty"`
	SecurityGroupIDs []string    `json:"securityGroupIDs,omitempty"`
}

const (
	InterfaceStatusCreating  = "Creating"
	InterfaceStatusAttached  = "Attaching"
	InterfaceStatusDeleting  = "Deleting"
	InterfaceStatusAvailable = "Available"
	InterfaceStatusInUse     = "InUse"

	InterfaceDetachStatusDetaching = "Detaching"
	InterfaceDetachStatusDetached  = "Detached"
	InterfaceDetachStatusAttached  = "Attached"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Interface is the Schema for the interfaces API
// +k8s:openapi-gen=true
type Interface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InterfaceSpec   `json:"spec,omitempty"`
	Status InterfaceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InterfaceList contains a list of Interface
type InterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Interface `json:"items"`
}

// PredictedInterfaceSpec defines the desired state of PredictedInterface
type PredictedInterfaceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	NodeSelector           *metav1.LabelSelector `json:"nodeSelector"`
	Replicas               int                   `json:"replicas"`
	VSwitchSelector        *metav1.LabelSelector `json:"vSwitchSelector"`
	SecurityGroupSelector  *metav1.LabelSelector `json:"securityGroupSelector"`
	NetworkInterfaceLabels map[string]string     `json:"networkInterfaceLabels"`
}

type PredictedInterfacePhase string

const (
	PredictedInterfacePhaseCreatingInterfaces PredictedInterfacePhase = "CreatingInterfaces"

	PredictedInterfacePhaseWaitingForInterfacesReady PredictedInterfacePhase = "WaitingForInterfacesReady"

	PredictedInterfacePhaseStable PredictedInterfacePhase = "Stable"
)

// PredictedInterfaceStatus defines the observed state of PredictedInterface
type PredictedInterfaceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Results []NodeInterfaceResult `json:"results,omitempty"`

	Phase PredictedInterfacePhase `json:"phase,omitempty"`
}

type NodeInterfaceResult struct {
	NodeName   string   `json:"nodeName,omitempty"`
	Interfaces []string `json:"interfaces,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PredictedInterface is the Schema for the predictedinterfaces API
// +k8s:openapi-gen=true
type PredictedInterface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PredictedInterfaceSpec   `json:"spec,omitempty"`
	Status PredictedInterfaceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PredictedInterfaceList contains a list of PredictedInterface
type PredictedInterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PredictedInterface `json:"items"`
}

// SecurityGroupSpec defines the desired state of SecurityGroup
type SecurityGroupSpec struct {
	VpcID             string `json:"vpcID"`
	Description       string `json:"description,omitempty"`
	SecurityGroupName string `json:"securityGroupName,omitempty"`
}

// SecurityGroupStatus defines the observed state of SecurityGroup
type SecurityGroupStatus struct {
	SecurityGroupID string `json:"securityGroupID,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SecurityGroup is the Schema for the securitygroups API
// +k8s:openapi-gen=true
type SecurityGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecurityGroupSpec   `json:"spec,omitempty"`
	Status SecurityGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SecurityGroupList contains a list of SecurityGroup
type SecurityGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecurityGroup `json:"items"`
}

// VSwitchSpec defines the desired state of VSwitch
type VSwitchSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	RegionID    string `json:"regionID"`
	VpcID       string `json:"vpcID"`
	VSwitchName string `json:"vSwitchName"`
	CIDRBlock   string `json:"cidrBlock"`
}

type VSwitchID string

// VSwitchStatus defines the observed state of VSwitch
type VSwitchStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	AvailableIPAddressCount int `json:"availableIPAddressCount,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VSwitch is the Schema for the vswitches API
// +k8s:openapi-gen=true
type VSwitch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VSwitchSpec   `json:"spec,omitempty"`
	Status VSwitchStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VSwitchList contains a list of VSwitch
type VSwitchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VSwitch `json:"items"`
}
