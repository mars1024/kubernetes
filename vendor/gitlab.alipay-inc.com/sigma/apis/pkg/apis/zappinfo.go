/*
Copyright 2019 The Alipay Authors. All Rights Reserved.
*/

package apis

type ZappinfoStatus string

var (
	// 未交付
	ZappinfoStatusUninit = ZappinfoStatus("uninit")
	// 已上线
	ZappinfoStatusOnline = ZappinfoStatus("online")
	// 临时下线
	ZappinfoStatusOffline = ZappinfoStatus("offline")
	// 已废弃
	ZappinfoStatusUseless = ZappinfoStatus("useless")
)

type ZappinfoServerType string

var (
	// DOCKER_VM
	ZappinfoServerTypeDockerVM = ZappinfoServerType("DOCKER_VM")
	// DOCKER
	ZappinfoServerTypeDocker = ZappinfoServerType("DOCKER")
)

type PodZappinfo struct {
	Spec   *PodZappinfoSpec   `json:"spec"`
	Status *PodZappinfoStatus `json:"status"`
}

type PodZappinfoSpec struct {
	AppName    string `json:"appName"`
	Zone       string `json:"zone"`
	ServerType string `json:"serverType"`
	Fqdn       string `json:"fqdn"`
}

type PodZappinfoStatus struct {
	Registered bool `json:"registered"`
}

type PodZappinfoMetaSpec struct {
	PodZappinfoSpec

	Status           ZappinfoStatus `json:"status"`
	Hostname         string         `json:"hostname"`
	Ip               string         `json:"ip"`
	Cluster          string         `json:"cluster"`
	HardwareTemplate string         `json:"hardwareTemplateName"`
	ParentSn         string         `json:"vmParent"`
	ParentIp         string         `json:"vmParentIp"`
	Platform         string         `json:"platform"`
	// TODO sigma 3.1 中没有直接对应的参数
	// https://lark.alipay.com/sys/sigma3.x/qr5egz#gva8rz
	CpuSetMode string `json:"cpuSetMode"`
}
