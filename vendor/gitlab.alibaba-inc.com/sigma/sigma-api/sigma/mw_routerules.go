package sigma

// 解决长久以来的中间件IP段提前注册带来的一系列问题
// 中间件去标逻辑：灰度去标的过程中，先通过临时的白名单方案，选择到正确的宿主机，选择到有效的容器IP
// 未来很长一段时间后，待中间件全部搞定后，未来再简化这段逻辑

// 设计细节见文档： http://docs.alibaba-inc.com/pages/viewpage.action?pageId=612697024#etcdinternal%E7%BA%A7%E5%88%AB%E6%8E%A5%E5%8F%A3%E5%AE%9A%E4%B9%89-%E3%80%90d403%E3%80%91etcdinternalrouterules%28%E4%B8%AD%E9%97%B4%E4%BB%B6%E8%B7%AF%E7%94%B1%E8%A7%84%E5%88%99%29

type RouteRules struct {
	UpdateTime string            `json:",omitempty"`
	Rules      []RouteRuleDetail `json:",omitempty"`
}

type RouteRuleDetail struct {
	AppUnit           string/**容器归属单元*/ `json:",omitempty"`
	AppEnv            string/**容器归属用途*/ `json:",omitempty"`
	PhyServerIdentity string/**宿主机筛选逻辑*/ `json:",omitempty"`
	PhyServerEnv      string/**宿主机筛选逻辑*/ `json:",omitempty"`
	IpLabel           string/**IP筛选逻辑*/ `json:",omitempty"`
}
