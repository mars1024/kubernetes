package util

import (
	"encoding/json"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/kubernetes/pkg/apis/core"
)

func PodAllocSpec(pod *core.Pod) (*sigmak8sapi.AllocSpec, error) {
	if v, exists := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]; exists {
		var allocSpec *sigmak8sapi.AllocSpec
		if err := json.Unmarshal([]byte(v), &allocSpec); err != nil {
			return nil, err
		}
		return allocSpec, nil
	}
	return nil, nil
}
