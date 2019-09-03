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
package priorities

import (
	"fmt"
	"github.com/golang/glog"
	cafelabels "gitlab.alipay-inc.com/antstack/cafe-k8s-api/pkg"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	schedulerapi "k8s.io/kubernetes/pkg/scheduler/api"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
)


// PodResourceBestFitPriorityMap checks the pod with label/value monotype=soft
// It prefers the nod which best fit the pod request and node capacity
func PodResourceBestFitPriorityMap(pod *v1.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (schedulerapi.HostPriority, error) {
	node := nodeInfo.Node()
	if node == nil {
		return schedulerapi.HostPriority{}, fmt.Errorf("node not found")
	}
	if pod == nil {
		return schedulerapi.HostPriority{}, fmt.Errorf("pod is nil")
	}
	isSoft := algorithm.IsPodMonotypeSoft(pod)

	if !isSoft {
		glog.V(5).Infof("pod doesn't container label/value: %s=%s, skipping", cafelabels.MonotypeLabelKey, cafelabels.MonotypeLabelValueSoft)
		return schedulerapi.HostPriority{
			Host:  node.Name,
			Score: 0,
		}, nil
	}
	podRequest := getNonZeroRequests(pod)
	// first check the host is empty
	if nodeInfo.RequestedResource().MilliCPU != 0 {
		return schedulerapi.HostPriority{
			Host:  node.Name,
			Score: 0,
		}, nil
	}
	if nodeInfo.RequestedResource().Memory != 0 {
		return schedulerapi.HostPriority{
			Host:  node.Name,
			Score: 0,
		}, nil
	}

	memScore := resourceApproximateRatio(podRequest.Memory, nodeInfo.AllocatableResource().Memory)
	cpuScore := resourceApproximateRatio(podRequest.MilliCPU, nodeInfo.AllocatableResource().MilliCPU)

	return schedulerapi.HostPriority{
		Host:  node.Name,
		Score: (memScore + cpuScore) / 2,
	}, nil
}

// resourceApproximateRatio calculates the score of request/capacity
// at the scale of 0-10
func resourceApproximateRatio(request int64, capacity int64) int {

	// If capacity is 10 times larger than request,
	// set it to 10 * request
	if request * 10 < capacity {
		capacity = request * 10
	}
	return int(float64(request) / float64(capacity) * schedulerapi.MaxPriority)
}
