package sigma

//AUTHOR： 智清
//需求背景：http://docs.alibaba-inc.com/pages/viewpage.action?pageId=656933898
//高级策略

type AdvancedStrategy struct {
	AdvancedParserConfig AdvancedParserConfig `json:"HostConfig"`
	ExtConfig            map[string]string    `json:"ExtConfig"`
	CandidatePlans       []*CandidatePlan     `json:"CandidatePlans"`
}

//对应高级策略中的需apiserver理解和解析的config
type AdvancedParserConfig struct {
	Privileged          bool                `json:"Privileged"`
	Runtime             string              `json:"Runtime"`
	MemoryWmarkRatio    float64             `json:"MemoryWmarkRatio"`
	UserDevices         []string            `json:"UserDevices"`
	Binds               []string            `json:"Binds"`
	NetworkMode         string              `json:"NetworkMode"`
	GpuSpec             GpuSpecStrategy     `json:"GpuSpec"`
	CpuSetModeConfig    map[string]string   `json:"CpuSetModeConfig"`
	CpuSetModeAdvConfig CpuSetModeAdvConfig `json:"CpuSetModeAdvConfig"` //优先使用
	NetPriority         map[string]string   `json:"NetPriority"`         //见： http://docs.alibaba-inc.com/pages/viewpage.action?pageId=671351156
}

type GpuSpecStrategy struct {
	Count       int     `json:"Count"`
	Memory      float64 `json:"Memory"`
	GpuMode     string  `json:"GpuMode"`
	ProductName string  `json:"ProductName"`
}

type CpuSetModeAdvConfig struct {
	DefaultAppCpuSetMode       string               `json:",omitempty"` //该应用默认的CPUSetMode
	DefaultNodeGroupCpuSetMode map[string]string    `json:",omitempty"` //该应用下某个DU默认的CpuSetMode
	Rules                      []AdvCpuSetModelRule `json:",omitempty"` //更复杂的规则
}

type AdvCpuSetModelRule struct {
	AppUnit    string/**归属单元，允许空字符串*/ `json:",omitempty"`
	AppEnv     string/**归属用途，允许空字符串*/ `json:",omitempty"`
	Cell       string/**归属机房，允许空字符串*/ `json:",omitempty"`
	NodeGroup  string/**归属分组，允许空字符串*/ `json:",omitempty"`
	CpuSetMode string/**所属的CPUSet，不允许空字符串*/ `json:",omitempty"`
}
