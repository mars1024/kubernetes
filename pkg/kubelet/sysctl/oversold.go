package sysctl

import (
	"fmt"

	"encoding/json"
	"strconv"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/lifecycle"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
)

const (
	// PodForbiddenReason pod brief reject reason.
	PodForbiddenReason = "OversoldForbidden"
)

// OversoldAdmitHandler oversold admit handler.
// TODO  now just check cpu, consider mem and disk
type OversoldAdmitHandler struct {
	// get node info.
	getNodeFunc GetNodeFunc
}

// GetNodeFunc which can get node info.
type GetNodeFunc func() (*v1.Node, error)

var _ lifecycle.PodAdmitHandler = &OversoldAdmitHandler{}

// NewOversoldAdmitHandler init a new OversoldAdmitHandler
func NewOversoldAdmitHandler(getNodeFunc func() (*v1.Node, error)) (*OversoldAdmitHandler, error) {
	return &OversoldAdmitHandler{getNodeFunc: getNodeFunc}, nil
}

// Admit checks whether the pod cpu oversold.
func (o *OversoldAdmitHandler) Admit(attrs *lifecycle.PodAdmitAttributes) lifecycle.PodAdmitResult {
	// 2.0 container upgrade to 3.1 , should be admit.
	// https://aone.alibaba-inc.com/req/17099502
	if attrs.Pod != nil || len(attrs.Pod.Annotations) > 0 {
		_, exist := attrs.Pod.Annotations[sigmak8sapi.AnnotationRebuildContainerInfo]
		if exist {
			glog.V(4).Infof("admin pod %s ,because it have %s annotation",
				format.Pod(attrs.Pod), sigmak8sapi.AnnotationRebuildContainerInfo)
			return lifecycle.PodAdmitResult{Admit: true}
		}
	}

	node, err := o.getNodeFunc()
	if err != nil {
		glog.Errorf("get node info err: %v", err)
		return lifecycle.PodAdmitResult{Admit: true}
	}

	if len(node.Labels) == 0 {
		glog.V(4).Info("node label is empty, no check")
		return lifecycle.PodAdmitResult{Admit: true}
	}

	cpuOverQuotaString, exist := node.Labels[sigmak8sapi.LabelCPUOverQuota]
	if !exist {
		glog.V(4).Infof("node label not contain %q, no check", sigmak8sapi.LabelCPUOverQuota)
		return lifecycle.PodAdmitResult{Admit: true}
	}

	cpuOverQuota, err := strconv.ParseFloat(cpuOverQuotaString, 64)
	if err != nil {
		glog.V(4).Infof("node label %q value is %q, convert to float err:%v ",
			sigmak8sapi.LabelCPUOverQuota, cpuOverQuotaString, err)
		return lifecycle.PodAdmitResult{Admit: true}
	}
	//TODO consider overquota >1 scenarios
	if cpuOverQuota > 1.0 {
		glog.V(4).Infof("node label %q value is %q, Ignore node  oversold scenarios",
			sigmak8sapi.LabelCPUOverQuota, cpuOverQuotaString)
		return lifecycle.PodAdmitResult{Admit: true}
	}

	cpuMap, success := updateCPUMapFromPodAnnotation(attrs.Pod, make(map[string]int, 4))
	if !success {
		return lifecycle.PodAdmitResult{Admit: true}
	}

	// cpuCountMap is cpu map, key is cpu id ,value is cpuID used count, init size is podNum * 4, which assume one pod
	// has one container, one container has 4 cpu.
	cpuCountMap := make(map[string]int, (len(attrs.OtherPods)+1)*4)
	for _, pod := range attrs.OtherPods {
		cpuCountMap, _ = updateCPUMapFromPodAnnotation(pod, cpuCountMap)
	}

	for cpuID, count := range cpuMap {
		usedCount, exist := cpuCountMap[cpuID]
		if exist {
			return lifecycle.PodAdmitResult{
				Admit:   false,
				Reason:  PodForbiddenReason,
				Message: fmt.Sprintf("cpuID %s will oversold to %d", cpuID, count+usedCount),
			}
		}
	}

	return lifecycle.PodAdmitResult{Admit: true}
}

// updateCPUMapFromPodAnnotation update cpuID used map from pod's annotation.
func updateCPUMapFromPodAnnotation(pod *v1.Pod, cpuCountMap map[string]int) (map[string]int, bool) {
	if pod.Annotations == nil {
		return cpuCountMap, false
	}
	// Get allocSpecStr from annotation.
	allocSpecStr, exists := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
	if !exists {
		return cpuCountMap, false
	}
	// Unmarshal allocSpecStr into struct.
	allocSpec := sigmak8sapi.AllocSpec{}
	if err := json.Unmarshal([]byte(allocSpecStr), &allocSpec); err != nil {
		glog.Errorf("pod %s annotation %q, value is %v  unmarshal err:%v",
			format.Pod(pod), sigmak8sapi.AnnotationPodAllocSpec, allocSpecStr, err)
		return cpuCountMap, false
	}

	for _, containerAlloc := range allocSpec.Containers {
		// Check nil CPUSet because CPUSet is a pointer.
		if containerAlloc.Resource.CPU.CPUSet == nil {
			continue
		}
		for _, cpuID := range containerAlloc.Resource.CPU.CPUSet.CPUIDs {
			cpuCountMap[strconv.Itoa(cpuID)]++
		}
	}
	return cpuCountMap, true
}
