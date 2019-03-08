// +build linux

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

package kuberuntime

import (
	"strconv"
	"time"

	"k8s.io/api/core/v1"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/qos"
	sigmautil "k8s.io/kubernetes/pkg/kubelet/sigma"
)

const (
	containerPouchCPUBvtWarpNsAnnotation = "customization.cpu_bvt_warp_ns"
	containerPouchMemorySwapAnnotation   = "io.alibaba.pouch.resources.memory-swap"
)

// applyPlatformSpecificContainerConfig applies platform specific configurations to runtimeapi.ContainerConfig.
func (m *kubeGenericRuntimeManager) applyPlatformSpecificContainerConfig(config *runtimeapi.ContainerConfig, container *v1.Container, pod *v1.Pod, uid *int64, username string) error {
	config.Linux = m.generateLinuxContainerConfig(container, pod, uid, username)

	applyExtendContainerConfig(pod, container, config)

	// For pouch
	if config.Linux.Resources.CpuBvtWarpNs != 0 {
		config.Annotations[containerPouchCPUBvtWarpNsAnnotation] = strconv.Itoa(int(config.Linux.Resources.CpuBvtWarpNs))
	}
	if config.Linux.Resources.MemorySwap > 0 {
		config.Annotations[containerPouchMemorySwapAnnotation] = strconv.Itoa(int(config.Linux.Resources.MemorySwap))
	}
	return nil
}

// generateLinuxContainerConfig generates linux container config for kubelet runtime v1.
// All supported resources will be generated here.
func (m *kubeGenericRuntimeManager) generateLinuxContainerConfig(container *v1.Container, pod *v1.Pod, uid *int64, username string) *runtimeapi.LinuxContainerConfig {
	lc := &runtimeapi.LinuxContainerConfig{
		Resources:       &runtimeapi.LinuxContainerResources{},
		SecurityContext: m.determineEffectiveSecurityContext(pod, container, uid, username),
	}

	// set linux container resources
	var cpuShares int64
	cpuRequest := container.Resources.Requests.Cpu()
	cpuLimit := container.Resources.Limits.Cpu()
	memoryLimit := container.Resources.Limits.Memory().Value()
	oomScoreAdj := int64(qos.GetContainerOOMScoreAdjust(pod, container,
		int64(m.machineInfo.MemoryCapacity)))

	// if cpuRequest.Amount is nil, then milliCPUToShares will return the minimal number
	// of CPU shares.
	cpuShares = milliCPUToShares(cpuRequest.MilliValue())
	lc.Resources.CpuShares = cpuShares

	if memoryLimit != 0 {
		lc.Resources.MemoryLimitInBytes = memoryLimit
	}
	// Set OOM score of the container based on qos policy. Processes in lower-priority pods should
	// be killed first if the system runs out of memory.
	lc.Resources.OomScoreAdj = oomScoreAdj

	if m.cpuCFSQuota {
		allocSpecResource := sigmautil.GetAllocResourceFromAnnotation(pod, container.Name)
		// Set CpuQuota as -1 if container's mode is "CpuSet".
		if allocSpecResource != nil && allocSpecResource.CPU.CPUSet != nil {
			lc.Resources.CpuPeriod = int64(m.cpuCFSQuotaPeriod.Duration / time.Microsecond)
			lc.Resources.CpuQuota = -1
		} else {
			// if cpuLimit.Amount is nil, then the appropriate default value is returned
			// to allow full usage of cpu resource.
			cpuPeriod := int64(m.cpuCFSQuotaPeriod.Duration / time.Microsecond)
			cpuQuota := milliCPUToQuota(cpuLimit.MilliValue(), cpuPeriod)
			lc.Resources.CpuQuota = cpuQuota
			lc.Resources.CpuPeriod = cpuPeriod
			AdjustResourcesByAnnotation(pod, container.Name, lc.Resources, cpuLimit.MilliValue())
		}
	}

	applyExtendContainerResource(pod, container, lc, m.cpuCFSQuota)

	return lc
}
