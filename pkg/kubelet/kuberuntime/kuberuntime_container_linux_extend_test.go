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
