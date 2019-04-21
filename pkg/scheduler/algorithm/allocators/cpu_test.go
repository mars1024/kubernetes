package allocators

import (
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/json"
	"testing"
)

func TestNewCPUAllocator_ShareCPUSet_First(t *testing.T) {

	nodeInfo, _ := makeNodeInfo()
	al := NewCPUAllocator(nodeInfo)
	newPod := makePodWithAlloc("testPod", "2000m", "2000m")
	result := al.Allocate(newPod)
	if len(result) <= 0 {
		t.Error("failed to allocate cpu for containers")
	}
	t.Logf("allocated %v for pod %s", result, newPod.Name)
	// Add this manual to mock real case
	by := GenAllocSpecAnnotation(newPod, result)
	t.Logf("pod will update with annotation %s", string(by))
	newPod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(by)
	nodeInfo.AddPod(newPod)
	alNew, _ := al.(*CPUAllocator)
	alNew.resetPool(nodeInfo)
	// Mock end
	newPod1 := makePodWithAlloc("testPod2", "8000m", "8000m")
	result = alNew.Allocate(newPod1)
	if len(result) <= 0 {
		t.Error("failed to allocate cpu for containers")
	}
	t.Logf("allocated %v for pod %s", result, newPod.Name)
}

func TestNewCPUAllocator_ShareCPUSet(t *testing.T) {
	pod := makePod("1000m", "2")
	pod2 := makePod("2000m", "2")
	nodeInfo, _ := makeNodeInfo(pod, pod2)
	al := NewCPUAllocator(nodeInfo)
	newPod := makePodWithAlloc("testPod", "1000m", "1000m")
	result := al.Allocate(newPod)
	if len(result) <= 0 {
		t.Error("failed to allocate cpu for containers")
	}
	t.Logf("allocated %v for pod %s", result, newPod.Name)
	// Add this manual to mock real case
	by := GenAllocSpecAnnotation(newPod, result)
	t.Logf("pod will update with annotation %s", string(by))
	newPod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(by)
	nodeInfo.AddPod(newPod)
	alNew, _ := al.(*CPUAllocator)
	alNew.resetPool(nodeInfo)
	// Mock end
	newPod1 := makePodWithAlloc("testPod2", "8000m", "8000m")
	result = alNew.Allocate(newPod1)
	if len(result) <= 0 {
		t.Error("failed to allocate cpu for containers")
	}
	t.Logf("allocated %v for pod %s", result, newPod.Name)
}

func TestNewCPUAllocator_Reallocate_First(t *testing.T) {
	//TODO(yuzhi.wx) add tests here
}

func TestNewCPUAllocator_Exclusive(t *testing.T) {
	pod := makePod("1000m", "2")
	pod2 := makePod("2000m", "2")
	nodeInfo, _ := makeNodeInfo(pod, pod2)
	al := NewCPUAllocator(nodeInfo)
	newPod := makePodWithAlloc("testPod", "2000m", "2000m")
	result := al.Allocate(newPod)
	if len(result) <= 0 {
		t.Error("failed to allocate cpu for containers")
	}
	t.Logf("allocated %v for pod %s", result, newPod.Name)
	// Add this manual to mock real case
	by := GenAllocSpecAnnotation(newPod, result)
	t.Logf("pod will update with annotation %s", string(by))
	newPod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(by)
	nodeInfo.AddPod(newPod)
	alNew, _ := al.(*CPUAllocator)
	alNew.resetPool(nodeInfo)
	// Mock end
	newPod1 := makePodWithAlloc("testPod2", "4000m", "4000m")
	setExclusivePod(newPod1)
	result = alNew.Allocate(newPod1)

	if !(len(result) > 0 && !result["testContainer"].Contains(0) && result["testContainer"].Size() == 4) {
		t.Errorf("failed to allocate cpu for containers, got %s", result)
		t.FailNow()
	}
	t.Logf("allocated %v for pod %s", result, newPod.Name)
}

func makePodWithAlloc(name, cpuRequest, cpuLimit string) *v1.Pod {
	alloc := sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{Name: "testContainer",
				Resource: sigmak8sapi.ResourceRequirements{
					CPU: sigmak8sapi.CPUSpec{
						CPUSet: &sigmak8sapi.CPUSetSpec{},
					},
				}},
		},
	}
	allocStr, err := json.Marshal(alloc)
	if err != nil {
		panic(err)
	}
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: string(allocStr),
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "testContainer",
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

func setExclusivePod(pod *v1.Pod) {
	if len(pod.Labels) == 0 {
		pod.Labels = make(map[string]string, 0)
	}
	pod.Labels[ExclusiveCPU] = "1"

}
