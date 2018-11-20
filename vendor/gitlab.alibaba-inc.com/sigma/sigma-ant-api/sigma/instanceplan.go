package sigma

//http://docs.alibaba-inc.com/pages/viewpage.action?pageId=501419951
//"Status":"allocated",       //取值范围：accepted|allocated|created|reclaimed|ask4reclaim|destroyed;
type InstanceStatus string

var (
	InstanceStatus_accepted        = InstanceStatus("accepted")        //请求已接受
	InstanceStatus_allocated       = InstanceStatus("allocated")       //资源已分配
	InstanceStatus_allocate_failed = InstanceStatus("allocate_failed") //资源分配失败
	InstanceStatus_ready           = InstanceStatus("ready")           //创建实例的所有数据都已准备好, 比如IP等已分配
	InstanceStatus_creating        = InstanceStatus("creating")        //实例创建中
	InstanceStatus_created         = InstanceStatus("created")         //实例已创建
	InstanceStatus_starting        = InstanceStatus("starting")        //实例启动中
	InstanceStatus_started         = InstanceStatus("started")         //实例已启动
	InstanceStatus_reclaimed       = InstanceStatus("reclaimed")       //资源已回收； 调度器要主动杀离线任务时，将其alloc值为reclaimed状态，等待执行器停止实例
	InstanceStatus_ask4reclaim     = InstanceStatus("ask4reclaim")     //资源已回收
	InstanceStatus_destroyed       = InstanceStatus("destroyed")       //实例已销毁
)

///instances/requirements/{site}/{DeployUnit}/{InstanceSn}

type InstancePlan struct {
	UpdateTime    string       //例："2016-06-29 19:59:22",
	RequirementId string       //例："123456" ，哪个请求最后确定的这个slot，方便问题排查
	Requirement   *Requirement `json:"-"` //方便存放Requirement对象, 不出现在json中
	Site          string       //冗余
	BizName       string       //二层业务域名称；zeue，Carbon，Captain
	AppName       string       //相当于Aone的app，一个app下可以由多个分组
	DeployUnit    string       //相当于电商一个应用下的armory分组，预发/小流量等，同分组下的实例用同一份代码+配置，例如:"buy_center_online"

	InstanceSn   string         //"15dfb7b48b3c517e184392ffedd30a464ccae8217cb8c120d1ee20574e1a2930"  //与armory中的sn一致，由apiserver生成
	AllocSpecKey string         //"4c8g60g_1",  //资源规格定义的key；和前缀拼起来得到规格明细的etcd路径：/applications/allocspecs/$site/4c8g60g_1
	AllocSpec    AllocSpec      //方便存放AllocSpec对象,
	Status       InstanceStatus //状态。

	InstanceType     InstanceType //"CONTAINER", //实例类型： TASK|CONTAINER|KVM|ECS
	InstanceName     string       //"buy010153023089" //对于容器来说，就是容器名ContainerName；kvm、ecs和job类型可以不填
	InstanceIp       string       //"10.185.162.130",  //容器和kvm，ecs类型必填，job类型可以不填
	InstanceHostName string       //"buy010153023089.et2",  //容器和kvm，ecs类型必填，job类型可以不填

	HostIp            string   //宿主机IP，例如: 11.136.23.69
	HostSn            string   //宿主机Sn，例如: 214247788-02A
	HostPath          string   //Host在etcd上的地址路径：如："/localinfos/$site/214247788-02A"
	SlotId            int      //如：宿主机上唯一
	CpuSet            []int    //例：[4,5,6,7],  //CPU具体核，允许空字符串
	CpuShare          int      //400 //  总共需要多少的cpu计算能力，如果CpuNum不为0， 则必须是100*CpuNum
	GpuSet            []string //例:[/dev/nvidia0, /dev/nvidia1]
	PlanPath          string   `json:"-"`
	LastRequirementId string   `json:"-"` //不出现在json中
}
