package sysctl

import (
	"testing"

	"reflect"

	"encoding/json"

	"fmt"

	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubelet/lifecycle"
)

func TestGetCPUsFromAnnotation(t *testing.T) {

	testCase := []struct {
		name         string
		pod          []*v1.Pod
		expectResult map[string]int
	}{
		{
			name: "annotation is nil, count map is nil",
			pod: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
			},
			expectResult: make(map[string]int),
		},
		{
			name: "not contain pod alloc annotation, count map is nil",
			pod: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							"testKey": "testValue",
						},
					},
				},
			},
			expectResult: make(map[string]int),
		},
		{
			name: "contain pod alloc annotation, but value is invalid, count map is nil",
			pod: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							sigmak8sapi.AnnotationPodAllocSpec: "testValue",
						},
					},
				},
			},
			expectResult: make(map[string]int),
		},
		{
			name: "contain pod alloc annotation, but value is invalid, count map is nil",
			pod: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							sigmak8sapi.AnnotationPodAllocSpec: "testValue",
						},
					},
				},
			},
			expectResult: make(map[string]int),
		},
		{
			name: "one pod ,cpu oversold",
			pod: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
						Annotations: map[string]string{
							sigmak8sapi.AnnotationPodAllocSpec: getAllocSpec1(),
						},
					},
				},
			},
			expectResult: map[string]int{
				"1": 1,
				"2": 1,
				"3": 2,
				"4": 1,
				"5": 1,
			},
		},
		{
			name: "two pod ,cpu oversold",
			pod: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test1",
						Annotations: map[string]string{
							sigmak8sapi.AnnotationPodAllocSpec: getAllocSpec1(),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test2",
						Annotations: map[string]string{
							sigmak8sapi.AnnotationPodAllocSpec: getAllocSpec2(),
						},
					},
				},
			},
			expectResult: map[string]int{
				"1": 1,
				"2": 1,
				"3": 2,
				"4": 1,
				"5": 2,
				"6": 1,
				"7": 1,
			},
		},
	}
	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			cpuCountMap := make(map[string]int, len(tc.pod)*4)
			for _, pod := range tc.pod {
				countMap, success := updateCPUMapFromPodAnnotation(pod, cpuCountMap)
				if !success {
					continue
				}
				cpuCountMap = countMap
			}
			assert.True(t, reflect.DeepEqual(cpuCountMap, tc.expectResult))
		})
	}

}
func getAllocSpec1() string {
	allocSpec := sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "test1",
				Resource: sigmak8sapi.ResourceRequirements{
					CPU: sigmak8sapi.CPUSpec{
						CPUSet: &sigmak8sapi.CPUSetSpec{},
					},
				},
			},
			{
				Name: "test2",
				Resource: sigmak8sapi.ResourceRequirements{
					CPU: sigmak8sapi.CPUSpec{
						CPUSet: &sigmak8sapi.CPUSetSpec{},
					},
				},
			},
		},
	}
	allocSpec.Containers[0].Resource.CPU.CPUSet.CPUIDs = []int{1, 2, 3}
	allocSpec.Containers[1].Resource.CPU.CPUSet.CPUIDs = []int{3, 4, 5}
	allocSpecByte, _ := json.Marshal(allocSpec)
	return string(allocSpecByte)
}

func getAllocSpec2() string {
	allocSpec := sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "test1",
				Resource: sigmak8sapi.ResourceRequirements{
					CPU: sigmak8sapi.CPUSpec{
						CPUSet: &sigmak8sapi.CPUSetSpec{},
					},
				},
			},
		},
	}
	allocSpec.Containers[0].Resource.CPU.CPUSet.CPUIDs = []int{5, 6, 7}
	allocSpecByte, _ := json.Marshal(allocSpec)
	return string(allocSpecByte)
}

func getAllocSpec3() string {
	allocSpec := sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "test1",
				Resource: sigmak8sapi.ResourceRequirements{
					CPU: sigmak8sapi.CPUSpec{
						CPUSet: &sigmak8sapi.CPUSetSpec{},
					},
				},
			},
		},
	}
	allocSpec.Containers[0].Resource.CPU.CPUSet.CPUIDs = []int{7, 8, 9}
	allocSpecByte, _ := json.Marshal(allocSpec)
	return string(allocSpecByte)
}

func getAllocSpec4() string {
	allocSpec := sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "test1",
				Resource: sigmak8sapi.ResourceRequirements{
					CPU: sigmak8sapi.CPUSpec{
						CPUSet: &sigmak8sapi.CPUSetSpec{},
					},
				},
			},
		},
	}
	allocSpec.Containers[0].Resource.CPU.CPUSet.CPUIDs = []int{10, 11, 12}
	allocSpecByte, _ := json.Marshal(allocSpec)
	return string(allocSpecByte)
}

func getAllocSpec5() string {
	allocSpec := sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "test1",
				Resource: sigmak8sapi.ResourceRequirements{
					CPU: sigmak8sapi.CPUSpec{},
				},
			},
		},
	}
	allocSpecByte, _ := json.Marshal(allocSpec)
	return string(allocSpecByte)
}

func TestOversoldAdmitHandler_Admit(t *testing.T) {
	testCase := []struct {
		name        string
		getNodeFunc GetNodeFunc
		attrs       *lifecycle.PodAdmitAttributes
		admit       bool
	}{
		{
			name:        "pod is 2.0 to 3.1, should admit",
			getNodeFunc: getNodeFunc,
			attrs: &lifecycle.PodAdmitAttributes{
				Pod: func() *v1.Pod {
					pod := getFakePod()
					pod.Annotations[sigmak8sapi.AnnotationRebuildContainerInfo] = " testValue"
					return pod
				}(),
				OtherPods: getFakeOtherPods(),
			},
			admit: true,
		},
		{
			name: "get node info err, should admit",
			getNodeFunc: func() (*v1.Node, error) {
				return nil, fmt.Errorf("testCase")
			},
			attrs: &lifecycle.PodAdmitAttributes{
				Pod:       getFakePod(),
				OtherPods: getFakeOtherPods(),
			},
			admit: true,
		},
		{
			name:        "node have no label, should admit",
			getNodeFunc: getNodeFunc,
			attrs: &lifecycle.PodAdmitAttributes{
				Pod:       getFakePod(),
				OtherPods: getFakeOtherPods(),
			},
			admit: true,
		},
		{
			name: "node have no cpu over quota label, should admit",
			getNodeFunc: func() (*v1.Node, error) {
				return &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"testLabelKey": "testLabelValue",
						},
					},
				}, nil
			},
			attrs: &lifecycle.PodAdmitAttributes{
				Pod:       getFakePod(),
				OtherPods: getFakeOtherPods(),
			},
			admit: true,
		},
		{
			name: "node have invalid cpu over quota label, should admit",
			getNodeFunc: func() (*v1.Node, error) {
				return &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							sigmak8sapi.LabelCPUOverQuota: "testLabelValue",
						},
					},
				}, nil
			},
			attrs: &lifecycle.PodAdmitAttributes{
				Pod:       getFakePod(),
				OtherPods: getFakeOtherPods(),
			},
			admit: true,
		},
		{
			name: "node cpu over quota value >1, should admit",
			getNodeFunc: func() (*v1.Node, error) {
				return &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							sigmak8sapi.LabelCPUOverQuota: "2",
						},
					},
				}, nil
			},
			attrs: &lifecycle.PodAdmitAttributes{
				Pod:       getFakePod(),
				OtherPods: getFakeOtherPods(),
			},
			admit: true,
		},
		{
			name: "node cpu over sold, should not admit",
			getNodeFunc: func() (*v1.Node, error) {
				return &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							sigmak8sapi.LabelCPUOverQuota: "1",
						},
					},
				}, nil
			},
			attrs: &lifecycle.PodAdmitAttributes{
				Pod:       getFakePod(),
				OtherPods: getFakeOtherPods(),
			},
			admit: false,
		},
		{
			name: "node cpu not over sold, should admit",
			getNodeFunc: func() (*v1.Node, error) {
				return &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							sigmak8sapi.LabelCPUOverQuota: "1.0",
						},
					},
				}, nil
			},
			attrs: &lifecycle.PodAdmitAttributes{
				Pod:       getFakePod2(),
				OtherPods: getFakeOtherPods(),
			},
			admit: true,
		},
		{
			name: "pod have no cpuID, should admit",
			getNodeFunc: func() (*v1.Node, error) {
				return &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							sigmak8sapi.LabelCPUOverQuota: "1.0",
						},
					},
				}, nil
			},
			attrs: &lifecycle.PodAdmitAttributes{
				Pod:       getFakePod3(),
				OtherPods: getFakeOtherPods(),
			},
			admit: true,
		},
		{
			name: "pod have no pod alloc annnotation, should admit",
			getNodeFunc: func() (*v1.Node, error) {
				return &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							sigmak8sapi.LabelCPUOverQuota: "1.0",
						},
					},
				}, nil
			},
			attrs: &lifecycle.PodAdmitAttributes{
				Pod:       &v1.Pod{},
				OtherPods: getFakeOtherPods(),
			},
			admit: true,
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			oversold, _ := NewOversoldAdmitHandler(tc.getNodeFunc)
			result := oversold.Admit(tc.attrs)
			assert.Equal(t, result.Admit, tc.admit)
		})
	}

}

func getNodeFunc() (*v1.Node, error) {
	return &v1.Node{}, nil
}

func getFakeOtherPods() []*v1.Pod {
	return []*v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					sigmak8sapi.AnnotationPodAllocSpec: getAllocSpec2(),
				},
			},
		},
	}
}

func getFakePod() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: getAllocSpec3(),
			},
		},
	}
}

func getFakePod2() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: getAllocSpec4(),
			},
		},
	}
}

func getFakePod3() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: getAllocSpec5(),
			},
		},
	}
}
