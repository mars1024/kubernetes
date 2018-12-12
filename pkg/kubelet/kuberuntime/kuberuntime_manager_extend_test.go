package kuberuntime

import (
	"testing"
	"time"

	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
)

func TestSyncPodExtension(t *testing.T) {
	fakeRuntime, _, m, err := createTestRuntimeManager()
	assert.NoError(t, err)

	containerNameFirst := "foo1"
	containerNameSecond := "foo2"
	containerNameThird := "foo3"

	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:         "12345678",
			Name:        "foo",
			Namespace:   "new",
			Annotations: make(map[string]string),
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            containerNameFirst,
					Image:           "alpine",
					ImagePullPolicy: v1.PullIfNotPresent,
				},
				{
					Name:            containerNameSecond,
					Image:           "alpine",
					ImagePullPolicy: v1.PullIfNotPresent,
				},
				{
					Name:            containerNameThird,
					Image:           "alpine",
					ImagePullPolicy: v1.PullIfNotPresent,
				},
			},
		},
	}

	containerStateFirst := runtimeapi.ContainerState_CONTAINER_EXITED
	containerStateSecond := runtimeapi.ContainerState_CONTAINER_EXITED
	containerStateThird := runtimeapi.ContainerState_CONTAINER_UNKNOWN

	templates := []containerTemplate{
		{pod: pod, container: &pod.Spec.Containers[0], attempt: 0, createdAt: 2, state: containerStateFirst},
		{pod: pod, container: &pod.Spec.Containers[1], attempt: 0, createdAt: 1, state: containerStateSecond},
		{pod: pod, container: &pod.Spec.Containers[2], attempt: 0, createdAt: 1, state: containerStateThird},
	}
	fakes := makeFakeContainers(t, m, templates)
	fakeRuntime.SetFakeContainers(fakes)

	var containerIDOfFirst, containerIDOfSecond, containerIDOfThird string

	for _, fakeContainer := range fakes {
		if fakeContainer.Metadata.Name == containerNameFirst {
			containerIDOfFirst = fakeContainer.Id
			continue
		}
		if fakeContainer.Metadata.Name == containerNameSecond {
			containerIDOfSecond = fakeContainer.Id
			continue
		}
		if fakeContainer.Metadata.Name == containerNameThird {
			containerIDOfThird = fakeContainer.Id
			continue
		}
	}

	status := &kubecontainer.PodStatus{
		ID:        pod.UID,
		Name:      pod.Name,
		Namespace: pod.Namespace,
		SandboxStatuses: []*runtimeapi.PodSandboxStatus{
			{
				Id:       "sandboxID",
				State:    runtimeapi.PodSandboxState_SANDBOX_READY,
				Metadata: &runtimeapi.PodSandboxMetadata{Name: pod.Name, Namespace: pod.Namespace, Uid: "sandboxuid", Attempt: uint32(0)},
				Network:  &runtimeapi.PodSandboxNetworkStatus{Ip: "10.0.0.1"},
			},
		},
		ContainerStatuses: []*kubecontainer.ContainerStatus{
			// container first state is exited
			{
				ID:    kubecontainer.ContainerID{ID: containerIDOfFirst},
				Name:  containerNameFirst,
				State: toKubeContainerState(containerStateFirst),
			},
			// container second state is running
			{
				ID:    kubecontainer.ContainerID{ID: containerIDOfSecond},
				Name:  containerNameSecond,
				State: toKubeContainerState(containerStateSecond),
			},
			// container third state is unknown
			{
				ID:    kubecontainer.ContainerID{ID: containerIDOfThird},
				Name:  containerNameThird,
				State: toKubeContainerState(containerStateThird),
			},
		},
	}

	containerStateSpec := sigmak8sapi.ContainerStateSpec{
		States: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerState{
			sigmak8sapi.ContainerInfo{Name: containerNameFirst}:  sigmak8sapi.ContainerStateExited,
			sigmak8sapi.ContainerInfo{Name: containerNameSecond}: sigmak8sapi.ContainerStateRunning,
			sigmak8sapi.ContainerInfo{Name: containerNameThird}:  sigmak8sapi.ContainerStateExited,
		},
	}

	podActions := podActions{
		ContainersToStartBecauseDesireState: make(map[kubecontainer.ContainerID]containerOperationInfo),
		ContainersToKillBecauseDesireState:  make(map[kubecontainer.ContainerID]containerOperationInfo),
		ContainersToUpdate:                  make(map[kubecontainer.ContainerID]containerOperationInfo),
		ContainersToUpgrade:                 make(map[kubecontainer.ContainerID]containerOperationInfo),
	}

	podActions.ContainersToStartBecauseDesireState[kubecontainer.ContainerID{ID: containerIDOfSecond}] =
		containerOperationInfo{&pod.Spec.Containers[1], containerNameSecond, "test"}

	podActions.ContainersToKillBecauseDesireState[kubecontainer.ContainerID{ID: containerIDOfThird}] =
		containerOperationInfo{&pod.Spec.Containers[2], containerNameThird, "test"}

	containerStateSpecByte, err := json.Marshal(containerStateSpec)
	assert.NoError(t, err)
	pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = string(containerStateSpecByte)

	containerNameFake := "foo-fake"
	stateStatusBefore := sigmak8sapi.ContainerStateStatus{
		Statuses: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerStatus{
			sigmak8sapi.ContainerInfo{
				Name: containerNameFake,
			}: {
				CurrentState:      sigmak8sapi.ContainerStateRunning,
				CreationTimestamp: time.Now(),
				Success:           true,
				Message:           "fake",
			},
			sigmak8sapi.ContainerInfo{
				Name: containerNameFirst,
			}: {
				CurrentState:      containerStateConvertFromRunTimeAPI(containerStateFirst),
				CreationTimestamp: time.Now(),
				Success:           true,
				Message:           "fake",
			},
			sigmak8sapi.ContainerInfo{
				Name: containerNameSecond,
			}: {
				CurrentState:      containerStateConvertFromRunTimeAPI(containerStateSecond),
				CreationTimestamp: time.Now(),
				Success:           true,
				Message:           "fake",
			},
			sigmak8sapi.ContainerInfo{
				Name: containerNameThird,
			}: {
				CurrentState:      containerStateConvertFromRunTimeAPI(containerStateThird),
				CreationTimestamp: time.Now(),
				Success:           true,
				Message:           "fake",
			},
		},
	}
	stateStatusByte, err := json.Marshal(stateStatusBefore)
	assert.NoError(t, err)
	pod.Annotations[sigmak8sapi.AnnotationPodUpdateStatus] = string(stateStatusByte)

	podSandboxConfig, err := m.generatePodSandboxConfig(pod, 0)
	assert.NoError(t, err)

	result := &kubecontainer.PodSyncResult{
		StateStatus: sigmak8sapi.ContainerStateStatus{
			Statuses: make(map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerStatus),
		},
	}

	m.SyncPodExtension(podSandboxConfig, pod, status, []v1.Secret{}, "", result, podActions, nil)

	assert.True(t, len(fakeRuntime.Containers) == len(templates), "container num should not change")

	assert.True(t, string(toKubeContainerState(fakeRuntime.Containers[containerIDOfFirst].State)) ==
		string(containerStateSpec.States[sigmak8sapi.ContainerInfo{Name: containerNameFirst}]),
		fmt.Sprintf("current status is %s, expect status is %s, not equal",
			string(toKubeContainerState(fakeRuntime.Containers[containerIDOfFirst].State)),
			string(containerStateSpec.States[sigmak8sapi.ContainerInfo{Name: containerNameFirst}])))

	assert.True(t, string(toKubeContainerState(fakeRuntime.Containers[containerIDOfSecond].State)) ==
		string(containerStateSpec.States[sigmak8sapi.ContainerInfo{Name: containerNameSecond}]),
		fmt.Sprintf("current status is %s, expect status is %s, not equal",
			string(toKubeContainerState(fakeRuntime.Containers[containerIDOfSecond].State)),
			string(containerStateSpec.States[sigmak8sapi.ContainerInfo{Name: containerNameSecond}])))

	assert.True(t, string(toKubeContainerState(fakeRuntime.Containers[containerIDOfThird].State)) ==
		string(containerStateSpec.States[sigmak8sapi.ContainerInfo{Name: containerNameThird}]),
		fmt.Sprintf("current status is %s, expect status is %s, not equal",
			string(toKubeContainerState(fakeRuntime.Containers[containerIDOfThird].State)),
			string(containerStateSpec.States[sigmak8sapi.ContainerInfo{Name: containerNameThird}])))

	result.ContainerStateClean(pod)
	result.UpdateStateToPodAnnotation(pod)
	stateStatusJSON, ok := pod.Annotations[sigmak8sapi.AnnotationPodUpdateStatus]
	assert.True(t, ok, "stateStatus annotation not exist")

	var stateStatus sigmak8sapi.ContainerStateStatus

	err = json.Unmarshal([]byte(stateStatusJSON), &stateStatus)
	assert.NoError(t, err)

	assert.True(t, len(stateStatus.Statuses) == 2, "state status should have 3 element")

	for containerInfo, state := range stateStatus.Statuses {
		assert.NotEqual(t, containerNameFake, containerInfo.Name, "containerNameFake should not exist")

		assert.True(t, state.Success,
			fmt.Sprintf("container %s should success, reason is %s", containerInfo.Name, state.Message))

		for statesContainerInfo, containerExpectState := range containerStateSpec.States {
			if strings.EqualFold(containerInfo.Name, statesContainerInfo.Name) {
				assert.True(t, state.CurrentState == containerExpectState,
					fmt.Sprintf("container %s expect status %s not equal current state %s",
						containerInfo.Name, containerExpectState, state.CurrentState))
			}
		}
	}
}

func TestComputeContainerAction(t *testing.T) {
	for _, test := range []struct {
		expectState  sigmak8sapi.ContainerState
		currentState kubecontainer.ContainerState
		action       ContainerAction
	}{
		//expect exited
		{
			expectState:  sigmak8sapi.ContainerStateExited,
			currentState: kubecontainer.ContainerStateRunning,
			action:       ContainerStop,
		},
		{
			expectState:  sigmak8sapi.ContainerStateExited,
			currentState: kubecontainer.ContainerStateExited,
			action:       ContainerDoNothing,
		},
		{
			expectState:  sigmak8sapi.ContainerStateExited,
			currentState: kubecontainer.ContainerStateCreated,
			action:       ContainerStop,
		},
		{
			expectState:  sigmak8sapi.ContainerStateExited,
			currentState: kubecontainer.ContainerStateUnknown,
			action:       ContainerStop,
		},
		//expect running
		{
			expectState:  sigmak8sapi.ContainerStateRunning,
			currentState: kubecontainer.ContainerStateRunning,
			action:       ContainerDoNothing,
		},
		{
			expectState:  sigmak8sapi.ContainerStateRunning,
			currentState: kubecontainer.ContainerStateExited,
			action:       ContainerStart,
		},
		{
			expectState:  sigmak8sapi.ContainerStateRunning,
			currentState: kubecontainer.ContainerStateCreated,
			action:       ContainerStart,
		},
		{
			expectState:  sigmak8sapi.ContainerStateRunning,
			currentState: kubecontainer.ContainerStateUnknown,
			action:       ContainerStart,
		},
		//expect created ,ignore  created，so do nothing
		{
			expectState:  sigmak8sapi.ContainerStateCreated,
			currentState: kubecontainer.ContainerStateRunning,
			action:       ContainerDoNothing,
		},
		{
			expectState:  sigmak8sapi.ContainerStateCreated,
			currentState: kubecontainer.ContainerStateExited,
			action:       ContainerDoNothing,
		},
		{
			expectState:  sigmak8sapi.ContainerStateCreated,
			currentState: kubecontainer.ContainerStateCreated,
			action:       ContainerDoNothing,
		},
		{
			expectState:  sigmak8sapi.ContainerStateCreated,
			currentState: kubecontainer.ContainerStateUnknown,
			action:       ContainerDoNothing,
		},
		//expect unknown, do nothing
		{
			expectState:  sigmak8sapi.ContainerStateUnknown,
			currentState: kubecontainer.ContainerStateRunning,
			action:       ContainerDoNothing,
		},
		{
			expectState:  sigmak8sapi.ContainerStateUnknown,
			currentState: kubecontainer.ContainerStateExited,
			action:       ContainerDoNothing,
		},
		{
			expectState:  sigmak8sapi.ContainerStateUnknown,
			currentState: kubecontainer.ContainerStateCreated,
			action:       ContainerDoNothing,
		},
		{
			expectState:  sigmak8sapi.ContainerStateUnknown,
			currentState: kubecontainer.ContainerStateUnknown,
			action:       ContainerDoNothing,
		},
	} {
		action := computeContainerAction(test.expectState, test.currentState)
		assert.True(t, action == test.action, "expect state is %s, current state is %s, "+
			"action should be %s, but actual action is %s", test.expectState, test.currentState, test.action, action)
	}

}

func TestCompareCurrentStateAndDesiredState(t *testing.T) {
	test := []struct {
		name               string
		containerStateSpec sigmak8sapi.ContainerStateSpec
		containerStatus    *kubecontainer.ContainerStatus
		containerName      string
		wantExist          bool
		wantAction         ContainerAction
	}{
		{
			"containerStatus is nil, so not exist, no action",
			sigmak8sapi.ContainerStateSpec{
				States: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerState{
					sigmak8sapi.ContainerInfo{Name: "foo"}: sigmak8sapi.ContainerStateExited,
				}},
			nil,
			"foo",
			false,
			ContainerDoNothing,
		},
		{
			"container name not exist in desire state, so not exist,no action",
			sigmak8sapi.ContainerStateSpec{
				States: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerState{
					sigmak8sapi.ContainerInfo{Name: "foo"}:  sigmak8sapi.ContainerStateExited,
					sigmak8sapi.ContainerInfo{Name: "foo1"}: sigmak8sapi.ContainerStateExited,
				}},
			&kubecontainer.ContainerStatus{Name: "fool", State: kubecontainer.ContainerStateUnknown},
			"fake",
			false,
			ContainerDoNothing,
		},
		{
			"container expect running, current status is unknown, so exist, and container start",
			sigmak8sapi.ContainerStateSpec{
				States: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerState{
					sigmak8sapi.ContainerInfo{Name: "foo"}: sigmak8sapi.ContainerStateRunning,
				}},
			&kubecontainer.ContainerStatus{Name: "fool", State: kubecontainer.ContainerStateUnknown},
			"foo",
			true,
			ContainerStart,
		},
		{
			"container expect running, current status is running ,so exist,and do nothing",
			sigmak8sapi.ContainerStateSpec{
				States: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerState{
					sigmak8sapi.ContainerInfo{Name: "foo"}: sigmak8sapi.ContainerStateRunning,
				}},
			&kubecontainer.ContainerStatus{Name: "fool", State: kubecontainer.ContainerStateRunning},
			"foo",
			true,
			ContainerDoNothing,
		},
		{
			"container expect exited, current status is running ,so exist,and do stop",
			sigmak8sapi.ContainerStateSpec{
				States: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerState{
					sigmak8sapi.ContainerInfo{Name: "foo"}: sigmak8sapi.ContainerStateExited,
				}},
			&kubecontainer.ContainerStatus{Name: "fool", State: kubecontainer.ContainerStateRunning},
			"foo",
			true,
			ContainerStop,
		},
	}
	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			exist, action := CompareCurrentStateAndDesiredState(tt.containerStateSpec, tt.containerStatus, tt.containerName)
			assert.Equal(t, tt.wantExist, exist)
			assert.Equal(t, tt.wantAction, action)
		})
	}
}

func TestComputePodActions_extension(t *testing.T) {
	_, _, m, err := createTestRuntimeManager()
	require.NoError(t, err)

	containerNameFirst := "foo1"
	containerNameSecond := "foo2"
	containerNameThird := "foo3"

	// Createing a pair reference pod and status for the test cases to refer
	// the specific fields.
	basePod, baseStatus := makeBasePodAndStatus()

	for desc, test := range map[string]struct {
		mutatePodFn    func(*v1.Pod)
		mutateStatusFn func(*kubecontainer.PodStatus)
		containerState sigmak8sapi.ContainerStateSpec
		actions        func(*v1.Pod, *kubecontainer.PodStatus) podActions
	}{
		"start pod sandbox and all containers for a new pod": {
			mutatePodFn: func(pod *v1.Pod) {},
			mutateStatusFn: func(status *kubecontainer.PodStatus) {
				for _, containerStatus := range status.ContainerStatuses {
					if containerStatus.Name == containerNameFirst {
						containerStatus.State = kubecontainer.ContainerStateExited
						continue
					}
					if containerStatus.Name == containerNameSecond {
						containerStatus.State = kubecontainer.ContainerStateRunning
						continue
					}
					if containerStatus.Name == containerNameThird {
						containerStatus.State = kubecontainer.ContainerStateExited
						continue
					}
				}
			},
			containerState: sigmak8sapi.ContainerStateSpec{
				States: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerState{
					sigmak8sapi.ContainerInfo{Name: containerNameFirst}:  sigmak8sapi.ContainerStateRunning,
					sigmak8sapi.ContainerInfo{Name: containerNameSecond}: sigmak8sapi.ContainerStateExited,
					sigmak8sapi.ContainerInfo{Name: containerNameThird}:  sigmak8sapi.ContainerStateExited,
				},
			},
			actions: func(pod *v1.Pod, status *kubecontainer.PodStatus) podActions {
				return podActions{
					KillPod:                             false,
					CreateSandbox:                       false,
					Attempt:                             uint32(0),
					ContainersToStart:                   []int{},
					ContainersToKill:                    make(map[kubecontainer.ContainerID]containerToKillInfo),
					ContainersToStartBecauseDesireState: getOperatorContainerMap(ContainerStart, []string{containerNameFirst}, basePod, baseStatus),
					ContainersToKillBecauseDesireState:  getOperatorContainerMap(ContainerStop, []string{containerNameSecond}, basePod, baseStatus),
					ContainersToUpdate:                  make(map[kubecontainer.ContainerID]containerOperationInfo),
					ContainersToUpgrade:                 make(map[kubecontainer.ContainerID]containerOperationInfo),
				}
			},
		},
	} {
		pod, _ := makeBasePodAndStatus()

		containerStateSpecByte, err := json.Marshal(test.containerState)
		assert.NoError(t, err)
		pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = string(containerStateSpecByte)

		if test.mutatePodFn != nil {
			test.mutatePodFn(pod)
		}
		if test.mutateStatusFn != nil {
			test.mutateStatusFn(baseStatus)
		}
		actions := m.computePodActions(pod, baseStatus)

		actionsFromFunc := test.actions(pod, baseStatus)

		assert.True(t, reflect.DeepEqual(actionsFromFunc.ContainersToKillBecauseDesireState, actions.ContainersToKillBecauseDesireState), desc)
		assert.True(t, reflect.DeepEqual(actionsFromFunc.ContainersToStartBecauseDesireState, actions.ContainersToStartBecauseDesireState), desc)
	}
}

func TestSyncPodExtensionWithUpdatedContainer(t *testing.T) {
	fakeRuntime, _, m, err := createTestRuntimeManager()
	assert.NoError(t, err)

	containerNameFirst := "foo"
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:         "12345678",
			Name:        "foo",
			Namespace:   "new",
			Annotations: make(map[string]string),
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:      containerNameFirst,
					Image:     "image1",
					Resources: getResourceRequirements(getResourceList("8", "8Gi"), getResourceList("8", "8Gi")),
				},
			},
		},
	}

	containerStateFirst := runtimeapi.ContainerState_CONTAINER_RUNNING
	templates := []containerTemplate{
		{pod: pod, container: &pod.Spec.Containers[0], attempt: 0, createdAt: 2, state: containerStateFirst},
	}
	fakes := makeFakeContainers(t, m, templates)
	fakeRuntime.SetFakeContainers(fakes)

	var containerIDOfFirst string
	for _, fakeContainer := range fakes {
		if fakeContainer.Metadata.Name == containerNameFirst {
			containerIDOfFirst = fakeContainer.Id
			continue
		}
	}

	status := &kubecontainer.PodStatus{
		ID:        pod.UID,
		Name:      pod.Name,
		Namespace: pod.Namespace,
		SandboxStatuses: []*runtimeapi.PodSandboxStatus{
			{
				Id:       "sandboxID",
				State:    runtimeapi.PodSandboxState_SANDBOX_READY,
				Metadata: &runtimeapi.PodSandboxMetadata{Name: pod.Name, Namespace: pod.Namespace, Uid: "sandboxuid", Attempt: uint32(0)},
				Network:  &runtimeapi.PodSandboxNetworkStatus{Ip: "10.0.0.1"},
			},
		},
		ContainerStatuses: []*kubecontainer.ContainerStatus{
			// container first state is running
			{
				ID:    kubecontainer.ContainerID{ID: containerIDOfFirst},
				Name:  containerNameFirst,
				State: toKubeContainerState(containerStateFirst),
				Hash:  2517316070,
				Resources: &runtimeapi.LinuxContainerResources{
					CpuShares:          4196,
					MemoryLimitInBytes: 4294967296,
				},
			},
		},
	}

	podActions := podActions{
		ContainersToUpdate:  make(map[kubecontainer.ContainerID]containerOperationInfo),
		ContainersToUpgrade: make(map[kubecontainer.ContainerID]containerOperationInfo),
	}

	podActions.ContainersToUpdate[kubecontainer.ContainerID{ID: containerIDOfFirst}] =
		containerOperationInfo{&pod.Spec.Containers[0], containerNameFirst, "test"}

	podSandboxConfig, err := m.generatePodSandboxConfig(pod, 0)
	assert.NoError(t, err)

	syncResult := &kubecontainer.PodSyncResult{
		StateStatus: sigmak8sapi.ContainerStateStatus{
			Statuses: make(map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerStatus),
		},
	}
	m.SyncPodExtension(podSandboxConfig, pod, status, []v1.Secret{}, "", syncResult, podActions, nil)
	updateContainerSyncResult := syncResult.SyncResults[0]
	assert.True(t, updateContainerSyncResult.Error == nil, "should update success")
}

func TestContainerChanged(t *testing.T) {
	_, _, m, err := createTestRuntimeManager()
	require.NoError(t, err)

	for desc, test := range map[string]struct {
		container       *v1.Container
		containerStatus *kubecontainer.ContainerStatus
		pod             *v1.Pod
		update          bool
	}{
		"container resource requirement changed": {
			container: &v1.Container{
				Name:      "foo",
				Image:     "image1",
				Resources: getResourceRequirements(getResourceList("8", "8Gi"), getResourceList("8", "8Gi")),
			},
			containerStatus: &kubecontainer.ContainerStatus{
				Hash: 3911781334,
				Resources: &runtimeapi.LinuxContainerResources{
					CpuShares:          4196,
					MemoryLimitInBytes: 4294967296,
				},
			},
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "foo-ns"},
				Spec:       v1.PodSpec{Containers: []v1.Container{}},
			},
			update: true,
		},
		"container image changed": {
			container: &v1.Container{
				Name:  "foo",
				Image: "imageNew",
			},
			containerStatus: &kubecontainer.ContainerStatus{
				Hash: 2517316070,
			},
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "foo-ns"},
				Spec:       v1.PodSpec{Containers: []v1.Container{}},
			},
			update: false,
		},
	} {
		_, _, _, needToResize := m.containerChanged(test.container, test.containerStatus, test.pod)
		assert.Equal(t, test.update, needToResize, desc)
	}
}

func getOperatorContainerMap(action ContainerAction, containerList []string, pod *v1.Pod, podStatus *kubecontainer.PodStatus) map[kubecontainer.ContainerID]containerOperationInfo {
	resultMap := make(map[kubecontainer.ContainerID]containerOperationInfo)

	for _, containerStatus := range podStatus.ContainerStatuses {
		for _, containerName := range containerList {
			if containerStatus.Name != containerName {
				continue
			}
			for idx, container := range pod.Spec.Containers {
				if containerName != container.Name {
					continue
				}
				resultMap[containerStatus.ID] = containerOperationInfo{
					container: &pod.Spec.Containers[idx],
					name:      containerName,
					message: func(action ContainerAction, containerName string, containerState kubecontainer.ContainerState) string {
						if action == ContainerStart {
							return fmt.Sprintf("container %s need to start by annotation, because  "+
								"current state is %s not equal to desire state", containerName, containerState)
						} else {
							return fmt.Sprintf("container %s need to stop by annotation, because "+
								" current state is %s not equal to desire state", containerName, containerState)
						}
					}(action, containerName, containerStatus.State),
				}
			}

		}
	}
	return resultMap
}

func getResourceRequirements(requests, limits v1.ResourceList) v1.ResourceRequirements {
	res := v1.ResourceRequirements{}
	res.Requests = requests
	res.Limits = limits
	return res
}

func getResourceList(cpu, memory string) v1.ResourceList {
	res := v1.ResourceList{}
	if cpu != "" {
		res[v1.ResourceCPU] = resource.MustParse(cpu)
	}
	if memory != "" {
		res[v1.ResourceMemory] = resource.MustParse(memory)
	}
	return res
}

func TestValidateMessage(t *testing.T) {
	testCases := []struct {
		name         string
		message      string
		expectResult bool
	}{
		{
			name:         "invalid message",
			message:      "ImagePullBackOff: Back-off pulling image \"test-image\"",
			expectResult: false,
		},
		{
			name:         "valid message",
			message:      "container start failed: PostStartHookError",
			expectResult: true,
		},
	}
	for _, cs := range testCases {
		t.Run(cs.name, func(t *testing.T) {
			result := validateMessage(cs.message)
			assert.Equal(t, cs.expectResult, result)
		})
	}
}
