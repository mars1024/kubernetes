package xvip

import "k8s.io/api/core/v1"

// XVIPSpec is the specification for XVIP
/*
VipBuType 定义：
	internet("互联网VIP","公网"),
	abroad("海外加速VIP", "公网"),
	mobile_internet("无线互联网VIP", "公网"),
	oss_internet("oss互联网VIP","公网"),
	office("办公网VIP","公网"),
	cdn_loop("cdn回源vip","公网"),
	oss_static("oss静态带宽vip", "公网"),
	cdn_static("cdn静态带宽vip", "公网"),

	internal("内部vip","私网"),
	special("专线vip","私网"),
	vpc("vpc vip","私网"),
	cross_domain("跨域vip","私网"),
	aliyun_control("阿里云管控vip","私网"),
	oss_internal("oss内部vip", "私网");
*/
type XVIPSpec struct {
	AppGroup        string         `json:"appGroupName,omitempty"`
	ApplyUser       string         `json:"applyUser,omitempty"`
	AppId           string         `json:"appId,omitempty"`
	VipBuType       string         `json:"vipBuType,omitempty"`
	Ip              string         `json:"ip,omitempty"`
	Port            int32          `json:"port,omitempty"`
	Protocol        v1.Protocol    `json:"protocol,omitempty"`
	RealServerList  RealServerList `json:"rs"`
	HealthcheckType string         `json:"hcType,omitempty"`
	HealthcheckPath string         `json:"hcPath,omitempty"`
	ReqAvgSize      int64          `json:"reqAvgSize,omitempty"`
	QpsLimit        int64          `json:"qpsLimit,omitempty"`

	ChangeOrderId string `json:"changeOrderId,omitempty"`

	LbName string `json:"lbName,omitempty"`
}

type XVIPSpecList []*XVIPSpec
type RealServerList []*RealServer

// RealServer is a struct that store backend real servers.
type RealServer struct {
	Ip     string `json:"ip,omitempty"`
	Port   int32  `json:"port,omitempty"`
	Status Status `json:"status,omitempty"`

	Op Operation `json:"op,omitempty"`
}

type Status string
type Operation string

var (
	StatusEnable  = Status("enable")
	StatusDisable = Status("disable")
	// decide by the first real server
	StatusDynamic = Status("dynamic")
)

var (
	OpAdd    = Operation("ADD")
	OpUpdate = Operation("UPDATE")
	OpDelete = Operation("DELETE")
)

type VIPOrder struct {
	Port    string `json:"port,omitempty"`
	OrderId string `json:"orderId,omitempty"`
}

type Client interface {
	// 增加 VIP
	AddVIP(spec *XVIPSpec) (ip string, err error)

	// 删除 VIP
	DeleteVIP(spec *XVIPSpec) error

	// 新增 RS
	AddRealServer(*XVIPSpec, ...*RealServer) error

	// 删除 RS，每次
	DeleteRealServer(*XVIPSpec, ...*RealServer) error

	// 启用 RS
	EnableRealServer(*XVIPSpec, ...*RealServer) error

	// 停用 RS
	DisableRealServer(*XVIPSpec, ...*RealServer) error

	// 获取任务信息
	GetTaskInfo(requestId string) (*TaskInfo, error)

	// 获取 RS 的信息
	GetRsInfo(spec *XVIPSpec) (XVIPSpecList, error)
}
