/*
Copyright 2018 The Alipay Authors.
*/

package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is a specification for a Cluster resource
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec"`
	Status ClusterStatus `json:"status"`
}

// ClusterSpec is the spec for a Cluster resource
type ClusterSpec struct {
	// Cluster whole resources when delete this Cluster CR
	ClearResourceAfterDeletion bool `json:"clearResourceAfterDeletion"`

	// The Node Selector for deploying Component
	NodeSelector map[string]string `json:"nodeSelector"`

	// DeployType means how to deploy this Cluster, Standard or Minimum
	DeployType ClusterDeployType `json:"deployType"`

	// the name of ClusterComponentVersion
	// 可以留空，使用单独组件版本模式
	ClusterVersionName string `json:"clusterVersionName"`

	// Config for components, including Version, CustomConfig, TemplateName and Disable.
	// 单独组件版本模式，每个组件版本在这里控制
	Components ClusterComponentConfig `json:"components"`

	// The config for etcd deploy
	ETCD ETCDConfig `json:"etcd"`

	// The config for pki
	PKI PKIConfig `json:"pki"`

	ClusterConfig ClusterConfig `json:"clusterConfig"`

	// the External Users(kubeconfig) need to be generated and stored to Secret
	ExternalUsers []ExternalUser `json:"externalUsers"`

	// how to send notify messages
	Notify []NotifyConfig `json:"notify"`

	// log config for master components
	Log LogConfig `json:"log"`

	// Server SSL PKI needed by component
	// Like CSR in k8s, but this certs pair will be generated in meta cluster
	// and can be referenced by CustomComponentsInMetaCluster
	ServerSSLs []ServerSSL `json:"serverSSLs"`
}

type ClusterDeployType string

const (
	// 标准部署模式, 3台ETCD， 3个apiserver，scheduler, controll-manager
	ClusterDeployTypeStandard ClusterDeployType = "Standard"

	// 最小化部署模式, 1台ETCD， 1个apiserver，scheduler, controll-manager
	ClusterDeployTypeMinimum ClusterDeployType = "Minimum"
)

// ComponentType 标记 Custom Component 发布到 业务集群 还是 元集群
type ComponentType string

const (
	// Master Component, 不用标记, 留着备用
	ComponentTypeMaster ComponentType = "Master"

	// Custom Component 发布到 业务集群
	ComponentTypeBizClusterComponent ComponentType = "BizClusterComponent"

	// Custom Component 发布到 元集群，使用kubeconfig连接 业务集群
	ComponentTypeMetaClusterComponent ComponentType = "MetaClusterComponent"
)

type ClusterComponentConfig struct {
	Master []ComponentConfig `json:"master"`

	Custom []ComponentConfig `json:"custom"`
}

type ComponentConfig struct {
	// Component Name, such as apiserver, etcd, etc.
	Name string `json:"componentName"`

	// Custom config of one component
	// Pre-defined the key of Master refer to:
	CustomConfig map[string]string `json:"customConfig"`

	// Disable the deployment of this Component
	Disable bool `json:"disable"`
}

type ServiceConfig struct {
	Type ServiceType `json:"type"`

	// Headless Service set up params
	Headless *HeadlessService `json:"headless"`

	// DNS RR Service set up params
	DNSRR *DNSRRService `json:"dnsrr"`

	// VIPServer Service set up params
	VIPServer *VIPServerService `json:"vipServer"`
}

type ServiceType string

const (
	// kube 原生的 headless service， 需要 kube-dns(core-dns) 支持
	ServiceTypeHeadless ServiceType = "HeadlessService"

	// 机房DNS RR, 蚂蚁方案, 需要蚂蚁Controller支持
	ServiceTypeIDCDNSRR ServiceType = "DNSRR"

	// VIPServer 方案, 需要VIPServer Controller支持
	ServiceTypeVIPServer ServiceType = "VIPServer"
)

type HeadlessService struct {
	// Cluster Domain 后缀(元集群), 不指定则使用短域名.
	// 即不指定，apiserver的域名就是apiserver, 指定了就是 apiserver.${Cluster.Name}.svc.${ClusterDomain}
	ClusterDomain string `json:"clusterDomain,omitempty"`
}

type DNSRRService struct {
	// 机房域名后缀
	DCDomain string `json:"dcDomain"`

	DCPrefix string `json:"dcPrefix"`
}

type VIPServerService struct {
	// 直接指定Apiserver域名
	Domain string `json:"domain"`

	// etcd域名format格式, 如 %s.su18.alibaba-inc.com, 那么 etcd1 的域名就是 etcd1.su18.alibaba-inc.com
	DomainLayout string `json:"domainLayout"`
}

type ETCDConfig struct {
	DataVolume ETCDDataVolume `json:"dataVolume"`

	Service *ServiceConfig `json:"service"`

	EndPoints string `json:"endPoints"`
}

type ETCDDataVolume struct {
	ClaimTemplate *v1.PersistentVolumeClaimSpec `json:"claimTemplate"`
	Source        v1.VolumeSource               `json:",inline"`
}

type PKIConfig struct {
	ExpireDays int64 `json:"expireDays"`
}

type ClusterConfig struct {
	ApiserverDomains []string `json:"apiserverDomains"`

	ExtensionApiserverDomains []string `json:"extensionApiserverDomains"`

	Site string `json:"site"`

	// Service Subnet of Cluster
	ServiceSubnet string `json:"serviceSubnet,omitempty"`

	// Cluster Domain of Cluster, needed by kube-dns or core-dns
	ClusterDomain string `json:"clusterDomain,omitempty"`
}

type ExternalUser struct {
	// The prefix of Secret Name.
	// ie. Name is admin, then the Secret Name is admin.kubeconfig
	Name string `json:"name"`

	// TLS Client Cert expire days
	ExpireDays int64 `json:"expireDays"`

	// RBAC Username
	Username string `json:"username"`

	// RBAC Groups
	Groups []string `json:"groups"`
}

type ServerSSL struct {
	// Secret name
	SecretName string `json:"secretName"`
	// Server Serve IPs or Domains
	Hosts []string `json:"hosts"`
	// Server SSL Cert expire days
	ExpireDays int64 `json:"expireDays"`
}

type NotifyConfig struct {
	Driver string            `json:"driver"`
	Config map[string]string `json:"config"`
}

type LogConfig struct {
	V        int64  `json:"v"`
	HostPath string `json:"hostPath"`
}

// ClusterStatus is the status for a Cluster resource
type ClusterStatus struct {
	Phase ClusterPhase `json:"phase"`

	// A human readable message indicating details about why the cluster is in this condition.
	// +optional
	Message string `json:"message"`

	// A brief CamelCase message indicating details about why the cluster is in this state.
	// e.g. 'Evicted'
	// +optional
	Reason string `json:"reason"`

	// Cluster Conditions
	Conditions []ClusterCondition `json:"conditions"`

	// Version publish history
	ComponentVersionHistory []ClusterVersionHistory `json:"versionHistory"`
}

type ClusterPhase string

const (
	// Cluster is processed by Operator, but not all components are ready
	ClusterPending ClusterPhase = "Pending"
	// All components are ready
	ClusterRunning ClusterPhase = "Running"
	// Cluster can not be inited, or occur the error that Operator can not recover
	ClusterError ClusterPhase = "Error"
)

type ClusterCondition struct {
	// Cluster Condition Type, such as Apiserver
	Type ClusterConditionType `json:"type"`

	// Cluster Condition Status
	// Can be True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`

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

type ClusterConditionType string

type ClusterVersionHistory struct {
	ClusterVersionName string                 `json:"clusterVersionName"`
	Components         ClusterComponentConfig `json:"components"`
	AppliedAt          metav1.Time            `json:"appliedAt"`

	// Username 是 Kok-Portal 通过 Label传递的，Username是在Portal中的用户名，非Kubernetes体系中的用户名(RBAC Username)。
	// 每次发布、更新前，Kok-Portal将操作者的Username打上Label，提交Cluster Update，cluster-controller执行完发布之后，会将信息压入history。
	Username string `json:"username"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList is a list of Cluster resources
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Cluster `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterVersionSpec `json:"spec"`
}

type ClusterVersionSpec struct {
	Master []ClusterVersionComponent `json:"master"`

	Custom []ClusterVersionComponent `json:"custom"`
}

type ClusterVersionComponent struct {
	// Name of component
	Name string `json:"name"`

	Args []ComponentVersionArg `json:"args"`

	// Component template files need to be compiled and applied by kubectl
	Templates []ComponentVersionTemplate `json:"templates"`
}

type ComponentVersionArg struct {
	// Name of Component Arg
	Name string `json:"name"`

	// Default value of this component
	DefaultValue *string `json:"defaultValue"`

	// Required
	Required bool `json:"required"`

	// The description of this Arg
	Description string `json:"description"`
}

type ComponentVersionTemplate struct {
	Name    string `json:"name"`
	File    string `json:"file"`
	Content string `json:"content"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList is a list of Cluster resources
type ClusterVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ClusterVersion `json:"items"`
}
