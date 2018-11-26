package violatecheck

import (
	"testing"
	"time"

	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	statsapi "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/autopilot/slo"
)

type TestCase struct {
	annotations map[string]string
	runStatus   bool
}

type MockViolateNeedData struct {
	mock.Mock
	slo.MockRuntimeSLOPredictValue
}

var _ ViolateNeedData = new(MockViolateNeedData)

func (d *MockViolateNeedData) GetPodByName(podnamespace string, podname string) (*v1.Pod, bool) {
	fmt.Println("just trace the GetPodByName action...")
	args := d.Called(podnamespace, podname)
	return args.Get(0).(*v1.Pod), args.Bool(1)
}

// ListPodStats returns the stats of all the containers managed by pods.
func (d *MockViolateNeedData) ListPodStats() ([]statsapi.PodStats, error) {
	fmt.Println("just trace the ListPodStats action...")
	args := d.Called()
	return args.Get(0).([]statsapi.PodStats), args.Error(1)
}

var (
	calcPeriod time.Duration
	alcPeriod  = 60 * time.Second
	pod        = &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Annotations: map[string]string{
			//AnnotationQPS:"100",
			AnnotationCPI: "2.5",
			AnnotationRT:  "50",
		}}, // map[string]string{"node.alpha.kubernetes.io/ttl": "60"},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{Name: "container-test1", ContainerID: "container-test1-UID"},
				{Name: "container-test2", ContainerID: "container-test2-UID"},
				{Name: "container-test3", ContainerID: "container-test3-UID"},
			},
		}}

	podStats = []statsapi.PodStats{
		{
			PodRef: statsapi.PodReference{
				Namespace: "test-namespace",
				Name:      "test-pod",
				UID:       "test-pod-UID"},
			Containers: []statsapi.ContainerStats{
				{Name: "container-test1"},
				{Name: "container-test2"},
			},
			StartTime: metav1.NewTime(time.Now()),
		},
	}
)

func TestDefaultViolateCheckManager_ContainerViolateState(t *testing.T) {
	inputData := new(MockViolateNeedData)
	recorder := new(record.FakeRecorder)
	calcPeriod := time.Second * 10
	inputData.On("ListPodStats").Return(podStats, nil).
		On("GetPodByName", "test-namespace", "test-pod").Return(pod, true)

	manager := NewDefaultViolateCheckManager(inputData, recorder, calcPeriod)

	testCases := []TestCase{
		{annotations: map[string]string{api.AnnotationAutopilot + ViolateKey: "{\"switchOn\":true}"},
			runStatus: true,
		},
		{annotations: map[string]string{api.AnnotationAutopilot + ViolateKey: "{\"switchOn\":false}"},
			runStatus: false,
		},
	}

	for _, testcase := range testCases {
		ids, err := manager.ViolateCheck(testcase.annotations)
		if err != nil {
			assert.Failf(t, "CheckFailed", "msg:%v", err)
		} else {
			assert.Equal(t, manager.runStatus, testcase.runStatus)

			t.Logf("ids:%v", ids)
		}
	}

}
