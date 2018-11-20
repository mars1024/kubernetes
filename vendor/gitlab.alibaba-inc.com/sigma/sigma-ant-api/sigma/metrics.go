package sigma

//etcd:/metrics/$site/$sn
type MachineMetrics struct {
	UpdateTime               string
	ContainerVersion         string //容器版本
	LoadAverageOneMinute     float64
	LoadAverageFiveMinute    float64
	LoadAverageFifteenMinute float64
	ProcCount                int
	Interfaces               []InterfaceMetrics
	Cpus                     []CpuCoreMetrics
	CpusHistory              []CpuMetricsHistory
	Memory                   MemMetrics
	DiskIos                  []DiskIoMetrics
	DiskSpaces               []DiskSpaceMetrics
	NetTrafficInfos          map[string]NetTrafficMetrics //网络消耗情况

	// InstanceInfos            []InstanceMetrics

}

type CpuMetricsHistory struct {
	UpdateTime string
	Cpus       []CpuCoreMetrics
}

// CPU: /metrics/$site/$sn/cpu
type CpuLoadMetrics struct {
	UpdateTime               string
	LoadAverageOneMinute     float64
	LoadAverageFiveMinute    float64
	LoadAverageFifteenMinute float64
	Cpus                     []CpuCoreMetrics
}

type CpuCoreMetrics struct {
	Name                           string
	Us, Sy, Ni, Id, Wa, Hi, Si, St float64
}
type MemMetrics struct {
	UpdateTime string

	Total     int64
	Used      int64
	Free      int64
	Cached    int64 //系统分配但未被使用的cache 数量。
	Available int64

	Swap     int64 //表示硬盘上交换分区的总量
	SwapUsed int64 //表示硬盘上已使用的交换分区
	SwapFree int64 //表示硬盘上未使用的交换分区
}

// 磁盘: /metrics/$site/$sn/disk
type DiskMetrics struct {
	UpdateTime string
	Spaces     []DiskSpaceMetrics `json:"DiskSpaceMetrics"`
	IO         []DiskIoMetrics    `json:"DiskIoMetrics"`
}

type DiskSpaceMetrics struct {
	FileSystem string
	Size       int64 //单位：字节
	Used       int64 //单位：字节
	Available  int64 //单位：字节
	MountPoint string
}

type DiskIoMetrics struct {
	FileSystems []string `json:"-"`
	Device      string
	Iops        float64
}

// 网络: /metrics/$site/$sn/net
type NetMetrics struct {
	UpdateTime string
	Traffic    NetTrafficMetrics
	Interfaces []InterfaceMetrics
}

type NetTrafficMetrics struct {
	BytinByte  int64
	BytoutByte int64
	PktinByte  int64
	PktoutByte int64
	PkterrByte int64
	PktdrpByte int64
}

type InterfaceMetrics struct {
	InterfaceName string
	LinkType      string //连接类型
	RxPackets     int64  //单位个
	TxPackets     int64  //单位个
	RxBytes       int64  //单位字节
	TxBytes       int64  //单位字节
}

type InstanceInfoMetrics struct {
	UpdateTime    string
	SlotId        int
	ContainerIp   string
	ContainerName string
	InstanceType  InstanceType
	BizName       string
	AppName       string
	HippoWorkDir  string
	HippoAppName  string
	HippoRoleName string
	DeployUnit    string
}

type InstanceCpuMetrics struct {
	UpdateTime               string
	LoadAverageOneMinute     float64
	LoadAverageFiveMinute    float64
	LoadAverageFifteenMinute float64
	CpuQuota                 float64
	CpuUser                  float64
	CpuSys                   float64
	CpuTotal                 float64
	CpuUserPercent           float64
	CpuSysPercent            float64
	CpuTotalPercent          float64
}

type InstanceMemoryMetrics struct {
	UpdateTime         string
	Total              int64
	Used               int64
	Free               int64
	Cached             int64
	Available          int64
	FailCnt            int64
	UnAvailablePercent float64
}

type InstanceNetMetrics struct {
	UpdateTime string

	Read               int64
	Read_Packets       int64
	Read_Errors        int64
	Read_Drops         int64
	Read_Speed         float64
	Read_Packets_Speed float64
	Read_Errors_Speed  float64
	Read_Drops_Speed   float64

	Write               int64
	Write_Packets       int64
	Write_Errors        int64
	Write_Drops         int64
	Write_Speed         float64
	Write_Packets_Speed float64
	Write_Errors_Speed  float64
	Write_Drops_Speed   float64
}

type PathQuotaUsageInfo struct {
	HostPath      string
	ContainerPath string
	Quota         int64
	UsedPercent   float64
}

type InstanceDiskMetrics struct {
	UpdateTime string

	Read             int64
	Read_Ops         int64
	Read_Speed       float64
	Read_Ops_Speed   float64

	Write            int64
	Write_Ops        int64
	Write_Speed      float64
	Write_Ops_Speed  float64

	Quota_Usage_Info []PathQuotaUsageInfo
}
