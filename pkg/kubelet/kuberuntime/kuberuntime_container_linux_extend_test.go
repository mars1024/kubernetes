// +build linux

package kuberuntime

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func TestAdjustResourcesByAnnotation(t *testing.T) {
	containerName := "foo"
	testCase := []struct {
		name            string
		annotationValue *sigmak8sapi.AllocSpec
		resources       *runtimeapi.LinuxContainerResources
		milliCPU        int64
		expectCpuPeriod int64
		expectCpuQuota  int64
	}{
		{
			name:            "original cpu period is 0",
			annotationValue: nil,
			resources:       &runtimeapi.LinuxContainerResources{CpuPeriod: 0},
			milliCPU:        0,
			expectCpuPeriod: 0,
			expectCpuQuota:  0,
		},
		{
			name:            "annotation is nil",
			annotationValue: nil,
			resources:       &runtimeapi.LinuxContainerResources{CpuPeriod: 100 * 1000, CpuQuota: 150 * 1000},
			milliCPU:        2000,
			expectCpuPeriod: 100 * 1000,
			expectCpuQuota:  150 * 1000,
		},
		{
			name:            "everything is ok",
			annotationValue: makeAllocSpecWithCpuPeriod(containerName, 150*1000),
			resources:       &runtimeapi.LinuxContainerResources{CpuPeriod: 100 * 1000},
			milliCPU:        2000,
			expectCpuPeriod: 150 * 1000,
			expectCpuQuota:  300 * 1000,
		},
	}

	for _, cs := range testCase {
		t.Run(cs.name, func(t *testing.T) {
			pod := &v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{Name: containerName}},
				},
			}
			if cs.annotationValue != nil {
				annotation, err := json.Marshal(cs.annotationValue)
				assert.NoError(t, err)

				pod.Annotations = map[string]string{
					sigmak8sapi.AnnotationPodAllocSpec: string(annotation),
				}
			}
			AdjustResourcesByAnnotation(pod, containerName, cs.resources, cs.milliCPU)
			assert.Equal(t, cs.resources.CpuPeriod, cs.expectCpuPeriod)
			assert.Equal(t, cs.resources.CpuQuota, cs.expectCpuQuota)
		})
	}
}

func makeAllocSpecWithCpuPeriod(containerName string, cpuPeriod int64) *sigmak8sapi.AllocSpec {
	return &sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: containerName,
				HostConfig: sigmak8sapi.HostConfigInfo{
					CpuPeriod: cpuPeriod,
				},
			},
		},
	}
}

func TestApplyDiskQuota(t *testing.T) {
	for desc, testCase := range map[string]struct {
		pod               *v1.Pod
		expectedDiskQuota map[string]string
	}{
		"pod has diskquota and the quota mode is not set": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "foo",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceEphemeralStorage: resource.MustParse("5Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceEphemeralStorage: resource.MustParse("5Gi"),
								},
							},
						},
					},
				},
			},
			expectedDiskQuota: map[string]string{".*": "5g"},
		},
		"pod has diskquota and the quota mode is '.*'": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: `{"containers":[{"name":"foo","hostConfig":{"diskQuotaMode":".*"}}]}`,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "foo",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceEphemeralStorage: resource.MustParse("5Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceEphemeralStorage: resource.MustParse("5Gi"),
								},
							},
						},
					},
				},
			},
			expectedDiskQuota: map[string]string{".*": "5g"},
		},
		"pod has diskquota and the quota mode is '/'": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: `{"containers":[{"name":"foo","hostConfig":{"diskQuotaMode":"/"}}]}`,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "foo",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceEphemeralStorage: resource.MustParse("5Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceEphemeralStorage: resource.MustParse("5Gi"),
								},
							},
						},
					},
				},
			},
			expectedDiskQuota: map[string]string{"/": "5g"},
		},
		"pod has diskquota and the quota mode is invalid": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: `{"containers":[{"name":"foo","hostConfig":{"diskQuotaMode":"invalid"}}]}`,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "foo",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceEphemeralStorage: resource.MustParse("5Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceEphemeralStorage: resource.MustParse("5Gi"),
								},
							},
						},
					},
				},
			},
			expectedDiskQuota: map[string]string{".*": "5g"},
		},
		"pod has no ResourceEphemeralStorage defined and the quota mode is '.*'": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: `containers":[{"name":"foo","hostConfig":{"diskQuotaMode":".*"}}]`,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "foo",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
						},
					},
				},
			},
			expectedDiskQuota: map[string]string{},
		},
		"pod has no ResourceEphemeralStorage defined and the quota mode is '/'": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: `containers":[{"name":"foo","hostConfig":{"diskQuotaMode":"/"}}]`,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "foo",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
						},
					},
				},
			},
			expectedDiskQuota: map[string]string{},
		},
	} {
		containerConfig := &runtimeapi.ContainerConfig{
			Linux: &runtimeapi.LinuxContainerConfig{
				Resources: &runtimeapi.LinuxContainerResources{},
			},
		}
		container := &testCase.pod.Spec.Containers[0]
		applyDiskQuota(testCase.pod, container, containerConfig.Linux)

		if len(containerConfig.Linux.Resources.DiskQuota) == 0 && len(testCase.expectedDiskQuota) == 0 {
			continue
		}

		if !reflect.DeepEqual(containerConfig.Linux.Resources.DiskQuota, testCase.expectedDiskQuota) {
			t.Errorf("test case: %v, expected DiskQuota %s, but got: %s", desc, testCase.expectedDiskQuota, containerConfig.Linux.Resources.DiskQuota)
		}
	}
}

func TestApplyExtendContainerResource(t *testing.T) {
	defaultCpuShares := int64(44444)
	hostConfigCpuShares := int64(12345)
	hostConfigCpuQuota := int64(450000)
	hostConfigCpuPeriod := int64(150000)
	hostConfigOomScoreAdj := int64(1000)
	hostConfigMemorySwappiness := int64(10000000)
	hostConfigMemorySwap := int64(20000000)
	hostConfigPidsLimit := uint16(65535)
	hostConfigCpuBvtWarpNs := -1

	annotationWithNilAllocSpect, _ := json.Marshal(nil)
	annotationWithEmptyAllocSpec, _ := json.Marshal(&sigmak8sapi.AllocSpec{})
	annotationWithEmptyContainers, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{},
	})
	annotationWithEmptyHostConfig1, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name:       "container-1",
				HostConfig: sigmak8sapi.HostConfigInfo{},
			},
		},
	})
	annotationWithEmptyHostConfig, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "container-1",
			},
		},
	})
	annotationWithCpuShares, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "container-1",
				HostConfig: sigmak8sapi.HostConfigInfo{
					CpuShares: hostConfigCpuShares,
				},
			},
			{
				Name: "container-2",
				HostConfig: sigmak8sapi.HostConfigInfo{
					CpuShares: 54321,
				},
			},
		},
	})
	annotationWithCpuQuota, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "container-1",
				HostConfig: sigmak8sapi.HostConfigInfo{
					CpuQuota: hostConfigCpuQuota,
				},
			},
			{
				Name: "container-2",
				HostConfig: sigmak8sapi.HostConfigInfo{
					CpuQuota: 5000,
				},
			},
		},
	})
	annotationWithCpuPeriod, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "container-1",
				HostConfig: sigmak8sapi.HostConfigInfo{
					CpuPeriod: hostConfigCpuPeriod,
				},
			},
			{
				Name: "container-2",
				HostConfig: sigmak8sapi.HostConfigInfo{
					CpuQuota: 2000,
				},
			},
		},
	})
	annotationWithAll, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name:       "container-2",
				HostConfig: sigmak8sapi.HostConfigInfo{},
			},
			{
				Name: "container-1",
				HostConfig: sigmak8sapi.HostConfigInfo{
					DefaultCpuShares: &defaultCpuShares,
					CpuShares:        hostConfigCpuShares,
					CpuQuota:         hostConfigCpuQuota,
					CpuPeriod:        hostConfigCpuPeriod,
					CPUBvtWarpNs:     hostConfigCpuBvtWarpNs,
					OomScoreAdj:      hostConfigOomScoreAdj,
					MemorySwappiness: hostConfigMemorySwappiness,
					MemorySwap:       hostConfigMemorySwap,
					PidsLimit:        hostConfigPidsLimit,
				},
			},
		},
	})
	annotationWithContainerNotExist, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "container-2",
				HostConfig: sigmak8sapi.HostConfigInfo{
					DefaultCpuShares: &defaultCpuShares,
					CpuShares:        hostConfigCpuShares,
					CpuQuota:         hostConfigCpuQuota,
					CpuPeriod:        hostConfigCpuPeriod,
					CPUBvtWarpNs:     hostConfigCpuBvtWarpNs,
					OomScoreAdj:      hostConfigOomScoreAdj,
					MemorySwappiness: hostConfigMemorySwappiness,
					MemorySwap:       hostConfigMemorySwap,
					PidsLimit:        hostConfigPidsLimit,
				},
			},
		},
	})
	annotationWithInvalidValue, _ := json.Marshal(&sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "container-1",
				HostConfig: sigmak8sapi.HostConfigInfo{
					CpuShares:    -1,
					CpuQuota:     -2,
					CpuPeriod:    -3,
					OomScoreAdj:  0,
					CPUBvtWarpNs: 0,
				},
			},
		},
	})

	for desc, testCase := range map[string]struct {
		pod                             *v1.Pod
		expectedContainerResourceConfig runtimeapi.LinuxContainerResources
	}{
		"pod with nil allocSpec": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithNilAllocSpect),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "container-1",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
			expectedContainerResourceConfig: runtimeapi.LinuxContainerResources{
				CpuShares:    5 * 1024,
				CpuQuota:     700000,
				CpuPeriod:    122333,
				CpuBvtWarpNs: 2,
				OomScoreAdj:  500,
			},
		},
		"pod with empty allocSpec": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithEmptyAllocSpec),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "container-1",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
			expectedContainerResourceConfig: runtimeapi.LinuxContainerResources{
				CpuShares:    5 * 1024,
				CpuQuota:     700000,
				CpuPeriod:    122333,
				CpuBvtWarpNs: 2,
				OomScoreAdj:  500,
			},
		},
		"pod with empty containers": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithEmptyContainers),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "container-1",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
			expectedContainerResourceConfig: runtimeapi.LinuxContainerResources{
				CpuShares:    5 * 1024,
				CpuQuota:     700000,
				CpuPeriod:    122333,
				CpuBvtWarpNs: 2,
				OomScoreAdj:  500,
			},
		},
		"pod with empty hostconfig1": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithEmptyHostConfig1),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "container-1",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
			expectedContainerResourceConfig: runtimeapi.LinuxContainerResources{
				CpuShares:        5 * 1024,
				CpuQuota:         700000,
				CpuPeriod:        122333,
				CpuBvtWarpNs:     2,
				OomScoreAdj:      500,
				MemorySwappiness: &runtimeapi.Int64Value{int64(0)},
			},
		},
		"pod with empty hostconfig2": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithEmptyHostConfig),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "container-1",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
			expectedContainerResourceConfig: runtimeapi.LinuxContainerResources{
				CpuShares:        5 * 1024,
				CpuQuota:         700000,
				CpuPeriod:        122333,
				CpuBvtWarpNs:     2,
				OomScoreAdj:      500,
				MemorySwappiness: &runtimeapi.Int64Value{int64(0)},
			},
		},
		"pod with hostConfig cpushares": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithCpuShares),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "container-1",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
									v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
								},
								Limits: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
									v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
								},
							},
						},
					},
				},
			},
			expectedContainerResourceConfig: runtimeapi.LinuxContainerResources{
				CpuShares:        hostConfigCpuShares,
				CpuQuota:         700000,
				CpuPeriod:        122333,
				CpuBvtWarpNs:     2,
				OomScoreAdj:      500,
				MemorySwappiness: &runtimeapi.Int64Value{int64(0)},
			},
		},
		"pod with hostConfig cpuQuota": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithCpuQuota),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "container-1",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
			expectedContainerResourceConfig: runtimeapi.LinuxContainerResources{
				CpuShares:        5 * 1024,
				CpuQuota:         hostConfigCpuQuota,
				CpuPeriod:        122333,
				CpuBvtWarpNs:     2,
				OomScoreAdj:      500,
				MemorySwappiness: &runtimeapi.Int64Value{int64(0)},
			},
		},
		"pod with hostConfig cpuPeriod": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithCpuPeriod),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "container-1",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
			expectedContainerResourceConfig: runtimeapi.LinuxContainerResources{
				CpuShares:        5 * 1024,
				CpuQuota:         700000,
				CpuPeriod:        hostConfigCpuPeriod,
				CpuBvtWarpNs:     2,
				OomScoreAdj:      500,
				MemorySwappiness: &runtimeapi.Int64Value{int64(0)},
			},
		},
		"pod with hostConfig cpuPeriod2": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithCpuPeriod),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "container-1",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
			expectedContainerResourceConfig: runtimeapi.LinuxContainerResources{
				CpuShares:        5 * 1024,
				CpuQuota:         hostConfigCpuPeriod * 100 / 1000,
				CpuPeriod:        hostConfigCpuPeriod,
				CpuBvtWarpNs:     2,
				OomScoreAdj:      500,
				MemorySwappiness: &runtimeapi.Int64Value{int64(0)},
			},
		},
		"pod with hostConfig all": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithAll),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "container-1",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
			expectedContainerResourceConfig: runtimeapi.LinuxContainerResources{
				CpuShares:        hostConfigCpuShares,
				CpuQuota:         hostConfigCpuQuota,
				CpuPeriod:        hostConfigCpuPeriod,
				CpuBvtWarpNs:     int64(hostConfigCpuBvtWarpNs),
				OomScoreAdj:      hostConfigOomScoreAdj,
				MemorySwap:       hostConfigMemorySwap,
				MemorySwappiness: &runtimeapi.Int64Value{hostConfigMemorySwappiness},
				PidsLimit:        int64(hostConfigPidsLimit),
			},
		},
		"pod with hostConfig container name not exist": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithContainerNotExist),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "container-1",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
			expectedContainerResourceConfig: runtimeapi.LinuxContainerResources{
				CpuShares:    5 * 1024,
				CpuQuota:     700000,
				CpuPeriod:    122333,
				CpuBvtWarpNs: 2,
				OomScoreAdj:  500,
			},
		},
		"pod with hostConfig invalid value": {
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "bar",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(annotationWithInvalidValue),
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "container-1",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
			expectedContainerResourceConfig: runtimeapi.LinuxContainerResources{
				CpuShares:        5 * 1024,
				CpuQuota:         700000,
				CpuPeriod:        122333,
				CpuBvtWarpNs:     2,
				OomScoreAdj:      500,
				MemorySwappiness: &runtimeapi.Int64Value{int64(0)},
			},
		},
	} {
		containerConfig := &runtimeapi.ContainerConfig{
			Linux: &runtimeapi.LinuxContainerConfig{
				Resources: &runtimeapi.LinuxContainerResources{
					CpuShares:    5 * 1024,
					CpuQuota:     700000,
					CpuPeriod:    122333,
					CpuBvtWarpNs: 2,
					OomScoreAdj:  500,
				},
			},
		}
		container := &testCase.pod.Spec.Containers[0]
		applyExtendContainerResource(testCase.pod, container, containerConfig.Linux, true)

		if reflect.DeepEqual(containerConfig.Linux.Resources.MemorySwappiness, testCase.expectedContainerResourceConfig.MemorySwappiness) {
			containerConfig.Linux.Resources.MemorySwappiness = nil
			testCase.expectedContainerResourceConfig.MemorySwappiness = nil
		} else {
			t.Errorf("test case: %v, expected MemorySwapiness %#v, but got: %#v",
				desc, testCase.expectedContainerResourceConfig.MemorySwappiness, containerConfig.Linux.Resources.MemorySwappiness)
		}
		if !reflect.DeepEqual(*containerConfig.Linux.Resources, testCase.expectedContainerResourceConfig) {
			t.Errorf("test case: %v, expected Resources %#v, but got: %#v",
				desc, testCase.expectedContainerResourceConfig, containerConfig.Linux.Resources)
		}
	}
}
