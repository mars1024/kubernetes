package sigma

import (
	"fmt"
)

type InstanceType string
type CpuStrategy string
type CpuSetMode string
type GpuSetMode string

var (
	InstanceType_TASK      = InstanceType("TASK")      // 会自己退出的，进程/组
	InstanceType_KVM       = InstanceType("KVM")       // KVM
	InstanceType_ECS       = InstanceType("ECS")       // ECS
	InstanceType_vLinux    = InstanceType("vLinux")    // vLinux
	InstaceType_Hippojob   = InstanceType("Hippojob")  // 容器+基线
	InstanceType_CONTAINER = InstanceType("CONTAINER") // 容器
	InstanceType_T4        = InstanceType("T4")        // 容器

	CpuStrategy_default       = CpuStrategy("default")       //一个实例的虚拟核(ht)优先按物理核打散
	CpuStrategy_sameCoreFirst = CpuStrategy("sameCoreFirst") //一个实例的虚拟核(ht)优先按物理核堆叠

	CpuSetMode_default = CpuSetMode("default") //默认
	CpuSetMode_share   = CpuSetMode("share")   //cpushare模式分配
	CpuSetMode_cpuset  = CpuSetMode("cpuset")  //cpuset模式分配

	CpuSetMode_mutex     = CpuSetMode("mutex")     //和同样是mutex模式的实例不共享核; 和共享模式的实例可以共享核
	CpuSetMode_exclusive = CpuSetMode("exclusive") //独占模式(exclusive)  绝对独占, 不做任何共享
)

/**App resource requirement info*/
type Requirement struct { //应用需求规格描述
	UpdateTime      string      `hash:"-"`
	RequirementId   string      `hash:"-"` //20160815153046 //请求的Id
	Site            string      //一个cluster可能跨多个小机房，二层请求传入的机房名称
	TargetReplica   int         //50, 目标实例个数
	IncreaseReplica int         // 实例的增量个数, apiserver 内部转化为TargetReplica后发给 master
	MinReplica      int         //10, 可接受的最小实例个数
	App             AppInfo     //应用信息
	AllocSpec       AllocSpec   //资源需求规格描述
	Spread          Spread      `json:",omitempty"` //应用实例如何分布
	Constraints     Constraints `json:",omitempty"` //应用对宿主机的要求
	Dependency      Dependency  `json:",omitempty"` //应用和应用/数据之间的依赖性
	Affinity        Affinity    `json:",omitempty"` //应用和应用/数据之间的亲和性
	Prohibit        Prohibit    `json:",omitempty"` //应用和应用/数据之间的互斥性
	CandidatePlans  []*CandidatePlan
}

type BatchStopPreview struct { //应用需求规格描述
	UpdateTime        string
	Type              string //如果Type的值是faultHostPriority，则会优先挑选故障主机，如果未设置，则使用之前的链路和算法
	RequirementId     string
	Site              string
	PhyLabels         map[string]string
	ExcludePhyLabels  map[string]string `json:",omitempty"`
	Constraints       map[string]StopConstraints
	LastRequirementId string
}

type StopConstraints struct {
	Labels map[string]string
	Count  int
}

type CandidatePlan struct {
	Spread      Spread      `json:",omitempty"` //应用实例如何分布
	Constraints Constraints `json:",omitempty"` //应用对宿主机的要求
	Dependency  Dependency  `json:",omitempty"` //应用和应用/数据之间的依赖性
	Affinity    Affinity    `json:",omitempty"` //应用和应用/数据之间的亲和性
	Prohibit    Prohibit    `json:",omitempty"` //应用和应用/数据之间的互斥性

}

type P0M0 map[string]int //级别->个数
//机型(F41,F45,A8)->级别(m0,p0, m0+p0)->个数
type DuMetaInfo struct { //DeployUnit meta信息，比如全局规则
	MetaKey            string
	UpdateTime         string
	PriorityClass      string
	PriorityConstaints map[string]P0M0
}

type GlobalRules struct {
	UpdateTime       string
	Monopolize       MonopolizeDecs
	CpuSetMutex      CpuSetMutexDecs      //cpu物理核互斥规则
	CpuSetMonopolize CpuSetMonopolizeDecs //cpu物理核独占规则
	RealTimeApps     []string
}

type MonopolizeDecs struct {
	AppConstraints []string
	DUConstraints  []string
}

type CpuSetMutexDecs struct {
	AppConstraints []string
	DUConstraints  []string
}

type CpuSetMonopolizeDecs struct {
	AppConstraints []string
	DUConstraints  []string
}

type AppInfo struct { //资源需求规格描述
	//AppName加AppDeployUnit相当于原Hippo的AppTag概念
	BizName       string            //二层业务域名称；zeue，Carbon，Captain
	AppName       string            //相当于Aone的app，一个app下可以由多个分组
	AppDeployUnit string            //相当于电商一个应用下的armory分组，预发/小流量等，同分组下的实例用同一份代码+配置，例如:"buy_center_online"
	DeployUnit    string            //等同于AppDeployUnit, 等全部系统改为DeployUnit后, 删除AppDeployUnit
	ImageName     string            `json:",omitempty"` //镜像uri, 例:"buy-20160918"
	InstanceGroup string            //容器/Job/ECS实例的armory2分组：生产好的容器/JOB实例会自动放到这个分组中, 例:"container-mw-eu13"
	InstanceType  InstanceType      //实例类型：TASK|CONTAINER|KVM
	Priority      InstancePriority  `json:",omitempty"` //实例优先级
	AppPorts      []int             `json:",omitempty"` //":[12200,8080] //App暴露的服务端口，启动后会写入nameserver
	RouteLabels   RouteLabels       `json:",omitempty"` //路由标签
	OverQuota     OverQuota         `json:",omitempty"` //允许的宿主机最大超卖比
	WorkDir       WorkDir           `json:",omitempty"` //应用在宿主机的工作路径
	AppLabels     map[string]string `json:",omitempty"` //{"label1":"value1"} 应用本身提供的自定义标签，用于亲近性相关。
}

func (app AppInfo) getAppSignature() string {
	return fmt.Sprintf("%v_%v_%v", app.BizName, app.AppName, app.AppDeployUnit)
}

type InstancePriority struct {
	MajorPriority int `json:",omitempty"` //32;主优先级，影响抢占：1-32为在线应用（在线应用之间目前都不互相抢占），33- 为离线应用，数字越小优先级越高。
	MinorPriority int `json:",omitempty"` //0 ;辅优先级，主优先级相同时，影响分配先后顺序
}

type RouteLabels struct {
	IpLabel string //:"et2_Unit_CENTER", //对容器ip段的要求，比如单元化中的单元名称
	Site    string //":"eu13",           //所在机房名称
	Stage   string `json:",omitempty"` //"pre",    //XIAOTAOBAO-小淘宝、PRE-预发、COLDBACK-冷备、ONLINE-正式
	Unit    string `json:",omitempty"` //"CENTER"  //CENTER-中心｜UNYUN-深圳云单元｜UNIT-杭州单元｜UNSZ-深圳单元
}

type OverQuota struct { //允许的宿主机最大超卖比
	Enable bool    //是否允许超卖
	Cpu    float32 `json:",omitempty"` //2表示2：1， 32核当64核用； 1个核上最多同时跑2个容器
	Memory float32 `json:",omitempty"` //1.5表示内存超卖50%， 120G，超卖到180G
	Disk   float32 `json:",omitempty"` //3表示磁盘超卖3倍。500G超卖到1.5T
	Gpu    float32 `json:",omitempty"` //4表示gpu设备数超卖4倍，2个GPU设备超卖到8个
}

type WorkDir struct {
	UseHostWorkDir bool   // 是否使用宿主机的workdir，若是，则单机单tag实例数最多为1，否则可指定个数。
	WorkDirTag     string `json:",omitempty"` // 搜索用的workdirTag，指定应用在宿主机的工作路径。为相对路径，可为空。
}

type FpgaSpec struct {
	FpgaCount int
}

type QatSpec struct {
	QatCount int
}

type AllocSpec struct { //资源需求规格描述
	// CPUResource specifies how CPU resources on a node are allocated.
	// If not nil, CPUResource overrides CpuSpec.
	CPUResource *CPUResource `json:"CPUResource,omitempty"`

	// MemResource specifies how Memory resources on a node are allocated.
	// If not nil, MemResource overrides MemorySpec.
	MemResource      *MemResource        `json:"MemResource,omitempty"`
	Cpu              CpuSpec             `json:"cpu,omitempty"`
	Memory           MemorySpec          `json:",omitempty"`
	Disk             map[string]DiskSpec `json:",omitempty"`
	Gpu              GpuSpec             `json:",omitempty"`
	Fpga             FpgaSpec            `json:",omitempty"`
	Qat              QatSpec             `json:",omitempty"`
	Volume           map[string]DiskSpec `json:",omitempty"`
	NetIo            NetIoSpec           `json:",omitempty"`
	IndependentIp    int
	AllocateMode     string   `json:",omitempty"`
	ActionType       string   `json:",omitempty"` // 针对instance需要执行的动作
	ReSchedulerParam string   `json:",omitempty"` // 扩展参数
	DeployUnits      []string `json:",omitempty"`
	PodSnList        []string `json:",omitempty"`
	ContainerSn      string   `json:",omitempty"`
}

// MemResource specifies how Mem resources on a node are allocated.
type MemResource struct {
	// Request specifies the minimum amount of mem resources.

	Request int64 `json:"request"`
	// Limit specifies the maximum amount of mem resources. Same unit as Request.
	// If Limit = 0, it means unlimited.
	// Mandatory: Limit >= Request, except when Limit = 0.
	// If Limit is omitted, it will be set to Request amount.
	Limit int64 `json:"limit"`
}

type Constraints struct { //应用对宿主机的要求
	NamedLabels              NamedLabels       `json:",omitempty"` //well-know标签
	CustomLabels             map[string]string `json:",omitempty"` //自定义标签
	ExtendScalarResources    map[string]int64  `json:",omitempty"` // 自定义扩展scalar类型(标量)资源
	SpecifiedNcIps           []string          `json:",omitempty"` //["10.0.1.1","10.0.1.2"],
	IgnoreLabelBySpecifiedIp bool
	MaxAllocatePercent       int `json:",omitempty"` //只能分配的物理资源百分比(cpu, mem, 暂时不支持disk)。
}

type NamedLabels struct {
	Kernel         string `json:",omitempty"` //"3.10",对内核版本的要求
	OS             string `json:",omitempty"` //:"alios7u2"  //对操作系统的要求
	IpLabel        string `json:",omitempty"` //":"et2_Unit_CENTER" //对容器ip段的要求，比如单元化中的单元名称
	MachineModel   string `json:",omitempty"` //":"D13"  //宿主机机型
	DiskType       string `json:",omitempty"` //":"SSD"  //磁盘类型,待废弃
	NetArchVersion string `json:",omitempty"` //":"3.5"  //网络架构版本：当前取值 3.0|3.5|4.0|4.0v
	NetCardType    string `json:",omitempty"` //WM:万兆网卡，KM：千兆网卡
}

type Spread struct {
	DisasterLevel         int  `json:",omitempty"` //例:1，实例容灾(铺开)等级，取值为1，2，3，4...
	MaxInstancePerHost    int  `json:",omitempty"` //例:1，应用在同一个宿主机上最多部署多少个实例
	MaxInstancePerPhyHost int  `json:",omitempty"` //例:1，应用在同一个ECS物理机上最多部署多少个实例
	MaxInstancePerFrame   int  `json:",omitempty"` //例:2，应用在同一个机框上最多部署多少个实例
	MaxInstancePerRack    int  `json:",omitempty"` //例:3，应用在同一个机柜上最多部署多少个实例
	MaxInstancePerAsw     int  `json:",omitempty"` //例:4，应用在同一组ASW上最多部署多少个实例
	Strictly              bool //例:1，true则Spread策略严格执行，无法满足就返回失败； false则Spread策略尽量执行，无法严格满足也降级返回成功
}

type Dependency struct {
	Application string            `json:",omitempty"` //"app01"        //必须调度到有app01实例的宿主机上
	Instance    string            `json:",omitempty"` //"06a1f75304ba" //必须调度到有这个实例的宿主机上
	Volume      []string          `json:",omitempty"` //search_data1"  //必须调度到有这个数据id的宿主机上
	Labels      map[string]string `json:",omitempty"` //自定义标签，必须调度到有label_name1=value1实例的宿主机上
	Devices     map[string]string `json:",omitempty"` //必须调度到满足 device_name=value1这样的宿主机上, 如DSW=1表示需要和存量容器调度再统一DSW的宿主机上
}

type Affinity struct {
	Application string            `json:",omitempty"` //app01"       //优先调度到有app01实例的宿主机上
	Instance    string            `json:",omitempty"` //06a1f75304ba"   //优先调度到有这个实例的宿主机上
	Volume      []string          `json:",omitempty"` // search_data1"     //优先调度到有这个数据id的宿主机上
	Image       string            `json:",omitempty"` //nginx"             //优先调度到有这个镜像的宿主机上
	Labels      map[string]string `json:",omitempty"` //自定义标签，优先调度到有label_name1=value1实例的宿主机上
	NodeLabels  map[string]string `json:",omitempty"`
}

type Prohibit struct {
	//应用不能部署到有App3应用实例的宿主机上;后面会废弃
	Application string `json:",omitempty"`

	AppConstraints map[string]int `json:",omitempty"`

	DUConstraints map[string]int `json:",omitempty"`

	//自定义标签，不能部署到有label_name1=value1实例的宿主机上
	Labels map[string]string `json:",omitempty"`

	//要排除的宿主机列表
	ExcludedNcIps []string `json:",omitempty"`
}

// CPUResource specifies how CPU resources on a node are allocated.
type CPUResource struct {
	// Request specifies the minimum amount of cpu resources.
	// The unit is MilliCPU. For example, 500 = (0.5 cpu).
	// If Request < 10, it will be set to 10.
	Request int64 `json:"request"`
	// Limit specifies the maximum amount of cpu resources. Same unit as Request.
	// If Limit = 0, it means unlimited.
	// Mandatory: Limit >= Request, except when Limit = 0.
	// If Limit is omitted, it will be set to Request amount.
	Limit int64 `json:"limit"`
}

type CpuSpec struct {
	CpuCount   int         //4   //  运行时占用/分布到4个cpu核(或超线程）；
	CpuQuota   int         //400 //  总共需要多少的cpu计算能力，如果CpuNum不为0， 则必须是100*CpuNum
	CpuSetMode CpuSetMode  //"default" //共享模式(default)，独占模式(exclusive)
	Strategy   CpuStrategy // "sameCoreFirst" // default, sameCoreFirst
	CpuSet     []int
}

type MemorySpec struct {
	HardLimit int64 //: 8589934592 //单位：字节。内存硬上限，，保底
	SoftLimit int64 `json:",omitempty"` //: 8589934592 //单位：字节。内存软上限，待定
	Predict   int64 `json:",omitempty"` //":   4294967296 //单位：字节。内存预期， 通常情况下的经验水位峰值
}

type DiskSpec struct {
	Seq                    int               `json:"-"`
	MountPoint             string            `json:",omitempty"`
	VolumeName             string            `json:",omitempty"`
	DiskSpecType           DiskSpecTypeEnum  `json:"DiskSpecType, omitempty"` // 0表示强制本地磁盘 1表示强制远程磁盘 2表示优先本地磁盘
	Size                   int64             //:61440000000 //单位：字节。磁盘空间大小
	Type                   string            `json:",omitempty"` //:"SSD", "SATA"
	Iops                   int               `json:",omitempty"` //单位：次/秒。应用单实例的磁盘iops配额， NC的iops能力要大于这个值才能被选中；如果后面有iops隔离，那么这个值就是磁盘iops的quota值
	Iobps                  int64             `json:",omitempty"` //单位：字节。应用单实例的磁盘带宽配额， NC的io吞吐能力要大于这个值才能被选中；如果后面有磁盘io带宽隔离，那么这个值就是磁盘带宽的quota值
	Rm                     string            `json:",omitempty"` //ro：readonly， rw：read write 含义等同于docker -v /a=/b:rw中冒号后面的部分
	Exclusive              string            `json:",omitempty"` //none:不独占，instance：实例独占；app：同一个appname可以共用，不和其他app共用
	Mandate                bool              `json:",omitempty"` //是否必需,默认为 true, 若非必需调度时可能不会给予分配
	Driver                 string            `json:",omitempty"` //磁盘driver 类型, 默认为local,
	IncludeVolumeInParam   bool              `json:",omitempty"` // 是否把docker create api 中的-v 选项指定的非bind mount volume 也使用这个磁盘
	IncludeVolumeInImage   bool              `json:",omitempty"` // 是否把镜像中的 volume 也使用这个磁盘
	Opt                    map[string]string `json:",omitempty"` //driver 的选项
	Label                  map[string]string `json:",omitempty"` //磁盘的 label
	AllowUseHostBootDisk   bool              `json:",omitempty"`
	AllowUseDockerRootDisk bool              `json:",omitempty"`
	ContainerPath          string            `json:"-"`
}

type DiskSpecTypeEnum int

const (
	DiskSpecTypeEnumForceLocal  DiskSpecTypeEnum = 0
	DiskSpecTypeEnumForceRemote DiskSpecTypeEnum = 1
	DiskSpecTypeEnumPreferLocal DiskSpecTypeEnum = 2
)

type GpuSpec struct {
	GpuCount int `json:",omitempty"` //gpu设备数
	GPUMem   int `json:",omitempty"` // 单容器gpu显存需求量，单位为MB, 最少200
}

type NetIoSpec struct {
	Bps      int64  `json:",omitempty"` //:30000000   //单位：字节。应用单实例的网络带宽配额（小b）， NC的网络吞吐能力要大于这个值才能被选中；如果后面有网络带宽隔离，那么这个值就是网络带宽的quota值
	Priority string `json:",omitempty"` //: "normal"  //流量优先级，用于网络qos控制；一般都设为normal； master之类的控制流设为high
}

type RequirementResponse struct {
	UpdateTime    string           //例："2016-06-29 19:59:22"
	RequirementId string           //20160815153046 //请求的Id
	Site          string           //一个cluster可能跨多个小机房，二层请求传入的机房名称
	BizName       string           //二层业务域名称；zeue，Carbon，Captain
	AppName       string           //相当于Aone的app，一个app下可以由多个分组
	DeployUnit    string           //相当于电商一个应用下的armory分组，预发/小流
	InstanceSn    string           //"15dfb7b48b3c517e184392ffedd30a464ccae8217cb8c120d1ee20574e1a2930"  //与armory中的sn一致，由apiserver生成
	AllocSpecKey  string           //"4c8g60g_1",  //资源规格定义的key；和前缀拼起来得到规格明细的etcd路径：/applications/allocspecs/$site/4c8g60g_1
	ErrorCode     int              //200
	ErrorMsg      string           `json:",omitempty"` //
	Statistics    []ScheduleResult //分配过程中用到的处理器，此为文本过滤器
	PlanPath      string
	PlanReqId     string
	MemVersion    int64
}

type ScalePreviewSample struct {
	HostSn string
	HostIp string
}

/**App resource requirement info*/
type ScalePreviewRequest struct { //应用需求规格描述
	Requirement
	DisableCache bool
}

/**App resource requirement info*/
type ScalePreviewResponse struct { //应用需求规格描述
	UpdateTime    string //例："2016-06-29 19:59:22",
	RequirementId string //例："123456" ，哪个请求最后确定的这个slot，方便问题排查

	Site       string //例:"et15sqa"
	BizName    string //二层业务域名称；zeus，Carbon，Captain
	AppName    string //相当于Aone的app，一个app下可以由多个分组
	DeployUnit string //相当于电商一个应用下的armory分组，预发/小流量等，同分组下的实例用同一份代码+配置，例如:"buy_center_online"

	AvailableReplicas int
	Samples           []ScalePreviewSample

	ErrorCode  int              //200
	ErrorMsg   string           `json:",omitempty"` //
	Statistics []ScheduleResult //分配过程中用到的处理器，此为文本过滤器
}

type PreviewResponse struct {
	UpdateTime        string           //例："2016-06-29 19:59:22"
	RequirementId     string           //20160815153046 //请求的Id
	Site              string           //一个cluster可能跨多个小机房，二层请求传入的机房名称
	BizName           string           //二层业务域名称；zeue，Carbon，Captain
	AppName           string           //相当于Aone的app，一个app下可以由多个分组
	DeployUnit        string           //相当于电商一个应用下的armory分组，预发/小流
	ErrorCode         int              //200
	ErrorMsg          string           `json:",omitempty"` //
	Statistics        []ScheduleResult //分配过程中用到的处理器，此为文本过滤器
	AvailableReplicas int
}

type BatchStopPreviewResponse struct {
	UpdateTime    string           //例："2016-06-29 19:59:22"
	Type          string           //如果Type的值是faultHostPriority，则会优先挑选故障主机，如果未设置，则使用之前的链路和算法
	RequirementId string           //20160815153046 //请求的Id
	Site          string           //一个cluster可能跨多个小机房，二层请求传入的机房名称
	BizName       string           //二层业务域名称；zeue，Carbon，Captain
	AppName       string           //相当于Aone的app，一个app下可以由多个分组
	DeployUnit    string           //相当于电商一个应用下的armory分组，预发/小流
	ErrorCode     int              //200
	ErrorMsg      string           `json:",omitempty"` //
	Statistics    []ScheduleResult //分配过程中用到的处理器，此为文本过滤器
	Instances     []BatchStopHost
	// RequireCount  int
	// actualCount   int
}

type BatchStopHost struct {
	Sn     string
	HostSn string
}

type ScheduleResult struct {
	Key            string //约束条件Key
	Value          string //约束条件值
	MatchCount     int
	UnmatchCount   int
	UnmatchSample  []string //不满足约束条件的机器IP(样例)
	ScoreHistogram []int    `json:",omitempty"` //":[ //本处理器通过后的机器的算分归一化后的分数直方图，21个桶，也即有21个值，桶间距0.05，只有ScalarFilter和DistinctSort有该项
}

type FilterResult []interface{}

//v2 api /instances/requirements/$site/$deployunit/$InstanceSn
type InstanceRequirement struct {
	UpdateTime    string //例："2016-06-29 19:59:22",
	RequirementId string //例："123456" ，哪个请求最后确定的这个slot，方便问题排查
	PlanReqId     string
	Site          string       //例:"et15sqa"
	BizName       string       //二层业务域名称；zeus，Carbon，Captain
	AppName       string       //相当于Aone的app，一个app下可以由多个分组
	DeployUnit    string       //相当于电商一个应用下的armory分组，预发/小流量等，同分组下的实例用同一份代码+配置，例如:"buy_center_online"
	AllocSpecKey  string       `json:",omitempty"` //资源规格定义的key；和前缀拼起来得到规格明细的etcd路径：/applications/allocspecs/$site/4c8g60g_1
	AllocSpec     AllocSpec    `json:",omitempty"` //方便存放AllocSpec对象, 不出现在json中
	InstanceSn    string       //与armory中的sn一致，由apiserver生成
	InstanceType  InstanceType //例："CONTAINER" //实例类型： TASK|CONTAINER|KVM
	HostIp        string       //指定宿主机扩容，例如: 11.136.23.69, 可用逗号分割标识 ip 列表
	HostSn        string       //指定宿主机扩容，例如: 11.136.23.69, 可用逗号分割标识 ip 列表
	SlotId        int          //兼容老版本 master, 一般为0
	NewSlotId     string
	CpuSet        []int          //指定 cpuset 扩容, 例：[4,5,6,7],  //CPU具体核，允许空字符串
	Status        InstanceStatus //accepted|allocated|reclaiming
	Requirement   *Requirement
}

type ReScheduleReq struct {
	ActionType       string
	Site             string
	HostSn           string
	DeployUnits      []string
	ContainerSn      string
	CPUResource      *CPUResource
	PodSnList        []string
	ReSchedulerParam string // 扩展参数
}

/**App resource requirement info*/
type Preview struct { //应用需求规格描述
	UpdateTime        string
	RequirementId     string      //20160815153046 //请求的Id
	Site              string      //一个cluster可能跨多个小机房，二层请求传入的机房名称
	TargetReplica     int         //50, 目标实例个数
	IncreaseReplica   int         // 实例的增量个数, apiserver 内部转化为TargetReplica后发给 master
	MinReplica        int         //10, 可接受的最小实例个数
	App               AppInfo     //应用信息
	AllocSpec         AllocSpec   //资源需求规格描述
	Spread            Spread      `json:",omitempty"` //应用实例如何分布
	Constraints       Constraints `json:",omitempty"` //应用对宿主机的要求
	Dependency        Dependency  `json:",omitempty"` //应用和应用/数据之间的依赖性
	Affinity          Affinity    `json:",omitempty"` //应用和应用/数据之间的亲和性
	Prohibit          Prohibit    `json:",omitempty"` //应用和应用/数据之间的互斥性
	CandidatePlans    []*CandidatePlan
	Status            string
	LastRequirementId string
	DisableCache      bool
	CacheKey          string
}
