package zappinfo

import (
	"fmt"
	"encoding/json"

	"k8s.io/api/core/v1"
	"github.com/golang/glog"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipayapis "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
)

type ClientConfig struct {
	Token string
}

type Client interface {
	// 写入服务器信息
	AddServer(server *alipayapis.PodZappinfoMetaSpec) error
	// 按主机名删除服务器
	DeleteServerByHostname(hostname string) error
	// 更新服务器状态
	UpdateServerStatus(hostname string, status alipayapis.ZappinfoStatus) error
	// 批量更新服务器状态
	UpdateMultiServerStatus(hostnames []string, status alipayapis.ZappinfoStatus) error
	// 按主机名查询服务器信息
	GetServerByHostname(hostname string) (*alipayapis.PodZappinfoMetaSpec, error)
	// 按IP查询服务器信息
	GetServerByIp(ip string) (*alipayapis.PodZappinfoMetaSpec, error)
}

func GetPodZappinfoMetaSpec(pod *v1.Pod, node *v1.Node, podZappinfo *alipayapis.PodZappinfo, networkStatus *sigmak8sapi.NetworkStatus) (*alipayapis.PodZappinfoMetaSpec, error) {
	meta := &alipayapis.PodZappinfoMetaSpec{
		PodZappinfoSpec: *podZappinfo.Spec,
	}
	if az, ok := pod.Annotations[alipayapis.AnnotationZappinfo]; ok {
		var info alipayapis.PodZappinfo
		glog.V(5).Infof("zappinfo: %s", string(az))
		if err := json.Unmarshal([]byte(az), &info); err != nil {
			return nil, err
		}
		meta.PodZappinfoSpec = *info.Spec
	}

	meta.Hostname = pod.Spec.Hostname
	meta.Ip = networkStatus.Ip

	meta.ParentSn = pod.Spec.NodeName
	if meta.ParentIp = detectNodeIp(node); meta.ParentIp == "" {
		return nil, fmt.Errorf("node %s ip not found from status or label %s", node.Name, sigmak8sapi.LabelNodeIP)
	}

	// FIXME
	meta.Status = detectStatus(pod)
	meta.CpuSetMode = "cpuShare"
	meta.Platform = "cloudprovision"

	var (
		cpu  int64
		mem  int64
		disk int64
	)

	for _, v := range pod.Spec.Containers {
		cpu += v.Resources.Requests.Cpu().Value()
		mem += v.Resources.Requests.Memory().Value()
		disk += v.Resources.Requests.StorageEphemeral().Value()
	}
	if cpu == 0 || mem == 0 || disk == 0 {
		return nil, fmt.Errorf("cpu, memory, or storage ephemeral is zero")
	}

	// TODO 是否需要取整
	meta.HardwareTemplate = fmt.Sprintf("VM_%dC%dG%dG", cpu, mem/(1024*1024*1024), disk/(1024*1024*1024))

	// check all value
	// we should do it in admission controller
	return meta, nil
}

func detectStatus(pod *v1.Pod) alipayapis.ZappinfoStatus {
	// FIXME
	return alipayapis.ZappinfoStatusUninit
}

func detectNodeIp(node *v1.Node) (ip string) {
	var exist bool
	if ip, exist = node.Labels[sigmak8sapi.LabelNodeIP]; exist && ip != "" {
		return ip
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type != v1.NodeInternalIP || addr.Address == "" {
			continue
		}
		ip = addr.Address
	}
	return ip
}