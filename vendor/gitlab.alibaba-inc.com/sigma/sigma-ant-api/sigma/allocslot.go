package sigma

//http://docs.alibaba-inc.com:8090/pages/viewpage.action?pageId=422874776#d0-etcd%E6%95%B0%E6%8D%AE%E7%BB%93%E6%9E%84%E6%8E%A5%E5%8F%A3%E5%AE%9A%E4%B9%89-%E3%80%90d103%E3%80%91etcdnodeAllocInfo%EF%BC%88%E8%B5%84%E6%BA%90%E5%88%86%E9%85%8D%E7%8E%B0%E7%8A%B6%EF%BC%89
type SlotStatus string

var (
	SlotStatus_allocated = SlotStatus("allocated") //资源已分配
	SlotStatus_rebind    = SlotStatus("rebind")    //资源重绑定
	SlotStatus_starting  = SlotStatus("starting")  //实例启动中
	SlotStatus_started   = SlotStatus("started")   //实例已启动完成
	SlotStatus_reclaimed = SlotStatus("reclaimed") //资源已回收； 调度器要主动杀离线任务时，将其alloc值为reclaimed状态，等待执行器停止实例
	SlotStatus_stopping  = SlotStatus("stopping")  //实例停止中
	SlotStatus_stopped   = SlotStatus("stopped")   //实例已停止
	SlotStatus_unknow    = SlotStatus("unknow")
)

type Allocinfos struct {
	RequirementId string //例："123456" ，哪个请求最后确定的这个slot，方便问题排查
	AllocCount    int
	Allocs        []string //每一项都是一个路径：allocinfos/$site/$sn/allocplans/$slotId
	ReleaseCount  int
	Releases      []string //已回收slots,格式同上：allocinfos/$site/$sn/allocplans/$slotId
}

type AllocPlan struct {
	//HippoTag        string           //拼装的hippo apptag, 拼装规则：AppName+"_"+AppDeployUnit+"_"+len(AppName)+"_"+len(AppDeployUnit)
	UpdateTime              string           //例："2016-06-29 19:59:22",
	RequirementId           string           //例："123456" ，哪个请求最后确定的这个slot，方便问题排查
	Requirement             *Requirement     `json:"-"` //方便存放Requirement对象, 不出现在json中
	Site                    string           //和path中的site一致, 方便编码；使代码更简洁一致
	BizName                 string           //二层业务域名称；zeue，Carbon，Captain
	AppName                 string           //相当于Aone的app，一个app下可以由多个分组
	DeployUnit              string           //相当于电商一个应用下的armory分组，预发/小流量等，同分组下的实例用同一份代码+配置，例如:"buy_center_online"
	AllocSpecKey            string           //资源需求规格描述Key
	AllocSpec               *AllocSpec       `json:"-"` //方便存放AllocSpec对象, 不出现在json中
	HostIp                  string           //宿主机IP，例如: 11.136.23.69
	HostSn                  string           //宿主机Sn，例如: 214247788-02A
	HostPath                string           //Host在etcd上的地址路径：如："$cell/$securityRegion/214247788-02A"
	SlotId                  int              //如：宿主机上唯一
	Status                  SlotStatus       //"allocated|starting|started|reclaimed|stopping|stopped", //状态。
	InstanceSn              string           //例：20170322093955333fe415840dc85 时间戳yyyyMMddHHmmssSSS+12位uuid：共29个字符
	InstanceType            InstanceType     //例："CONTAINER" //实例类型： TASK|CONTAINER|KVM
	Priority                InstancePriority //例：1           //实例优先级
	CpuSet                  []int            //例：[4,5,6,7],  //CPU具体核，允许空字符串
	CpusetMode              CpuSetMode
	CpusetStrategy          CpuStrategy
	CpuQuota                int                  //400 //  总共需要多少的cpu计算能力，如果CpuNum不为0， 则必须是100*CpuNum
	Memory                  int64                //例：8589934592, //内存配额，单位：字节
	DiskQuota               map[string]DiskQuota //例：80960,      //硬盘空间配额
	GpuDevicesSet           []string             //gpu设备和gpu控制设备，如[/dev/nvidia0, /dev/nvidia1, /dev/nvidia-uvm]
	GpuSetMode              GpuSetMode
	GpuQuota                map[string]GpuQuota
	GpuVolumes              []string //gpu驱动所在的数据卷
	GpuVolumeDriver         string
	NetBandwidthQuota       int64 //例：10240,      //网络带宽配额
	MatchRequirement        bool  `json:"ResourceNoLongerMatchRequirement"` //资源规格调整失败时会设置为true，其他时候为false
	PlanPath                string
	ExtendScalarResources   map[string]int64
	Resources               []Resource
	Declarations            []Resource
	AppLabels               map[string]string
	DuMetaInfo              *DuMetaInfo `json:"-"`
	OldCpuSet               []int       `json:"-"`
	NewCpuSet               []int       `json:"-"`
	ContainerMandatoryLabel string      `json:",omitempty"` //弹性资源池标签
}

type GpuQuota struct {
	DevicePath string
	CacheSize  int64
}

type Resource struct {
	Name   string `json:"Name"`
	Type   string `json:"Type"`
	Amount int64  `json:"Amount"`
	Value  string `json:"Value"`
}

type AllocAddr struct {
	UpdateTime  string //例："2016-06-29 19:59:22",
	ContainerIp string //":"10.185.162.130"},
	ContainerSn string //":"t4_10.185.162.130"},
}

type DiskQuota struct {
	HostPath string
	DiskSize int64
}

type SlotState struct {
	AllocPlan
	NcHostname    string //本地取到的Hostname
	DaemonPort    int    //本地取到的Daemon端口
	ContainerInfo ContainerInfo
	TaskInfo      TaskInfo
}

type SlotSetMessage struct {
	UpdateTime string
	Slots      []int
}

type ContainerInfo struct {
	AppNodeGroup      string //":"autoscalingtest2host"},
	AppBizType        string //":"core"},
	ContainerName     string //":"buy010153023089"} //容器名
	ContainerId       string //":"asfdasfasfdasfdas"} //容器名
	ContainerIp       string //":"10.185.162.130"},
	ContainerSn       string //":"t4_10.185.162.130"},
	ContainerHostName string //":"buy010153023089.et2"},
	//ContainerServerId string //":"8885094"},
}

type PackageStatus string

var (
	PackageStatus_UNKNOWN    = PackageStatus("UNKNOWN")    //未知
	PackageStatus_WAITING    = PackageStatus("WAITING")    //等待
	PackageStatus_INSTALLING = PackageStatus("INSTALLING") //安装中
	PackageStatus_INSTALLED  = PackageStatus("INSTALLED")  //已安装
	PackageStatus_FAILED     = PackageStatus("FAILED")     //已失败
	PackageStatus_CANCELLED  = PackageStatus("CANCELLED")  //已取消
)

type TaskInfo struct {
	ApplicationId   string            `json:"applicationId"`   //":"123124214"},
	LaunchSignature int64             `json:"launchSignature"` //":"12312"},
	PackageChecksum string            `json:"packageChecksum"` //":"12312"},
	PackageStatus   map[string]string `json:"packageStatus"`   //
	ProcessStatus   []ProcessStatus   `json:"processStatus"`   //
	DataStatus      []DataStatus      `json:"dataStatus"`      //":""}
	SlotId          int               `json:"slotId"`          // 0,
}

type ProcessStatus struct {
	ExitCode     int    `json:"exitCode"`     // 0,
	InstanceId   int64  `json:"instanceId"`   // 0,
	IsDaemon     bool   `json:"isDaemon"`     //true
	Pid          int    `json:"pid"`          // 0,
	ProcessName  string `json:"processName"`  // 0,
	RestartCount int    `json:"restartCount"` // 0,
	StartTime    int64  `json:"startTime"`    // 0,
	status       string `json:"status"`       // 0,
}

type DataStatus struct {
	Name          string `json:"name"`          // 0,
	Src           string `json:"src"`           // 0,
	Dst           string `json:"dst"`           // 0,
	CurVersion    int    `json:"curVersion"`    // 0,
	TargetVersion int    `json:"targetVersion"` // 0,
	DeployStatus  string `json:"deployStatus"`  // 0,
	AttemptId     int    `json:"attemptId"`     // 0,
	RetryCount    int    `json:"retryCount"`    // 0,
}

type ReclaimInfo SlotInfo
type ReleaseInfo SlotInfo
type SlotInfo struct {
	Site       string //"et15sqa",               // 机房
	BizName    string //"ha3",                 // 业务名称
	AppName    string
	DeployUnit string
	HostIp     string //"10.0.0.1",             // release slot所在宿主机的ip地址，可选，因为找路径主要依赖sn
	HostSn     string //"sn-1234567890",        // release slot所在宿主机的sn编号
	SlotId     int    //21                     // slot在这台slave上的id编号
}
