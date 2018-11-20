package sigma

//etcd:/nodes/status/$site/$sn
type NodeStatus struct {
	UpdateTime    string //"2016-06-28 19:31:04"
	Site          string
	Sn            string            // 统一成命名成HostSn，Hostname; "宿主机的sn" 都改成HostSn后可以删除
	HostSn        string            // 统一成命名成HostSn，Hostname; 同Sn,
	Hostname      string            //"rs3b17035.et2sqa",
	Status        string            //"uninit|locked|unavailable|available|dead", //只有available的机器才是可用的，其它状态只是为了便于管控区分使用。
	Version       string            //"1489393873189"
	Zone          string            //"ONLINE", //ONLINE, OFFLINE, HYBRID,
	OfflineStatus string            //"uninit|locked|unavailable|available|dead", //只有available的机器才是可用的，其它状态只是为了便于管控区分使用。
	ErrorList     map[string]string //
	WarnningList  map[string]string //
}

type MasterNodeStatus struct {
	UpdateTime    string //"2016-06-28 19:31:04"
	OnlineVersion string
}
