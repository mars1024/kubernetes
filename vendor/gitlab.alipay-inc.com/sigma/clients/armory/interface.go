package armory

type Client interface {
	//查询armory device信息
	QueryDevice(query string) (*DeviceInfo, error)

	//查询armory networkcluster信息
	QueryNetWorkCluster(query string) (*NetWorkClusterInfo, error)
}
