package ase

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"net/http"
)

type KubeResource interface {
	GetObjectKind() schema.ObjectKind
	GetObjectMeta() metav1.Object
	DeepCopyObject() runtime.Object
	GetName() string
	GetLabels() map[string]string
	SetLabels(map[string]string)
	GetAnnotations() map[string]string
	SetAnnotations(map[string]string)
}

type VolumeMountConfigItem struct {
	VolumeName string `json:"volumeName,omitempty"`
	MountAs    string `json:"mountAs,omitempty"`
}

type PodLogConfig struct {
	LogAgentRequestedResource LogAgentResourceQuota    `json:"logAgentRequestedResource,omitempty"`
	DefaultLogUserId          string                   `json:"defaultLogUserId,omitempty"`
	DefaultLogProject         string                   `json:"defaultLogProject,omitempty"`
	DefaultLogStore           string                   `json:"defaultLogStore,omitempty"`
	DefaultLogConfig          string                   `json:"defaultLogConfig,omitempty"`
	DefaultImage              string                   `json:"defaultImage,omitempty"`
	DefaultProjectRegionId    string                   `json:"defaultProjectRegionId,omitempty"`
	DefaultTenantId           string                   `json:"defaultTenantId,omitempty"`
	DefaultVolumeMountConfigs *[]VolumeMountConfigItem `json:"defaultVolumeMountConfigs,omitempty"`
}

type LogAgentContext struct {
	UserId             string                   `json:"userId,omitempty"`
	UserDefinedId      string                   `json:"userDefinedId,omitempty"`
	Config             string                   `json:"config,omitempty"`
	LogProjectName     string                   `json:"logProjectName,omitempty"`
	LogStoreName       string                   `json:"logStoreName,omitempty"`
	TenantId           string                   `json:"tenantId,omitempty"`
	Image              string                   `json:"image,omitempty"`
	ProjectRegionId    string                   `json:"projectRegionId,omitempty"`
	VolumeMountConfigs *[]VolumeMountConfigItem `json:"volumeMountConfigs,omitempty"`
}

type LogAgentResourceQuota struct {
	Cpu     string `json:"cpu,omitempty"`
	Memory  string `json:"memory,omitempty"`
	Storage string `json:"storage,omitempty"`
}

type SchedulerConfig struct {
	Predicates           []interface{} `json:"predicates,omitempty"`
	Priorities           []interface{} `json:"priorities,omitempty"`
	CpuOverScheduleRatio float64       `json:"cpuOverScheduleRatio,omitempty"`
	HardCpuOverSchedule  bool          `json:"hardCpuOverSchedule,omitempty"`
}

type ManagedSubCluster struct {
	SubClusterName string `json:"subClusterName,omitempty"`
}

type ManagedClusterInfo struct {
	ClusterId                string              `json:"clusterId,omitempty"`
	ClusterName              string              `json:"clusterName,omitempty"`
	ClusterTenantId          string              `json:"clusterTenantId,omitempty"`
	ClusterTenantName        string              `json:"clusterTenantName,omitempty"`
	ClusterWorkspaceId       string              `json:"clusterWorkspaceId,omitempty"`
	ClusterWorkspaceIdentity string              `json:"clusterWorkspaceIdentity,omitempty"`
	ClusterRegionId          string              `json:"clusterRegionId,omitempty"`
	ManagedSubClusters       []ManagedSubCluster `json:"managedSubClusters,omitempty"`
}

type CheckImagePermissionPayload struct {
	ImageUrl          string `json:"imageUrl,omitempty"`
	ContainerName     string `json:"containerName,omitempty"`
	ClusterId         string `json:"clusterId,omitempty"`
	ClusterTenantName string `json:"clusterTenantName,omitempty"`
	ClusterWorkspace  string `json:"clusterWorkspace,omitempty"`
}

type ImageConfig struct {
	CheckImagePermissionUrl string `json:"checkImagePermissionUrl,omitempty"`
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}
