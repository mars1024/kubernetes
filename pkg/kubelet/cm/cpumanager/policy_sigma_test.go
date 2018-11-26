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
	"reflect"
	"strconv"
	"strings"
	"testing"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/state"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/topology"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

type sigmaPolicyTest struct {
	description      string
	topo             *topology.CPUTopology
	containerID      string
	stAssignments    state.ContainerCPUAssignments
	stDefaultCPUSet  cpuset.CPUSet
	pod              *v1.Pod
	podUpdated       *v1.Pod
	expErr           error
	expCPUAlloc      bool
	expCSet          cpuset.CPUSet
	expPanic         bool
	expIsChanged     bool
	expDefaultCPUSet cpuset.CPUSet
}

func TestSigmaPolicyName(t *testing.T) {
	policy := NewSigmaPolicy(&fake.Clientset{}, "", topoSingleSocketHT)

	policyName := policy.Name()
	if policyName != "sigma" {
		t.Errorf("SigmaPolicy Name() error. expected: sigma, returned: %v",
			policyName)
	}
}

func TestCheckAndCorrectDefaultCPUSet(t *testing.T) {
	for _, testCase := range []struct {
		description string
		topo        *topology.CPUTopology
		node        *v1.Node
		expCSet     cpuset.CPUSet
	}{
		{
			description: "valid sharepool",
			topo:        topoSingleSocketHT,
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "host1",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationNodeCPUSharePool: "{\"cpuIDs\":[1,2,3]}",
					},
				},
			},
			expCSet: cpuset.NewCPUSet(1, 2, 3),
		},
		{
			description: "invalid sharepool",
			topo:        topoSingleSocketHT,
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "host1",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationNodeCPUSharePool: "{\"CPUID\":[1,2,3]}",
					},
				},
			},
			expCSet: cpuset.NewCPUSet(),
		},
	} {
		kubeClient := fake.NewSimpleClientset()
		kubeClient.CoreV1().Nodes().Create(testCase.node)
		policy := NewSigmaPolicy(kubeClient, types.NodeName(testCase.node.Name), testCase.topo).(PolicyExtend)
		st := &mockState{
			assignments:   state.ContainerCPUAssignments{},
			defaultCPUSet: cpuset.NewCPUSet(),
		}
		policy.CheckAndCorrectDefaultCPUSet(st)
		if !st.GetDefaultCPUSet().Equals(testCase.expCSet) {
			t.Errorf("State CPUSet is different than expected. Have %q wants: %q", st.GetDefaultCPUSet(),
				testCase.expCSet)
		}

	}
}

func TestSigmaPolicyAdd(t *testing.T) {
	testCases := []sigmaPolicyTest{
		{
			description:     "GuPodSingleCore, SingleSocketHT, ExpectAllocCPU",
			topo:            topoSingleSocketHT,
			containerID:     "container1",
			stAssignments:   state.ContainerCPUAssignments{},
			stDefaultCPUSet: cpuset.NewCPUSet(),
			pod:             makePodWithAllocSpec("container1", "1,2", sigmak8sapi.CPUBindStrategyDefault),
			expErr:          nil,
			expCPUAlloc:     true,
			expCSet:         cpuset.NewCPUSet(1, 2),
		},
		{
			description: "GuPodMultipleCores, SingleSocketHT, ExpectAllocForOverSell",
			topo:        topoSingleSocketHT,
			containerID: "container1",
			stAssignments: state.ContainerCPUAssignments{
				"fakeID100": cpuset.NewCPUSet(2, 3, 6, 7),
			},
			stDefaultCPUSet: cpuset.NewCPUSet(),
			pod:             makePodWithAllocSpec("container1", "2,3,4,5", sigmak8sapi.CPUBindStrategyDefault),
			expErr:          nil,
			expCPUAlloc:     true,
			expCSet:         cpuset.NewCPUSet(2, 3, 4, 5),
		},
		{
			description: "GuPodMultipleCores, SingleSocketHT, InvalidAllocSpec",
			topo:        topoSingleSocketHT,
			containerID: "container1",
			stAssignments: state.ContainerCPUAssignments{
				"fakeID100": cpuset.NewCPUSet(2, 3, 6, 7),
			},
			stDefaultCPUSet: cpuset.NewCPUSet(),
			pod:             makePodWithoutAllocSpec("container1"),
			expErr:          nil,
			expCPUAlloc:     false,
			expCSet:         cpuset.NewCPUSet(0, 1, 4, 5),
		},
		{
			description: "GuPodMultipleCores, SingleSocketHT, UpdateAllocSpec",
			topo:        topoSingleSocketHT,
			containerID: "container1",
			stAssignments: state.ContainerCPUAssignments{
				"container1": cpuset.NewCPUSet(2, 3, 6, 7),
			},
			stDefaultCPUSet: cpuset.NewCPUSet(),
			pod:             makePodWithAllocSpec("container1", "2,3,4,5", sigmak8sapi.CPUBindStrategyDefault),
			expErr:          nil,
			expCPUAlloc:     true,
			expCSet:         cpuset.NewCPUSet(2, 3, 4, 5),
		},
		{
			description: "GuPodMultipleCores, SingleSocketHT, BindAllCPU",
			topo:        topoSingleSocketHT,
			containerID: "container1",
			stAssignments: state.ContainerCPUAssignments{
				"container1": cpuset.NewCPUSet(2, 3, 6, 7),
			},
			stDefaultCPUSet: cpuset.NewCPUSet(),
			pod:             makePodWithAllocSpec("container1", "2,3,4,5", sigmak8sapi.CPUBindStrategyAllCPUs),
			expErr:          nil,
			expCPUAlloc:     true,
			expCSet:         cpuset.NewCPUSet(0, 1, 2, 3, 4, 5, 6, 7),
		},
	}

	for _, testCase := range testCases {
		policy := NewSigmaPolicy(&fake.Clientset{}, "", testCase.topo)

		st := &mockState{
			assignments:   testCase.stAssignments,
			defaultCPUSet: testCase.stDefaultCPUSet,
		}

		container := &testCase.pod.Spec.Containers[0]
		err := policy.AddContainer(st, testCase.pod, container, testCase.containerID)
		if err != nil {
			t.Errorf("SigmaPolicy AddContainer() error (%v). expected add error: %v but got: %v",
				testCase.description, testCase.expErr, err)
		}

		if testCase.expCPUAlloc {
			cset, found := st.assignments[testCase.containerID]
			if !found {
				t.Errorf("SigmaPolicy AddContainer() error (%v). expected container id %v to be present in assignments %v",
					testCase.description, testCase.containerID, st.assignments)
			}

			if !reflect.DeepEqual(cset, testCase.expCSet) {
				t.Errorf("SigmaPolicy AddContainer() error (%v). expected cpuset %v but got %v",
					testCase.description, testCase.expCSet, cset)
			}
		}

		if !testCase.expCPUAlloc {
			_, found := st.assignments[testCase.containerID]
			if found {
				t.Errorf("SigmaPolicy AddContainer() error (%v). Did not expect container id %v to be present in assignments %v",
					testCase.description, testCase.containerID, st.assignments)
			}
		}
	}
}

func TestSigmaPolicyIsCPUSetChanged(t *testing.T) {
	testCases := []sigmaPolicyTest{
		{
			description:   "No assignment",
			topo:          topoSingleSocketHT,
			containerID:   "container1",
			stAssignments: state.ContainerCPUAssignments{},
			pod:           makePodWithAllocSpec("container1", "1,2", sigmak8sapi.CPUBindStrategyDefault),
			expIsChanged:  false,
		},
		{
			description: "No alloc spec",
			topo:        topoSingleSocketHT,
			containerID: "container1",
			stAssignments: state.ContainerCPUAssignments{
				"container1": cpuset.NewCPUSet(1, 2, 3),
			},
			pod:          makePodWithoutAllocSpec("container1"),
			expIsChanged: false,
		},
		{
			description: "Expect cpuset changes",
			topo:        topoSingleSocketHT,
			containerID: "container1",
			stAssignments: state.ContainerCPUAssignments{
				"container1": cpuset.NewCPUSet(1, 2, 3),
			},
			pod:          makePodWithAllocSpec("container1", "1,2", sigmak8sapi.CPUBindStrategyDefault),
			expIsChanged: true,
		},
		{
			description: "Expect cpuset doesn't change",
			topo:        topoSingleSocketHT,
			containerID: "container1",
			stAssignments: state.ContainerCPUAssignments{
				"container1": cpuset.NewCPUSet(1, 2),
			},
			pod:          makePodWithAllocSpec("container1", "1,2", sigmak8sapi.CPUBindStrategyDefault),
			expIsChanged: false,
		},
	}

	for _, testCase := range testCases {
		policy := NewSigmaPolicy(&fake.Clientset{}, "", testCase.topo).(PolicyExtend)

		st := &mockState{
			assignments:   testCase.stAssignments,
			defaultCPUSet: testCase.stDefaultCPUSet,
		}

		container := &testCase.pod.Spec.Containers[0]
		isChanged := policy.IsCPUSetChanged(st, testCase.pod, container, testCase.containerID)
		if isChanged != testCase.expIsChanged {
			t.Errorf("SigmaPolicy IsCPUSetChanged() error (%v). container id: %v",
				testCase.description, testCase.containerID)
		}
	}
}

func TestSigmaPolicyUpdateAllocSpec(t *testing.T) {
	largeTopoBuilder := cpuset.NewBuilder()
	largeTopoSock0Builder := cpuset.NewBuilder()
	largeTopoSock1Builder := cpuset.NewBuilder()
	largeTopo := *topoQuadSocketFourWayHT
	for cpuid, val := range largeTopo.CPUDetails {
		largeTopoBuilder.Add(cpuid)
		if val.SocketID == 0 {
			largeTopoSock0Builder.Add(cpuid)
		} else if val.SocketID == 1 {
			largeTopoSock1Builder.Add(cpuid)
		}
	}

	testCases := []sigmaPolicyTest{
		{
			description:     "GuPodSingleCore, SingleSocketHT, ExpectAllocCPU",
			topo:            topoSingleSocketHT,
			containerID:     "fakeID1",
			stAssignments:   state.ContainerCPUAssignments{},
			stDefaultCPUSet: cpuset.NewCPUSet(0, 1, 2, 3, 4, 5, 6, 7),
			pod:             makePodWithAllocSpec("container1", "1,2", sigmak8sapi.CPUBindStrategyDefault),
			podUpdated:      makePodWithAllocSpec("container1", "1,2,3,4", sigmak8sapi.CPUBindStrategyDefault),
			expErr:          nil,
			expCPUAlloc:     true,
			expCSet:         cpuset.NewCPUSet(1, 2, 3, 4),
		},
	}

	for _, testCase := range testCases {
		policy := NewSigmaPolicy(&fake.Clientset{}, "", testCase.topo)

		st := &mockState{
			assignments:   testCase.stAssignments,
			defaultCPUSet: testCase.stDefaultCPUSet,
		}

		container := &testCase.pod.Spec.Containers[0]
		err := policy.AddContainer(st, testCase.pod, container, testCase.containerID)
		if err != nil {
			t.Errorf("SigmaPolicy AddContainer() error (%v). expected add error: %v but got: %v",
				testCase.description, testCase.expErr, err)
		}

		containerUpdated := &testCase.podUpdated.Spec.Containers[0]
		err = policy.AddContainer(st, testCase.podUpdated, containerUpdated, testCase.containerID)
		if err != nil {
			t.Errorf("SigmaPolicy AddContainer() error (%v). expected add error: %v but got: %v",
				testCase.description, testCase.expErr, err)
		}

		if testCase.expCPUAlloc {
			cset, found := st.assignments[testCase.containerID]
			if !found {
				t.Errorf("SigmaPolicy AddContainer() error (%v). expected container id %v to be present in assignments %v",
					testCase.description, testCase.containerID, st.assignments)
			}

			if !reflect.DeepEqual(cset, testCase.expCSet) {
				t.Errorf("SigmaPolicy AddContainer() error (%v). expected cpuset %v but got %v",
					testCase.description, testCase.expCSet, cset)
			}
		}

		if !testCase.expCPUAlloc {
			_, found := st.assignments[testCase.containerID]
			if found {
				t.Errorf("SigmaPolicy AddContainer() error (%v). Did not expect container id %v to be present in assignments %v",
					testCase.description, testCase.containerID, st.assignments)
			}
		}
	}
}

func TestSigmaPolicyRemove(t *testing.T) {
	testCases := []sigmaPolicyTest{
		{
			description: "SingleSocketHT, DeAllocOneContainer",
			topo:        topoSingleSocketHT,
			containerID: "fakeID1",
			stAssignments: state.ContainerCPUAssignments{
				"fakeID1": cpuset.NewCPUSet(1, 2, 3),
			},
			stDefaultCPUSet: cpuset.NewCPUSet(),
		},
		{
			description: "SingleSocketHT, DeAllocOneContainer, BeginEmpty",
			topo:        topoSingleSocketHT,
			containerID: "fakeID1",
			stAssignments: state.ContainerCPUAssignments{
				"fakeID1": cpuset.NewCPUSet(0, 1, 2, 3),
				"fakeID2": cpuset.NewCPUSet(4, 5, 6, 7),
			},
			stDefaultCPUSet: cpuset.NewCPUSet(),
		},
		{
			description: "SingleSocketHT, DeAllocOneContainer",
			topo:        topoSingleSocketHT,
			containerID: "fakeID1",
			stAssignments: state.ContainerCPUAssignments{
				"fakeID1": cpuset.NewCPUSet(1, 3, 5),
				"fakeID2": cpuset.NewCPUSet(2, 4),
			},
			stDefaultCPUSet: cpuset.NewCPUSet(),
		},
		{
			description: "SingleSocketHT, DeAllocOneContainer, Oversell",
			topo:        topoSingleSocketHT,
			containerID: "fakeID1",
			stAssignments: state.ContainerCPUAssignments{
				"fakeID1": cpuset.NewCPUSet(0, 1, 2, 3),
				"fakeID2": cpuset.NewCPUSet(2, 3, 4, 5),
			},
			stDefaultCPUSet: cpuset.NewCPUSet(),
		},
	}

	for _, testCase := range testCases {
		policy := NewSigmaPolicy(&fake.Clientset{}, "", testCase.topo)

		st := &mockState{
			assignments:   testCase.stAssignments,
			defaultCPUSet: testCase.stDefaultCPUSet,
		}

		policy.RemoveContainer(st, testCase.containerID)

		if _, found := st.assignments[testCase.containerID]; found {
			t.Errorf("SigmaPolicy RemoveContainer() error (%v). expected containerID %v not be in assignments %v",
				testCase.description, testCase.containerID, st.assignments)
		}
	}
}

func generateAllocSpec(containerName string, cpus []int, bindingStrategy sigmak8sapi.CPUBindingStrategy) *sigmak8sapi.AllocSpec {
	allocSpec := sigmak8sapi.AllocSpec{}
	container := sigmak8sapi.Container{
		Name: containerName,
	}
	cpuSet := sigmak8sapi.CPUSetSpec{
		CPUIDs: cpus,
	}
	container.Resource.CPU.CPUSet = &cpuSet
	container.Resource.CPU.BindingStrategy = bindingStrategy
	allocSpec.Containers = append(allocSpec.Containers, container)

	return &allocSpec

}

func makePodWithAllocSpec(containerName string, cpusStr string, bindingStrategy sigmak8sapi.CPUBindingStrategy) *v1.Pod {
	cpus := []int{}
	for _, char := range strings.Split(cpusStr, ",") {
		value, _ := strconv.Atoi(char)
		cpus = append(cpus, value)
	}
	allocSpec := generateAllocSpec(containerName, cpus, bindingStrategy)
	allocSpecBytes, _ := json.Marshal(allocSpec)
	allocSpecStr := string(allocSpecBytes)
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "testPod",
			Namespace:   "testNamespace",
			Annotations: map[string]string{sigmak8sapi.AnnotationPodAllocSpec: allocSpecStr},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: containerName,
				},
			},
		},
	}
}

func makePodWithoutAllocSpec(containerName string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testPod",
			Namespace: "testNamespace",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: containerName,
				},
			},
		},
	}
}