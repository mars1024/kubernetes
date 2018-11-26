// +build linux

package kuberuntime

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
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
