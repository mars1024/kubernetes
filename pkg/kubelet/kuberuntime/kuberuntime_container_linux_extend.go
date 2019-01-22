// +build linux

package kuberuntime

import (
	"k8s.io/api/core/v1"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	sigmautil "k8s.io/kubernetes/pkg/kubelet/sigma"
)

// AdjustResourcesByAnnotation adjusts container resource requirement(currently only adjust cpu period) according
// to pod annotations.
func AdjustResourcesByAnnotation(pod *v1.Pod, containerName string, resources *runtimeapi.LinuxContainerResources, milliCPU int64) {
	currentCpuPeriod := resources.CpuPeriod
	if currentCpuPeriod == 0 {
		return
	}
	newCpuPeriod := sigmautil.GetCpuPeriodFromAnnotation(pod, containerName)
	if newCpuPeriod < quotaPeriod {
		return
	}
	newCpuQuota := (milliCPU * newCpuPeriod) / milliCPUToCPU
	if newCpuQuota < minQuotaPeriod {
		newCpuQuota = minQuotaPeriod
	}
	resources.CpuPeriod = newCpuPeriod
	resources.CpuQuota = newCpuQuota
}

// applyExtendContainerConfig can merge extended feilds of hostconfig into container config of CRI.
func applyExtendContainerConfig(pod *v1.Pod, container *v1.Container, config *runtimeapi.ContainerConfig) {
	// Set NetPriority field
	netpriority := int64(sigmautil.GetNetPriorityFromAnnotation(pod))
	config.NetPriority = netpriority
}

// applyExtendContainerResource can merge extended resource feilds into container config of CRI.
func applyExtendContainerResource(pod *v1.Pod, container *v1.Container, config *runtimeapi.ContainerConfig) {
	hostConfig := sigmautil.GetHostConfigFromAnnotation(pod, container.Name)
	if hostConfig == nil {
		return
	}

	config.Linux.Resources.MemorySwappiness = &runtimeapi.Int64Value{int64(hostConfig.MemorySwappiness)}
	config.Linux.Resources.MemorySwap = hostConfig.MemorySwap
	config.Linux.Resources.CpuBvtWarpNs = int64(hostConfig.CPUBvtWarpNs)
	config.Linux.Resources.PidsLimit = int64(hostConfig.PidsLimit)
}
