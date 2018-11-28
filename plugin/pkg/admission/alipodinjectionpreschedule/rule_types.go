package alipodinjectionpreschedule

type appUnitStageConstraint struct {
	StageToDefault       []string /**环境转换为默认，比如线上把BACKUP/YACE/BETA_PUBLISH等环境转换为PUBLISH*/
	UnitToCenterForDaily []string /**在DAILY环境下，这些单元要转为中心单元*/
}

type dynamicStrategyType map[string]map[string]map[string]string

type dynamicStrategy struct {
	RefreshTime string                       `json:"RefreshTime"`
	UpdateTime  string                       `json:"UpdateTime"`
	Constraints dynamicStrategyType          `json:"Constraints"`
	ExtRules    map[string]map[string]string `json:"ExtRules"`
}

type appMetaInfo struct {
	UpdateTime string            `json:"UpdateTime"`
	AppName    string            `json:"AppName"`
	ExtInfo    map[string]string `json:"ExtInfo"`
}

type cpuSetModeAdvConfig struct {
	AppName        string               `json:",omitempty"` //该应用默认的CPUSetMode
	AppRule        string               `json:",omitempty"` //该应用下某个DU默认的CpuSetMode
	NodeGroupRules []advCpuSetModelRule `json:",omitempty"` //更复杂的规则
}

type advCpuSetModelRule struct {
	Cell       string/**归属机房，允许空字符串*/ `json:",omitempty"`
	NodeGroup  string/**归属分组，允许空字符串*/ `json:",omitempty"`
	AppUnit    string/**归属单元，允许空字符串*/ `json:",omitempty"`
	CpuSetMode string/**所属的CPUSet，不允许空字符串,share|cpuset|default */ `json:",omitempty"`
	/**为了兼容两块需求，并且快速上线，将原来面向应用的结构，增加ResourceLevel属性。
	面向分组的S10、S20、S30 服务计费和管理，一步到位聚焦cpuSet、cpuShare。
	缺陷：固定死了约定关系。S10 只能唯一cpuSet。S20 唯一cpuShare、S30 唯一cpuShare。
	备注：resource level 代表资源等级，没有想与具体的资源策略绑定。目前这个level实际是与CPUPolicy 有了关联的。可以直接理解是CPUPolicy。假如未来新增了MemPolicy，此时，对S10、S20、S30面向用户的，期望是保持不变，映射关系和灵活性交给scheduler。另外一种：直接显示MemPolicy，去掉S10-S20-S30这些，直接面向具体维度资源策略。目前K8S其实是后者，但是K8S里面的CPUPolicy 比Sigma这里cpuSet or cpuShare 更丰富，例如有Numa的信息。
	*/
	Modifier   string `json:",omitempty"`
	ModifyTime string `json:",omitempty"`
}

type appNamingMockRules struct {
	Rules []appNamingMockRuleDetail `json:",omitempty"`
}

type appNamingMockRuleDetail struct {
	AppUnit           string/**容器归属单元*/ `json:",omitempty"`
	AppEnv            string/**容器归属用途*/ `json:",omitempty"`
	PhyServerIdentity string/**宿主机筛选逻辑*/ `json:",omitempty"`
	PhyServerEnv      string/**宿主机筛选逻辑*/ `json:",omitempty"`
	NamingUnit        string /**注册armory用的标*/
	NamingEnv         string /**注册armory用的标*/
}
