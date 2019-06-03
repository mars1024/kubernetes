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
	"errors"
	"fmt"
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
	containerPouchMemoryWMarkRatio       = "customization.memory_wmark_ratio"
	containerPouchIntelRdtMba            = "customization.intel_rdt_mba"
	containerPouchIntelRdtGroup          = "customization.intel_rdt_group"
	containerPouchPidsLimit              = "io.alibaba.pouch.resources.pids-limit"
)

// applyPlatformSpecificContainerConfig applies platform specific configurations to runtimeapi.ContainerConfig.
func (m *kubeGenericRuntimeManager) applyPlatformSpecificContainerConfig(config *runtimeapi.ContainerConfig, container *v1.Container, pod *v1.Pod, uid *int64, username string) error {
	// Get image status
	imageSpec := &runtimeapi.ImageSpec{
		Image: container.Image,
	}
	imageStatus, err := m.imageService.ImageStatus(imageSpec)
	if err != nil {
		return err
	}
	if imageStatus == nil {
		msg := fmt.Sprintf("image %s not found", container.Image)
		return errors.New(msg)
	}

	config.Linux = m.generateLinuxContainerConfig(container, pod, imageStatus, uid, username)

	applyExtendContainerConfig(pod, container, config)

	// Generate annotations for pouch.
	applyPlatformAnnotationForPouch(config)
	return nil
}

// applyPlatformAnnotationForPouch will generate some annotations for pouch.
// https://yuque.antfin-inc.com/huamin.thm/gfg57i/gg1y2a#522ef370
// https://yuque.antfin-inc.com/pouchcontainer/cncf/cfen40
func applyPlatformAnnotationForPouch(config *runtimeapi.ContainerConfig) {
	if config.Linux.Resources.CpuBvtWarpNs != 0 {
		config.Annotations[containerPouchCPUBvtWarpNsAnnotation] = strconv.Itoa(int(config.Linux.Resources.CpuBvtWarpNs))
	}
	if config.Linux.Resources.MemorySwap != 0 {
		config.Annotations[containerPouchMemorySwapAnnotation] = strconv.Itoa(int(config.Linux.Resources.MemorySwap))
	}
	if config.Linux.Resources.MemoryWmarkRatio > 0 {
		config.Annotations[containerPouchMemoryWMarkRatio] = strconv.Itoa(int(config.Linux.Resources.MemoryWmarkRatio))
	}
	if config.Linux.IntelRdt != nil {
		if len(config.Linux.IntelRdt.IntelRdtMba) != 0 {
			config.Annotations[containerPouchIntelRdtMba] = config.Linux.IntelRdt.IntelRdtMba
		}
		if len(config.Linux.IntelRdt.IntelRdtGroup) != 0 {
			config.Annotations[containerPouchIntelRdtGroup] = config.Linux.IntelRdt.IntelRdtGroup
		}
	}
	if config.Linux.Resources.PidsLimit > 0 {
		config.Annotations[containerPouchPidsLimit] = strconv.FormatInt(config.Linux.Resources.PidsLimit, 10)
	}
}

// generateLinuxContainerConfig generates linux container config for kubelet runtime v1.
// All supported resources will be generated here.
func (m *kubeGenericRuntimeManager) generateLinuxContainerConfig(container *v1.Container, pod *v1.Pod, imageStatus *runtimeapi.Image, uid *int64, username string) *runtimeapi.LinuxContainerConfig {
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

	applyExtendContainerResource(pod, container, imageStatus, lc, m.cpuCFSQuota)

	return lc
}
