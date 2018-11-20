package sigma

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

	DaemonPort   int //本地取到的Daemon端口
	CpuNum       int
	Memory       int64
	IsECS        bool
	DiskInfos    []DiskInfo
	NetInfos     []NetInfo
	GpuInfos     GpuInfos
	CpuTopoInfos []CpuTopoInfo
	TunedLabels  map[string]string // 中转
	UltronSize   int64
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
