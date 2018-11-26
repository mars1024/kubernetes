package throttle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/kubelet/apis/cri"

	"github.com/google/cadvisor/events"
	cv1 "github.com/google/cadvisor/info/v1"
	cv2 "github.com/google/cadvisor/info/v2"
	"github.com/google/gofuzz"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	critest "k8s.io/kubernetes/pkg/kubelet/apis/cri/testing"
	statsapi "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
	cadvisortest "k8s.io/kubernetes/pkg/kubelet/cadvisor/testing"
	statstest "k8s.io/kubernetes/pkg/kubelet/server/stats/testing"
)

var (
	calcPeriod time.Duration
	alcPeriod  = 60 * time.Second
	node       = &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "test-node"}}
	pod        = &v1.Pod{
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{Name: "container-test1", ContainerID: "docker://container-UID-test1"},
				{Name: "container-test2", ContainerID: "docker://container-UID-test2"},
				{Name: "container-test3", ContainerID: "pouch://container-UID-test3"},
			},
		}}
	podStats = []statsapi.PodStats{
		{
			PodRef:      statsapi.PodReference{Name: "test-pod", Namespace: "test-namespace", UID: "UID_test-pod"},
			StartTime:   metav1.NewTime(time.Now()),
			Containers:  getContainerStats(),
			Network:     getNetworkStats(),
			VolumeStats: []statsapi.VolumeStats{*getVolumeStats()},
		},
	}
)

var recorder record.EventRecorder
var containerManager cri.ContainerManager

func TestThrottleStatus(t *testing.T) {
	var vcpuNum = 32
	containerManager = new(critest.FakeRuntimeService)
	var recoverPri ContainerThrottlePriority = new(DefaultThrottlePriority)
	var throttlePri ContainerThrottlePriority = new(DefaultThrottlePriority)
	var defaultInputData = &DefaultThrottleInputData{
		StatsClient:    new(statstest.StatsProvider),
		CadvisorClient: new(cadvisortest.Mock),
	}
	inputData := InputData(defaultInputData)

	throttle := NewThrottleCPUByLoadManager(recorder, containerManager,
		calcPeriod, vcpuNum, inputData, recoverPri, throttlePri)

	//throttle.Start()

	statusDefault := throttle.ThrottleStatus()
	assert.Equal(t, statusDefault, false)

	throttle.Enable()
	statusEnable := throttle.ThrottleStatus()
	assert.Equal(t, statusEnable, true)

	throttle.Disable()
	statusDisable := throttle.ThrottleStatus()
	assert.Equal(t, statusDisable, false)
}

func TestCheckNodeLoad(t *testing.T) {
	//var statsProvider *stats.StatsProvider
	var vcpuNum = 32
	age := 0 * time.Second
	option := cv2.RequestOptions{IdType: "name", Count: 2, Recursive: false, MaxAge: &age}
	mockCadvisorClient := new(cadvisortest.Mock)
	mockCadvisorClient.On("ContainerInfoV2", "/", option).Return(
		map[string]cv2.ContainerInfo{"/": cv2.ContainerInfo{
			Stats: []*cv2.ContainerStats{
				{Load: &cv1.LoadStats{NrSleeping: 0, NrRunning: 1, NrStopped: 0, NrUninterruptible: 0, NrIoWait: 10},
					Cpu: &cv1.CpuStats{LoadAverage: 11}},
				{Load: &cv1.LoadStats{NrSleeping: 0, NrRunning: 1, NrStopped: 0, NrUninterruptible: 0, NrIoWait: 10},
					Cpu: &cv1.CpuStats{LoadAverage: 6}},
			}}}, nil)

	containerManager = new(critest.FakeRuntimeService)
	var recoverPri ContainerThrottlePriority = new(DefaultThrottlePriority)
	var throttlePri ContainerThrottlePriority = new(DefaultThrottlePriority)

	var defaultInputData = &DefaultThrottleInputData{
		StatsClient:    new(statstest.StatsProvider),
		CadvisorClient: mockCadvisorClient,
	}
	mockInputData := InputData(defaultInputData)

	throttle := NewThrottleCPUByLoadManager(recorder, containerManager,
		calcPeriod, vcpuNum, mockInputData, recoverPri, throttlePri)

	loadValue, err := throttle.GetNodeLoad()
	if err != nil {
		assert.Error(t, err, "GetLoad failed.")
		return
	}
	assert.Equal(t, int32(11), loadValue)

	isWarn := throttle.CheckLoadWarn()
	assert.Equal(t, false, isWarn)
}

func TestUpdateResource(t *testing.T) {
	mockStatsProvider := new(statstest.StatsProvider)
	mockStatsProvider.
		On("GetNode").Return(node, nil).
		On("ListPodStats").Return(podStats, nil).
		On("GetPodByName", "test-namespace", "test-pod").Return(pod, true)

	containerManager = new(critest.FakeRuntimeService)
	var vcpuNum = 32
	age := 0 * time.Second
	option := cv2.RequestOptions{IdType: "name", Count: 2, Recursive: false, MaxAge: &age}
	var recoverPri ContainerThrottlePriority = new(DefaultThrottlePriority)
	var throttlePri ContainerThrottlePriority = new(DefaultThrottlePriority)

	cadvisorMock := new(cadvisortest.Mock)
	cadvisorMock.On("ContainerSpec", "container-test1", option).Return(
		map[string]cv2.ContainerSpec{
			"container-test1": {Cpu: cv2.CpuSpec{Quota: 1024}},
			"container-test2": {Cpu: cv2.CpuSpec{Quota: 0}},
			"container-test3": {Cpu: cv2.CpuSpec{Quota: 0}},
		}, nil)

	var defaultInputData = &DefaultThrottleInputData{
		StatsClient:    mockStatsProvider,
		CadvisorClient: cadvisorMock,
	}
	mockInputData := InputData(defaultInputData)

	throttle := NewThrottleCPUByLoadManager(recorder, containerManager,
		calcPeriod, vcpuNum, mockInputData, recoverPri, throttlePri)

	contUID := "container-test-UID"
	err := throttle.ThrottleCgroupToSpecQuota(contUID, 0)
	if err != nil {
		assert.Error(t, err, "throttle cgroup to spec file.")
	} else {
		newSpec, errSpec := throttle.inputData.ContainerSpec("container-test1", option)
		if errSpec != nil {
			assert.Error(t, err, "containerSpec error.")
			return
		}
		if targetSpec, exist := newSpec["container-test1"]; exist {
			assert.Equal(t, uint64(1024), targetSpec.Cpu.Quota)
		} else {
			assert.Fail(t, "not find the container spec. ContainerSpec failed.")
		}
	}
}

func TestThrottleCPUByLoad(t *testing.T) {

	mockStatsProvider := new(statstest.StatsProvider)
	mockStatsProvider.
		On("GetNode").Return(node, nil).
		On("ListPodStats").Return(podStats, nil).
		On("GetPodByName", "test-namespace", "test-pod").Return(pod, true)

	containerManager = new(critest.FakeRuntimeService)

	type testCase struct {
		vcpuNum       int
		containerName string
		cadvisor      *mockCadvisor
		recoverUID    string
	}

	cadvisorTest := &mockCadvisor{
		containerSpecMap: map[string](map[string]cv2.ContainerSpec){
			"container-test1": {"container-test1": cv2.ContainerSpec{Cpu: cv2.CpuSpec{Quota: 1024}}},
			"container-test2": {"container-test2": cv2.ContainerSpec{Cpu: cv2.CpuSpec{Quota: 0}}},
			"container-test3": {"container-test3": cv2.ContainerSpec{}},
		},
		containerInfoMap: map[string](map[string]cv2.ContainerInfo){
			"container-test1": {"container-test1": cv2.ContainerInfo{
				Stats: []*cv2.ContainerStats{
					{Load: &cv1.LoadStats{NrSleeping: 0, NrRunning: 1, NrStopped: 0, NrUninterruptible: 0, NrIoWait: 10},
						Cpu: &cv1.CpuStats{LoadAverage: 11}},
					{Load: &cv1.LoadStats{NrSleeping: 0, NrRunning: 1, NrStopped: 0, NrUninterruptible: 0, NrIoWait: 10},
						Cpu: &cv1.CpuStats{LoadAverage: 6}},
				}}},
			"container-test2": {"container-test2": cv2.ContainerInfo{
				Stats: []*cv2.ContainerStats{
					{Load: &cv1.LoadStats{NrSleeping: 0, NrRunning: 1, NrStopped: 0, NrUninterruptible: 0, NrIoWait: 10},
						Cpu: &cv1.CpuStats{LoadAverage: 11}},
					{Load: &cv1.LoadStats{NrSleeping: 0, NrRunning: 1, NrStopped: 0, NrUninterruptible: 0, NrIoWait: 10},
						Cpu: &cv1.CpuStats{LoadAverage: 6}},
				}}},
			"container-test3": {"container-test3": cv2.ContainerInfo{
				Stats: []*cv2.ContainerStats{
					{Load: &cv1.LoadStats{NrSleeping: 0, NrRunning: 1, NrStopped: 0, NrUninterruptible: 0, NrIoWait: 10},
						Cpu: &cv1.CpuStats{LoadAverage: 11}},
					{Load: &cv1.LoadStats{NrSleeping: 0, NrRunning: 1, NrStopped: 0, NrUninterruptible: 0, NrIoWait: 10},
						Cpu: &cv1.CpuStats{LoadAverage: 6}},
				}}},
		},
	}

	testCases := []testCase{
		{vcpuNum: 32, containerName: "container-test1", cadvisor: cadvisorTest, recoverUID: "container-UID-test1"},
		{vcpuNum: 48, containerName: "container-test2", cadvisor: cadvisorTest, recoverUID: "container-UID-test1"},
	}

	var recoverPri ContainerThrottlePriority = new(DefaultRecoverPriority)
	var throttlePri ContainerThrottlePriority = new(DefaultThrottlePriority)
	for _, testParam := range testCases {
		//t.Logf("testParam.recoverUID:%s\n",testParam.recoverUID )
		var defaultInputData = &DefaultThrottleInputData{
			StatsClient:    mockStatsProvider,
			CadvisorClient: testParam.cadvisor,
		}
		mockInputData := InputData(defaultInputData)

		throttle := NewThrottleCPUByLoadManager(recorder, containerManager,
			calcPeriod, testParam.vcpuNum, mockInputData, recoverPri, throttlePri)

		//throttle.Start()

		containerName, err := throttle.SelectThrottledContainerToRecover(throttle.inputData, throttle.recoverPriority)
		if err != nil {
			//fmt.Printf( "Select container to recover error:%v\n",err)
			t.Logf("find container to recover failed:%v", err)
		} else {
			assert.Equal(t, testParam.recoverUID, containerName)
			//fmt.Printf("select containerUID:%s\n",containerName)
		}

		cont, cpuSpec, err := throttle.SelectTheContainerToThrottle(throttle.inputData, throttle.throttledPriority)
		if err != nil {
			assert.Error(t, err, "Select the container to throttle.")
		} else {
			if cont == "" {
				continue
			}
			assert.Equal(t, "container-UID-test1", cont)
			assert.Equal(t, int64(1024), cpuSpec)
		}
	}
}

// Mock can not support for param. So just mock cadvisor here.
type mockCadvisor struct {
	containerSpecMap map[string](map[string]cv2.ContainerSpec)
	containerInfoMap map[string](map[string]cv2.ContainerInfo)
}

var _ cadvisor.Interface = new(mockCadvisor)

func (c *mockCadvisor) Start() error {
	return nil
}

func (c *mockCadvisor) ContainerInfo(name string, req *cv1.ContainerInfoRequest) (*cv1.ContainerInfo, error) {
	return new(cv1.ContainerInfo), nil
}

func (c *mockCadvisor) ContainerInfoV2(name string, options cv2.RequestOptions) (map[string]cv2.ContainerInfo, error) {
	if confor, ok := c.containerInfoMap[name]; ok {
		return confor, nil
	}
	return nil, nil
}

func (c *mockCadvisor) SubcontainerInfo(name string, req *cv1.ContainerInfoRequest) (map[string]*cv1.ContainerInfo, error) {
	return map[string]*cv1.ContainerInfo{}, nil
}

func (c *mockCadvisor) DockerContainer(name string, req *cv1.ContainerInfoRequest) (cv1.ContainerInfo, error) {
	return cv1.ContainerInfo{}, nil
}

func (c *mockCadvisor) MachineInfo() (*cv1.MachineInfo, error) {
	return nil, nil
}

func (c *mockCadvisor) VersionInfo() (*cv1.VersionInfo, error) {
	return new(cv1.VersionInfo), nil
}

func (c *mockCadvisor) ImagesFsInfo() (cv2.FsInfo, error) {
	return cv2.FsInfo{}, nil
}

func (c *mockCadvisor) RootFsInfo() (cv2.FsInfo, error) {
	return cv2.FsInfo{}, nil
}

func (c *mockCadvisor) WatchEvents(request *events.Request) (*events.EventChannel, error) {
	return new(events.EventChannel), nil
}

func (c *mockCadvisor) GetDirFsInfo(path string) (cv2.FsInfo, error) {
	return cv2.FsInfo{}, nil
}

func (c *mockCadvisor) ContainerSpec(containerName string, options cv2.RequestOptions) (map[string]cv2.ContainerSpec, error) {
	mapResult, ok := c.containerSpecMap[containerName]
	if ok {
		return mapResult, nil
	}
	return map[string]cv2.ContainerSpec{}, nil
}

func getVolumeStats() *statsapi.VolumeStats {
	f := fuzz.New().NilChance(0)
	v := &statsapi.VolumeStats{}
	f.Fuzz(v)
	return v
}

func getNetworkStats() *statsapi.NetworkStats {
	f := fuzz.New().NilChance(0)
	v := &statsapi.NetworkStats{}
	f.Fuzz(v)
	return v
}

func getContainerStats() []statsapi.ContainerStats {
	f := fuzz.New().NilChance(0)
	v := []statsapi.ContainerStats{{Name: "container-test1"}, {Name: "container-test2"}, {Name: "container-test3"}}
	f.Fuzz(&v)
	return v
}

// Client represents the base URL for a cAisor client.
type FakeHTTPClient struct {
}

// NewClient returns a new client with the specified base URL.
func newFakeHTTPClient() (*FakeHTTPClient, error) {
	return &FakeHTTPClient{}, nil
}

// MachineStats returns the JSON machine statistics for this client.
// A non-nil error result indicates a problem with obtainingc
// the JSON machine information data.
func (fakeClient *FakeHTTPClient) MachineStats() ([]cv2.MachineStats, error) {
	var ret []cv2.MachineStats
	ret = append(ret, cv2.MachineStats{
		Load: &cv1.LoadStats{
			NrSleeping:        0,
			NrRunning:         1,
			NrStopped:         0,
			NrUninterruptible: 1,
			NrIoWait:          10,
		},
	})
	return ret, nil
}
