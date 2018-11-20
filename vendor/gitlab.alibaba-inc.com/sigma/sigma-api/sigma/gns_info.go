package sigma

//http://docs.alibaba-inc.com/pages/viewpage.action?pageId=417009425#i9-%E5%A4%96%E9%83%A8%E6%8E%A5%E5%8F%A3%E5%AE%9A%E4%B9%89-i911%E8%BE%93%E5%85%A5%E5%8F%82%E6%95%B0%28%E8%AF%B7%E6%B1%82%29
/*
POST /gns/create HTTP/1.1
Content-Type: application/json
{
    "AppName":"simple_app",                 //容器应用名
    "AppDeployUnit":""               //相当于电商一个应用下的armory分组，预发/小流量等，同分组下的实例用同一份代码+配置，例如:"buy_center_online"
    "RouteLabels": {                         //路由标签
        "IpLabel":"et2_Unit_CENTER", //对容器ip段的要求，比如单元化中的单元名称
        "Site":"eu13",               //机房信息
        "Stage": "pre",              //XIAOTAOBAO-小淘宝、PRE-预发、COLDBACK-冷备、ONLINE-正式
        "Unit":"CENTER"              //CENTER-中心｜UNYUN-深圳云单元｜UNIT-杭州单元｜UNSZ-深圳单元
    }
    "AppLabels":{                     // 应用本身提供的标签，用于亲近性相关。
        "label1":"value1",            // 强烈建议被其他app所依赖的应用，将自己的app声明掉
        "label2":"value2"            // volume
    }
    "AppPorts":[12200,8080],         //App暴露的服务端口，启动后会写入nameserver
    "InstanceGroup": "container-mw-eu13",   //容器应用分组
    "HostIp": "nc1"                      //宿主机ip
    //"HostName": "nc1"                    //宿主机Hostname
    //"ImageId": "abce09192",  //容器运行镜像id
    "ImageName": "imagename1",         //容器启动时指定的镜像名
    "StartTime": "1469158914",         //容器启动时间
    "ContainerIp": "t4_abcd..."
    "ContainerSn": "t4_abcd..."
    "ContainerName": "cname1",         //容器名
    "ContainerHostName":"buy010153023089.et2",
    "ContainerStatus":"allocated", //容器运行状态
}
*/

type GnsInfo struct {
	AppName       string
	AppDeployUnit string            //相当于电商一个应用下的armory分组，预发/小流量等，同分组下的实例用同一份代码+配置，例如:"buy_center_online"
	RouteLabels   RouteLabels       //路由标签
	AppLabels     map[string]string //{"label1":"value1"} 应用本身提供的自定义标签，用于亲近性相关。
	AppPorts      []int             //":[12200,8080] //App暴露的服务端口，启动后会写入nameserver
	InstanceGroup string            //容器/Job/ECS实例的armory2分组：生产好的容器/JOB实例会自动放到这个分组中, 例:"container-mw-eu13"
	HostIp        string            //宿主机ip
	HostSn        string            //宿主机sn
	ImageName     string            //镜像uri, 例:"buy-20160918"
	//StartTime         int //容器启动时间
	ContainerIp       string            //":"10.185.162.130"},
	ContainerSn       string            //":"t4_10.185.162.130"},
	ContainerName     string            //":"buy010153023089"} //容器名
	ContainerHostName string            //":"buy010153023089.et2"},
	ContainerStatus   string            //":"allocated"},
	ContainerNetwork  string            //bridge, host
	Tag2Host          bool              //true为是，false为否，默认false，为true时不创建Armory记录
	Properties        map[string]string //需要记录在arc中的信息
	Model             string
	Site              string
}
