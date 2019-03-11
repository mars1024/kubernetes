/*
Copyright 2019 The Alipay Authors. All Rights Reserved.
*/

package apis

const (
	// Sigma best effort resource name which used as extended resource.
	SigmaBEResourceName     = "resource.sigma.ali/running-cpu-quota"
	ResourceConfigNamespace = "resource-system"
	ResourceConfigDataKey   = "config"
)

const (
	CGROUP_PARENT_OFFLINE = "sigma-stream"
	CGROUP_PARENT_ONLINE  = "sigma"
	CGROUP_PARENT_SYS     = "system-agent"
)

const (
	CPU_CGROUP_ISOLATION     = "CPUCgroupIsolation"
	MEMORY_CGROUP_ISOLATION  = "MemoryCgroupIsolation"
	NET_AIS_ISOLATION        = "NetAisIsolation"
	NET_CGROUP_ISOLATION     = "NetCgroupIsolation"
	BLKIO_CGROUP_ISOLATION   = "BlkIOCgroupIsolation"
	CPUCAT_CGROUP_ISOLATION  = "CPUCatCgroupIsolation"
	CPUCAT_RESCTRL_ISOLATION = "CPUCatResctrlIsolation"
)

type ResourceConfig struct {
	Version   string      `json:"version"`
	Residents []*Resident `json:"residents"`
}

type Resident struct {
	Name         string     `json:"name"`
	CgroupParent string     `json:"cgroupParent"`
	Isolation    *Isolation `json:"isolation"`
}

type Isolation struct {
	CPUCgroupIsolation           *CPUCgroupIsolation           `json:"cpuCgroupIsolation"`
	MemoryCgroupIsolation        *MemoryCgroupIsolation        `json:"memoryCgroupIsolation"`
	NetAisIsolation              *NetAisIsolation              `json:"netAisIsolation"`
	NetCgroupIsolation           *NetCgroupIsolation           `json:"netCgroupIsolation"`
	BlkIOCgroupIsolation         *BlkIOCgroupIsolation         `json:"blkIOCgroupIsolation"`
	BlkIOCgroupBufferedIsolation *BlkIOCgroupBufferedIsolation `json:"blkIOCgroupBufferedIsolation"`
	CPUCatCgroupIsolation        *CPUCatCgroupIsolation        `json:"cpuCatCgroupIsolation"`
	CPUCatResctrlIsolation       *CPUCatResctrlIsolation       `json:"cpuCatResctrlIsolation"`
}

type CPUCgroupIsolation struct {
	CPUShares int `json:"cpuShares"`
	CfsQuota  int `json:"cfsQuota"`
	CfsPeriod int `json:"cfsPeriod"`
	BvtWarpNs int `json:"bvtWarpNs"`
}

type MemoryCgroupIsolation struct {
	MemoryRatio float64 `json:"memoryRatio"`
}

type NetAisIsolation struct {
	TotalSpeedRatio     float64 `json:"totalSpeedRatio"`
	HighPriorityRatio   float64 `json:"highPriorityRatio"`
	MediumPriorityRatio float64 `json:"mediumPriorityRatio"`
	LowPriorityRatio    float64 `json:"lowPriorityRatio"`
}

type NetCgroupIsolation struct {
	TotalSpeedRatio     float64 `json:"totalSpeedRatio"`
	HighPriorityRatio   float64 `json:"highPriorityRatio"`
	MediumPriorityRatio float64 `json:"mediumPriorityRatio"`
	LowPriorityRatio    float64 `json:"lowPriorityRatio"`
}

type BlkIOCgroupIsolation struct {
	ReadBPS   int64 `json:"readBPS"`
	WriteBPS  int64 `json:"writeBPS"`
	ReadIOPS  int64 `json:"readIOPS"`
	WriteIOPS int64 `json:"writeIOPS"`
}

type BlkIOCgroupBufferedIsolation struct {
	BufferedWriteSwitch bool  `json:"bufferedWriteSwitch"`
	BufferedWriteBPS    int64 `json:"bufferedWriteBPS"`
}

type CPUCatCgroupIsolation struct {
}

type CPUCatResctrlIsolation struct {
}
