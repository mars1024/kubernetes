/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package predicates

import (
	"fmt"
	"github.com/golang/glog"
	cafelabels "gitlab.alipay-inc.com/antstack/cafe-k8s-api/pkg"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	"k8s.io/kubernetes/pkg/scheduler/algorithm/allocators"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
	schedulerutil "k8s.io/kubernetes/pkg/scheduler/util"
	"math"
)

const (
	// PodCPUSetResourceFitPred defines the name of predicate PodCPUSetResourceFit.
	PodCPUSetResourceFitPred = "PodCPUSetResourceFit"
	PodResourceBestFitPred   = "PodResourceBestFit"

	AllowedMemoryOverhead      = 0.06
	AdjustedMemoryOverhead     = 0.06
	AbsoluteMinDiffMemoryBytes = 400 * 1024 * 1024 // Minimum absolute memory diff = 400MiB
	AbsoluteMaxDiffMemoryBytes = 5 * 1024 * 1024 * 1024 // Maximum absolute memory diff = 5GiB
)

//Pod fields used:
//- resource request (cpu)
//- overquota
func PodCPUSetResourceFit(pod *v1.Pod, meta algorithm.PredicateMetadata, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	//if util.AllocSpecFromPod(pod) == nil {
	//	// Native pod should be check CPUSet
	//	return true, []algorithm.PredicateFailureReason{}, nil
	//}
	podRequest := GetResourceRequest(pod)
	if podRequest.MilliCPU == 0 && podRequest.Memory == 0 && podRequest.EphemeralStorage == 0 &&
		len(podRequest.ScalarResources) == 0 {
		return true, []algorithm.PredicateFailureReason{}, nil
	}
	var predicateFails []algorithm.PredicateFailureReason

	nodePool := allocators.NewCPUPool(nodeInfo)
	overRatio := nodePool.NodeOverRatio()
	// max Pod limit should not exceed non-exclusive pool size
	nonExclusivePoolSize := nodePool.GetNonExclusiveCPUSet().Size()
	if isCPUSetPod(pod) {
		glog.V(5).Infof("[PodCPUSetResourceFit]predicating CPUSet pod %s/%s", pod.Namespace, pod.Name)
		if allocators.IsExclusiveContainer(pod, nil) {
			exCPUs := (podRequest.MilliCPU + 999) / 1000
			actualCPUs := nodePool.AvailableCPUs()
			if int(exCPUs) > actualCPUs {
				predicateFails = append(predicateFails, NewInsufficientCPUSetError("exclusive-cpuset", exCPUs, int64(actualCPUs), int64(nodePool.GetNodeCPUSet().Size())))
			}
		} else { // over ratio
			// 1. check maxLimit of pod
			maxLimit := getPodMaxCPUCount(pod)
			allowedLimit := nonExclusivePoolSize - nodePool.CPUShareOccupiedCPUs()
			glog.V(5).Infof("[DEBUG]maxLimit=%d, nonExclusivePoolSize=%d, allowedLimit=%d, GetAllocatedCPUShare=%d",
				maxLimit, nonExclusivePoolSize, allowedLimit, nodePool.GetAllocatedCPUShare())
			if maxLimit > allowedLimit {
				predicateFails = append(predicateFails, NewInsufficientCPUSetError("shared-cpuset", int64(maxLimit), int64(nodePool.GetSharedCPUSet().Size()), int64(allowedLimit)))
			}
			// 2. check request
			maxRequestMilli := int64(float64(allowedLimit) * overRatio * 1000)
			if podRequest.MilliCPU+nodePool.GetAllocatedSharedCPUSetReq() > maxRequestMilli {
				predicateFails = append(predicateFails, NewInsufficientCPUSetError("shared-request", podRequest.MilliCPU, nodePool.GetAllocatedSharedCPUSetReq(), maxRequestMilli))
			}
		}

	} else {
		// native containers in CPUSet pod, only check CPUShare
		glog.V(5).Infof("[PodCPUSetResourceFit]predicating CPUShare pod %s/%s", pod.Namespace, pod.Name)
		newRequested := podRequest.MilliCPU + nodePool.GetAllocatedCPUShare()
		allowedCPUSharePoolSize := nodePool.GetCPUSharePoolCPUSet().Size()
		allowedMillCPUShare := int64(float64(allowedCPUSharePoolSize) * overRatio * 1000)
		if newRequested > allowedMillCPUShare {
			predicateFails = append(predicateFails, NewInsufficientResourceError(v1.ResourceCPU, podRequest.MilliCPU, nodePool.GetAllocatedCPUShare(), allowedMillCPUShare))
		}
	}
	return len(predicateFails) == 0, predicateFails, nil
}

func isCPUSetPod(pod *v1.Pod) bool {
	return allocators.IsExclusiveContainer(pod, nil) || allocators.IsSharedCPUSetPod(pod)
}

// getPodMaxLimitCpuCount only calculate the share pod stats
func getPodMaxCPUCount(pod *v1.Pod) int {
	var maxMil int64 = 0
	for _, container := range pod.Spec.Containers {
		req := container.Resources.Requests.Cpu().Value()
		limit := container.Resources.Limits.Cpu().Value()

		if limit > maxMil {
			maxMil = limit
		}

		if req > maxMil {
			maxMil = req
		}
	}

	return int(maxMil)
}

// PodResourceBestFit predicates that fit the capacity of host which best fit the pod request
func PodResourceBestFit(pod *v1.Pod, meta algorithm.PredicateMetadata, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	glog.V(5).Infof("entering PodResourceBestFit for monotype")
	node := nodeInfo.Node()
	if pod == nil {
		return false, nil, fmt.Errorf("pod is nil")
	}
	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}
	var predicateFails []algorithm.PredicateFailureReason

	if !algorithm.IsPodMonotypeHard(pod) {
		glog.V(5).Infof("pod doesn't container label/value: %s=%s, skipping", cafelabels.MonotypeLabelKey, cafelabels.MonotypeLabelValueHard)
		return true, predicateFails, nil
	}
	// TODO(yuzhi.wx) Check disk is matched
	podRequest := GetResourceRequest(pod)
	// We enlarge the node memory if pod container monotype=hard
	overcommitted, _ := schedulerutil.MemoryOverQuotaRatio(nodeInfo.Node())
	if overcommitted == 1.0 {
		overcommitted += AdjustedMemoryOverhead
	}
	// first check the host is empty
	if nodeInfo.RequestedResource().MilliCPU != 0 {
		cpuEmpty := NewMonotypeMismatchedError(v1.ResourceCPU, podRequest.MilliCPU, nodeInfo.AllocatableResource().MilliCPU, nodeInfo.AllocatableResource().MilliCPU)
		predicateFails = append(predicateFails, cpuEmpty)
	}
	if nodeInfo.RequestedResource().Memory != 0 {
		memoryEmpty := NewMonotypeMismatchedError(v1.ResourceMemory, podRequest.Memory, nodeInfo.AllocatableResource().Memory, nodeInfo.AllocatableResource().Memory)
		predicateFails = append(predicateFails, memoryEmpty)
	}

	if !IsResourceApproximate(int64(float64(nodeInfo.AllocatableResource().Memory)*overcommitted), podRequest.Memory, AllowedMemoryOverhead) {
		glog.V(3).Infof("node memory does not match pod request: %q", nodeInfo.Node().Name)
		memoryMatchError := NewMonotypeMismatchedError(v1.ResourceMemory, podRequest.Memory, nodeInfo.RequestedResource().Memory, nodeInfo.AllocatableResource().Memory)
		predicateFails = append(predicateFails, memoryMatchError)
	}
	if nodeInfo.AllocatableResource().MilliCPU != podRequest.MilliCPU {
		cpuMatchError := NewMonotypeMismatchedError(v1.ResourceCPU, podRequest.Memory, nodeInfo.RequestedResource().MilliCPU, nodeInfo.AllocatableResource().MilliCPU)
		predicateFails = append(predicateFails, cpuMatchError)
	}
	return len(predicateFails) == 0, predicateFails, nil
}

func IsResourceApproximate(data1, data2 int64, limit float64) bool {
	delta := math.Abs(float64(data1 - data2))
	ratio := delta / float64(data2)
	result := ratio <= limit
	if !result {
		glog.V(5).Infof("ratio is too large(%f %%), will compare absolute memory delta in byte: %f", ratio*100, delta)
		return delta <= AbsoluteMinDiffMemoryBytes
	}
	glog.V(5).Infof("absolute memory delta in byte: %f", delta)
	return delta <= AbsoluteMaxDiffMemoryBytes
}
