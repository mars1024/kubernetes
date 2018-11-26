package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/json"
)

func TestPodSyncResult_ContainerStateClean(t *testing.T) {
	result := PodSyncResult{
		StateStatus: sigmak8sapi.ContainerStateStatus{
			Statuses: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerStatus{
				sigmak8sapi.ContainerInfo{Name: "container-a"}: {Success: true},
				sigmak8sapi.ContainerInfo{Name: "container-c"}: {Success: false},
			},
		},
	}
	pod := &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "container-a",
				},
				{
					Name: "container-b",
				},
			},
		},
	}
	expectState := sigmak8sapi.ContainerStateStatus{
		Statuses: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerStatus{
			sigmak8sapi.ContainerInfo{Name: "container-a"}: {Success: true},
		},
	}
	result.ContainerStateClean(pod)
	assert.Equal(t, result.StateStatus, expectState)
}

func TestPodSyncResult_UpdateStateToPodAnnotation(t *testing.T) {
	result := PodSyncResult{
		StateStatus: sigmak8sapi.ContainerStateStatus{
			Statuses: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerStatus{
				sigmak8sapi.ContainerInfo{Name: "container-a"}: {Success: true},
				sigmak8sapi.ContainerInfo{Name: "container-c"}: {Success: false},
			},
		},
	}
	testCases := []struct {
		name string
		pod  *v1.Pod
	}{
		{
			name: "annotation is nil",
			pod:  &v1.Pod{},
		},
		{
			name: "annotation is not nil",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: make(map[string]string),
				},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result.UpdateStateToPodAnnotation(testCase.pod)
			jsonValue := testCase.pod.Annotations[sigmak8sapi.AnnotationPodUpdateStatus]
			actualValue := &sigmak8sapi.ContainerStateStatus{}
			err := json.Unmarshal([]byte(jsonValue), actualValue)
			assert.NoError(t, err)
			assert.Equal(t, result.StateStatus, *actualValue)
		})
	}
}
