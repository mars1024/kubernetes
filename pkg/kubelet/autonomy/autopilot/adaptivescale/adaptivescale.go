/*
Copyright 2018 The Kubernetes Authors.

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

package adaptivescale

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	nodeutil "k8s.io/kubernetes/pkg/api/v1/node"
	v1qos "k8s.io/kubernetes/pkg/apis/core/v1/helper/qos"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	statsapi "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	kubepod "k8s.io/kubernetes/pkg/kubelet/pod"
	"k8s.io/kubernetes/pkg/kubelet/server/stats"
)

var (
	// containerScaleUpMemoryRange is range of scaling up memory.
	containerScaleUpMemoryRange = 0.25
	// containerUnderMemoryPressureCriticalValue is critical value of check if node is under memory pressure.
	containerUnderMemoryPressureCriticalValue = 0.2

	adaptivescaleName = "adaptivescale"
)

type runtimeService interface {
	// UpdateContainerResources updates cgroupfs resource.
	// id is contatiner ID.
	// resources are cgroupfs specified resources.
	UpdateContainerResources(id string, resources *runtimeapi.LinuxContainerResources) error
}

// ResourceAdjustController defines the necessary parameters for
// adjusting the resource limit
type ResourceAdjustController struct {
	name string
	node *v1.Node

	// podManager is a facade that abstracts away the various sources of pods
	// this Kubelet services.
	podManager kubepod.Manager

	// summaryProvider is used to measure usage stats on system
	summaryProvider stats.SummaryProvider

	// resourceAdjust is the container runtime service interface needed
	// to make UpdateContainerResources() calls against the containers.
	containerRuntime runtimeService

	// runStatus defaults to be false
	runStatus bool

	// executionIntervalSeconds is adaptive scale period
	executionIntervalSeconds time.Duration
}

// NewController returns a configured ResourceAdjustController.
func NewController(node *v1.Node,
	podManager kubepod.Manager,
	summaryProvider stats.SummaryProvider,
	containerRuntime runtimeService,
	runStatus bool,
	executionIntervalSeconds time.Duration,
) *ResourceAdjustController {
	controller := &ResourceAdjustController{
		node:                     node,
		podManager:               podManager,
		summaryProvider:          summaryProvider,
		containerRuntime:         containerRuntime,
		runStatus:                false,
		executionIntervalSeconds: executionIntervalSeconds,
		name: adaptivescaleName,
	}
	return controller
}

// Name returns autopilot controller name.
func (r *ResourceAdjustController) Name() string {
	return r.name
}

// GetNodeInfo returns node info
func (r *ResourceAdjustController) GetNodeInfo() *v1.Node {
	return r.node
}

// Recover syncs containers cgroups from master.
func (r *ResourceAdjustController) Recover() {
	return
}

// Operate runs the AutopilotService object.
func (r *ResourceAdjustController) Operate(annotations map[string]string) {
	r.AdaptivescaleService(annotations)
	return
}

// Start sets the runStatus to true.
func (r *ResourceAdjustController) Start(executionIntervalSeconds time.Duration) {
	r.runStatus = true
	r.executionIntervalSeconds = executionIntervalSeconds
	r.Exec()
}

// Stop sets the runStatus to false.
func (r *ResourceAdjustController) Stop() {
	r.runStatus = false
}

// IsRunning returns runStatus.
func (r *ResourceAdjustController) IsRunning() bool {
	return r.runStatus
}

// Exec is the executor for adjusting resource.
func (r *ResourceAdjustController) Exec() error {
	// set a ticker to trigger resource adjustment
	tickerInterval := time.Duration(r.executionIntervalSeconds) * time.Second
	go wait.Forever(func() {
		if !r.runStatus {
			glog.V(4).Info("AdaptiveScale feature is closed.")
			return
		}
		err := r.resourceAdjust()
		if err != nil {
			glog.Errorf("adjust resource failed: %v", err)
		}
	}, tickerInterval)
	return nil
}

func (r *ResourceAdjustController) resourceAdjust() error {
	if !r.runStatus {
		glog.V(4).Info("AdaptiveScale feature is closed.")
		return nil
	}

	// IsNodeReady returns true if a node is ready; false otherwise.
	if !nodeutil.IsNodeReady(r.node) {
		return fmt.Errorf("node is not ready: %+v, do not adjust any resources", r.node)
	}

	updateStats := true
	summary, err := r.summaryProvider.Get(updateStats)
	if err != nil {
		return fmt.Errorf("resourceAdjustController: failed to get summary stats: %v", err)
	}

	if !isNodeMemorySufficient(summary) {
		// if memory resource is not sufficent, do not scale up container's memory
		glog.V(4).Info("node memory is sufficient, stop adaptive scaling this time.")
		return nil
	}

	pods := getAllPodsOnThisNode(r.podManager)
	if pods == nil || len(pods) == 0 {
		return nil
	}

	burstablePods := filterBurstablePods(pods)

	// Loop the pods containers, check these real resource usage status
	// if container has no limit, skip it;
	for i := range burstablePods.Items {
		for _, container := range burstablePods.Items[i].Spec.Containers {
			var containerMemoryLimit int64
			if checkResourceLimitMemoryExist(container) {
				containerMemoryLimit = getContainerResourceLimitMemory(container)
			} else {
				glog.V(4).Info("container has no limit memory.")
				continue
			}

			if isContainerUnderMemoryPressure(summary, container.Name) {
				recommendedMemoryValue := calcRecommendedMemoryValue(summary, containerMemoryLimit)
				if recommendedMemoryValue == 0 {
					glog.V(4).Info("node does not have enough memory to scale up resource.")
					continue
				}
				containerID := getContainerIDByContainerName(burstablePods.Items[i], container.Name)
				if containerID == "" {
					glog.Errorf("container %s can not get containerID", container.Name)
					continue
				}
				err := r.scaleUpMemory(containerID, recommendedMemoryValue)
				if err != nil {
					glog.Errorf("container %s scales up memory limit failed, error: %v", container.Name, err)
				} else {
					glog.V(0).Info("container %s scales up memory limit successfully.", container.Name)
				}
			}
		}
	}
	return nil
}

func checkResourceLimitMemoryExist(container v1.Container) bool {
	_, ok := container.Resources.Limits[v1.ResourceMemory]
	return ok
}

func getContainerResourceLimitMemory(container v1.Container) int64 {
	return container.Resources.Limits.Memory().Value()
}

func (r *ResourceAdjustController) scaleUpMemory(containerID string, recommendedMemoryValue int64) error {
	return r.containerRuntime.UpdateContainerResources(
		containerID,
		&runtimeapi.LinuxContainerResources{
			MemoryLimitInBytes: recommendedMemoryValue,
		})
}

func filterBurstablePods(pods []*v1.Pod) *v1.PodList {
	var burstablePodsItem []v1.Pod
	for i := range pods {
		if isBurstablePod(pods[i]) {
			burstablePodsItem = append(burstablePodsItem, *pods[i])
		}
	}
	return &v1.PodList{
		Items: burstablePodsItem,
	}
}

func isContainerUnderMemoryPressure(summary *statsapi.Summary, containerName string) bool {
	for _, podStats := range summary.Pods {
		for _, containerStats := range podStats.Containers {
			if containerStats.Name == containerName {
				return float64(*containerStats.Memory.AvailableBytes)/(float64(*containerStats.Memory.AvailableBytes)+float64(*containerStats.Memory.UsageBytes)) < containerUnderMemoryPressureCriticalValue
			}
		}
	}
	return false
}

func calcRecommendedMemoryValue(summary *statsapi.Summary, containerMemoryLimit int64) int64 {
	memAvailableBytes := float64(*summary.Node.Memory.AvailableBytes)
	recommendedAddMemoryValue := float64(containerMemoryLimit) * containerScaleUpMemoryRange
	if memAvailableBytes > recommendedAddMemoryValue {
		return int64(recommendedAddMemoryValue) + containerMemoryLimit
	}
	glog.V(4).Info("Node does not have enough memory to scale up resource to container.")
	return 0
}

func isNodeMemorySufficient(summary *statsapi.Summary) bool {
	if summary.Node.Memory == nil || summary.Node.Memory.AvailableBytes == nil || summary.Node.Memory.WorkingSetBytes == nil {
		glog.V(4).Info("summary is incomplete, and stop resource adjust this time")
		return false
	}
	memCapacity := float64(*summary.Node.Memory.AvailableBytes + *summary.Node.Memory.WorkingSetBytes)
	memAvailableBytes := float64(*summary.Node.Memory.AvailableBytes)

	return (memAvailableBytes / memCapacity) > containerScaleUpMemoryRange
}

func getContainerIDByContainerName(pod v1.Pod, containerName string) string {
	for _, value := range pod.Status.ContainerStatuses {
		if value.Name == containerName {
			return value.ContainerID
		}
	}

	return ""
}

func getAllPodsOnThisNode(podManager kubepod.Manager) []*v1.Pod {
	return podManager.GetPods()
}

func isBurstablePod(pod *v1.Pod) bool {
	return v1qos.GetPodQOS(pod) == v1.PodQOSBurstable
}
