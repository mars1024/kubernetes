/*
Copyright 2017 The Kubernetes Authors.

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

package cpumanager

import (
	"encoding/json"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/state"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/topology"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
)

// PolicySigma is the name of the sigma policy
const PolicySigma policyName = "sigma"

// Check sigmaPolicy has implemented PolicyExtend interface
var _ PolicyExtend = &sigmaPolicy{}

type sigmaPolicy struct {
	// client to get node info
	// https://yuque.antfin-inc.com/sigma.pouch/sigma3.x/tsmf32#cpu-sharepool
	kubeClient clientset.Interface
	// name of the node
	nodeName types.NodeName
	// cpu socket topology
	topology *topology.CPUTopology
}

// NewSigmaPolicy returns a CPU manager policy that does not change CPU
// assignments for exclusively pinned guaranteed containers after the main
// container process starts.
// SigmaPolicy reads the CPU assignments from pod's annotation.
func NewSigmaPolicy(kubeClient clientset.Interface, nodeName types.NodeName, topology *topology.CPUTopology) Policy {
	return &sigmaPolicy{
		kubeClient: kubeClient,
		nodeName:   nodeName,
		topology:   topology,
	}
}

// Name returns sigmaPolicy's name.
func (p *sigmaPolicy) Name() string {
	return string(PolicySigma)
}

// Start checks sigma policy and does some initialized works.
func (p *sigmaPolicy) Start(s state.State) {
	node, err := p.kubeClient.Core().Nodes().Get(string(p.nodeName), metav1.GetOptions{})
	if err != nil {
		glog.Warningf("[cpumanager] sigma policy can't get node: %s, set defaultCPUSet to all cpus", node.Name)
		s.SetDefaultCPUSet(p.topology.CPUDetails.CPUs())
		return
	}

	expectDefaultCPUSet, exists := p.getDefaultCPUSetFromNodeAnnotation(node)
	if !exists {
		glog.Warningf("[cpumanager] sigma policy can't get defultCPUSet from node: %s, set defaultCPUSet to all cpus", node.Name)
		s.SetDefaultCPUSet(p.topology.CPUDetails.CPUs())
		return
	}
	s.SetDefaultCPUSet(expectDefaultCPUSet)
}

// CheckAndCorrectDefaultCPUSet can check DefaultCPUSet. If DefaultCPUSet is not correct, then correct it.
func (p *sigmaPolicy) CheckAndCorrectDefaultCPUSet(s state.State) {
	node, err := p.kubeClient.Core().Nodes().Get(string(p.nodeName), metav1.GetOptions{})
	if err != nil {
		glog.Errorf("[cpumanager] sigma policy can't get node: %s", node.Name)
		return
	}

	expectDefaultCPUSet, exists := p.getDefaultCPUSetFromNodeAnnotation(node)
	if !exists {
		return
	}
	currentDefaultCPUSet := s.GetDefaultCPUSet()
	if currentDefaultCPUSet.Equals(expectDefaultCPUSet) {
		return
	}
	s.SetDefaultCPUSet(expectDefaultCPUSet)
}

func (p *sigmaPolicy) getDefaultCPUSetFromNodeAnnotation(node *v1.Node) (cpuset.CPUSet, bool) {
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
		glog.Errorf("[cpumanager] sigma policy invalid data in annotation of node: %s", node.Name)
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

// IsCPUSetChanged check whether container's expect cpuset is changed or not.
// If expect cpuset is equal to cpuset in assignment, then return true; else return false.
func (p *sigmaPolicy) IsCPUSetChanged(s state.State, pod *v1.Pod, container *v1.Container, containerID string) bool {
	// Get expect cpuset
	expectCPUSet, exists := p.getCPUsFromAnnotation(s, pod, container)
	if !exists {
		return false
	}
	// Get current cpuset
	currentCPUSet, ok := s.GetCPUSet(containerID)
	if !ok {
		return false
	}
	// Compare current cpuset with expect cpuset
	if currentCPUSet.Equals(expectCPUSet) {
		return false
	}

	return true
}

// AddContainer can set a container's cpuset or put this container into shared pool.
func (p *sigmaPolicy) AddContainer(s state.State, pod *v1.Pod, container *v1.Container, containerID string) error {
	// Get expect cpuset
	expectCPUSet, exists := p.getCPUsFromAnnotation(s, pod, container)
	if !exists {
		return nil
	}
	// Set cpuset for containerID in state.
	s.SetCPUSet(containerID, expectCPUSet)

	// Check and correct default cpuset.
	p.CheckAndCorrectDefaultCPUSet(s)
	return nil
}

// getCPUsFromAnnotation gets container's cpuset from pod's annotation.
// If the second return value is true, then this container is regarded as "cpuset",
// else container is regarded as "sharedpool".
func (p *sigmaPolicy) getCPUsFromAnnotation(s state.State, pod *v1.Pod, container *v1.Container) (cpuset.CPUSet, bool) {
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
		glog.Errorf("[cpumanager] sigma policy invalid data in annotation from %s in %s", container.Name, format.Pod(pod))
		return cpuset.NewCPUSet(), false
	}
	// Get container's CPUIDs.
	containerName := container.Name
	cpus := []int{}
	for _, containerAlloc := range allocSpec.Containers {
		if containerName != containerAlloc.Name {
			continue
		}
		// Bind container to all cpus if binding strategy is CPUBindStrategyAllCPUs
		if containerAlloc.Resource.CPU.BindingStrategy == sigmak8sapi.CPUBindStrategyAllCPUs {
			return p.topology.CPUDetails.CPUs(), true
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

// RemoveContainer delete a container from state
func (p *sigmaPolicy) RemoveContainer(s state.State, containerID string) error {
	glog.Infof("[cpumanager] sigma policy: RemoveContainer (container id: %s)", containerID)
	// step1: Check current cpuset in state
	_, ok := s.GetCPUSet(containerID)
	if ok {
		// step2: Delete cpuset from state
		s.Delete(containerID)

		// step3: Check and correct default cpuset.
		p.CheckAndCorrectDefaultCPUSet(s)
	}
	return nil
}
