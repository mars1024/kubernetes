package swarm

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"
)

type ContainerResult struct {
	containerHostName string
	ContainerIP       string
	ContainerName     string
	ContainerSN       string
	ContainerID       string
	ContainerHN       string
	DeployUnit        string
	Site              string
	CPUSet            string `json: "CpuSet"`
	HostIP            string `json: "HostIP"`
	HostSN            string `json:"HostSn"`
}

// Host is the container scheduled host
type Host struct {
	HostIP string
	HostSN string
}

// ContainerCreationResp wraps swarm creation response body.
// NOTE: For now, we does not consider multiple-containers creation situation
type ContainerCreationResp struct {
	// container ID
	ID         string `json:"Id"`
	Warnings   []string
	Containers map[string]ContainerResult `json:"Containers"`
}

// PreviewContainerCreationResp wraps swarm preview creation response body.
type PreviewContainerCreationResp struct {
	Count   int
	Samples []map[string]string
	Warning string
}

// ContainerQueryResp wraps swarm query container info.
type ContainerQueryResp struct {
	ID     string          `json:"Id"`
	Config ContainerConfig `json:"Config"`
}

type ContainerConfig struct {
	Image string   `json:"Image"`
	Env   []string `json:"Env"`
}

// Host returns which host the container is scheduled to
func (c *ContainerCreationResp) Host() *Host {
	return &Host{
		HostIP: c.Containers[c.ID].HostIP,
		HostSN: strings.ToLower(c.Containers[c.ID].HostSN),
	}
}

// IsScheduled returns whether the container is scheduled or not.
func (c *ContainerCreationResp) IsScheduled() bool {
	// 集团创建成功返回的 body 不包括 hostip 等字段
	// 通过 containerID 和 warning 判断是一种通用的做法
	return c.ID != "" && len(c.Warnings) == 0
}

// ParseResponseBody creates a Response result from http response body data
func ParseResponseBody(body io.ReadCloser) (*ContainerCreationResp, error) {
	defer body.Close()

	bodyBytes, _ := ioutil.ReadAll(body)
	bodyString := string(bodyBytes)
	fmt.Printf("response body %s", bodyString)

	resp := &ContainerCreationResp{}
	err := json.Unmarshal([]byte(bodyString), resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ParsePreviewResponseBody creates a Response result from http response body data
func ParsePreviewResponseBody(body io.ReadCloser) (*PreviewContainerCreationResp, error) {
	defer body.Close()

	bodyBytes, _ := ioutil.ReadAll(body)
	bodyString := string(bodyBytes)
	fmt.Printf("response body %s", bodyString)

	resp := &PreviewContainerCreationResp{}
	err := json.Unmarshal([]byte(bodyString), resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type AllocResult struct {
	ResultCode    int    /* 0表示成功，非0为错误码 */
	ErrorMsg      string /* 错误信息 */
	Site          string `json:"-"`          //Site
	HostSn        string `json:",omitempty"` /* 宿主机ip */
	HostIp        string `json:",omitempty"` /* 宿主机ip */
	SlotId        string `json:"-"`          //如：宿主机上唯一
	RequirementId string `json:",omitempty"` //当前AllocPlan的RequirementId

	ContainerSn         string `json:",omitempty"` /* armory中容器的sn */
	ContainerHn         string `json:",omitempty"` /* armory中容器的hostname */
	ContainerId         string `json:",omitempty"` /* docker的容器id */
	ContainerName       string `json:",omitempty"` /* docker容器的name */
	ContainerIp         string `json:",omitempty"` /* 容器ip */
	ContainerVlan       string `json:",omitempty"` /* 容器ip对应的VlanId*/
	ContainerGw         string `json:",omitempty"` /* 容器ip对应的Gateway */
	ContainerMask       string `json:",omitempty"` /* 容器ip对应的Mask*/
	ContainerMacAddress string `json:",omitempty"` /* 容器对应的Mac地址*/

	IpLabel string `json:",omitempty"` /* 容器ip对应的ip label*/

	ContainerPublicIp   string `json:",omitempty"` /* 容器公网ip */
	ContainerPublicVlan string `json:",omitempty"` /* 公网ip对应的VlanId*/
	ContainerPublicGw   string `json:",omitempty"` /* 公网ip对应的Gateway */
	ContainerPublicMask string `json:",omitempty"` /* 公网ip对应的Mask*/

	CpuSet   string                  `json:",omitempty"` /* 容器要绑定到的cpu核列表 */
	CpuQuota int                     `json:",omitempty"` /* 容器cpu核的总配额 */
	GnsState string                  `json:"-"`          /* 容器在GNS注册的状态 */
	Volume   map[string]*VolumeAlloc `json:",omitempty"`

	GpuSet      []string `json:",omitempty"` //容器分配到的gpu设备(含绝对路径)
	GpuCtrlDevs []string `json:",omitempty"` //容器分配到的gpu控制设备(含绝对路径)
	GpuVolumes  []string `json:",omitempty"` //容器挂载的gpu驱动数据卷

	FpgaSet map[string][]string `json:",omitempty"`
	QatSet  map[string][]string `json:",omitempty"`

	OverlayNetwork    string `json:",omitempty"` /* 是否加入Overlay "true" or "false" */
	OverlayNetworkVer string `json:",omitempty"` /* Overlay网络实现版本 */
	VPortId           string `json:",omitempty"` /* Overlay网络VPort ID */
	VPortToken        string `json:",omitempty"` /* Overlay网络VPort创建Token */
	VSwitchId         string `json:",omitempty"` /* Overlay网络或ECS创建时指定的VSwitch ID */
	EcsInstanceId     string `json:",omitempty"` /* 容器所在的ECS宿主机的instanceId */
	EnId              string `json:",omitempty"`

	Invisible bool `json:",omitempty"` /* 用以标记action中的记录是否对上层可见 */
}

type VolumeAlloc struct {
	Address    string
	VolumeName string /* 已创建的 volume 名,若只分配了名字未创建,名字为空 */
}

type Request struct {
	UUid       string
	Type       string
	Object     string
	State      string
	Body       string
	Msg        string
	TraceId    string
	CreateTime time.Time
	Modified   time.Time
	Actions    []*Action
}

type Action struct {
	UUid                  string
	Object                string
	RequestId             string
	State                 string
	Stage                 string
	Result                string
	CreateTime            time.Time
	Modified              time.Time
	ElapsedTimesInSeconds map[string]float64
}

type GlobalRules struct {
	UpdateTime       string
	Monopolize       MonopolizeDecs
	CpuSetMutex      CpuSetMutexDecs      //cpu物理核互斥规则
	CpuSetMonopolize CpuSetMonopolizeDecs //cpu物理核独占规则
}

type MonopolizeDecs struct {
	AppConstraints []string
	DUConstraints  []string
}

type CpuSetMutexDecs struct {
	AppConstraints []string
	DUConstraints  []string
}

type CpuSetMonopolizeDecs struct {
	AppConstraints []string
	DUConstraints  []string
}
