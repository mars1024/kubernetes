package sigma

import (
	"k8s.io/api/core/v1"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

// GetSNFromLabel can get SN from pod.
func GetSNFromLabel(pod *v1.Pod) (string, bool) {
	if pod.Labels == nil {
		return "", false
	}
	snKey := sigmak8sapi.LabelPodSn
	value, exists := pod.Labels[snKey]
	return value, exists
}
