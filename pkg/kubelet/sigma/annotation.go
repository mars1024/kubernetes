package sigma

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/util/format"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

// GetStatusFromAnnotation can get container's status from certain annotation.
func GetStatusFromAnnotation(pod *v1.Pod, containerName string) *sigmak8sapi.ContainerStatus {
	key := sigmak8sapi.AnnotationPodUpdateStatus
	statusStr, exists := pod.Annotations[key]
	if !exists {
		return nil
	}
	containerStatuses := sigmak8sapi.ContainerStateStatus{}
	err := json.Unmarshal([]byte(statusStr), &containerStatuses)
	if err != nil {
		return nil
	}
	containerStatus, exists := containerStatuses.Statuses[sigmak8sapi.ContainerInfo{containerName}]
	if !exists {
		return nil
	}
	return &containerStatus
}

// GetSpecHashFromAnnotation can get user defined spec hash from certain annotation.
func GetSpecHashFromAnnotation(pod *v1.Pod) (string, bool) {
	if pod.Annotations == nil {
		return "", false
	}
	specHashKey := sigmak8sapi.AnnotationPodSpecHash
	hashStr, specHashExists := pod.Annotations[specHashKey]
	return hashStr, specHashExists
}

// GetContainerDesiredStateFromAnnotation parse whether the pod have valid annotation which represent containers
// desired state, and get the value of containers desired state,
// the value of containers status in annotation which update by previous process.
func GetContainerDesiredStateFromAnnotation(pod *v1.Pod) (haveContainerStateAnnotation bool,
	containerDesiredState sigmak8sapi.ContainerStateSpec, stateStatus sigmak8sapi.ContainerStateStatus) {

	containerDesiredState =
		sigmak8sapi.ContainerStateSpec{States: make(map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerState)}
	stateStatus =
		sigmak8sapi.ContainerStateStatus{Statuses: make(map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerStatus)}
	if pod == nil || len(pod.Annotations) == 0 {
		glog.V(4).Infof("pod %v is nil, or pod annotation's length is zero", pod)
		return false, containerDesiredState, stateStatus
	}

	haveContainerStateAnnotation = false

	// get container desired state through annotation
	containerStateSpec, ok := pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec]
	if ok {
		glog.V(4).Infof("pod %s has annotation: %s, value is %s",
			format.Pod(pod), sigmak8sapi.AnnotationContainerStateSpec, containerStateSpec)
		// unmarshal user's desired.
		if err := json.Unmarshal([]byte(containerStateSpec), &containerDesiredState); err != nil {
			glog.Errorf("unmarshal container state spec err: %v", err)
		} else {
			haveContainerStateAnnotation = true
		}
	} else {
		glog.V(4).Infof("pod %s has no annotation: %s",
			format.Pod(pod), sigmak8sapi.AnnotationContainerStateSpec)
	}

	// get stateStatus through annotation
	stateStatusJSONFromAnnotation, ok := pod.Annotations[sigmak8sapi.AnnotationPodUpdateStatus]
	if ok {
		if err := json.Unmarshal([]byte(stateStatusJSONFromAnnotation), &stateStatus); err != nil {
			glog.Errorf("unmarshal container state status err :%v", err)
		}
	}
	return haveContainerStateAnnotation, containerDesiredState, stateStatus
}

// GetRebuildContainerIDFromPodAnnotation get the source cotainer id in sigma2
func GetRebuildContainerIDFromPodAnnotation(pod *v1.Pod) string {
	rebuildContainerStr, exists := pod.Annotations[sigmak8sapi.AnnotationRebuildContainerInfo]
	if !exists {
		return ""
	}
	rebuildContainerInfo := sigmak8sapi.RebuildContainerInfo{}
	if err := json.Unmarshal([]byte(rebuildContainerStr), &rebuildContainerInfo); err != nil {
		glog.Errorf("Failed to get rebuildContainerInfo from pod %s because of invalid data", format.Pod(pod))
		return ""
	}

	return rebuildContainerInfo.ContainerID
}

// GetContainerRebuildInfoFromAnnotation get container which create by sigma2.0 info annotation.
func GetContainerRebuildInfoFromAnnotation(pod *v1.Pod) (*sigmak8sapi.RebuildContainerInfo, error) {
	if pod == nil || len(pod.Annotations) == 0 {
		return nil, fmt.Errorf("pod %v is nil, or pod annotation's length is zero", pod)
	}
	// get container info which come from sigma3.0 container through annotation
	rebuildContainerInfoJSON, ok := pod.Annotations[sigmak8sapi.AnnotationRebuildContainerInfo]
	if !ok {
		return nil, fmt.Errorf("pod %s has no annotation :%s",
			format.Pod(pod), sigmak8sapi.AnnotationRebuildContainerInfo)
	}
	glog.V(4).Infof("pod %s has annotation : %s,value is %s",
		format.Pod(pod), sigmak8sapi.AnnotationRebuildContainerInfo, rebuildContainerInfoJSON)

	rebuildContainerInfo := &sigmak8sapi.RebuildContainerInfo{}
	// unmarshal container info .
	if err := json.Unmarshal([]byte(rebuildContainerInfoJSON), rebuildContainerInfo); err != nil {
		// Because can't unmarshal annotation content, so assume container info annotation is invalid.
		return nil, fmt.Errorf("unmarshal container info spec err :%v", err)
	}
	return rebuildContainerInfo, nil
}

// GetUlimitsFromAnnotation extracts ulimits settings from pod annotations
func GetUlimitsFromAnnotation(container *v1.Container, pod *v1.Pod) []sigmak8sapi.Ulimit {
	podAllocSpecJSON, ok := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
	if !ok {
		return []sigmak8sapi.Ulimit{}
	}
	podAllocSpec := &sigmak8sapi.AllocSpec{}
	// unmarshal pod allocation spec
	if err := json.Unmarshal([]byte(podAllocSpecJSON), podAllocSpec); err != nil {
		glog.Errorf("could not get ulimits, unmarshal alloc spec err: %v", err)
		return []sigmak8sapi.Ulimit{}
	}
	for _, containerInfo := range podAllocSpec.Containers {
		if containerInfo.Name == container.Name {
			return containerInfo.HostConfig.Ulimits
		}
	}
	return []sigmak8sapi.Ulimit{}
}

// GetNetworkStatusFromAnnotation can get network status from certain annotation.
func GetNetworkStatusFromAnnotation(pod *v1.Pod) *sigmak8sapi.NetworkStatus {
	key := sigmak8sapi.AnnotationPodNetworkStats
	statusStr, exists := pod.Annotations[key]
	if !exists {
		glog.V(4).Infof("No network status found in pod: %v", format.Pod(pod))
		return nil
	}
	networkStatus := &sigmak8sapi.NetworkStatus{}
	err := json.Unmarshal([]byte(statusStr), networkStatus)
	if err != nil {
		glog.Errorf("Failed to unmarshal %s from pod %s into NetworkStatus: %v", statusStr, format.Pod(pod), err)
		return nil
	}
	return networkStatus
}

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
		glog.Errorf("Failed to unmarshal alloc spec, err: %v", err)
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

// GetAllocSpecFromAnnotation can get alloc-spec infomation from pod annotation.
// AnnotationPodAllocSpec = "pod.beta1.sigma.ali/alloc-spec"
func GetAllocSpecFromAnnotation(pod *v1.Pod) *sigmak8sapi.AllocSpec {
	if pod == nil || pod.Annotations == nil {
		return nil
	}

	allocSpecStr, exists := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
	if !exists {
		glog.V(4).Infof("No alloc-spec found in pod: %s", format.Pod(pod))
		return nil
	}
	allocSpec := &sigmak8sapi.AllocSpec{}
	err := json.Unmarshal([]byte(allocSpecStr), allocSpec)
	if err != nil {
		glog.V(4).Infof("Failed to unmarshal %s from pod %s into AllocSpec: %v", allocSpecStr, format.Pod(pod), err)
		return nil
	}
	return allocSpec
}

// GetHostConfigFromAnnotation can get host config infomation from pod annotation.
func GetHostConfigFromAnnotation(pod *v1.Pod, containerName string) *sigmak8sapi.HostConfigInfo {
	allocSpec := GetAllocSpecFromAnnotation(pod)
	if allocSpec == nil {
		return nil
	}

	for _, container := range allocSpec.Containers {
		if container.Name == containerName {
			return &container.HostConfig
		}
	}

	glog.V(4).Infof("No HostConfig found in pod %s for container %s", format.Pod(pod), containerName)
	return nil
}

// GetAllocResourceFromAnnotation can get alloc spec resource field from pod annotation.
func GetAllocResourceFromAnnotation(pod *v1.Pod, containerName string) *sigmak8sapi.ResourceRequirements {
	allocSpec := GetAllocSpecFromAnnotation(pod)
	if allocSpec == nil {
		return nil
	}

	for _, container := range allocSpec.Containers {
		if container.Name == containerName {
			return &container.Resource
		}
	}

	glog.V(4).Infof("No Resource found in pod %s for container %s", format.Pod(pod), containerName)
	return nil
}

// GetDanglingPodsFromNodeAnnotation can get danglingPod information from node's annotation.
func GetDanglingPodsFromNodeAnnotation(node *v1.Node) ([]sigmak8sapi.DanglingPod, error) {
	if node == nil {
		return []sigmak8sapi.DanglingPod{}, fmt.Errorf("invalid node: %v", node)
	}
	if len(node.Annotations) == 0 {
		return []sigmak8sapi.DanglingPod{}, nil
	}

	danglingPodsStr, exists := node.Annotations[sigmak8sapi.AnnotationDanglingPods]
	if !exists {
		return []sigmak8sapi.DanglingPod{}, nil
	}
	var danglingPods []sigmak8sapi.DanglingPod
	if err := json.Unmarshal([]byte(danglingPodsStr), &danglingPods); err != nil {
		glog.Errorf("[DanglingPod] Failed to unmarshal dangling pods: %s", danglingPodsStr)
		return []sigmak8sapi.DanglingPod{}, err
	}
	return danglingPods, nil
}

// GetPodAnnotationByName get annotation value by annotation name
func GetPodAnnotationByName(pod *v1.Pod, annotationName string) string {
	if pod == nil {
		glog.V(4).Info("invalid pod, pod is nil")
		return ""
	}
	if len(pod.Annotations) == 0 {
		glog.V(4).Infof("pod %s, annotation is nil", format.Pod(pod))
		return ""
	}
	return pod.Annotations[annotationName]
}

// GetAllocSpecFromAnnotation can get netpriority from pod annotation.
// Default value: 0
func GetNetPriorityFromAnnotation(pod *v1.Pod) int {
	if pod == nil || pod.Annotations == nil {
		return 0
	}

	netpriorityStr, exists := pod.Annotations[sigmak8sapi.AnnotationNetPriority]
	if !exists {
		glog.V(4).Infof("No netpriority definition found in pod: %s", format.Pod(pod))
		return 0
	}
	netpriority, err := strconv.Atoi(netpriorityStr)
	if err != nil {
		glog.V(4).Infof("Failed to convert netpriority string %s of pod %s into int: %v", netpriorityStr, format.Pod(pod), err)
		return 0
	}
	return netpriority
}
