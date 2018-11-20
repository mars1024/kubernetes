package sigma

/*
目前的元信息中只有一个QuotaGroup
/applications/$site/$bizname
{
    "QuotaGroup":"zeus_trade"  //由管控工具修改。全局唯一。为便于一目了然了解其含义，格式要求： bizname_子业务标示_....（支持多级子group）
    "BizName":"$bizname" //与路径中的bizname一致
}
/applications/$site/$bizname/$appname/
{
    "QuotaGroup":"zeus_trade"  //由管控工具修改。全局唯一。为便于一目了然了解其含义，格式要求： bizname_子业务标示_....（支持多级子group）
    "BizName":"$bizname" //与路径中的bizname一致
    "AppName":"$appname" //与路径中的appname一致
}
*/

type AppMetaInfo struct {
	UpdateTime string
	QuotaGroup string //"zeus_trade"  //由管控工具修改。全局唯一。为便于一目了然了解其含义，格式要求： bizname_子业务标示_....（支持多级子group）
	BizName    string //"$bizname" //与路径中的bizname一致
	AppName    string //"$appname" //与路径中的appname一致
	DswCluster string //如果请求constraint中不指定DswCluster,则用app元信息中的DswCluster
}

type AppOperationFailure struct {
	UpdateTime string //更新时间
	Action     string //失败的动作， 目前仅考虑start(启动容器）
	Message    string //失败的错误原因

	DeployUnit string //相当于电商一个应用下的armory分组，预发/小流量等，同分组下的实例用同一份代码+配置，例如:"buy_center_online"

	HostSn string //操作失败的机器sn
	HostIp string //操作失败的机器 ip

	InstanceIp   string //容器和kvm，ecs类型必填，job类型可以不填
	InstanceSn   string //与armory中的sn一致，由apiserver生成
	InstanceName string //对于容器来说，就是容器名ContainerName；kvm、ecs和job类型可以不填
}
