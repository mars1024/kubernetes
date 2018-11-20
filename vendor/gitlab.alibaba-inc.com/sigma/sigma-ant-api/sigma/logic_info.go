package sigma

//http://docs.alibaba-inc.com/pages/viewpage.action?pageId=422874776#d0-etcd%E6%95%B0%E6%8D%AE%E7%BB%93%E6%9E%84%E6%8E%A5%E5%8F%A3%E5%AE%9A%E4%B9%89-%E3%80%90d202%E3%80%91etcdnodeslogicinfos%EF%BC%88%E6%9C%BA%E4%BD%8D%E8%99%9A%E6%8B%9F%E6%AF%94%E7%AD%89%E9%80%BB%E8%BE%91%E4%BF%A1%E6%81%AF%EF%BC%89
//etcd:/nodes/logicinfos/$site/$sn
type LogicInfo struct {
	UpdateTime            string                       `json:",omitempty"` // "UpdateTime": "2016-06-29 13:14:16","2016-06-29 13:14:16"
	ServerCreateTime      string                       `json:",omitempty"` // "ServerCreateTime": "2013-02-25 22:39:58"
	ServerModifyTime      string                       `json:",omitempty"` // "ServerModifyTime": "2015-12-10 15:16:55"
	Site                  string                       `json:",omitempty"` // "Site": "et15sqa", //机房
	Region                string                       `json:",omitempty"` // "Region": "****", //所属域：集团生产|阿里云|蚂蚁
	GeogRegion            string                       `json:",omitempty"` // "GeogRegion": "****", //地理位置简称
	NcServerId            int64                        `json:",omitempty"` // "NcServerId": 689511, //在cmdb系统中的唯一id, int64
	NcSn                  string                       `json:",omitempty"` // 将废弃, 统一成命名成HostSn，Hostname, HostIp; 用HostSn替代
	NcHostname            string                       `json:",omitempty"` // 将废弃, 统一成命名成HostSn，Hostname, HostIp; 用Hostname替代
	NcIp                  string                       `json:",omitempty"` // 将废弃, 统一成命名成HostSn，Hostname, HostIp; 用HostIp替代
	HostSn                string                       `json:",omitempty"` // "HostSn": "Armory取到的sn信息",
	Hostname              string                       `json:",omitempty"` // "Hostname": "e19h19470.et15sqa",
	HostIp                string                       `json:",omitempty"` // "HostIp": "100.81.153.117",
	ParentServiceTag      string                       `json:",omitempty"` // "ParentServiceTag": "", //父设备, 物理机为机框sn，容器为宿主机sn
	Room                  string                       `json:",omitempty"` // "Room": "A2-3.Eu6", //房间
	Rack                  string                       `json:",omitempty"` // "Rack": "H21", //机架
	NetArchVersion        string                       `json:",omitempty"` // "NetArchVersion": "3.0/3.5/4.0",
	UplinkHostName        string                       `json:",omitempty"` // "UplinkHostName": "PSW-A5-1-H19.ET15", //上联设备主机名。 上联设备一般就是PSW
	UplinkIp              string                       `json:",omitempty"` // "UplinkIp": "", //上联设备IP
	UplinkSn              string                       `json:",omitempty"` // "UplinkSn": "", //上联设备Sn
	ASW                   string                       `json:",omitempty"` // "ASW": "ASW-A5-1-H19.ET15", //ASW
	LogicPod              string                       `json:",omitempty"` // "LogicPod": "6", //逻辑Pod，比如： 6
	Pod                   string                       `json:",omitempty"` //"Pod": "6", //Pod，比如： 6
	DswCluster            string                       `json:",omitempty"` // "DswCluster": "EU6-APPDB-G1", //网络集群（网络核心）比如：EU6-APPDB-G1
	NetLogicSite          string                       `json:",omitempty"` // "NetLogicSite": "EU6-MASTER", //网络上的逻辑机房，与网络相关比如：EU6-MASTER
	SmName                string                       `json:",omitempty"` // "SmName": "A8", //机型简称
	Model                 string                       `json:",omitempty"` // "Model": "PowerEdge R510", //机型
	IdcManagerState       string                       `json:",omitempty"` // "IdcManagerState": "READY|REPAIR", //IDC管控状态，用于标记是否存在硬件故障
	EnableOverQuota       bool                         `json:",omitempty"` // "EnableOverQuota": true|false, //是否超卖
	CpuOverQuota          float64                      `json:",omitempty"` // "CpuOverQuota": 3.5,
	MemoryOverQuota       float64                      `json:",omitempty"` // "MemoryOverQuota": 3.5,
	DiskOverQuota         float64                      `json:",omitempty"` // "DiskOverQuota": 3.5,n
	GpuOverQuota          float64                      `json:",omitempty"` // "GpuOverQuota": 3.5,
	ExtLabels             map[string]string            `json:",omitempty"` // *可变的扩展标签
	ExtendScalarResources map[string]int64             `json:",omitempty"`
	MandatoryLabel        string                       `json:",omitempty"` //"MandatoryLabel":"MixRun" //强制匹配标，最多只允许一个
	MandatoryLabels       map[string]string            //强制标，由boss强制必填的标签和用户自定义标签。
	DiskLabels            map[string]map[string]string `json:",omitempty"`
}
