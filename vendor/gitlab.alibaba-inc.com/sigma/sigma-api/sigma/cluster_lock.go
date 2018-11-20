package sigma

// @author: 智清
// APIServer主备切换

type CrossClusterLock struct {
	UpdateTime string
	ClusterID  string
}

type ClusterLockIpsWhite struct {
	UpdateTime           string/**更改时间*/ `json:",omitempty"`
	ApiserverIpWhiteList []string/**APIServer白名单*/ `json:",omitempty"`
}

type ClusterLockHeartBeat struct {
	UpdateTime        string/**更改时间*/ `json:",omitempty"`
	ActiveHeartBeatIp string/**ActiveHeartBeatIp*/ `json:",omitempty"`
}
