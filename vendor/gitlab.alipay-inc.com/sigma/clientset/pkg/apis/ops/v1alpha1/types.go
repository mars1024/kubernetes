/*
Copyright 2018 The Alipay Authors.
*/

package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	IP  string `json:"ip"`
	SN  string `json:"sn"`
	IDC string `json:"idc"`

	PackageCustomConfigs []PackageCustomConfig `json:"packageCustomConfigs"`
	MachineDriverOptions MachineDriverOptions  `json:"machineDriverOptions"`
}

// MachineStatus is the status for a Machine resource
type MachineStatus struct {
	Phase MachinePhase `json:"phase"`

	// A human readable message indicating details about why the machine is in this condition.
	// +optional
	Message string `json:"message"`

	// A brief CamelCase message indicating details about why the machine is in this state.
	// e.g. 'Evicted'
	// +optional
	Reason string `json:"reason"`

	Packages            []PackageConfig      `json:"packages"`
	BetaPublishVersions []BetaPublishVersion `json:"betaPublishVersions"`

	// The conditions of packages of this machine
	PackageConditions []PackageCondition `json:"packageConditions"`

	// where get this info ?
	ClusterInfo ClusterInfo `json:"clusterInfo"`

	NodeLabelSet string `json:"nodeLabelSet"`
}

type BetaPublishVersion struct {
	BetaPublishName string `json:"betaPublishName"`
}

// PackageConditions describes the state of a package at a certain point.
type PackageCondition struct {
	// Type of package condition.
	Type PackageConditionType `json:"type"`
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

type PackageConditionType string

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineList is a list of Machine resources
type MachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Machine `json:"items"`
}

type MachineDriverOptions struct {
	Driver     string            `json:"driver"`
	Config     map[string]string `json:"config"`
	SecretName string            `json:"secretName"`
}

type PackageCustomConfig struct {
	PackageName  string            `json:"packageName"`
	CustomConfig map[string]string `json:"customConfig"`
	Disable      bool              `json:"disable"`
}

type PackageConfig struct {
	PackageName string `json:"packageName"`
	VersionName string `json:"versionName"`
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
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MachinePackageVersionSpec `json:"spec"`
}

type MachinePackageVersionSpec struct {
	PackageName    string            `json:"packageName"`
	PackageVersion string            `json:"packageVersion"`
	Config         map[string]string `json:"config"`
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

type ClusterMachinePackageVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterMachinePackageVersionSpec `json:"spec"`
}

type ClusterMachinePackageVersionSpec struct {
	Packages []PackageConfig `json:"packages"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachinePackageVersionList is a list of MachinePackageVersion resources
type ClusterMachinePackageVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ClusterMachinePackageVersion `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachinePackageBetaPublish is a specification for a Package Beta Publish for some machines resource
type MachinePackageBetaPublish struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachinePackageBetaPublishSpec   `json:"spec"`
	Status MachinePackageBetaPublishStatus `json:"status"`
}

type MachinePackageBetaPublishSpec struct {
	// Label selector for Nodes.
	Selector *metav1.LabelSelector `json:"selector"`

	// 随机挑选灰度发布的机器。
	// Value can be an absolute number (ex: 5) or a percentage of desired Machines (ex: 10%).
	RandomPick *intstr.IntOrString `json:"randomPick"`

	// 指定Machine发布
	Machines []string `json:"machines"`

	// 指定灰度发布对象
	Packages []PackageConfig `json:"packages"`

	// 发布策略
	Strategy MachinePackageBetaPublishStrategy `json:"strategy"`

	// Indicates that the beta publish is paused.
	// +optional
	Paused bool `json:"paused"`
}

type MachinePackageBetaPublishStrategy struct {
	// Type of publish. Can be "OneByOne" or "Batch". Default is Batch.
	// +optional
	Type MachinePackageBetaPublishStrategyType `json:"type"`

	BatchPublish *BatchPublish `json:"batchPublish"`
}

type MachinePackageBetaPublishStrategyType string

const (
	OneByOneBetaPublishStrategyType MachinePackageBetaPublishStrategyType = "OneByOne"

	BatchBetaPublishStrategyType MachinePackageBetaPublishStrategyType = "Batch"
)

type BatchPublish struct {
	// Default is 10.
	MaxBatchSize int64 `json:"maxBatchSize"`

	// 最大失败忍受数目
	// -1 代表不管失败多少都将剩下机器继续执行
	// +optional
	MaxFailedTolerate int64 `json:"maxFailedTolerate"`
}

type MachinePackageBetaPublishStatus struct {
	Phase MachinePackageBetaPublishPhase `json:"phase"`

	// Machine list in this Beta-Publish
	Machines []string `json:"machines"`

	// List of machines being upgraded
	Upgrading []string `json:"upgrading"`

	// List of machines that have been successfully upgraded
	Succeeded []string `json:"succeeded"`

	// List of machines that failed to upgrade
	Failed []string `json:"failed"`

	RandomPick *intstr.IntOrString `json:"randomPick"`
}

type MachinePackageBetaPublishPhase string

const (
	// 正在发布
	BetaPublishPhaseUpgrading MachinePackageBetaPublishPhase = "Upgrading"

	// 发布成功，并结束。
	// 发布失败的Machine数目小于 Spec.Strategy.MaxFailedTolerate
	BetaPublishPhaseSucceeded MachinePackageBetaPublishPhase = "Succeeded"

	// 发布失败，可能并未全部发布完成
	// 发布失败的Machine数目大于 Spec.Strategy.MaxFailedTolerate, 停止剩下的发布.
	BetaPublishPhaseFailed MachinePackageBetaPublishPhase = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachinePackageBetaPublishList is a list of MachinePackageBetaPublish resources
type MachinePackageBetaPublishList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MachinePackageBetaPublish `json:"items"`
}
