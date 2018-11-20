package cmdb

type Client interface {
	// 注册cmdb信息
	AddContainerInfo(reqInfo []byte) error
	// 更新cmdb信息
	UpdateContainerInfo(reqInfo []byte) error
	// 删除cmdb信息
	DeleteContainerInfo(sn string) error
	// 获取cmdb信息
	GetContainerInfo(sn string) (*CMDBResp, error)
}
