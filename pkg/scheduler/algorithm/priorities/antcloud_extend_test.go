package priorities

import (
	cafelabels "gitlab.alipay-inc.com/antstack/cafe-k8s-api/pkg"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
	"testing"
)

func TestPodResourceBestFitPriorityMap(t *testing.T) {
	pod := makeSoftPod("4", "4Gi")
	nodeInfo := makeNodeInfo()
	//nodeInfo.AddPod(pod)
	result, err := PodResourceBestFitPriorityMap(pod, nil, nodeInfo)
	t.Logf("log result: %#v", result)
	if err != nil {
		t.Errorf("failed to calculate the priority: %s", err.Error())
	}
	if result.Score != 10 {
		t.Errorf("expected score is 10, actual %d", result.Score)
	}

	pod1 := makeSoftPod("2", "4Gi")
	result1, err1 := PodResourceBestFitPriorityMap(pod1, nil, nodeInfo)
	t.Logf("log result: %#v", result1)
	if err1 != nil {
		t.Errorf("failed to calculate the priority: %s", err1.Error())
	}
	if result1.Score != 7 {
		t.Errorf("expected score is 7, actual %d", result1.Score)
	}

}

func makeSoftPod(cpu, memory string) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				cafelabels.MonotypeLabelKey: cafelabels.MonotypeLabelValueSoft,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse(cpu),
							v1.ResourceMemory: resource.MustParse(memory),
						},
					},
				},
			},
		},
	}
	return pod
}

func makeNodeInfo() *schedulercache.NodeInfo {
	mem := resource.MustParse("4Gi")
	node := makeNode("testNode", 4000, mem.Value())
	nodeInfo := schedulercache.NewNodeInfo()
	_ = nodeInfo.SetNode(node)
	return nodeInfo
}
