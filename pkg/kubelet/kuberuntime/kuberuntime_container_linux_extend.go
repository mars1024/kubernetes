// +build linux

package kuberuntime

import (
	"k8s.io/api/core/v1"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"

	annotationutil "k8s.io/kubernetes/pkg/kubelet/util/annotation"
)

// AdjustResourcesByAnnotation adjusts container resource requirement(currently only adjust cpu period) according
// to pod annotations.
func AdjustResourcesByAnnotation(pod *v1.Pod, containerName string, resources *runtimeapi.LinuxContainerResources, milliCPU int64) {
	currentCpuPeriod := resources.CpuPeriod
	if currentCpuPeriod == 0 {
		return
	}
	newCpuPeriod := annotationutil.GetCpuPeriodFromAnnotation(pod, containerName)
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
