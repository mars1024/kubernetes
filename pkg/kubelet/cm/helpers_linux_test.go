// +build linux

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

package cm

import (
	"reflect"
	"strconv"
	"testing"
	"time"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	utilfeaturetesting "k8s.io/apiserver/pkg/util/feature/testing"
	pkgfeatures "k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/json"
)

// getResourceList returns a ResourceList with the
// specified cpu and memory resource values
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

// getResourceRequirements returns a ResourceRequirements object
func getResourceRequirements(requests, limits v1.ResourceList) v1.ResourceRequirements {
	res := v1.ResourceRequirements{}
	res.Requests = requests
	res.Limits = limits
	return res
}

func TestResourceConfigForPod(t *testing.T) {
	defaultQuotaPeriod := uint64(100 * time.Millisecond / time.Microsecond)
	tunedQuotaPeriod := uint64(5 * time.Millisecond / time.Microsecond)

	minShares := uint64(MinShares)
	burstableShares := MilliCPUToShares(100)
	memoryQuantity := resource.MustParse("200Mi")
	burstableMemory := memoryQuantity.Value()
	burstablePartialShares := MilliCPUToShares(200)
	burstableQuota := MilliCPUToQuota(200, int64(defaultQuotaPeriod))
	guaranteedShares := MilliCPUToShares(100)
	// guaranteedQuota := MilliCPUToQuota(100, int64(defaultQuotaPeriod))
	// guaranteedTunedQuota := MilliCPUToQuota(100, int64(tunedQuotaPeriod))
	memoryQuantity = resource.MustParse("100Mi")
	cpuNoLimit := int64(-1)
	guaranteedMemory := memoryQuantity.Value()
	testCases := map[string]struct {
		pod              *v1.Pod
		expected         *ResourceConfig
		enforceCPULimits bool
		quotaPeriod      uint64
	}{
		"besteffort": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("", ""), getResourceList("", "")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &minShares},
		},
		"burstable-no-limits": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("", "")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstableShares},
		},
		"burstable-with-limits": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstableShares, CpuQuota: &burstableQuota, CpuPeriod: &defaultQuotaPeriod, Memory: &burstableMemory},
		},
		"burstable-with-limits-no-cpu-enforcement": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
						},
					},
				},
			},
			enforceCPULimits: false,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstableShares, CpuQuota: &cpuNoLimit, CpuPeriod: &defaultQuotaPeriod, Memory: &burstableMemory},
		},
		"burstable-partial-limits": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
						},
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("", "")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstablePartialShares},
		},
		"burstable-with-limits-with-tuned-quota": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstableShares, CpuQuota: &burstableQuota, CpuPeriod: &tunedQuotaPeriod, Memory: &burstableMemory},
		},
		"burstable-with-limits-no-cpu-enforcement-with-tuned-quota": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
						},
					},
				},
			},
			enforceCPULimits: false,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstableShares, CpuQuota: &cpuNoLimit, CpuPeriod: &tunedQuotaPeriod, Memory: &burstableMemory},
		},
		"burstable-partial-limits-with-tuned-quota": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
						},
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("", "")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstablePartialShares},
		},
		"guaranteed": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &guaranteedShares, CpuQuota: &cpuNoLimit, CpuPeriod: &defaultQuotaPeriod, Memory: &guaranteedMemory},
		},
		"guaranteed-no-cpu-enforcement": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: false,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &guaranteedShares, CpuQuota: &cpuNoLimit, CpuPeriod: &defaultQuotaPeriod, Memory: &guaranteedMemory},
		},
		"guaranteed-with-tuned-quota": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &guaranteedShares, CpuQuota: &cpuNoLimit, CpuPeriod: &tunedQuotaPeriod, Memory: &guaranteedMemory},
		},
		"guaranteed-no-cpu-enforcement-with-tuned-quota": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: false,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &guaranteedShares, CpuQuota: &cpuNoLimit, CpuPeriod: &tunedQuotaPeriod, Memory: &guaranteedMemory},
		},
	}

	for testName, testCase := range testCases {

		actual := ResourceConfigForPod(testCase.pod, testCase.enforceCPULimits, testCase.quotaPeriod)

		if !reflect.DeepEqual(actual.CpuPeriod, testCase.expected.CpuPeriod) {
			t.Errorf("unexpected result, test: %v, cpu period not as expected", testName)
		}
		if !reflect.DeepEqual(actual.CpuQuota, testCase.expected.CpuQuota) {
			t.Errorf("unexpected result, test: %v, cpu quota not as expected", testName)
		}
		if !reflect.DeepEqual(actual.CpuShares, testCase.expected.CpuShares) {
			t.Errorf("unexpected result, test: %v, cpu shares not as expected", testName)
		}
		if !reflect.DeepEqual(actual.Memory, testCase.expected.Memory) {
			t.Errorf("unexpected result, test: %v, memory not as expected", testName)
		}
	}
}

func TestResourceConfigForPodWithCustomCPUCFSQuotaPeriod(t *testing.T) {
	defaultQuotaPeriod := uint64(100 * time.Millisecond / time.Microsecond)
	tunedQuotaPeriod := uint64(5 * time.Millisecond / time.Microsecond)
	tunedQuota := int64(1 * time.Millisecond / time.Microsecond)

	utilfeaturetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, pkgfeatures.CPUCFSQuotaPeriod, true)
	defer utilfeaturetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, pkgfeatures.CPUCFSQuotaPeriod, false)

	minShares := uint64(MinShares)
	burstableShares := MilliCPUToShares(100)
	memoryQuantity := resource.MustParse("200Mi")
	burstableMemory := memoryQuantity.Value()
	burstablePartialShares := MilliCPUToShares(200)
	burstableQuota := MilliCPUToQuota(200, int64(defaultQuotaPeriod))
	guaranteedShares := MilliCPUToShares(100)
	guaranteedQuota := MilliCPUToQuota(100, int64(defaultQuotaPeriod))
	//guaranteedTunedQuota := MilliCPUToQuota(100, int64(tunedQuotaPeriod))
	memoryQuantity = resource.MustParse("100Mi")
	cpuNoLimit := int64(-1)
	guaranteedMemory := memoryQuantity.Value()
	defaultCpuShares := int64(12345)
	expectedDefaultCpuShares := uint64(12345)
	expectedDefaultCpuSharesDouble := expectedDefaultCpuShares * 2
	hostConfigCpushares := int64(54321)
	expectedhostConfigCpushares := uint64(54321)
	expectedhostConfigCpusharesTriple := expectedhostConfigCpushares * 3

	annotationWithEmptyAllocSpec, _ := json.Marshal(&sigmak8sapi.AllocSpec{})
	annotationWithEmptyContainers, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{},
	})
	annotationWithEmptyHostConfig, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{},
			{},
		},
	})
	annotationWithDefaultCpuShares, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				HostConfig: sigmak8sapi.HostConfigInfo{
					DefaultCpuShares: &defaultCpuShares,
				},
			},
			{
				HostConfig: sigmak8sapi.HostConfigInfo{
					DefaultCpuShares: &defaultCpuShares,
				},
			},
		},
	})
	annotationWithHostConfigCpushares, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				HostConfig: sigmak8sapi.HostConfigInfo{
					CpuShares: hostConfigCpushares,
				},
			},
			{
				HostConfig: sigmak8sapi.HostConfigInfo{
					CpuShares: hostConfigCpushares * 2,
				},
			},
		},
	})
	annotationWithDefaultAndCpuShares, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				HostConfig: sigmak8sapi.HostConfigInfo{
					DefaultCpuShares: &defaultCpuShares,
					CpuShares:        hostConfigCpushares,
					CpuQuota:         450000,
					CpuPeriod:        150000,
				},
			},
			{
				HostConfig: sigmak8sapi.HostConfigInfo{
					DefaultCpuShares: &defaultCpuShares,
					CpuShares:        0,
					CpuQuota:         450000,
					CpuPeriod:        150000,
				},
			},
		},
	})
	testCases := map[string]struct {
		pod              *v1.Pod
		expected         *ResourceConfig
		enforceCPULimits bool
		quotaPeriod      uint64
	}{
		"besteffort": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("", ""), getResourceList("", "")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &minShares},
		},
		"burstable-no-limits": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("", "")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstableShares},
		},
		"burstable-with-limits": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstableShares, CpuQuota: &burstableQuota, CpuPeriod: &defaultQuotaPeriod, Memory: &burstableMemory},
		},
		"burstable-with-limits-no-cpu-enforcement": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
						},
					},
				},
			},
			enforceCPULimits: false,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstableShares, CpuQuota: &cpuNoLimit, CpuPeriod: &defaultQuotaPeriod, Memory: &burstableMemory},
		},
		"burstable-partial-limits": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
						},
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("", "")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstablePartialShares},
		},
		"burstable-with-limits-with-tuned-quota": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstableShares, CpuQuota: &tunedQuota, CpuPeriod: &tunedQuotaPeriod, Memory: &burstableMemory},
		},
		"burstable-with-limits-no-cpu-enforcement-with-tuned-quota": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
						},
					},
				},
			},
			enforceCPULimits: false,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstableShares, CpuQuota: &cpuNoLimit, CpuPeriod: &tunedQuotaPeriod, Memory: &burstableMemory},
		},
		"burstable-partial-limits-with-tuned-quota": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("200m", "200Mi")),
						},
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("", "")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &burstablePartialShares},
		},
		"guaranteed": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &guaranteedShares, CpuQuota: &cpuNoLimit, CpuPeriod: &defaultQuotaPeriod, Memory: &guaranteedMemory},
		},
		"guaranteed-no-cpu-enforcement": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: false,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &guaranteedShares, CpuQuota: &cpuNoLimit, CpuPeriod: &defaultQuotaPeriod, Memory: &guaranteedMemory},
		},
		"guaranteed-with-tuned-quota": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &guaranteedShares, CpuQuota: &cpuNoLimit, CpuPeriod: &tunedQuotaPeriod, Memory: &guaranteedMemory},
		},
		"guaranteed-no-cpu-enforcement-with-tuned-quota": {
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: false,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &guaranteedShares, CpuQuota: &cpuNoLimit, CpuPeriod: &tunedQuotaPeriod, Memory: &guaranteedMemory},
		},
		"empty-allocSpec": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithEmptyAllocSpec),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &guaranteedShares, CpuQuota: &cpuNoLimit, CpuPeriod: &tunedQuotaPeriod, Memory: &guaranteedMemory},
		},
		"empty-containers": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithEmptyContainers),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &guaranteedShares, CpuQuota: &cpuNoLimit, CpuPeriod: &tunedQuotaPeriod, Memory: &guaranteedMemory},
		},
		"empty-hostConfig": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithEmptyHostConfig),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("100m", "100Mi"), getResourceList("100m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      tunedQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &guaranteedShares, CpuQuota: &cpuNoLimit, CpuPeriod: &tunedQuotaPeriod, Memory: &guaranteedMemory},
		},
		"reset-cpushares-with-hostConfig-defaultCpuShares": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithDefaultCpuShares),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("0m", "50Mi"), getResourceList("40m", "50Mi")),
						},
						{
							Resources: getResourceRequirements(getResourceList("0m", "50Mi"), getResourceList("60m", "50Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &expectedDefaultCpuSharesDouble, CpuQuota: &guaranteedQuota, CpuPeriod: &defaultQuotaPeriod, Memory: &guaranteedMemory},
		},
		"reset-cpushares-with-hostConfig-cpuShares": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithHostConfigCpushares),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: getResourceRequirements(getResourceList("0m", "100Mi"), getResourceList("0m", "100Mi")),
						},
						{
							Resources: getResourceRequirements(getResourceList("0m", "100Mi"), getResourceList("0m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: false,
			quotaPeriod:      tunedQuotaPeriod,
			// when request cpu = 0, CpuQuota=CpuPeriod=0
			expected: &ResourceConfig{CpuShares: &expectedhostConfigCpusharesTriple, Memory: &burstableMemory},
		},
		"reset-cpushares-with-hostConfig-defaultCpuShares-cpuShares": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithDefaultAndCpuShares),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:      "",
							Resources: getResourceRequirements(getResourceList("0m", "100Mi"), getResourceList("100m", "100Mi")),
						},
						{
							Resources: getResourceRequirements(getResourceList("0m", "100Mi"), getResourceList("100m", "100Mi")),
						},
					},
				},
			},
			enforceCPULimits: true,
			quotaPeriod:      defaultQuotaPeriod,
			expected:         &ResourceConfig{CpuShares: &expectedhostConfigCpushares, CpuQuota: &burstableQuota, CpuPeriod: &defaultQuotaPeriod, Memory: &burstableMemory},
		},
	}

	for testName, testCase := range testCases {

		actual := ResourceConfigForPod(testCase.pod, testCase.enforceCPULimits, testCase.quotaPeriod)

		if !reflect.DeepEqual(actual.CpuPeriod, testCase.expected.CpuPeriod) {
			var actVal, expectVal uint64
			if actual.CpuPeriod != nil {
				actVal = *actual.CpuPeriod
			}
			if testCase.expected.CpuPeriod != nil {
				expectVal = *testCase.expected.CpuPeriod
			}
			t.Errorf("unexpected result, test: %v, cpu period: %v not as expected: %v", testName, actVal, expectVal)
		}
		if !reflect.DeepEqual(actual.CpuQuota, testCase.expected.CpuQuota) {
			var actVal, expectVal int64
			if actual.CpuQuota != nil {
				actVal = *actual.CpuQuota
			}
			if testCase.expected.CpuQuota != nil {
				expectVal = *testCase.expected.CpuQuota
			}
			t.Errorf("unexpected result, test: %v, cpu quota: %v not as expected: %v", testName, actVal, expectVal)
		}
		if !reflect.DeepEqual(actual.CpuShares, testCase.expected.CpuShares) {
			t.Errorf("unexpected result, test: %v, cpu shares: %v not as expected: %v", testName, *actual.CpuShares, *testCase.expected.CpuShares)
		}
		if !reflect.DeepEqual(actual.Memory, testCase.expected.Memory) {
			t.Errorf("unexpected result, test: %v, memory: %v not as expected: %v", testName, *actual.Memory, *testCase.expected.Memory)
		}
	}
}

func TestMilliCPUToQuota(t *testing.T) {
	testCases := []struct {
		input  int64
		quota  int64
		period uint64
	}{
		{
			input:  int64(0),
			quota:  int64(0),
			period: uint64(0),
		},
		{
			input:  int64(5),
			quota:  int64(1000),
			period: uint64(100000),
		},
		{
			input:  int64(9),
			quota:  int64(1000),
			period: uint64(100000),
		},
		{
			input:  int64(10),
			quota:  int64(1000),
			period: uint64(100000),
		},
		{
			input:  int64(200),
			quota:  int64(20000),
			period: uint64(100000),
		},
		{
			input:  int64(500),
			quota:  int64(50000),
			period: uint64(100000),
		},
		{
			input:  int64(1000),
			quota:  int64(100000),
			period: uint64(100000),
		},
		{
			input:  int64(1500),
			quota:  int64(150000),
			period: uint64(100000),
		},
	}
	for _, testCase := range testCases {
		quota := MilliCPUToQuota(testCase.input, int64(testCase.period))
		if quota != testCase.quota {
			t.Errorf("Input %v and %v, expected quota %v, but got quota %v", testCase.input, testCase.period, testCase.quota, quota)
		}
	}
}

func TestHugePageLimits(t *testing.T) {
	Mi := int64(1024 * 1024)
	type inputStruct struct {
		key   string
		input string
	}

	testCases := []struct {
		name     string
		inputs   []inputStruct
		expected map[int64]int64
	}{
		{
			name: "no valid hugepages",
			inputs: []inputStruct{
				{
					key:   "2Mi",
					input: "128",
				},
			},
			expected: map[int64]int64{},
		},
		{
			name: "2Mi only",
			inputs: []inputStruct{
				{
					key:   v1.ResourceHugePagesPrefix + "2Mi",
					input: "128",
				},
			},
			expected: map[int64]int64{2 * Mi: 128},
		},
		{
			name: "2Mi and 4Mi",
			inputs: []inputStruct{
				{
					key:   v1.ResourceHugePagesPrefix + "2Mi",
					input: "128",
				},
				{
					key:   v1.ResourceHugePagesPrefix + strconv.FormatInt(2*Mi, 10),
					input: "256",
				},
				{
					key:   v1.ResourceHugePagesPrefix + "4Mi",
					input: "512",
				},
				{
					key:   "4Mi",
					input: "1024",
				},
			},
			expected: map[int64]int64{2 * Mi: 384, 4 * Mi: 512},
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			resourceList := v1.ResourceList{}

			for _, input := range testcase.inputs {
				value, err := resource.ParseQuantity(input.input)
				if err != nil {
					t.Fatalf("error in parsing hugepages, value: %s", input.input)
				} else {
					resourceList[v1.ResourceName(input.key)] = value
				}
			}

			resultValue := HugePageLimits(resourceList)

			if !reflect.DeepEqual(testcase.expected, resultValue) {
				t.Errorf("unexpected result, expected: %v, actual: %v", testcase.expected, resultValue)
			}
		})

	}
}
