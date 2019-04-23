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
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	"k8s.io/kubernetes/pkg/scheduler/algorithm/allocators"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
)

const (
	// PodCPUSetResourceFitPred defines the name of predicate PodCPUSetResourceFit.
	PodCPUSetResourceFitPred = "PodCPUSetResourceFit"
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

	overRatio, _ := CPUOverQuotaRatio(nodeInfo)
	nodePool := allocators.NewCPUPool(nodeInfo)
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
			allowedLimit := nonExclusivePoolSize - int(float64(nodePool.GetAllocatedCPUShare()+int64(overRatio*float64(1000)))/overRatio)
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
