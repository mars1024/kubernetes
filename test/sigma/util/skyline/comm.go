package skyline

import (
	"strings"
	"time"
)

const (
	SkylineCpuSetModeNameSpace = "cpu_set_mode"
	SkylineCpuSetModeCpuSet    = "cpu_set_mode.cpuset"
	SkylineCpuSetModeCpuShare  = "cpu_set_mode.cpushare"
)

const (
	EQ = "EQ"
	IN = "IN"
)

const (
	addTagUri      = "%s/h/server/tag_add"             // 打标
	delTagUri      = "%s/h/server/tag_del"             // 去标
	vmAddUri       = "%s/openapi/device/vm/add"        // 新增
	vmDeleteUri    = "%s/openapi/device/vm/delete"     // 删除
	queryUri       = "%s/item/query"                   // 查询
	updateUri      = "%s/openapi/device/server/update" // 更新
	timeOutDefault = time.Duration(60) * time.Second   // 限流器锁超时
)

const (
	updateNodeOperatorType = 1      // update时的Operator类型
	defaultOperatorType    = "USER" // 通用的Operator类型
)

const (
	NsUnknownStatus  = "unknown"
	NsCreatedStatus  = "allocated"
	NsPreparedStatus = "prepared"
	NsRunningStatus  = "running"
	NsStoppedStatus  = "stopped"
)

// 参考https://yuque.antfin-inc.com/at7ocb/qbn0oy/dwuaod#9c1zin
const (
	SelectCabinetNum      = "cabinet.cabinet_num"       // 机柜编号
	SelectCabinetPosition = "cabinet_position"          // 设备在机柜中位置号
	SelectAppName         = "app.name"                  // 应用名称
	SelectAppGroup        = "app_group.name"            // 应用分组
	SelectAppUseType      = "app_use_type"              // 应用用途
	SelectCpuCount        = "total_cpu_count"           // 总核数
	SelectDiskSize        = "total_disk_size"           // 磁盘大小
	SelectMemorySize      = "total_memory_size"         // 内存大小
	SelectAbb             = "room.abbreviation"         // 物理房间缩写
	SelectSmName          = "standard_model.sm_name"    // 标准机型名称
	SelectCreate          = "gmt_create"                // 创建时间
	SelectModify          = "gmt_modified"              // 上次修改时间
	SelectParentSn        = "parent_service_tag"        // 父sn,物理机为机框sn,虚拟机为服务器sn
	SelectSn              = "sn"                        // 容器或者宿主机sn
	SelectIp              = "ip"                        // 容器或者宿主机ip
	SelectHostName        = "host_name"                 // hostname
	SelectAppServerState  = "app_server_state"          // 应用状态
	SelectSecurityDomain  = "security_domain"           // 安全域
	SelectSite            = "cabinet.logic_region"      // 机房
	SelectModel            = "device_model.model_name" // 机房
)

// 参考https://yuque.antfin-inc.com/at7ocb/qbn0oy/qbdtlf#serverapiparamconstantserver
const (
	ApiParamSn               = "sn"               // sn
	ApiParamParentSn         = "parentSn"         // 父类sn
	ApiParamAppGroup         = "appGroup"         // 应用分组
	ApiParamHostName         = "hostName"         // 主机名
	ApiParamDeviceModel      = "deviceModel"      // 设备模式
	ApiParamDeviceType       = "device_type"      // 设备类型
	ApiParamIp               = "ip"               // ip
	ApiParamIpSecurityDomain = "ipSecurityDomain" // 安全域
	ApiParamAppServerState   = "appServerState"   // 服务状态
	ApiParamTotalCpuCount    = "totalCpuCount"    // 总核数
	ApiParamTotalMemorySize  = "totalMemorySize"  // 内存大小单位M
	ApiParamTotalDiskSize    = "totalDiskSize"    // 磁盘大小单位G
	ApiParamAppUseType       = "appUseType"       // 应用状态
	ApiParamResourceOwner    = "resourceOwner"    // 资源归属
	ApiParamContainerState   = "containerState"   // 容器状态
)

var SelectDefault = strings.Join([]string{SelectCabinetNum, SelectCabinetPosition, SelectAppName, SelectAppUseType, SelectCpuCount,
	SelectDiskSize, SelectMemorySize, SelectAbb, SelectSmName, SelectCreate, SelectModify,
	SelectParentSn}, ",")

type ArmoryAppState string

var (
	Armory_UNKNOWN         = ArmoryAppState("unknown")         // "未知"
	Armory_WAIT_ONLINE     = ArmoryAppState("wait_online")     // "应用等待在线"
	Armory_WORKING_ONLINE  = ArmoryAppState("working_online")  // "应用在线"
	Armory_WORKING_OFFLINE = ArmoryAppState("working_offline") // "应用离线"
	Armory_READY           = ArmoryAppState("ready")           // "准备中"),
	Armory_BUFFER          = ArmoryAppState("buffer")          // "闲置"),
	Armory_BROKEN          = ArmoryAppState("broken")          // "损坏，维修"),
	Armory_LOCK            = ArmoryAppState("lock")            // "锁定"),
	Armory_UNUSE           = ArmoryAppState("unuse")           // "停用");

	ArmoryRegisterStateMap = map[ArmoryAppState]int{
		Armory_READY:           1,
		Armory_WAIT_ONLINE:     1,
		Armory_WORKING_ONLINE:  1,
		Armory_WORKING_OFFLINE: 1,
		Armory_BUFFER:          1,
	}

	armoryStateMap = map[string]ArmoryAppState{
		NsCreatedStatus:  Armory_WORKING_OFFLINE,
		NsPreparedStatus: Armory_WAIT_ONLINE,
		NsStoppedStatus:  Armory_WORKING_OFFLINE,
		NsRunningStatus:  Armory_WORKING_ONLINE,
		NsUnknownStatus:  Armory_UNKNOWN,
	}

	gnsStateMap = map[ArmoryAppState]string{
		Armory_WORKING_OFFLINE: NsCreatedStatus,
		Armory_READY:           NsPreparedStatus,
		Armory_WORKING_ONLINE:  NsRunningStatus,
		Armory_UNKNOWN:         NsUnknownStatus,
	}
)

type auth struct {
	Account   string `json:"account"`
	AppName   string `json:"appName"`
	Signature string `json:"signature"`
	Timestamp int64  `json:"timestamp"`
}

// TagAdd or Delete
type TagParam struct {
	Auth        *auth        `json:"auth"`
	SkyOperator *skyOperator `json:"operator"`
	Sn          string       `json:"sn"`
	Tag         string       `json:"tag"`
	TagValue    string       `json:"tagValue"`
}

type skyOperator struct {
	Type     interface{} `json:"type"`
	Nick     string      `json:"nick"`
	WorkerId string      `json:"workerId"`
}

// vmAdd
type VmAddParam struct {
	Auth        *auth        `json:"auth"`
	SkyOperator *skyOperator `json:"operator"`
	SkyItem     *skyItem     `json:"item"`
}

type skyItem struct {
	DeviceType  string                 `json:"deviceType"`
	PropertyMap map[string]interface{} `json:"propertyMap"`
}

// vmDelete
type VmDeleteParam struct {
	Auth        *auth        `json:"auth"`
	SkyOperator *skyOperator `json:"operator"`
	SkyItem     string       `json:"item"`
}

// query
type QueryParam struct {
	Auth *auth      `json:"auth"`
	Item *QueryItem `json:"item"`
}

type QueryItem struct {
	From      string `json:"from"`      // 查询那个表;底层自动根据选择的字段进行join
	Select    string `json:"select"`    // 查询哪些columns;目标表可以直接取属性名,其他关联表必须以类目名作为前缀.分隔 "sn,ip,node_type,app_group.name"
	Condition string `json:"condition"` // 查询条件;暂时只支持AND;左表达式为字段名;表达式: =, !=, >, >=, <, <=, IN, LIKE;右表达式为值字符串和数字,字符串用''包起来,其他像boolean都当成字符类型
	Page      int    `json:"page"`      // 页码;第几页;从1开始
	Num       int    `json:"num"`       // 每页大小
	NeedTotal bool   `json:"needTotal"` // 是否需要总数;为true才承诺给准确总数;不需要总数默认false;有利于底层优化
}

// 所有api返回的通用结构
type Result struct {
	Success      bool         `json:"success"`
	Value        *ResultValue `json:"value"`
	ErrorCode    int          `json:"errorCode"`
	ErrorMessage string       `json:"errorMessage"`
}

type ResultValue struct {
	TotalCount int                      `json:"totalCount"`
	HasMore    bool                     `json:"hasMore"`
	ItemList   []map[string]interface{} `json:"itemList"`
}

type ResultItem struct {
	Ip         string // 容器或者物理机ip
	Sn         string // 容器或者物理机sn
	HostName   string // 主机名
	NodeGroup  string // 应用分组
	ParentSn   string // 父sn,物理机为机框sn,虚拟机为服务器sn
	State      string // 应用状态
	Site       string // 逻辑机房
	Model      string // 型号名
	AppName    string // 应用名称
	AppUseType string // 应用用途
}

// update
type UpdateParam struct {
	Auth        *auth        `json:"auth"`
	SkyOperator *skyOperator `json:"operator"`
	UpdateItem  *updateItem  `json:"item"`
}

type updateItem struct {
	Sn          string                 `json:"sn"`
	PropertyMap map[string]interface{} `json:"propertyMap"`
}
