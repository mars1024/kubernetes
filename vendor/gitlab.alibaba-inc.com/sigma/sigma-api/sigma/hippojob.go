package sigma

type HippojobStartRequest struct {
	ApplicationId  string         `json:"applicationId"` //":"Biz1",
	SlotId         int            `json:"slotId"`        //:1,
	ProcessContext ProcessContext `json:"processContext"`
}

type ProcessContext struct {
	RequiredPackages []RequiredPackage `json:"requiredPackages,omitempty"`
	Processes        []Processe        `json:"processes,omitempty"`
	RequiredDatas    []RequiredData    `json:"requiredDatas,omitempty"`
}

type RequiredPackage struct {
	PackageURI string `json:"packageURI"`
	Type       string `json:"type"` //RPM,ARCHIVE,IMAGE
}

type Processe struct {
	IsDaemon    bool   `json:"isDaemon"`
	ProcessName string `json:"processName"`
	Cmd         string `json:"cmd"`
	Args        []KV   `json:"args,omitempty"`
	Envs        []KV   `json:"envs,omitempty"`

	InstanceId        int64 `json:"instanceId"`        //int64
	restartInterval   int   `json:"restartInterval"`   //10
	restartCountLimit int   `json:"restartCountLimit"` //10
	procStopSig       int   `json:"slotId"`            //10:SIGUSR1 15:SIGTERM
}

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RequiredData struct {
	Name            string `json:"name"`            //
	Src             string `json:"src"`             //路径
	Dst             string `json:"dst"`             //路径
	Version         string `json:"version"`         //
	NormalizeType   string `json:"normalizeType"`   //" : "NONE" // 可选为NONE， COLON_REPLACED
	AttemptId       int64  `json:"attemptId"`       //整数
	expireTime      int64  `json:"expireTime"`      //int64
	RetryCountLimit int    `json:"retryCountLimit"` //0, -1 means retrying never end
	//Visibility      string `json:"isDaemon"`        //"PUBLIC" //可选为PUBLIC,PRIVATE...

}
