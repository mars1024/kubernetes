package alipaymeta

import (
	"k8s.io/api/core/v1"

	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"gitlab.alipay-inc.com/sigma/apis/pkg/apis"
)

const (
	LabelPodFQDN = apis.FQDN
)

func GetPodSN(pod *v1.Pod) string {
	if pod.Labels == nil {
		return ""
	}
	return pod.Labels[api.LabelPodSn]
}

func GetPodFQDN(pod *v1.Pod) string {
	if pod.Labels != nil {
		return ""
	}
	return pod.Labels[apis.FQDN]
}
