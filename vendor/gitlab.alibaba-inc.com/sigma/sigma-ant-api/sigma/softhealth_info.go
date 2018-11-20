package sigma

type SoftHealthInfo struct {
	UpdateTime          string
	FileSystem          []MountInfo
	HostSn              string
	Memory              Memory
	Swap                Swap
	Loads               Loads
	ProcessNum          int64
	CreateInstanceNum   int64
	IsECS               bool
	ContainerStatus     ContainerStatus
	LxcInfo             CommonInfo
	SshdInfo            CommonInfo
	DockerDaemonInfo    CommonInfo
	HippoSalveInfo      CommonInfo
	YumInfo             CommonInfo
	OssInfo             CommonInfo
	HostCgroupInfo      CommonInfo
	ContainerCgroupInfo ContainerCgroupInfo
}

type MountInfo struct {
	MountPoint  string
	Filesystem  string
	SizeMega    int64
	UsedMega    int64
	UsedPercent int64 // 96标示96%
}
type Memory struct {
	TotalMega     int64
	UsedMega      int64
	FreeMega      int64
	SharedMega    int64
	BuffCacheMega int64
}

type Swap struct {
	TotalMega int64
	UsedMega  int64
	FreeMega  int64
}
type Loads struct {
	One     float64
	Five    float64
	Fifteen float64
}
type ContainerStatus struct {
	Total      int64
	Running    int64
	Created    int64
	Exited     int64
	Paused     int64
	Restarting int64
}

type CommonInfo struct {
	Status      bool
	Version     string
	Description string
}

type ContainerCgroupInfo struct {
	Status bool
	Total  int64
	OK     int64
}
