package resourcemutationbe

import (
	"encoding/json"
	"fmt"
	"io"

	log "github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apiserver/pkg/admission"
)

const (
	// PluginName is the name for current plugin, it should be unique among all plugins.
	PluginName                 = "AlipayResourceMutationBestEffort"
	bestEffortCGroupParentName = "/sigma-stream"
	bestEffortCPUBvtWarpNs     = int(-1)
	bestEffortOOMScoreAdj      = int64(1000)
)

const (
	// Taken from lmctfy https://github.com/google/lmctfy/blob/master/lmctfy/controllers/cpu_controller.cc
	MinShares     = 2
	SharesPerCPU  = 1024
	MilliCPUToCPU = 1000

	// 100000 is equivalent to 100ms
	QuotaPeriod    = 150000
	MinQuotaPeriod = 1000
)

// MilliCPUToQuota converts milliCPU to CFS quota and period values.
func MilliCPUToQuota(milliCPU int64, period int64) (quota int64) {
	// CFS quota is measured in two values:
	//  - cfs_period_us=100ms (the amount of time to measure usage across given by period)
	//  - cfs_quota=20ms (the amount of cpu time allowed to be used across a period)
	// so in the above example, you are limited to 20% of a single CPU
	// for multi-cpu environments, you just scale equivalent amounts
	// see https://www.kernel.org/doc/Documentation/scheduler/sched-bwc.txt for details

	if milliCPU == 0 {
		return
	}

	// we then convert your milliCPU to a value normalized over a period
	quota = (milliCPU * period) / MilliCPUToCPU

	// quota needs to be a minimum of 1ms.
	if quota < MinQuotaPeriod {
		quota = MinQuotaPeriod
	}
	return
}

// MilliCPUToShares converts the milliCPU to CFS shares.
func MilliCPUToShares(milliCPU int64) int64 {
	if milliCPU == 0 {
		// Docker converts zero milliCPU to unset, which maps to kernel default
		// for unset: 1024. Return 2 here to really match kernel default for
		// zero milliCPU.
		return MinShares
	}
	// Conceptually (milliCPU / milliCPUToCPU) * sharesPerCPU, but factored to improve rounding.
	shares := (milliCPU * SharesPerCPU) / MilliCPUToCPU
	if shares < MinShares {
		return MinShares
	}
	return int64(shares)
}

var (
	_ admission.MutationInterface = &AlipayResourceMutationBestEffort{}
)

// Register is used to register current plugin to APIServer.
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName,
		func(config io.Reader) (admission.Interface, error) {
			return newAlipayResourceMutationBestEffort(), nil
		})
}

// AlipayResourceMutationBestEffort is the main struct to mutate pod resource setting.
type AlipayResourceMutationBestEffort struct {
	*admission.Handler
}

func newAlipayResourceMutationBestEffort() *AlipayResourceMutationBestEffort {
	return &AlipayResourceMutationBestEffort{
		Handler: admission.NewHandler(admission.Create),
	}
}

// Mutate resource if this is a best effort request.
// 1. Add extended resource with name: SigmaBEResourceName and value: cpu request.
// 2. Unset cpu resource value.
func (a *AlipayResourceMutationBestEffort) Admit(attr admission.Attributes) (err error) {
	if shouldIgnore(attr) {
		return nil
	}

	pod, ok := attr.GetObject().(*v1.Pod)
	if !ok {
		return admission.NewForbidden(attr, fmt.Errorf("unexpected resource"))
	}

	if err = mutatePodResource(pod); err != nil {
		return admission.NewForbidden(attr, err)
	}
	return nil
}

func mutatePodResource(pod *v1.Pod) error {
	allocSpec, err := podAllocSpec(pod)
	if err != nil {
		return err
	}

	if allocSpec == nil {
		return fmt.Errorf("alloc spec must be set before best effort admission")
	}

	if len(allocSpec.Containers) != len(pod.Spec.Containers) {
		return fmt.Errorf("illegal alloc spec, length of containers not equal to pod spec")
	}

	// Mutate resources and cgroup values.
	for i, c := range pod.Spec.Containers {
		// Set best effort resource value.
		cpuRequestMilliValue := c.Resources.Requests.Cpu().MilliValue()
		pod.Spec.Containers[i].Resources.Requests[apis.SigmaBEResourceName] =
			*resource.NewMilliQuantity(cpuRequestMilliValue, resource.DecimalSI)
		cpuLimitMilliValue := c.Resources.Limits.Cpu().MilliValue()
		pod.Spec.Containers[i].Resources.Limits[apis.SigmaBEResourceName] =
			*resource.NewMilliQuantity(cpuLimitMilliValue, resource.DecimalSI)

		// Unset cpu resource value.
		pod.Spec.Containers[i].Resources.Requests[v1.ResourceCPU] =
			*resource.NewQuantity(0, resource.DecimalSI)
		pod.Spec.Containers[i].Resources.Limits[v1.ResourceCPU] =
			*resource.NewQuantity(0, resource.DecimalSI)

		// Mutate cgroup values in host config.
		for i, ac := range allocSpec.Containers {
			if ac.Name == c.Name {
				continue
			}
			log.Infof("mutate cgroup values in host config")
			// Set cgroup parent.
			allocSpec.Containers[i].HostConfig.CgroupParent = bestEffortCGroupParentName
			allocSpec.Containers[i].HostConfig.CPUBvtWarpNs = bestEffortCPUBvtWarpNs
			allocSpec.Containers[i].HostConfig.CpuPeriod = QuotaPeriod
			allocSpec.Containers[i].HostConfig.CpuQuota =
				MilliCPUToQuota(cpuRequestMilliValue, QuotaPeriod)
			allocSpec.Containers[i].HostConfig.CpuShares =
				MilliCPUToShares(cpuRequestMilliValue)
			allocSpec.Containers[i].HostConfig.OomScoreAdj = bestEffortOOMScoreAdj
		}
	}

	data, err := json.Marshal(allocSpec)
	if err != nil {
		return err
	}
	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)

	return nil
}

// isSigmaBestEffortPod
// return true if podQOSClass == SigmaQOSBestEffort
func isSigmaBestEffortPod(pod *v1.Pod) bool {
	return sigmak8sapi.GetPodQOSClass(pod) == sigmak8sapi.SigmaQOSBestEffort
}

func shouldIgnore(a admission.Attributes) bool {
	resource := a.GetResource().GroupResource()
	if resource != v1.Resource("pods") {
		return true
	}

	if a.GetSubresource() != "" {
		return true
	}

	pod, ok := a.GetObject().(*v1.Pod)
	if !ok {
		log.Errorf("expected pod but got %s", a.GetKind().Kind)
		return true
	}

	if a.GetOperation() != admission.Create {
		// only admit created pod.
		return true
	}

	if !isSigmaBestEffortPod(pod) {
		// only admit sigma best effort pod.
		return true
	}
	return false
}

func podAllocSpec(pod *v1.Pod) (*sigmak8sapi.AllocSpec, error) {
	if v, exists := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]; exists {
		var allocSpec *sigmak8sapi.AllocSpec
		if err := json.Unmarshal([]byte(v), &allocSpec); err != nil {
			return nil, err
		}
		return allocSpec, nil
	}
	return nil, nil
}
