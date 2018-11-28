package swarm

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/kubernetes/test/sigma/util"
)

//LogicInfo mock logicinfo with some field not omitempty, in case cerebellum panic.
type LogicInfo struct {
	UpdateTime            string            `json:",omitempty"` // "UpdateTime": "2016-06-29 13:14:16","2016-06-29 13:14:16"
	ServerCreateTime      string            `json:",omitempty"` // "ServerCreateTime": "2013-02-25 22:39:58"
	ServerModifyTime      string            `json:",omitempty"` // "ServerModifyTime": "2015-12-10 15:16:55"
	Site                  string            `json:",omitempty"` // "Site": "et15sqa", //机房
	Region                string            `json:",omitempty"` // "Region": "****", //所属域：集团生产|阿里云|蚂蚁
	GeogRegion            string            `json:",omitempty"` // "GeogRegion": "****", //地理位置简称
	NcServerId            int64             `json:",omitempty"` // "NcServerId": 689511, //在cmdb系统中的唯一id, int64
	NcSn                  string            `json:",omitempty"` // 将废弃, 统一成命名成HostSn，Hostname, HostIp; 用HostSn替代
	NcHostname            string            `json:",omitempty"` // 将废弃, 统一成命名成HostSn，Hostname, HostIp; 用Hostname替代
	NcIp                  string            `json:",omitempty"` // 将废弃, 统一成命名成HostSn，Hostname, HostIp; 用HostIp替代
	HostSn                string            `json:",omitempty"` // "HostSn": "Armory取到的sn信息",
	Hostname              string            `json:",omitempty"` // "Hostname": "e19h19470.et15sqa",
	HostIp                string            `json:",omitempty"` // "HostIp": "100.81.153.117",
	ParentServiceTag      string            `json:",omitempty"` // "ParentServiceTag": "", //父设备, 物理机为机框sn，容器为宿主机sn
	Room                  string            `json:",omitempty"` // "Room": "A2-3.Eu6", //房间
	Rack                  string            `json:",omitempty"` // "Rack": "H21", //机架
	NetArchVersion        string            `json:",omitempty"` // "NetArchVersion": "3.0/3.5/4.0",
	UplinkHostName        string            `json:",omitempty"` // "UplinkHostName": "PSW-A5-1-H19.ET15", //上联设备主机名。 上联设备一般就是PSW
	UplinkIp              string            `json:",omitempty"` // "UplinkIp": "", //上联设备IP
	UplinkSn              string            `json:",omitempty"` // "UplinkSn": "", //上联设备Sn
	ASW                   string            `json:",omitempty"` // "ASW": "ASW-A5-1-H19.ET15", //ASW
	LogicPod              string            `json:",omitempty"` // "LogicPod": "6", //逻辑Pod，比如： 6
	Pod                   string            `json:",omitempty"` //"Pod": "6", //Pod，比如： 6
	DswCluster            string            `json:",omitempty"` // "DswCluster": "EU6-APPDB-G1", //网络集群（网络核心）比如：EU6-APPDB-G1
	NetLogicSite          string            `json:",omitempty"` // "NetLogicSite": "EU6-MASTER", //网络上的逻辑机房，与网络相关比如：EU6-MASTER
	SmName                string            `json:",omitempty"` // "SmName": "A8", //机型简称
	Model                 string            `json:",omitempty"` // "Model": "PowerEdge R510", //机型
	IdcManagerState       string            `json:",omitempty"` // "IdcManagerState": "READY|REPAIR", //IDC管控状态，用于标记是否存在硬件故障
	EnableOverQuota       bool              `json:",omitempty"` // "EnableOverQuota": true|false, //是否超卖
	CpuOverQuota          float64           `json:",omitempty"` // "CpuOverQuota": 3.5,
	MemoryOverQuota       float64           `json:",omitempty"` // "MemoryOverQuota": 3.5,
	DiskOverQuota         float64           `json:",omitempty"` // "DiskOverQuota": 3.5,n
	GpuOverQuota          float64           `json:",omitempty"` // "GpuOverQuota": 3.5,
	ExtLabels             map[string]string // *可变的扩展标签
	ExtendScalarResources map[string]int64
	MandatoryLabel        string            //"MandatoryLabel":"MixRun" //强制匹配标，最多只允许一个
	MandatoryLabels       map[string]string //强制标，由boss强制必填的标签和用户自定义标签。
	DiskLabels            map[string]map[string]string
}

type LocalInfo struct {
	UpdateTime    string
	OS            string
	Kernel        string
	CpuClockSpeed string

	NcIp       string //统一成命名成HostSn，Hostname, HostIp; 同HostIp; 都修改完后删除
	NcSn       string //统一成命名成HostSn，Hostname, HostIp; 同HostSn; 都修改完后删除
	NcHostname string //统一成命名成HostSn，Hostname, HostIp; 同Hostname; 都修改完后删除
	HostIp     string //统一成命名成HostSn，Hostname, HostIp; 同NcIp;
	HostSn     string //统一成命名成HostSn，Hostname, HostIp; 同NcSn,
	Hostname   string //统一成命名成HostSn，Hostname, HostIp; 同NcHostname,

	DaemonPort          int //本地取到的Daemon端口
	CpuNum              int
	Memory              int64
	IsECS               bool
	DiskInfos           []DiskInfo
	NetInfos            []NetInfo
	GpuInfos            GpuInfos
	DevicePluginMessage map[string][]PluginDeviceInfo
	CpuTopoInfos        []CpuTopoInfo
	TunedLabels         map[string]string // 中转
}

type DiskInfo struct {
	FileSystem string
	Type       string
	Size       int64
	MountPoint string
	DiskType   string
	IsBootDisk bool
}

type NetInfo struct {
	// Device name
	Name string

	// Mac Address
	MacAddress string

	// Speed in MBits/s
	Speed int64

	// Maximum Transmission Unit
	Mtu int64
}

type GpuInfo struct {
	ProductName string
	Count       int
}

type PluginDeviceInfo struct {
	ID     string
	Health string
}

type CpuTopoInfo struct {
	CpuId    int
	CoreId   int
	SocketId int
}

type DockerInfo struct {
	UpdateTime string
	Version    string
	Volumes    []VolumeInfo
	HostSn     string
}

type VolumeInfo struct {
	Driver string
	Name   string
	Size   int64
}

/*new gpuinfos add more info*/
type GpuInfos struct {
	Version        GVersion
	ControlDevices []string
	VolumeDriver   string
	Volumes        []string
	Devices        []SGDevice
}

/*simple gpu device. no pci,clocks,topology according to GDevice */
type SGDevice struct {
	UUID        string
	Path        string
	Model       string
	Power       int
	CPUAffinity int
	Family      string
	Arch        string
	Cores       int
	Memory      GMemory
}

type GpuFullVolum struct {
	VolumeDriver string
	Volumes      []string
	Devices      []string
}

type GpuFullInfo struct {
	Version GVersion
	Devices []GDevice
}

type GDevice struct {
	SGDevice
	PCI      GPCI
	Clocks   GClock
	Topology []GTopology
}

type GPCI struct {
	BusID     string
	BAR1      int
	Bandwidth int
}

type GClock struct {
	Cores  int
	Memory int
}

type GTopology struct {
	BusID string
	Link  int
}

type GMemory struct {
	ECC       bool
	Global    int
	Shared    int
	Constant  int
	L2Cache   int
	Bandwidth int
}

type GVersion struct {
	Driver string
	CUDA   string
}

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

type InstanceType string
type CpuStrategy string
type GpuSetMode string

type DiskQuota struct {
	HostPath string
	DiskSize int64
}

type GpuQuota struct {
	DevicePath string
	CacheSize  int64
}

type ScalarResource struct {
	Name   string `json:"Name"`
	Type   string `json:"Type"`
	Amount int64  `json:"Amount"`
	Value  string `json:"Value"`
}

type AllocPlan struct {
	UpdateTime              string       //例："2016-06-29 19:59:22",
	RequirementId           string       //例："123456" ，哪个请求最后确定的这个slot，方便问题排查
	Site                    string       //和path中的site一致, 方便编码；使代码更简洁一致
	BizName                 string       //二层业务域名称；zeue，Carbon，Captain
	AppName                 string       //相当于Aone的app，一个app下可以由多个分组
	DeployUnit              string       //相当于电商一个应用下的armory分组，预发/小流量等，同分组下的实例用同一份代码+配置，例如:"buy_center_online"
	AllocSpecKey            string       //资源需求规格描述Key
	HostIp                  string       //宿主机IP，例如: 11.136.23.69
	HostSn                  string       //宿主机Sn，例如: 214247788-02A
	HostPath                string       //Host在etcd上的地址路径：如："$cell/$securityRegion/214247788-02A"
	SlotId                  int          //如：宿主机上唯一
	Status                  SlotStatus   //"allocated|starting|started|reclaimed|stopping|stopped", //状态。
	InstanceSn              string       //例：20170322093955333fe415840dc85 时间戳yyyyMMddHHmmssSSS+12位uuid：共29个字符
	InstanceType            InstanceType //例："CONTAINER" //实例类型： TASK|CONTAINER|KVM
	CpuSet                  []int        //例：[4,5,6,7],  //CPU具体核，允许空字符串
	CpusetMode              string
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
	Resources               []ScalarResource
	Declarations            []ScalarResource
	AppLabels               map[string]string
	ContainerMandatoryLabel string `json:",omitempty"` //弹性资源池标签
}

// Node represents a sigma2.0 node
type Node struct {
	LogicInfo *LogicInfo
	LocalInfo *LocalInfo
}

// CreateOrUpdateNodeLogicInfoSmName updates sigma node LogicInfo
func CreateOrUpdateNodeLogicInfoSmName(nodeName, value string) error {
	node := GetNode(nodeName)

	node.LogicInfo.SmName = value
	updateNode(nodeName, node)

	return nil
}

// CreateOrUpdateNodeLabel updates sigma node labels
func CreateOrUpdateNodeLabel(nodeName string, labels map[string]string) error {
	node := GetNode(nodeName)
	if node.LogicInfo.ExtLabels == nil {
		node.LogicInfo.ExtLabels = make(map[string]string)
	}

	for key, value := range labels {
		node.LogicInfo.ExtLabels[key] = value
	}

	updateNode(nodeName, node)
	return nil
}

// EnsureNodeHasLabels checks node has the specified labels
func EnsureNodeHasLabels(nodeName string, labels map[string]string) error {
	node := GetNode(nodeName)

	for key, value := range labels {
		data, ok := node.LogicInfo.ExtLabels[key]
		Expect(ok).Should(Equal(true))
		Expect(data).Should(Equal(value))
	}
	return nil
}

// DeleteNodeLabels removes label from sigma node
func DeleteNodeLabels(nodeName string, labelNames ...string) error {
	node := GetNode(nodeName)

	for _, labelKey := range labelNames {
		delete(node.LogicInfo.ExtLabels, labelKey)
	}

	updateNode(nodeName, node)
	return nil
}

// CreateOrUpdateNodeMandatoryLabel updates sigma node mandatory labels
func CreateOrUpdateNodeMandatoryLabel(nodeName string, labels map[string]string) error {
	node := GetNode(nodeName)

	for key, value := range labels {
		node.LogicInfo.MandatoryLabels[key] = value
	}

	updateNode(nodeName, node)

	return nil
}

// EnsureNodeHasMandatoryLabels checks node has the specified mandatory labels
func EnsureNodeHasMandatoryLabels(nodeName string, labels map[string]string) error {
	node := GetNode(nodeName)

	for key, value := range labels {
		data, ok := node.LogicInfo.MandatoryLabels[key]
		Expect(ok).Should(Equal(true))
		Expect(data).Should(Equal(value))
	}
	return nil
}

// DeleteNodeMandatoryLabels removes mandatory label from sigma node
func DeleteNodeMandatoryLabels(nodeName string, labelNames ...string) error {
	node := GetNode(nodeName)

	for _, labelKey := range labelNames {
		delete(node.LogicInfo.MandatoryLabels, labelKey)
	}

	updateNode(nodeName, node)
	return nil
}

// GetNode returns a sigma 2.0 node
func GetNode(nodeName string) *Node {
	node := &Node{
		LogicInfo: &LogicInfo{
			ExtLabels:             map[string]string{},
			ExtendScalarResources: map[string]int64{},
			MandatoryLabels:       map[string]string{},
			DiskLabels:            map[string]map[string]string{},
		},
	}
	// query the site info from armory
	nsInfo, err := util.QueryArmory(fmt.Sprintf("sn=='%v'", nodeName))
	Expect(err).NotTo(HaveOccurred(), "query armory return error:%s", err)

	site := strings.ToLower(nsInfo[0].Site)
	key := "/nodes/logicinfos/" + site + "/" + nsInfo[0].ServiceTag
	data, err := EtcdGet(key)
	Expect(err).NotTo(HaveOccurred())

	err = json.Unmarshal(data, node.LogicInfo)
	Expect(err).NotTo(HaveOccurred())

	key = "/nodes/localinfos/" + site + "/" + nsInfo[0].ServiceTag
	data, err = EtcdGet(key)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("EtcdGet key:%s, data:%s", key, data))

	node.LocalInfo = &LocalInfo{}
	err = json.Unmarshal(data, node.LocalInfo)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Unmarshal key:%s, data:%s", key, data))
	return node
}

// /nodes/allocplans/et15/818210969/
func GetAllocPlans(nodeName string) []*AllocPlan {
	// query the site info from armory
	nsInfo, err := util.QueryArmory(fmt.Sprintf("sn=='%v'", nodeName))
	Expect(err).NotTo(HaveOccurred(), "query armory return error:%s", err)

	site := strings.ToLower(nsInfo[0].Site)
	key := "/nodes/allocplans/" + site + "/" + nsInfo[0].ServiceTag
	dataKeys, err := EtcdGetPrefix(key)
	Expect(err).NotTo(HaveOccurred())

	ret := []*AllocPlan{}

	By("get allocplans:" + string(len(dataKeys)))
	for _, keyValue := range dataKeys {
		data, err := EtcdGet(string(keyValue.Key))
		By("get:" + string(keyValue.Key))
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("EtcdGet key:%s, data:%s", string(keyValue.Key), data))

		plan := &AllocPlan{}
		err = json.Unmarshal(data, plan)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Unmarshal key:%s, data:%s", string(keyValue.Key), data))
		ret = append(ret, plan)
	}
	return ret
}

func updateNode(nodename string, node *Node) {
	logicInfoData, err := json.Marshal(node.LogicInfo)
	Expect(err).ShouldNot(HaveOccurred())

	// query the site info from armory
	nsInfo, err := util.QueryArmory(fmt.Sprintf("sn=='%v'", nodename))
	Expect(err).NotTo(HaveOccurred(), "query armory return error:%s", err)
	site := strings.ToLower(nsInfo[0].Site)
	key := "/nodes/logicinfos/" + site + "/" + nsInfo[0].ServiceTag

	err = etcdPut(key, string(logicInfoData))
	Expect(err).ShouldNot(HaveOccurred())
}

// GetNodeLogicinfos
func GetNodeLogicinfos(nodename string) (key string, logicinfo *LogicInfo) {
	// query the site info from armory
	nsInfo, err := util.QueryArmory(fmt.Sprintf("sn=='%v'", nodename))
	Expect(err).NotTo(HaveOccurred(), "query armory return error:%s", err)

	site := strings.ToLower(nsInfo[0].Site)
	key = "/nodes/logicinfos/" + site + "/" + nsInfo[0].ServiceTag
	value, err := EtcdGet(key)
	Expect(err).NotTo(HaveOccurred(), "get key %s from etcd return error:%s", key, err)

	var nodeLogicInfo LogicInfo
	err = json.Unmarshal(value, &nodeLogicInfo)
	Expect(err).ShouldNot(HaveOccurred())

	return key, &nodeLogicInfo
}

// SetNodeOverQuota set the node to be overquota
func SetNodeOverQuota(nodename string, cpuOverQuota float64, memoryOverQuota float64) {
	_, nodeLogicInfo := GetNodeLogicinfos(nodename)

	nodeLogicInfo.EnableOverQuota = true
	nodeLogicInfo.CpuOverQuota = cpuOverQuota
	nodeLogicInfo.MemoryOverQuota = memoryOverQuota

	node := Node{}
	node.LogicInfo = nodeLogicInfo
	updateNode(nodename, &node)
}

// SetNodeToNotOverQuota
func SetNodeToNotOverQuota(nodename string) {
	_, nodeLogicInfo := GetNodeLogicinfos(nodename)

	nodeLogicInfo.EnableOverQuota = false
	nodeLogicInfo.CpuOverQuota = 1.0
	nodeLogicInfo.MemoryOverQuota = 1.0

	node := Node{}
	node.LogicInfo = nodeLogicInfo
	updateNode(nodename, &node)
}
