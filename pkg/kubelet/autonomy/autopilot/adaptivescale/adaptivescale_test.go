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
	"reflect"
	"testing"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	statsapi "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/configmap"
	kubepod "k8s.io/kubernetes/pkg/kubelet/pod"
	podtest "k8s.io/kubernetes/pkg/kubelet/pod/testing"
	"k8s.io/kubernetes/pkg/kubelet/secret"
)

func testIsBurstablePod(t *testing.T) {
	testCases := []struct {
		pod      *v1.Pod
		expected bool
	}{
		{
			pod: newPod("guaranteed", []v1.Container{
				newContainer("guaranteed", getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
			}),
			expected: false,
		},
		{
			pod: newPod("guaranteed-with-gpu", []v1.Container{
				newContainer("guaranteed", getResourceList("100m", "100Mi"), addResource("nvidia-gpu", "2", getResourceList("100m", "100Mi"))),
			}),
			expected: false,
		},
		{
			pod: newPod("guaranteed-guaranteed", []v1.Container{
				newContainer("guaranteed", getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
				newContainer("guaranteed", getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
			}),
			expected: false,
		},
		{
			pod: newPod("guaranteed-guaranteed-with-gpu", []v1.Container{
				newContainer("guaranteed", getResourceList("100m", "100Mi"), addResource("nvidia-gpu", "2", getResourceList("100m", "100Mi"))),
				newContainer("guaranteed", getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
			}),
			expected: false,
		},
		{
			pod: newPod("best-effort-best-effort", []v1.Container{
				newContainer("best-effort", getResourceList("", ""), getResourceList("", "")),
				newContainer("best-effort", getResourceList("", ""), getResourceList("", "")),
			}),
			expected: false,
		},
		{
			pod: newPod("best-effort-best-effort-with-gpu", []v1.Container{
				newContainer("best-effort", getResourceList("", ""), addResource("nvidia-gpu", "2", getResourceList("", ""))),
				newContainer("best-effort", getResourceList("", ""), getResourceList("", "")),
			}),
			expected: false,
		},
		{
			pod: newPod("best-effort-with-gpu", []v1.Container{
				newContainer("best-effort", getResourceList("", ""), addResource("nvidia-gpu", "2", getResourceList("", ""))),
			}),
			expected: false,
		},
		{
			pod: newPod("best-effort-burstable", []v1.Container{
				newContainer("best-effort", getResourceList("", ""), addResource("nvidia-gpu", "2", getResourceList("", ""))),
				newContainer("burstable", getResourceList("1", ""), getResourceList("2", "")),
			}),
			expected: true,
		},
		{
			pod: newPod("best-effort-guaranteed", []v1.Container{
				newContainer("best-effort", getResourceList("", ""), addResource("nvidia-gpu", "2", getResourceList("", ""))),
				newContainer("guaranteed", getResourceList("10m", "100Mi"), getResourceList("10m", "100Mi")),
			}),
			expected: false,
		},
		{
			pod: newPod("burstable-cpu-guaranteed-memory", []v1.Container{
				newContainer("burstable", getResourceList("", "100Mi"), getResourceList("", "100Mi")),
			}),
			expected: true,
		},
		{
			pod: newPod("burstable-no-limits", []v1.Container{
				newContainer("burstable", getResourceList("100m", "100Mi"), getResourceList("", "")),
			}),
			expected: true,
		},
		{
			pod: newPod("burstable-guaranteed", []v1.Container{
				newContainer("burstable", getResourceList("1", "100Mi"), getResourceList("2", "100Mi")),
				newContainer("guaranteed", getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
			}),
			expected: true,
		},
		{
			pod: newPod("burstable-unbounded-but-requests-match-limits", []v1.Container{
				newContainer("burstable", getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
				newContainer("burstable-unbounded", getResourceList("100m", "100Mi"), getResourceList("", "")),
			}),
			expected: true,
		},
		{
			pod: newPod("burstable-1", []v1.Container{
				newContainer("burstable", getResourceList("10m", "100Mi"), getResourceList("100m", "200Mi")),
			}),
			expected: true,
		},
		{
			pod: newPod("burstable-2", []v1.Container{
				newContainer("burstable", getResourceList("0", "0"), addResource("nvidia-gpu", "2", getResourceList("100m", "200Mi"))),
			}),
			expected: true,
		},
		{
			pod: newPod("best-effort-hugepages", []v1.Container{
				newContainer("best-effort", addResource("hugepages-2Mi", "1Gi", getResourceList("0", "0")), addResource("hugepages-2Mi", "1Gi", getResourceList("0", "0"))),
			}),
			expected: false,
		},
	}
	for id, testCase := range testCases {
		if actual := isBurstablePod(testCase.pod); testCase.expected != actual {
			t.Errorf("[%d]: invalid qos pod %s, expected: %t, actual: %t", id, testCase.pod.Name, testCase.expected, actual)
		}
	}
}

func getResourceList(cpu, memory string) v1.ResourceList {
	res := v1.ResourceList{}
	if cpu != "" {
		res[v1.ResourceCPU] = resource.MustParse(cpu)
	}
	if memory != "" {
		res[v1.ResourceMemory] = resource.MustParse(memory)
	}
	return res
}

func addResource(rName, value string, rl v1.ResourceList) v1.ResourceList {
	rl[v1.ResourceName(rName)] = resource.MustParse(value)
	return rl
}

func getResourceRequirements(requests, limits v1.ResourceList) v1.ResourceRequirements {
	res := v1.ResourceRequirements{}
	res.Requests = requests
	res.Limits = limits
	return res
}

func newContainer(name string, requests v1.ResourceList, limits v1.ResourceList) v1.Container {
	return v1.Container{
		Name:      name,
		Resources: getResourceRequirements(requests, limits),
	}
}

func newPod(name string, containers []v1.Container) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PodSpec{
			Containers: containers,
		},
	}
}

func TestGetContainerIDByContainerName(t *testing.T) {
	type args struct {
		pod           v1.Pod
		containerName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get-container-id",
			args: args{
				pod: v1.Pod{
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{
							{
								Name:        "busybox-1",
								ContainerID: "112233",
							},
							{
								Name:        "busybox-0",
								ContainerID: "001122",
							},
						},
					},
				},
				containerName: "busybox-1",
			},
			want: "112233",
		},
		{
			name: "do-not-get-container-id",
			args: args{
				pod: v1.Pod{
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{
							{
								Name:        "busybox-1",
								ContainerID: "112233",
							},
						},
					},
				},
				containerName: "busybox-11",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getContainerIDByContainerName(tt.args.pod, tt.args.containerName); got != tt.want {
				t.Errorf("getContainerIDByContainerName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNodeMemorySufficient(t *testing.T) {
	type args struct {
		summary *statsapi.Summary
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nodeMemoryIsSufficient",
			args: args{
				summary: &statsapi.Summary{
					Node: statsapi.NodeStats{
						Memory: &statsapi.MemoryStats{
							AvailableBytes:  uint64Ptr(50),
							WorkingSetBytes: uint64Ptr(50),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "nodeMemoryIsNotSufficient",
			args: args{
				summary: &statsapi.Summary{
					Node: statsapi.NodeStats{
						Memory: &statsapi.MemoryStats{
							AvailableBytes:  uint64Ptr(10),
							WorkingSetBytes: uint64Ptr(90),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "summaryIsIncomplete",
			args: args{
				summary: &statsapi.Summary{
					Node: statsapi.NodeStats{
						Memory: &statsapi.MemoryStats{
							AvailableBytes:  uint64Ptr(10),
							WorkingSetBytes: uint64Ptr(90),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "summaryIsIncomplete",
			args: args{
				summary: &statsapi.Summary{
					Node: statsapi.NodeStats{
						Memory: &statsapi.MemoryStats{
							AvailableBytes: uint64Ptr(10),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "summaryIsIncomplete",
			args: args{
				summary: &statsapi.Summary{
					Node: statsapi.NodeStats{
						Memory: &statsapi.MemoryStats{
							WorkingSetBytes: uint64Ptr(90),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "summaryIsIncomplete",
			args: args{
				summary: &statsapi.Summary{
					Node: statsapi.NodeStats{},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNodeMemorySufficient(tt.args.summary); got != tt.want {
				t.Errorf("isNodeMemorySufficient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func uint64Ptr(i uint64) *uint64 {
	return &i
}

func TestCalcRecommendedMemoryValue(t *testing.T) {
	type args struct {
		summary              *statsapi.Summary
		containerMemoryLimit int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "containerScaleUpMemoryLimit",
			args: args{
				summary: &statsapi.Summary{
					Node: statsapi.NodeStats{
						Memory: &statsapi.MemoryStats{
							AvailableBytes: uint64Ptr(100),
						},
					},
				},
				containerMemoryLimit: int64(300),
			},
			want: 375,
		},
		{
			name: "containerDoNotScaleUpMemoryLimit",
			args: args{
				summary: &statsapi.Summary{
					Node: statsapi.NodeStats{
						Memory: &statsapi.MemoryStats{
							AvailableBytes: uint64Ptr(100),
						},
					},
				},
				containerMemoryLimit: int64(400),
			},
			want: 0,
		},
		{
			name: "containerDoNotScaleUpMemoryLimit",
			args: args{
				summary: &statsapi.Summary{
					Node: statsapi.NodeStats{
						Memory: &statsapi.MemoryStats{
							AvailableBytes: uint64Ptr(100),
						},
					},
				},
				containerMemoryLimit: int64(500),
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calcRecommendedMemoryValue(tt.args.summary, tt.args.containerMemoryLimit); got != tt.want {
				t.Errorf("calcRecommendedMemoryValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsContainerUnderMemoryPressure(t *testing.T) {
	type args struct {
		summary       *statsapi.Summary
		containerName string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "containerIsUnderMemoryPressure",
			args: args{
				summary: &statsapi.Summary{
					Pods: []statsapi.PodStats{
						{
							Containers: []statsapi.ContainerStats{
								{
									Name: "aaa",
									Memory: &statsapi.MemoryStats{
										AvailableBytes: uint64Ptr(10),
										UsageBytes:     uint64Ptr(10),
									},
								},
								{
									Name: "aab",
									Memory: &statsapi.MemoryStats{
										AvailableBytes: uint64Ptr(10),
										UsageBytes:     uint64Ptr(10),
									},
								},
							},
						},
						{
							Containers: []statsapi.ContainerStats{
								{
									Name: "bbb",
									Memory: &statsapi.MemoryStats{
										AvailableBytes: uint64Ptr(10),
										UsageBytes:     uint64Ptr(10),
									},
								},
								{
									Name: "bbc",
									Memory: &statsapi.MemoryStats{
										AvailableBytes: uint64Ptr(10),
										UsageBytes:     uint64Ptr(90),
									},
								},
							},
						},
					},
				},
				containerName: "bbc",
			},
			want: true,
		},
		{
			name: "containerIsNotUnderMemoryPressure",
			args: args{
				summary: &statsapi.Summary{
					Pods: []statsapi.PodStats{
						{
							Containers: []statsapi.ContainerStats{
								{
									Name: "aaa",
									Memory: &statsapi.MemoryStats{
										AvailableBytes: uint64Ptr(10),
										UsageBytes:     uint64Ptr(10),
									},
								},
								{
									Name: "aab",
									Memory: &statsapi.MemoryStats{
										AvailableBytes: uint64Ptr(10),
										UsageBytes:     uint64Ptr(10),
									},
								},
							},
						},
						{
							Containers: []statsapi.ContainerStats{
								{
									Name: "bbb",
									Memory: &statsapi.MemoryStats{
										AvailableBytes: uint64Ptr(10),
										UsageBytes:     uint64Ptr(10),
									},
								},
								{
									Name: "bbc",
									Memory: &statsapi.MemoryStats{
										AvailableBytes: uint64Ptr(10),
										UsageBytes:     uint64Ptr(90),
									},
								},
							},
						},
					},
				},
				containerName: "bbb",
			},
			want: false,
		},
		{
			name: "summaryHasNoContainerStats",
			args: args{
				summary: &statsapi.Summary{
					Pods: []statsapi.PodStats{
						{
							Containers: []statsapi.ContainerStats{},
						},
						{
							Containers: []statsapi.ContainerStats{},
						},
					},
				},
				containerName: "",
			},
			want: false,
		},
		{
			name: "summaryHasNoPodStats",
			args: args{
				summary: &statsapi.Summary{
					Pods: []statsapi.PodStats{},
				},
				containerName: "",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isContainerUnderMemoryPressure(tt.args.summary, tt.args.containerName); got != tt.want {
				t.Errorf("isContainerUnderMemoryPressure() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterBurstablePods(t *testing.T) {
	type args struct {
		pods []*v1.Pod
	}

	pod1 := newPod("burstable-1", []v1.Container{
		newContainer("burstable", getResourceList("10m", "100Mi"), getResourceList("100m", "200Mi")),
	})

	pod2 := newPod("burstable-2", []v1.Container{
		newContainer("burstable", getResourceList("0", "0"), addResource("nvidia-gpu", "2", getResourceList("100m", "200Mi"))),
	})

	pod3 := newPod("guaranteed", []v1.Container{
		newContainer("guaranteed", getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
	})

	tests := []struct {
		name string
		args args
		want *v1.PodList
	}{
		{
			name: "getBurstablePods",
			args: args{
				pods: []*v1.Pod{
					pod1,
					pod2,
					pod3,
				},
			},
			want: &v1.PodList{
				Items: []v1.Pod{
					*pod1,
					*pod2,
				},
			},
		},
		{
			name: "getBurstablePods",
			args: args{
				pods: []*v1.Pod{
					pod3,
				},
			},
			want: &v1.PodList{
				Items: []v1.Pod{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterBurstablePods(tt.args.pods)
			if len(got.Items) != len(tt.want.Items) {
				t.Errorf("filterBurstablePods() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeSummaryProvider struct {
	result *statsapi.Summary
}

func (f *fakeSummaryProvider) Get(updateStats bool) (*statsapi.Summary, error) {
	return f.result, nil
}

type mockContainerRuntime struct {
	err error
}

func (rt mockContainerRuntime) UpdateContainerResources(id string, resources *runtimeapi.LinuxContainerResources) error {
	return rt.err
}

func mockResourceAdjustController() *ResourceAdjustController {
	cpm := podtest.NewMockCheckpointManager()
	podManager := kubepod.NewBasicPodManager(podtest.NewFakeMirrorClient(), secret.NewFakeManager(), configmap.NewFakeManager(), cpm)
	return &ResourceAdjustController{
		node: &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "127.0.0.1",
			},
		},
		podManager:       podManager,
		summaryProvider:  &fakeSummaryProvider{},
		containerRuntime: mockContainerRuntime{},
		runStatus:        false,
	}
}

func Test_checkResourceLimitMemoryExist(t *testing.T) {
	type args struct {
		container v1.Container
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "limitMemoryExist",
			args: args{
				container: newContainer("burstable-01", getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
			},
			want: true,
		},
		{
			name: "limitMemoryDoNotExist",
			args: args{
				container: newContainer("burstable-02", getResourceList("100m", "100Mi"), getResourceList("", "")),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkResourceLimitMemoryExist(tt.args.container); got != tt.want {
				t.Errorf("checkResourceLimitMemoryExist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getContainerResourceLimitMemory(t *testing.T) {
	type args struct {
		container v1.Container
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "abc",
			args: args{
				container: newContainer("burstable-01", getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
			},
			want: 209715200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getContainerResourceLimitMemory(tt.args.container); got != tt.want {
				t.Errorf("getContainerResourceLimitMemory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceAdjustController_scaleUpMemory(t *testing.T) {
	type args struct {
		containerID               string
		recommendedAddMemoryValue int64
	}
	tests := []struct {
		name    string
		fields  ResourceAdjustController
		args    args
		wantErr error
	}{
		{
			name:   "nil",
			fields: *mockResourceAdjustController(),
			args: args{
				containerID:               "abc",
				recommendedAddMemoryValue: int64(100),
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ResourceAdjustController{
				node:             tt.fields.node,
				podManager:       tt.fields.podManager,
				summaryProvider:  tt.fields.summaryProvider,
				containerRuntime: tt.fields.containerRuntime,
				runStatus:        tt.fields.runStatus,
			}
			if err := r.scaleUpMemory(tt.args.containerID, tt.args.recommendedAddMemoryValue); err != tt.wantErr {
				t.Errorf("ResourceAdjustController.scaleUpMemory() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func mockResourceAdjustControllerOfRunningStatus() *ResourceAdjustController {
	cpm := podtest.NewMockCheckpointManager()
	podManager := kubepod.NewBasicPodManager(podtest.NewFakeMirrorClient(), secret.NewFakeManager(), configmap.NewFakeManager(), cpm)
	return &ResourceAdjustController{
		node:             &v1.Node{},
		podManager:       podManager,
		summaryProvider:  &fakeSummaryProvider{},
		containerRuntime: mockContainerRuntime{},
		runStatus:        true,
	}
}

func TestNewController(t *testing.T) {
	tests := []struct {
		name string
		args *ResourceAdjustController
		want *ResourceAdjustController
	}{
		{
			name: "nil",
			args: mockResourceAdjustController(),
			want: mockResourceAdjustControllerOfRunningStatus(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewController(tt.args.node, tt.args.podManager, tt.args.summaryProvider, tt.args.containerRuntime, tt.args.runStatus, 10); reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected result: two objects are not equal")
			}
		})
	}
}

func TestResourceAdjustController_Stop(t *testing.T) {
	type fields struct {
		runStatus bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "stop",
			fields: fields{
				runStatus: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ResourceAdjustController{
				runStatus: tt.fields.runStatus,
			}
			r.Stop()
			if r.runStatus == true {
				t.Errorf("Stop() runs failed.")
			}
		})
	}
}

func TestResourceAdjustController_IsRunning(t *testing.T) {
	type fields struct {
		runStatus bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "isRunning",
			fields: fields{
				runStatus: true,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ResourceAdjustController{
				runStatus: tt.fields.runStatus,
			}
			if got := r.IsRunning(); got != tt.want {
				t.Errorf("ResourceAdjustController.IsRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getAllPodsOnThisNode(t *testing.T) {
	cpm := podtest.NewMockCheckpointManager()
	podManager := kubepod.NewBasicPodManager(podtest.NewFakeMirrorClient(), secret.NewFakeManager(), configmap.NewFakeManager(), cpm)
	type args struct {
		podManager kubepod.Manager
	}
	tests := []struct {
		name string
		args args
		want []*v1.Pod
	}{
		{
			name: "getNoPods",
			args: args{
				podManager: podManager,
			},
			want: []*v1.Pod{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAllPodsOnThisNode(tt.args.podManager); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAllPodsOnThisNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceAdjustController_resourceAdjust(t *testing.T) {
	tests := []struct {
		name   string
		fields ResourceAdjustController
	}{
		{
			name:   "nil",
			fields: *mockResourceAdjustController(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ResourceAdjustController{
				node:             tt.fields.node,
				podManager:       tt.fields.podManager,
				summaryProvider:  tt.fields.summaryProvider,
				containerRuntime: tt.fields.containerRuntime,
				runStatus:        tt.fields.runStatus,
			}
			r.resourceAdjust()
		})
	}
}

func TestResourceAdjustController_Exec(t *testing.T) {
	type fields struct {
		runStatus bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "nil",
			fields: fields{
				runStatus: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ResourceAdjustController{
				runStatus: tt.fields.runStatus,
			}
			r.Exec()
		})
	}
}

func TestResourceAdjustController_Start(t *testing.T) {
	type fields struct {
		runStatus bool
		node      *v1.Node
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "nil",
			fields: fields{
				runStatus: false,
				node: &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeOutOfDisk, Status: v1.ConditionTrue},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ResourceAdjustController{
				runStatus: tt.fields.runStatus,
				node:      tt.fields.node,
			}
			r.Start(10 * time.Second)
		})
	}
}
