package annotation

import (
	"encoding/json"
	"strconv"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/util/format"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

// GetTimeoutSecondsFromPodAnnotation can get timeout value from pod annotation.
// The return value unit is 'second'
func GetTimeoutSecondsFromPodAnnotation(pod *v1.Pod, containerName string, timeoutItem string) int {
	extraConfigStr, exists := pod.Annotations[sigmak8sapi.AnnotationContainerExtraConfig]
	if !exists {
		return 0
	}
	extraConfig := sigmak8sapi.ContainerExtraConfig{}
	if err := json.Unmarshal([]byte(extraConfigStr), &extraConfig); err != nil {
		glog.Errorf("Failed to get custom config from pod %s because of invalid data", format.Pod(pod))
		return 0
	}

	itemTimeoutSeconds := 0
	for containerInfo, containerConfig := range extraConfig.ContainerConfigs {
		if containerInfo.Name != containerName {
			continue
		}
		timeoutSecondsStr, exists := containerConfig[timeoutItem]
		if !exists {
			glog.V(4).Infof("Can't get timeout value for %s, use 0 by default", timeoutItem)
			return 0
		}
		timeoutSeconds, err := strconv.Atoi(timeoutSecondsStr)
		if err != nil {
			glog.Errorf("Failed to get %s from pod %s because of invalid data", timeoutItem, format.Pod(pod))
			return 0
		}
		itemTimeoutSeconds = timeoutSeconds
	}
	return itemTimeoutSeconds
}

// GetCpuPeriodFromAnnotation extracts custom cpu period from pod annotations
func GetCpuPeriodFromAnnotation(pod *v1.Pod, containerName string) int64 {
	podAllocSpecJSON, ok := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
	if !ok {
		return 0
	}
	podAllocSpec := &sigmak8sapi.AllocSpec{}
	// unmarshal pod allocation spec
	if err := json.Unmarshal([]byte(podAllocSpecJSON), podAllocSpec); err != nil {
		glog.Errorf("could not get cpu period, unmarshal alloc spec err: %v", err)
		return 0
	}
	var newCpuPeriod int64
	for _, containerInfo := range podAllocSpec.Containers {
		if containerInfo.Name == containerName {
			newCpuPeriod = containerInfo.HostConfig.CpuPeriod
			break
		}
	}
	return newCpuPeriod
}
