/*
Copyright 2018 The Alipay Authors.
*/

package v1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Machine is a specification for a Machine resource
type Machine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSpec   `json:"spec"`
	Status MachineStatus `json:"status"`
}

// MachineSpec is the spec for a Machine resource
type MachineSpec struct {
	IP                   string                `json:"ip"`
	SN                   string                `json:"sn"`
	IDC                  string                `json:"idc"`
	Versions             map[string]string     `json:"versions"`
	Taints               []v1.Taint            `json:"taints,omitempty"`
	PackageCustomConfigs []PackageCustomConfig `json:"packageCustomConfigs"`
}

type PackageCustomConfig struct {
	PackageName  string            `json:"packageName"`
	CustomConfig map[string]string `json:"customConfig"`
	Disable      bool              `json:"disable"`
}

// MachineStatus is the status for a Machine resource
type MachineStatus struct {
	Phase MachinePhase `json:"phase"`

	// A human readable message indicating details about why the machine is in this condition.
	// +optional
	Message string `json:"message"`

	// machine package suite versions of the machine
	Versions map[string]string `json:"versions"`

	// A brief CamelCase message indicating details about why the machine is in this state.
	// e.g. 'Evicted'
	// +optional
	Reason string `json:"reason"`

	Packages []PackageConfig `json:"packages"`

	// The conditions of components of this machine
	ComponentConditions []ComponentCondition `json:"componentConditions"`

	// The readiness gates of this machine
	ReadinessGates []MachineReadinessGate `json:"readinessGates"`
}

type MachineReadinessGate struct {
	ConditionType ComponentConditionType `json:"conditionType"`
}

// ComponentConditions describes the state of a component at a certain point.
type ComponentCondition struct {
	// Type of component condition.
	Type ComponentConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	// These are valid condition statuses. "ConditionTrue" means a resource is in the condition.
	// "ConditionFalse" means a resource is not in the condition. "ConditionUnknown" means kubernetes
	Status v1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// The reason for the condition's last transition.
	Reason string `json:"reason"`
	// A human readable message indicating details about the transition.
	Message string `json:"message"`
}

type ComponentConditionType string

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineList is a list of Machine resources
type MachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Machine `json:"items"`
}

type PackageConfig struct {
	PackageName string            `json:"packageName"`
	VersionName string            `json:"versionName"`
	Config      map[string]string `json:"config"`
}

type ClusterInfo struct {
	CaCert         string `json:"caCert"`
	Apiserver      string `json:"apiserver"`
	BootstrapToken string `json:"bootstrapToken"`
}

type MachinePhase string

const (
	// Pending means the Machine has been accepted by the system, package(s) are installed correctly
	MachinePending MachinePhase = "Pending"

	// Running means the Node which the Machine present is running.
	MachineRunning MachinePhase = "Running"

	// Running means the Node which the Machine present is in error state, which can not recover successfully.
	MachineError MachinePhase = "Error"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachinePackageVersion is a specification for a Package Version resource
type MachinePackageVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec MachinePackageVersionSpec `json:"spec"`
}

type MachinePackageVersionSpec struct {
	PackageName    string             `json:"packageName" yaml:"packageName"`
	PackageVersion string             `json:"packageVersion" yaml:"packageVersion"`
	ConfigMaps     []PackageConfigMap `json:"configMaps" yaml:"configMaps"`
	Config         map[string]string  `json:"config" yaml:"config"`
}

type PackageConfigMap struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachinePackageVersionList is a list of MachinePackageVersion resources
type MachinePackageVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MachinePackageVersion `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachinePackage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MachinePackageSpec `json:"spec"`
}

type MachinePackageSpec struct {
	Package PackageConfig `json:"package"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachinePackageVersionList is a list of MachinePackageVersion resources
type MachinePackageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MachinePackage `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineOps is a specification for a MachineOps resource
type MachineOps struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineOpsSpec   `json:"spec"`
	Status MachineOpsStatus `json:"status"`
}

// MachineOpsSpec is the spec for a MachineOps resource
type MachineOpsSpec struct {
	// Required. Type of this ops operation, should be same
	// as OpsType name.
	Type string `json:"type"`
	// Indicates that the MachineOps is paused.
	// +optional
	Paused bool `json:"paused,omitempty"`
	// List of ops action step to skip.
	// +optional
	SkipSteps []string `json:"skipSteps,omitempty"`
	// Default time out number of seconds after which waiting for the action finished and return.
	// +optional
	DefaultTimeoutSeconds int32 `json:"defaultTimeoutSeconds,omitempty"`
	// RetrySteps of one Ops, contains the name of the ops actions and start time.
	RetrySteps RetrySteps `json:"retrySteps,omitempty"`
	// Total Params map
	Params map[string]string `json:"params"`
}

type RetrySteps struct {
	// Name of the Retry Step
	Steps []string `json:"steps,omitempty"`
	// TimeStamps of this retry operation
	RetryTimeStamp metav1.Time `json:"retryTimeStamp,omitempty"`
}

// OpsStateType defines the state of MachineOps.
type OpsStateType string

const (
	OpsStateRunning     OpsStateType = "Running"
	OpsStateInitialized OpsStateType = "Initialized"
	OpsStateFailed      OpsStateType = "Failed"
	OpsStateSucceeded   OpsStateType = "Succeeded"
	OpsStatePaused      OpsStateType = "Paused"
)

// MachineOpsStatus is the status for a MachineOps resource
type MachineOpsStatus struct {
	// Required. State of this machine ops
	OpsState OpsStateType `json:"opsState"`
	// List of ops action step status.
	StepStatus []OpsActionStepStatus `json:"stepStatus"`
	// Machine Ops finished time
	FinishedTime metav1.Time `json:"finishedTime,omitempty"`
	// A human readable message indicating details about why the machineOps is in this condition.
	// +optional
	Message string `json:"message"`
	// A brief CamelCase message indicating details about why the machineOps is in this state.
	// +optional
	Reason string `json:"reason"`

	GlobalDataContext
}

// GlobalDataContext is a global map to contains all the essential data
// that from DataContext. MachineOps engine will take charge of merge all the
// field.
// New data will override the old data when their keys are the same in DataContext.
// string: MachineOps Name
// DataContext: Merge DataContext for a special MachineOps
type GlobalDataContext map[string]string

type OpsActionStepStatus struct {
	// Required. Name of this ops action.
	Name string `json:"name"`
	// Description of this ops action
	Desc string `json:"desc"`
	// Last time the state transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Last time the state transitioned from one status to another.
	// +optional
	StartOpsActionTime metav1.Time `json:"startOpsActionTime,omitempty"`
	// Required. State of this ops action.
	OpsState OpsStateType `json:"opsState"`
	// Detailed execution result of this ops action.
	// +optional
	Result OpsActionResult `json:"result"`
	// Retry count
	RetryCount int32 `json:"retryCount,omitempty"`
	// Useful data for each action.
	Data map[string]string `json:"data"`
}

type OpsActionResult struct {
	// The execution code for the ops action.
	// +optional
	Code string `json:"code"`
	// The reason for the result's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineOpsList is a list of MachineOps resources
type MachineOpsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MachineOps `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OpsType represents some kind ops operation type and each OpsType contains list of
// OpsActions. Well known OpsType includes 'Create', 'Delete', 'Reboot', etc.
type OpsType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OpsTypeSpec `json:"spec"`
}

// OpsTypeSpec is the spec for a OpsType resource
type OpsTypeSpec struct {
	// List of ops actions
	OpsActions []OpsAction `json:"steps"`
}

// Data configuration that get from OpsType, usually it contains the data that
// user will change, such as the action silent time.
type DefaultActionConfig map[string]string

// OpsAction represents a single ops action, for each ops action, there should be a
// correlative action handler which is responsible for action handling.
type OpsAction struct {
	// Name of this ops action, should be unique.
	Name string `json:"name"`
	// Name of this opsaction action handler
	ActionHandlerName string `json:"actionHandlerName"`
	// Some descriptive language of this ops action.
	Description string `json:"description,omitempty"`
	// ops action could execute in async manner
	Async bool `json:"async,omitempty"`
	// Number of seconds after which waiting for the action finished and return times out.
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`

	DefaultActionConfig `json:"defaultActionConfig"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OpsTypeList is a list of OpsType resources
type OpsTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []OpsType `json:"items"`
}

type ReadinessGatesConfig struct {
	HeartBeatInterval int32 `json:"heartBeatInterval"`
	TaintByCondition  bool  `json:"taintByCondition"`
	Offlined          bool  `json:"offlined"`
}
