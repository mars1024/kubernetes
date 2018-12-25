/*
Copyright 2015 The Kubernetes Authors.

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

package sigmascheduling

import (
	"encoding/json"
	"strings"
	"testing"

	sigmaapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	kubeletapis "k8s.io/kubernetes/pkg/kubelet/apis"
)

// TestAdmitPod verifies all update requests for pods result in every container's labels
func TestAdmitPod(t *testing.T) {
	namespace := "test"
	handler := &SigmaScheduling{}
	tests := []struct {
		name              string
		pod               api.Pod
		allocInput        sigmaapi.AllocSpec
		allocOutput       sigmaapi.AllocSpec
		annotationsOutput map[string]string
		err               string
	}{
		{
			name: "admit default",
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
						{Name: "name2"},
					},
				},
			},
			allocInput: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
					},
					{
						Name: "name2",
					},
				},
			},
			allocOutput: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
					{
						Name: "name2",
						Resource: sigmaapi.ResourceRequirements{
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			annotationsOutput: map[string]string{
				sigmaapi.AnnotationNetPriority: "5",
			},
			err: "",
		},
		{
			name: "set cpu set spread strategy, disk type, gpu share mode, net-priority and cgroupParent",
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "123",
					Namespace: namespace,
					Annotations: map[string]string{
						sigmaapi.AnnotationNetPriority: "3",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			allocInput: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			allocOutput: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			annotationsOutput: map[string]string{
				sigmaapi.AnnotationNetPriority: "3",
			},
			err: "",
		},
		{
			name: "container not found in spec",
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			allocInput: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name0",
					},
				},
			},
			annotationsOutput: map[string]string{
				sigmaapi.AnnotationNetPriority: "5",
			},
			err: "container name0 not found",
		},
		{
			name: "nil annotation",
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "123",
					Namespace: namespace,
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			annotationsOutput: map[string]string{
				sigmaapi.AnnotationNetPriority: "5",
			},
		},
	}
	for _, tc := range tests {
		allocBytes, _ := json.Marshal(tc.allocInput)
		if tc.pod.Annotations != nil {
			tc.pod.Annotations[sigmaapi.AnnotationPodAllocSpec] = string(allocBytes)
		}

		err := handler.Admit(admission.NewAttributesRecord(&tc.pod, nil,
			api.Kind("Pod").WithVersion("version"), tc.pod.Namespace, tc.pod.Name,
			api.Resource("pods").WithVersion("version"), "", admission.Create, false, nil))
		if tc.err != "" {
			if err != nil {
				if !strings.Contains(err.Error(), tc.err) {
					t.Errorf("%s, Unexpected error returned from admission handler: %s, expect: %s", tc.name, err, tc.err)
				}
			} else {
				t.Errorf("%s, Missing error returned from admission handler, expect: %s", tc.name, tc.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s, Unexpected error returned from admission handler: %s, expect: %s", tc.name, err, tc.err)
			continue
		}

		var allocSpec sigmaapi.AllocSpec
		json.Unmarshal([]byte(tc.pod.Annotations[sigmaapi.AnnotationPodAllocSpec]), &allocSpec)
		if !apiequality.Semantic.DeepEqual(allocSpec, tc.allocOutput) {
			t.Errorf("%s, mismatch allocspec:\n%#v\n%#v", tc.name, allocSpec, tc.allocOutput)
		}

		delete(tc.pod.Annotations, sigmaapi.AnnotationPodAllocSpec)
		if !apiequality.Semantic.DeepEqual(tc.pod.Annotations, tc.annotationsOutput) {
			t.Errorf("%s, mismatch annotations:\n%#v\n%#v", tc.name, tc.pod.Annotations, tc.annotationsOutput)
		}
	}
}

func TestValidatePod(t *testing.T) {
	namespace := "test"
	handler := &SigmaScheduling{}
	tests := []struct {
		name     string
		pod      api.Pod
		oldPod   *api.Pod
		alloc    sigmaapi.AllocSpec
		oldAlloc *sigmaapi.AllocSpec
		err      string
		action   admission.Operation
	}{
		{
			name:   "validate default",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
						{Name: "name2"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name:     "name1",
						Resource: sigmaapi.ResourceRequirements{},
					},
					{
						Name: "name2",
					},
				},
			},
			err: `Pod "123" is invalid: [allocSpec.containers[0].resource.gpu.shareMode: Invalid value: "", allocSpec.containers[1].resource.gpu.shareMode: Invalid value: ""]`,
		},
		{
			name:   "set cpu set spread strategy, disk type, gpu share mode, net-priority and cgroupParent",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "123",
					Namespace: namespace,
					Annotations: map[string]string{
						sigmaapi.AnnotationNetPriority: "3",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			err: "",
		},
		{
			name:   "container not found in spec",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name0",
					},
				},
			},
			err: "\"name0\": container not found",
		},
		{
			name:   "nil annotation",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "123",
					Namespace: namespace,
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
		},
		{
			name:   "pod wrong cpu spread strategy",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "14",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: "",
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			err: `Pod "14" is invalid: allocSpec.containers[0].resource.cpu.cpuset.spreadStrategy: Invalid value: "": [sameCoreFirst, spread]`,
		},
		{
			name:   "pod wrong cpu spread strategy",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "15",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: "default",
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			err: `Pod "15" is invalid: allocSpec.containers[0].resource.cpu.cpuset.spreadStrategy: Invalid value: "default": [sameCoreFirst, spread]`,
		},
		{
			name:   "pod with null cpuids",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name: "name1",
							Resources: api.ResourceRequirements{
								Requests: api.ResourceList{
									api.ResourceCPU: *resource.NewQuantity(2, resource.DecimalSI),
								},
							},
						},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
		},
		{
			name:   "pod with duplicity cpuids",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					NodeName: "node",
					Containers: []api.Container{
						{
							Name: "name1",
							Resources: api.ResourceRequirements{
								Requests: api.ResourceList{
									api.ResourceCPU: *resource.NewQuantity(2, resource.DecimalSI),
								},
							},
						},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
									CPUIDs:         []int{1, 1},
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			err: "Pod \"123\" is invalid: allocSpec.containers[0].resource.cpu.cpuset.cpuIDs: Invalid value: \"[1 1]\": duplicity cpuIDs `1`",
		},
		{
			name:   "pod with cpuids but nodename is not specified",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name: "name1",
							Resources: api.ResourceRequirements{
								Requests: api.ResourceList{
									api.ResourceCPU: *resource.NewQuantity(2, resource.DecimalSI),
								},
							},
						},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
									CPUIDs:         []int{0, 1},
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			err: "Pod \"123\" is invalid: allocSpec.containers[0].resource.cpu.cpuset.cpuIDs: Invalid value: \"[0 1]\": the pod is created with specified CPUIDs, but the NodeName of this pod is not specified",
		},
		{
			name:   "pod with invalid cpuids",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					NodeName: "node",
					Containers: []api.Container{
						{
							Name: "name1",
							Resources: api.ResourceRequirements{
								Requests: api.ResourceList{
									api.ResourceCPU: resource.MustParse("2m"),
								},
							},
						},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
									CPUIDs:         []int{1, 2},
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			err: "spec.containers[0].resources.requests.cpu: Invalid value: \"2m\": pod spec is invalid, must be integer",
		},
		{
			name:   "pod with mismatch cpuids count",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					NodeName: "node",
					Containers: []api.Container{
						{
							Name: "name1",
							Resources: api.ResourceRequirements{
								Requests: api.ResourceList{
									api.ResourceCPU: resource.MustParse("3"),
								},
							},
						},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
									CPUIDs:         []int{1, 2},
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			err: "the count of cpuIDs is not match pod spec and this pod is not in inplace update process",
		},
		{
			name:   "pod update error",
			action: admission.Update,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name: "name1",
							Resources: api.ResourceRequirements{
								Requests: api.ResourceList{
									api.ResourceCPU: resource.MustParse("2"),
								},
							},
						},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
									CPUIDs:         []int{1, 2},
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			oldPod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name: "name1",
							Resources: api.ResourceRequirements{
								Requests: api.ResourceList{
									api.ResourceCPU: resource.MustParse("2"),
								},
							},
						},
					},
				},
			},
			oldAlloc: &sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySpread,
									CPUIDs:         []int{1, 2},
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			err: "can not UPDATE annotation pod.beta1.sigma.ali/alloc-spec due to only cpuIDs can update",
		},
		{
			name:   "pod update with cpuids",
			action: admission.Update,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name: "name1",
							Resources: api.ResourceRequirements{
								Requests: api.ResourceList{
									api.ResourceCPU: resource.MustParse("2"),
								},
							},
						},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			oldPod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name: "name1",
							Resources: api.ResourceRequirements{
								Requests: api.ResourceList{
									api.ResourceCPU: resource.MustParse("2"),
								},
							},
						},
					},
				},
			},
			oldAlloc: &sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
									CPUIDs:         []int{1, 2},
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
		},
		{
			name:   "net-priority parse error",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "123",
					Namespace: namespace,
					Annotations: map[string]string{
						sigmaapi.AnnotationNetPriority: "somestring",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			err: "net-priority must be integer",
		},
		{
			name:   "net-priority range error",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "123",
					Namespace: namespace,
					Annotations: map[string]string{
						sigmaapi.AnnotationNetPriority: "16",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			err: "net-priority must be with range of 0-15",
		},
		{
			name:   "wrong inplace update state",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "123",
					Namespace: namespace,
					Annotations: map[string]string{
						sigmaapi.AnnotationPodInplaceUpdateState: "wrong",
					},
				},
			},
			err: "[created, accepted, failed, succeeded]",
		},
		{
			name:   "pod with mismatch cpuids count but in inplace update",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "123",
					Namespace: namespace,
					Annotations: map[string]string{
						sigmaapi.AnnotationPodInplaceUpdateState: sigmaapi.InplaceUpdateStateCreated,
					},
				},
				Spec: api.PodSpec{
					NodeName: "node",
					Containers: []api.Container{
						{
							Name: "name1",
							Resources: api.ResourceRequirements{
								Requests: api.ResourceList{
									api.ResourceCPU: resource.MustParse("3"),
								},
							},
						},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
									CPUIDs:         []int{1, 2},
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
			},
			err: "",
		},
	}
	for i, tc := range tests {
		allocBytes, _ := json.Marshal(tc.alloc)
		if tc.pod.Annotations != nil {
			tc.pod.Annotations[sigmaapi.AnnotationPodAllocSpec] = string(allocBytes)
		}

		if tc.oldPod != nil && tc.oldAlloc != nil {
			oldAllocBytes, _ := json.Marshal(tc.oldAlloc)
			if tc.oldPod.Annotations != nil {
				tc.oldPod.Annotations[sigmaapi.AnnotationPodAllocSpec] = string(oldAllocBytes)
			}
		}

		err := handler.Validate(admission.NewAttributesRecord(&tc.pod, tc.oldPod,
			api.Kind("Pod").WithVersion("version"), tc.pod.Namespace, tc.pod.Name,
			api.Resource("pods").WithVersion("version"), "", tc.action, false, nil))
		if tc.err != "" {
			if err != nil {
				if !strings.Contains(err.Error(), tc.err) {
					t.Errorf("Case[%d]: %s, Unexpected error: %s, expect: %s", i, tc.name, err, tc.err)
				}
			} else {
				t.Errorf("Case[%d]: %s, Missing error, expect: %s", i, tc.name, tc.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("Case[%d]: %s, Unexpected error: %s, expect: %s", i, tc.name, err, tc.err)
		}
	}
}

func TestValidatePodAffinity(t *testing.T) {
	namespace := "test"
	handler := &SigmaScheduling{}
	tests := []struct {
		name     string
		pod      api.Pod
		oldPod   *api.Pod
		alloc    sigmaapi.AllocSpec
		oldAlloc *sigmaapi.AllocSpec
		err      string
		action   admission.Operation
	}{
		{
			name:   "pod affinity",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
				Affinity: &sigmaapi.Affinity{
					PodAntiAffinity: &sigmaapi.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []sigmaapi.PodAffinityTerm{
							{
								PodAffinityTerm: v1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "key2",
												Operator: metav1.LabelSelectorOpExists,
											},
										},
									},
									TopologyKey: kubeletapis.LabelHostname,
									Namespaces:  []string{"ns"},
								},
								MaxPercent: 10,
							},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []sigmaapi.WeightedPodAffinityTerm{
							{
								WeightedPodAffinityTerm: v1.WeightedPodAffinityTerm{
									Weight: 10,
									PodAffinityTerm: v1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      "key2",
													Operator: metav1.LabelSelectorOpNotIn,
													Values:   []string{"value1", "value2"},
												},
											},
										},
										TopologyKey: kubeletapis.LabelHostname,
										Namespaces:  []string{"ns"},
									},
								},
								MaxPercent: 10,
							},
						},
					},
				},
			},
			err: "",
		},
		{
			name:   "pod affinity with wrong maxPercent",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
				Affinity: &sigmaapi.Affinity{
					PodAntiAffinity: &sigmaapi.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []sigmaapi.PodAffinityTerm{
							{
								PodAffinityTerm: v1.PodAffinityTerm{

									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "key2",
												Operator: metav1.LabelSelectorOpExists,
											},
										},
									},
									TopologyKey: kubeletapis.LabelHostname,
									Namespaces:  []string{"ns"},
								},
								MaxPercent: 101,
							},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []sigmaapi.WeightedPodAffinityTerm{
							{
								WeightedPodAffinityTerm: v1.WeightedPodAffinityTerm{
									Weight: 10,
									PodAffinityTerm: v1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      "key2",
													Operator: metav1.LabelSelectorOpNotIn,
													Values:   []string{"value1", "value2"},
												},
											},
										},
										TopologyKey: kubeletapis.LabelHostname,
										Namespaces:  []string{"ns"},
									},
								},
								MaxPercent: 101,
							},
						},
					},
				},
			},
			err: "Pod \"123\" is invalid: [allocSpec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution.maxPercent: Invalid value: 101: must be between 0 and 100, inclusive, allocSpec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.maxPercent: Invalid value: 101: must be between 0 and 100, inclusive]",
		},
		{
			name:   "pod affinity with wrong requiredDuringSchedulingIgnoredDuringExecution",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
				Affinity: &sigmaapi.Affinity{
					PodAntiAffinity: &sigmaapi.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []sigmaapi.PodAffinityTerm{
							{
								PodAffinityTerm: v1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key: "key2",
												//Operator: metav1.LabelSelectorOpExists,
											},
										},
									},
									TopologyKey: kubeletapis.LabelHostname,
									Namespaces:  []string{"ns"},
								},
								MaxPercent: 10,
							},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []sigmaapi.WeightedPodAffinityTerm{
							{
								WeightedPodAffinityTerm: v1.WeightedPodAffinityTerm{
									Weight: 10,
									PodAffinityTerm: v1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key: "key2",
													//Operator: metav1.LabelSelectorOpNotIn,
													Values: []string{"value1", "value2"},
												},
											},
										},
										TopologyKey: kubeletapis.LabelHostname,
										Namespaces:  []string{"ns"},
									},
								},
								MaxPercent: 10,
							},
						},
					},
				},
			},
			err: `allocSpec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].matchExpressions.matchExpressions[0].operator: Invalid value: "": not a valid selector operator, allocSpec.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.matchExpressions.matchExpressions[0].operator: Invalid value: "": not a valid selector operator`,
		},
		{
			name:   "pod affinity with wrong topologyKey",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySameCoreFirst,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
				Affinity: &sigmaapi.Affinity{
					PodAntiAffinity: &sigmaapi.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []sigmaapi.PodAffinityTerm{
							{
								PodAffinityTerm: v1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "key2",
												Operator: metav1.LabelSelectorOpExists,
											},
										},
									},
									TopologyKey: "zone",
									Namespaces:  []string{"ns"},
								},
								MaxPercent: 10,
							},
						},
					},
				},
			},
			err: `allocSpec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey: Required value: has topologyKey "zone" but only key "kubernetes.io/hostname" is allowed`,
		},
		{
			name:   "pod cpuAntiAffinity physical core",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "9",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySpread,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
				Affinity: &sigmaapi.Affinity{
					CPUAntiAffinity: &sigmaapi.CPUAntiAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{{
							Weight: int32(10),
							PodAffinityTerm: v1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{
										{
											Key:      "sigma.ali/app-name",
											Operator: metav1.LabelSelectorOpIn,
											Values:   []string{"value1", "value2"},
										},
									},
								},
								TopologyKey: sigmaapi.TopologyKeyPhysicalCore,
								Namespaces:  []string{"ns"},
							},
						}},
					},
				},
			},
			err: "",
		},
		{
			name:   "pod cpuAntiAffinity logical core",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "10",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySpread,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
				Affinity: &sigmaapi.Affinity{
					CPUAntiAffinity: &sigmaapi.CPUAntiAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{{
							Weight: int32(10),
							PodAffinityTerm: v1.PodAffinityTerm{
								TopologyKey: sigmaapi.TopologyKeyLogicalCore,
							},
						}},
					},
				},
			},
			err: "",
		},
		{
			name:   "pod with wrong cpuAntiAffinity topo-key",
			action: admission.Create,
			pod: api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "17",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Name: "name1"},
					},
				},
			},
			alloc: sigmaapi.AllocSpec{
				Containers: []sigmaapi.Container{
					{
						Name: "name1",
						Resource: sigmaapi.ResourceRequirements{
							CPU: sigmaapi.CPUSpec{
								CPUSet: &sigmaapi.CPUSetSpec{
									SpreadStrategy: sigmaapi.SpreadStrategySpread,
								},
							},
							GPU: sigmaapi.GPUSpec{
								ShareMode: sigmaapi.GPUShareModeExclusive,
							},
						},
					},
				},
				Affinity: &sigmaapi.Affinity{
					CPUAntiAffinity: &sigmaapi.CPUAntiAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{{
							Weight: int32(10),
							PodAffinityTerm: v1.PodAffinityTerm{
								TopologyKey: "unknown-key",
							},
						}},
					},
				},
			},
			err: `Pod "17" is invalid: allocSpec.affinity.cpuAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.topologyKey: Required value: has topologyKey "unknown-key" but only key ["sigma.ali/logical-core", "sigma.ali/physical-core"] is allowed`,
		},
	}
	for _, tc := range tests {
		allocBytes, _ := json.Marshal(tc.alloc)
		if tc.pod.Annotations != nil {
			tc.pod.Annotations[sigmaapi.AnnotationPodAllocSpec] = string(allocBytes)
		}

		if tc.oldPod != nil && tc.oldAlloc != nil {
			oldAllocBytes, _ := json.Marshal(tc.oldAlloc)
			if tc.oldPod.Annotations != nil {
				tc.oldPod.Annotations[sigmaapi.AnnotationPodAllocSpec] = string(oldAllocBytes)
			}
		}

		err := handler.Validate(admission.NewAttributesRecord(&tc.pod, tc.oldPod,
			api.Kind("Pod").WithVersion("version"), tc.pod.Namespace, tc.pod.Name,
			api.Resource("pods").WithVersion("version"), "", tc.action, false, nil))
		if tc.err != "" {
			if err != nil {
				if !strings.Contains(err.Error(), tc.err) {
					t.Errorf("%s, Unexpected error: %s, expect: %s", tc.name, err, tc.err)
				}
			} else {
				t.Errorf("%s, Missing error, expect: %s", tc.name, tc.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s, Unexpected error: %s, expect: %s", tc.name, err, tc.err)
		}
	}
}

func TestValidateNodeCreate(t *testing.T) {
	namespace := "test"
	handler := &SigmaScheduling{}
	tests := []struct {
		name        string
		node        api.Node
		local       sigmaapi.LocalInfo
		localString string
		err         string
	}{
		{
			name: "correct labels",
			node: api.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
					Labels: map[string]string{
						sigmaapi.LabelCPUOverQuota:  "1.0",
						sigmaapi.LabelDiskOverQuota: "1.0",
						sigmaapi.LabelMemOverQuota:  "1.0",
					},
				},
			},
		}, {
			name: "wrong labels",
			node: api.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
					Labels: map[string]string{
						sigmaapi.LabelCPUOverQuota:  "0.0",
						sigmaapi.LabelDiskOverQuota: "-1.0",
						sigmaapi.LabelMemOverQuota:  "@.0",
					},
				},
			},
			err: "must greater then 1.0",
		}, {
			name: "correct local info",
			node: api.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
					Labels:      map[string]string{},
				},
			},
			local: sigmaapi.LocalInfo{
				CPUInfos: []sigmaapi.CPUInfo{
					{
						CPUID:    1,
						CoreID:   1,
						SocketID: 1,
					},
				},
			},
		}, {
			name: "wrong local info",
			node: api.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
					Labels:      map[string]string{},
				},
			},
			localString: "x",
			err:         "can not CREATE due to annotation node.beta1.sigma.ali/local-info json unmarshal error invalid character 'x' looking for beginning of value",
		},
	}

	for _, tc := range tests {
		if tc.localString == "" && tc.node.Annotations != nil {
			localBytes, _ := json.Marshal(tc.local)
			tc.node.Annotations[sigmaapi.AnnotationLocalInfo] = string(localBytes)
		} else if tc.localString != "" {
			tc.node.Annotations[sigmaapi.AnnotationLocalInfo] = tc.localString
		}

		err := handler.Validate(admission.NewAttributesRecord(&tc.node, nil,
			api.Kind("Node").WithVersion("version"), tc.node.Namespace, tc.node.Name,
			api.Resource("nodes").WithVersion("version"), "", admission.Create, false, nil))
		if tc.err != "" {
			if err != nil {
				if !strings.Contains(err.Error(), tc.err) {
					t.Errorf("%s, unexpected error: %s, expect: %s", tc.name, err, tc.err)
				}
			} else {
				t.Errorf("%s, missing error, expect: %s", tc.name, tc.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s, unexpected error: %s", tc.name, err)
		}
	}
}

func TestValidateNodeUpdate(t *testing.T) {
	namespace := "test"
	handler := &SigmaScheduling{}
	tests := []struct {
		name           string
		node           api.Node
		oldNode        api.Node
		local          sigmaapi.LocalInfo
		localString    string
		oldLocal       sigmaapi.LocalInfo
		oldLocalString string
		err            string
	}{
		{
			name: "correct labels",
			node: api.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
					Labels: map[string]string{
						sigmaapi.LabelCPUOverQuota:  "1.0",
						sigmaapi.LabelDiskOverQuota: "1.0",
						sigmaapi.LabelMemOverQuota:  "1.0",
					},
				},
			},
			oldNode: api.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
					Labels: map[string]string{
						sigmaapi.LabelCPUOverQuota:  "1.0",
						sigmaapi.LabelDiskOverQuota: "2.0",
						sigmaapi.LabelMemOverQuota:  "3.0",
					},
				},
			},
		}, {
			name: "wrong labels",
			node: api.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
					Labels: map[string]string{
						sigmaapi.LabelCPUOverQuota:  "0.0",
						sigmaapi.LabelDiskOverQuota: "-1.0",
						sigmaapi.LabelMemOverQuota:  "@.0",
					},
				},
			},
			oldNode: api.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
					Labels: map[string]string{
						sigmaapi.LabelCPUOverQuota:  "1.0",
						sigmaapi.LabelDiskOverQuota: "2.0",
						sigmaapi.LabelMemOverQuota:  "3.0",
					},
				},
			},
			err: "must greater then 1.0",
		}, {
			name: "correct local info",
			node: api.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
					Labels:      map[string]string{},
				},
			},
			local: sigmaapi.LocalInfo{
				CPUInfos: []sigmaapi.CPUInfo{
					{
						CPUID:    1,
						CoreID:   1,
						SocketID: 1,
					},
				},
			},
		}, {
			name: "wrong local info",
			node: api.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "123",
					Namespace:   namespace,
					Annotations: map[string]string{},
					Labels:      map[string]string{},
				},
			},
			localString: "x",
			err:         "can not UPDATE due to annotation node.beta1.sigma.ali/local-info json unmarshal error invalid character 'x' looking for beginning of value",
		},
	}

	for _, tc := range tests {
		if tc.localString == "" && tc.node.Annotations != nil {
			localBytes, _ := json.Marshal(tc.local)
			tc.node.Annotations[sigmaapi.AnnotationLocalInfo] = string(localBytes)
		} else if tc.localString != "" {
			tc.node.Annotations[sigmaapi.AnnotationLocalInfo] = tc.localString
		}

		if tc.oldLocalString == "" && tc.oldNode.Annotations != nil {
			localBytes, _ := json.Marshal(tc.oldLocal)
			tc.oldNode.Annotations[sigmaapi.AnnotationLocalInfo] = string(localBytes)
		} else if tc.oldLocalString != "" {
			tc.oldNode.Annotations[sigmaapi.AnnotationLocalInfo] = tc.oldLocalString
		}

		err := handler.Validate(admission.NewAttributesRecord(&tc.node, &tc.oldNode,
			api.Kind("Node").WithVersion("version"), tc.node.Namespace, tc.node.Name,
			api.Resource("nodes").WithVersion("version"), "", admission.Update, false, nil))
		if tc.err != "" {
			if err != nil {
				if !strings.Contains(err.Error(), tc.err) {
					t.Errorf("%s, unexpected error: %s, expect: %s", tc.name, err, tc.err)
				}
			} else {
				t.Errorf("%s, missing error, expect: %s", tc.name, tc.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s, unexpected error: %s", tc.name, err)
		}
	}
}

// TestOtherResources ensures that this admission controller is a no-op for other resources,
// subresources, and non-pods.
func TestOtherResources(t *testing.T) {
	namespace := "testnamespace"
	name := "testname"
	pod := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{Name: "ctr2", Image: "image", ImagePullPolicy: api.PullNever},
			},
		},
	}
	node := &api.Node{}
	tests := []struct {
		name        string
		kind        string
		resource    string
		subresource string
		object      runtime.Object
		expectError bool
	}{
		{
			name:     "non-pod resource",
			kind:     "Foo",
			resource: "foos",
			object:   pod,
		},
		{
			name:        "pod subresource",
			kind:        "Pod",
			resource:    "pods",
			subresource: "exec",
			object:      pod,
		},
		{
			name:     "node",
			kind:     "Node",
			resource: "nodes",
			object:   node,
		},
		{
			name:        "non-pod object",
			kind:        "Pod",
			resource:    "pods",
			object:      &api.Service{},
			expectError: true,
		},
	}

	for _, tc := range tests {
		handler := &SigmaScheduling{}

		err := handler.Validate(admission.NewAttributesRecord(tc.object, nil, api.Kind(tc.kind).WithVersion("version"), namespace, name, api.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Update, false, nil))

		if tc.expectError {
			if err == nil {
				t.Errorf("%s: unexpected nil error", tc.name)
			}
			continue
		}

		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
			continue
		}

		err = handler.Admit(admission.NewAttributesRecord(tc.object, nil, api.Kind(tc.kind).WithVersion("version"), namespace, name, api.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Update, false, nil))

		if tc.expectError {
			if err == nil {
				t.Errorf("%s: unexpected nil error", tc.name)
			}
			continue
		}

		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
			continue
		}
	}
}
