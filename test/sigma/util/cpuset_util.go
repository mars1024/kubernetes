package util

import (
	"encoding/json"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
	"k8s.io/kubernetes/test/e2e/framework"
)

// Get all cpus from node's annotation.
func GetNodeAllCPUs(node *v1.Node) (cpuset.CPUSet, bool) {
	if node.Annotations == nil {
		return cpuset.NewCPUSet(), false
	}
	// Get local-info string from annotation.
	localInfoStr, exists := node.Annotations[sigmak8sapi.AnnotationLocalInfo]
	if !exists {
		return cpuset.NewCPUSet(), false
	}
	// Unmarshal local-info string into struct.
	localInfo := sigmak8sapi.LocalInfo{}
	if err := json.Unmarshal([]byte(localInfoStr), &localInfo); err != nil {
		glog.Errorf("Invalid localinfo data in node %s", node.Name)
		return cpuset.NewCPUSet(), false
	}
	// Get CPUID list.
	CPUIDs := make([]int, len(localInfo.CPUInfos))
	for i, CPUInfo := range localInfo.CPUInfos {
		CPUIDs[i] = int(CPUInfo.CPUID)
	}
	cpusetBuilder := cpuset.NewBuilder()
	cpusetBuilder.Add(CPUIDs...)
	return cpusetBuilder.Result(), true
}

// Get container's cpuset.
func GetContainerCPUSet(f *framework.Framework, pod *v1.Pod, containerName string) cpuset.CPUSet {
	command := "cat /sys/fs/cgroup/cpuset/cpuset.cpus"
	containerCPUSetStr := f.ExecShellInContainer(pod.Name, containerName, command)
	return cpuset.MustParse(containerCPUSetStr)
}

// GetCPUsFromPodAnnotation gets container's cpuset from pod's annotation.
func GetCPUsFromPodAnnotation(pod *v1.Pod, containerName string) (cpuset.CPUSet, bool) {
	if pod.Annotations == nil {
		return cpuset.NewCPUSet(), false
	}
	// Get allocSpecStr from annotation.
	allocSpecStr, exists := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
	if !exists {
		return cpuset.NewCPUSet(), false
	}
	// Unmarshal allocSpecStr into struct.
	allocSpec := sigmak8sapi.AllocSpec{}
	if err := json.Unmarshal([]byte(allocSpecStr), &allocSpec); err != nil {
		glog.Errorf("sigma policy invalid data in annotation from %s in %s", containerName, format.Pod(pod))
		return cpuset.NewCPUSet(), false
	}

	cpus := []int{}
	for _, containerAlloc := range allocSpec.Containers {
		if containerName != containerAlloc.Name {
			continue
		}
		// Bind container to all cpus if binding strategy is CPUBindStrategyAllCPUs
		if containerAlloc.Resource.CPU.BindingStrategy == sigmak8sapi.CPUBindStrategyAllCPUs {
			return cpuset.NewCPUSet(), true
		}
		// Check nil CPUSet because CPUSet is a pointer.
		if containerAlloc.Resource.CPU.CPUSet == nil {
			return cpuset.NewCPUSet(), false
		}
		cpus = containerAlloc.Resource.CPU.CPUSet.CPUIDs
		break
	}

	// If there is no cpu assigned in allocSpec, just return.
	if len(cpus) == 0 {
		return cpuset.NewCPUSet(), true
	}
	// Create cpuset by cpusetBuilder.
	cpusetBuilder := cpuset.NewBuilder()
	cpusetBuilder.Add(cpus...)
	return cpusetBuilder.Result(), true
}

func GetDefaultCPUSetFromNodeAnnotation(node *v1.Node) (cpuset.CPUSet, bool) {
	if node.Annotations == nil {
		return cpuset.NewCPUSet(), false
	}
	// Get defaultCPUSetStr from annotation.
	defaultCPUSetStr, exists := node.Annotations[sigmak8sapi.AnnotationNodeCPUSharePool]
	if !exists {
		return cpuset.NewCPUSet(), false
	}
	// Unmarshal defaultCPUSetStr into struct.
	defaultCPUSet := sigmak8sapi.CPUSharePool{}
	if err := json.Unmarshal([]byte(defaultCPUSetStr), &defaultCPUSet); err != nil {
		glog.Errorf("Invalid data in annotation of node: %s", node.Name)
		return cpuset.NewCPUSet(), false
	}

	// Convert int32 to int
	CPUIDs := []int{}
	for _, CPUID := range defaultCPUSet.CPUIDs {
		CPUIDs = append(CPUIDs, int(CPUID))
	}

	// Create cpuset by cpusetBuilder.
	cpusetBuilder := cpuset.NewBuilder()
	cpusetBuilder.Add(CPUIDs...)
	return cpusetBuilder.Result(), true
}
