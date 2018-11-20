package sigma

//http://docs.alibaba-inc.com:8090/pages/viewpage.action?pageId=422874776#d0-etcd%E6%95%B0%E6%8D%AE%E7%BB%93%E6%9E%84%E6%8E%A5%E5%8F%A3%E5%AE%9A%E4%B9%89-%E3%80%90d107%E3%80%91etcdapplicationsimages%EF%BC%88%E5%BA%94%E7%94%A8%E9%95%9C%E5%83%8F%2F%E5%9F%BA%E7%BA%BF%2Fpackagein...
//etcd:/applications/$site/$appname/$apptag/imageinfo
type ImageInfo struct {
	ImageName    string
	PackageInfos PackageInfo
}

type PackageInfo struct {
	PackageConfigs []PackageConfig
	DataConfigs    []DataConfig
	ProcessConfigs []ProcessConfig
}

type PackageConfig struct {
	PackageType       string //: "RPM", //可选为FILE，DIR，RPM，ARCHIVE...
	PackageVisibility string //: "PUBLIC" , //可选为PUBLIC,PRIVATE...
	PackageURI        string //":""
}

type DataConfig struct {
	Name            string //
	Src             string //路径
	Dst             string //路径
	Version         string //
	AttemptId       int    //整数
	Visibility      string //"PUBLIC" //可选为PUBLIC,PRIVATE...
	RetryCountLimit int    //": 0 // -1 means retrying never end
	NormalizeType   string //" : "NONE" // 可选为NONE， COLON_REPLACED
}

type ProcessConfig struct {
	IsDaemon          bool              //": "true", //可选为true, false
	Name              string            //": "" ,
	Cmd               string            //": "",
	Envs              map[string]string //": {"env1": "xxxx","env2": "yyyy"},
	Args              map[string]string //": {"arg1": "xxxx","arg2": "yyyy"}
	StopTimeout       int               //": 100,
	RestartInterval   int               //": 10,
	RestartCountLimit int               //": 10,
	ProcStopSig       int               //": 10  // 10 mean SIGUSR1
}
