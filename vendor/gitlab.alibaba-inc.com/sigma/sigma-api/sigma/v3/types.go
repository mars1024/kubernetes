package sigma_v3

import (
	"gitlab.alibaba-inc.com/sigma/sigma-api/sigma"
	"time"
)

type Pod struct {
	ObjectMeta
	PodStatus PodStatus //pod状态
	Spec      PodSpec   //pod终态

}

type ObjectMeta struct {
	PodSn           string            //全局唯一保证         // podSn
	Labels          map[string]string //必需包括Site，ContainerType，DeployUnit，BizName， InstanceGroup， AppName
	UpdateTime      time.Time
	CreationTime    time.Time
	DeletionTime    *time.Time
	ResourceVersion string
	MemVersion      int64
	WaitReqId       string
	WaitTime        *time.Time
	Annotations     map[string]string
	NodeName        string
}

type PodSpec struct {
	Requirement Requirement
	Containers  []*Container `json:",omitempty"` //容器列表
}

type PodStatus struct {
	Phase             PodPhase
	Condition         PodCondition
	HostIp            string //分配的宿主机IP地址
	HostSn            string
	PodIp             string
	NetInfo           NetInfo            //网络信息
	ContainerStatuses []*ContainerStatus `json:",omitempty"`
}

type ContainerStatus struct {
	Image         string        `json:",omitempty"`
	ContainerID   string        `json:",omitempty"`
	AllocResource AllocResource //分配的具体资源
}

type NetInfo struct {
	PublicIp          string //公网IP地址
	ArmoryModel       string `json:",omitempty"`
	IPAMType          string `json:",omitempty"`
	OverlayNetwork    string `json:",omitempty"`
	OverlayNetworkVer string `json:",omitempty"`
	VPortId           string `json:",omitempty"` /* Overlay网络VPort ID */
	VPortToken        string `json:",omitempty"` /* Overlay网络VPort创建Token */
	VSwitchId         string `json:",omitempty"` /* Overlay网络或ECS创建时指定的VSwitch ID */
	EcsInstanceId     string `json:",omitempty"`
}

type Requirement struct {
	OverQuota      sigma.OverQuota   `json:",omitempty"` //允许的宿主机最大超卖比
	Spread         sigma.Spread      `json:",omitempty"` //应用实例如何分布
	Constraints    sigma.Constraints `json:",omitempty"` //应用对宿主机的要求
	Dependency     sigma.Dependency  `json:",omitempty"` //应用和应用/数据之间的依赖性
	Affinity       sigma.Affinity    `json:",omitempty"` //应用和应用/数据之间的亲和性
	Prohibit       sigma.Prohibit    `json:",omitempty"` //应用和应用/数据之间的互斥性
	CandidatePlans []*sigma.CandidatePlan
}

type Container struct {
	Image     string            `json:",omitempty"` //镜像名称
	Labels    map[string]string `json:",omitempty"` //用来存放相关容器的标签，这里面的信息会被打入container
	AllocSpec sigma.AllocSpec   `json:",omitempty"` //资源的规格描述
}

type PodResponse struct {
	HostIp             string `json:",omitempty"`
	HostSn             string `json:",omitempty"`
	RequirementId      string `json:",omitempty"`
	UpdateTime         string `json:",omitempty"`
	MemVersion         int64  `json:",omitempty"`
	WaitReqId          string `json:",omitempty"`
	ErrorCode          int
	ErrorMsg           string                 `json:",omitempty"`
	Statistics         []sigma.ScheduleResult `json:",omitempty"` //分配过程中用到的处理器，此为文本过滤器
	ContainerResponses []*ContainerResponse   `json:",omitempty"` //容器列表
	CpuSetMode         sigma.CpuSetMode       `json:",omitempty"`

	//这三字段主要是给OtherPod使用的
	PodSn      string
	Site       string
	DeployUnit string
	OtherPod   []*PodResponse
}

type ContainerResponse struct {
	AllocResource AllocResource //分配的具体资源
}

type PodCondition struct {
	Type PodConditionType
	// +optional
	Reason string
	// +optional
	Message string
}

type PodConditionType string

const (
	PodScheduled           PodConditionType = "PodScheduled"
	PodScheduledError      PodConditionType = "PodScheduledError"
	PodReady               PodConditionType = "Ready"
	PodInitialized         PodConditionType = "Initialized"
	PodReasonUnschedulable PodConditionType = "Unschedulable"
)

type AllocResource struct {
	/* 内存 */
	Mem int64 //内存大小单位byte

	/* cpu */
	CpuSet      []int //cpuset方式的cpu个数
	CpuSetShare []int //cpushare方式的cpu个数
	CpuQuota    int   //cpu的计算能力
	CPURequest  int64 `json:"cpuRequest,omitempty"`
	CPULimit    int64 `json:"cpuLimit,omitempty"`

	/* disk */
	DiskQuota map[string]sigma.DiskQuota // key:MountPoint

	/* gpu */
	GpuCount       int      // 用户指定需要多少的gpu数量
	GpuDevicesSet  []string `json:",omitempty"` //gpu设备，如[/dev/nvidia0]
	GpuCtrlDevices []string `json:",omitempty"` //gpu控制设备，如[/dev/nvidia-uvm]
	GpuVolumes     []string `json:",omitempty"` //gpu驱动所在的数据卷

	/*Fpga*/
	FpgaSet map[string][]string `json:",omitempty"`
}

type PodTaskList struct {
	Values []*PodTask
}

type PodTask struct {
	PodPath    string //pod的etcd路径
	PodSn      string //唯一编号
	Site       string //机房
	UpdateTime int64
	PodStatus  PodStatus //pod的状态信息
}

const Etcd_key_pod_response = "/response/pod/%v/%v/%v" //site, deployUnit, reqId:自己构建的唯一标记uuid算法
const Etcd_key_pod_response_head = "/response/pod/%v"  //site, deployUnit, instanceSn
const Etcd_key_pod = "/pod/%v/%v/%v"                   //site, deployUnit, instanceSn
const Etcd_key_pod_list_url = "/pod/%v/"               //site, deployUnit, instanceSn
const Etcd_key_task_pod = "/task/pod/%v/%v/%v"         //site , apiServerIp地址 递增ID
const Etcd_key_iplist = "/iplist/apiserver/%v/"        //site
const Etcd_key_iplist_ip = "/iplist/apiserver/%v/%v"   //site apiServerIp地址

type PodPhase string

const (
	PodPending   PodPhase = "Pending"
	PodRunning   PodPhase = "Running"
	PodSucceeded PodPhase = "Succeeded"
	PodFailed    PodPhase = "Failed"
	PodUnknown   PodPhase = "Unknown"
	PodDestroyed PodPhase = "Destroyed"
)
