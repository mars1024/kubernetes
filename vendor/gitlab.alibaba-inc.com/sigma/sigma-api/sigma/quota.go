package sigma

//etcd:/quota/groups/$groupname
type QuotaGroup struct {
	QuotaAssigned
	DiskSpaceQuota int64 //1024000000, //磁盘quota，单位按默认的字节
	DiskIopsQuota  int   //1000, //disk_iops quota
	DiskIobpsQuota int64 //1000, //disk bps quota ，单位bps
	CpuNumQuota    int   //100000, //CPU核数量quota
	MemQuota       int64 //1000, //内存quota，单位按默认的字节
	IpNumQuota     int   //1000, //（保留字段，目前暂不使用）。需要占用多少IP资源，对于容器实际上就是容器数
	NetIobpsQuota  int64 //1000, //net io bps
}

//etcd:/quota/allocated/$site/$bizname/$appname/$deployunit
type QuotaAssigned struct {
	UpdateTime    string
	DiskSpaceAssigned int64 //1024, //已分配磁盘，单位按默认的字节
	DiskIopsAssigned  int   //1000, //已分配disk iops
	DiskIobpsAssigned int64 //1000, //已分配disk_bps，单位bps
	CpuNumAssigned    int   //100000, //已分配cpu核数
	MemAssigned       int64 //1000, //已分配内存数，单位按默认的字节
	NetIobpsAssigned  int64 //1000, //已分配Net bps
	IpNumAssigned     int   //1000, //（保留字段，目前暂不使用）。已占用多少IP资源，对于容器实际上就是容器数
}
