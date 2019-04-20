package allocators

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"
)

func TestContainerCPUCount(t *testing.T) {
	pod := makePod("100m", "1700m")
	ret := ContainerCPUCount(&pod.Spec.Containers[0])
	if ret != 2 {
		t.Errorf("the cpu count should be 2, acutal %d", ret)
	}
}

func TestContainerCPUCount_Guaranteed(t *testing.T) {
	pod := makePod("3", "3")
	ret := ContainerCPUCount(&pod.Spec.Containers[0])
	if ret != 3 {
		t.Errorf("the cpu count should be 3, acutal %d", ret)
	}
}

func makePod(cpuRequest, cpuLimit string) *v1.Pod {
	return &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceName(v1.ResourceCPU):    resource.MustParse(cpuRequest),
							v1.ResourceName(v1.ResourceMemory): resource.MustParse("1G"),
						},
						Limits: v1.ResourceList{
							v1.ResourceName(v1.ResourceCPU):    resource.MustParse(cpuLimit),
							v1.ResourceName(v1.ResourceMemory): resource.MustParse("1G"),
						},
					},
				},
			},
		},
	}
}
