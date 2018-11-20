package sigma

import (
	"fmt"
	"time"
)

//http://docs.alibaba-inc.com/pages/viewpage.action?pageId=501419951
//"Status":"allocated",       //取值范围：accepted|allocated|created|reclaimed|ask4reclaim|destroyed;
type InstanceStatus string

var (
	InstanceStatus_accepted    = InstanceStatus("accepted")    //请求已接受
	InstanceStatus_allocated   = InstanceStatus("allocated")   //资源已分配
	InstanceStatus_ready       = InstanceStatus("ready")       //创建实例的所有数据都已准备好, 比如IP等已分配
	InstanceStatus_creating    = InstanceStatus("creating")    //实例创建中
	InstanceStatus_created     = InstanceStatus("created")     //实例已创建
	InstanceStatus_starting    = InstanceStatus("starting")    //实例启动中
	InstanceStatus_started     = InstanceStatus("started")     //实例已启动
	InstanceStatus_reclaimed   = InstanceStatus("reclaimed")   //资源已回收； 调度器要主动杀离线任务时，将其alloc值为reclaimed状态，等待执行器停止实例
	InstanceStatus_ask4reclaim = InstanceStatus("ask4reclaim") //资源已回收
	InstanceStatus_destroyed   = InstanceStatus("destroyed")   //实例已销毁
)

type InstancePlan struct {
	AllocPlan
	InstanceId         string
	InstanceIp         string
	InstanceVlan       string `json:",omitempty"` /* 容器ip对应的VlanId*/
	InstanceGw         string `json:",omitempty"` /* 容器ip对应的Gateway */
	InstanceMask       string `json:",omitempty"` /* 容器ip对应的Mask*/
	InstanceMacAddress string `json:",omitempty"` /* 容器ip对应的Mac 地址*/

	InstancePublicIp   string
	InstancePublicVlan string `json:",omitempty"` /* 容器ip对应的VlanId*/
	InstancePublicGw   string `json:",omitempty"` /* 容器ip对应的Gateway */
	InstancePublicMask string `json:",omitempty"` /* 容器ip对应的Mask*/

	InstanceName     string
	InstanceHostName string

	SlotId string         //如：宿主机上唯一
	Status InstanceStatus //"allocated|starting|started|reclaimed|stopping|stopped", //状态。

	ArmoryModel string `json:",omitempty"`
	IPAMType    string `json:",omitempty"`

	OverlayNetwork    string `json:",omitempty"`
	OverlayNetworkVer string `json:",omitempty"`
	VPortId           string `json:",omitempty"` /* Overlay网络VPort ID */
	VPortToken        string `json:",omitempty"` /* Overlay网络VPort创建Token */
	VSwitchId         string `json:",omitempty"` /* Overlay网络或ECS创建时指定的VSwitch ID */
	EcsInstanceId     string `json:",omitempty"`
	EnId              string `json:",omitempty"` /* 弹性网卡的ID，加速弹性网卡的操作，避免查询 */

	PlanReqId      string `json:"-"`
	RequiremewntId string `json:"-"`
}

type InstancePlanSch struct {
	UpdateTime    string //例："2016-06-29 19:59:22",
	RequirementId string
	PlanReqId     string
	Requirement   *Requirement //方便存放Requirement对象, 不出现在json中
	Site          string       //冗余
	BizName       string       //二层业务域名称；zeue，Carbon，Captain
	AppName       string       //相当于Aone的app，一个app下可以由多个分组
	DeployUnit    string       //相当于电商一个应用下的armory分组，预发/小流量等，同分组下的实例用同一份代码+配置，例如:"buy_center_online"

	InstanceSn   string         //"15dfb7b48b3c517e184392ffedd30a464ccae8217cb8c120d1ee20574e1a2930"  //与armory中的sn一致，由apiserver生成
	AllocSpecKey string         //"4c8g60g_1",  //资源规格定义的key；和前缀拼起来得到规格明细的etcd路径：/applications/allocspecs/$site/4c8g60g_1
	AllocSpec    AllocSpec      //方便存放AllocSpec对象,
	Status       InstanceStatus //状态。

	InstanceType     InstanceType //"CONTAINER", //实例类型： TASK|CONTAINER|KVM|ECS
	InstanceIp       string       //"10.185.162.130",  //容器和kvm，ecs类型必填，job类型可以不填
	InstanceHostName string       //"buy010153023089.et2",  //容器和kvm，ecs类型必填，job类型可以不填

	HostIp     string   //宿主机IP，例如: 11.136.23.69
	HostSn     string   //宿主机Sn，例如: 214247788-02A
	SlotId     int      //如：宿主机上唯一
	GpuSet     []string //例:[/dev/nvidia0, /dev/nvidia1]
	PlanPath   string
	MemVersion int64
}

var loc, _ = time.LoadLocation("Asia/Shanghai")

func (ins *InstancePlan) IsValid() bool {
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", ins.UpdateTime, loc)
	return err == nil && time.Now().Sub(updateTime).Minutes() < 30.0
}

func (ins *InstancePlan) GetAppKey() string {
	return fmt.Sprintf("%v/%v", ins.Site, ins.DeployUnit)
}

func (ins *InstancePlanSch) GetAppKey() string {
	return fmt.Sprintf("%v/%v", ins.Site, ins.DeployUnit)
}

func (ins *InstancePlanSch) IsValid() bool {
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", ins.UpdateTime, loc)
	return err == nil && time.Now().Sub(updateTime).Minutes() < 30.0
}
