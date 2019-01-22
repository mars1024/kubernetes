package sigma

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetStatusFromAnnotation(t *testing.T) {
	annotations := map[string]string{sigmak8sapi.AnnotationPodUpdateStatus: "{\"statuses\":{\"test\":{\"creationTimestamp\":\"2018-07-18T18:28:28.039600678+08:00\",\"finishTimestamp\":\"2018-07-18T18:28:28.403536805+08:00\",\"currentState\":\"created\",\"lastState\":\"created\",\"message\":\"exit status 1\",\"specHash\":\"cf455531\"}}}"}
	testPodWithStatuses := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "bar",
			Namespace:   "default",
			Annotations: annotations,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "foo",
					Image: "busybox",
				},
			},
		},
	}
	// Get right container's status
	containerName := "test"
	status := GetStatusFromAnnotation(testPodWithStatuses, containerName)
	if status == nil {
		t.Errorf("Failed to get update-satus of %v from pod: %v", containerName, testPodWithStatuses)
	}
	// Get wrong container's status
	containerName = "test1"
	status = GetStatusFromAnnotation(testPodWithStatuses, containerName)
	if status != nil {
		t.Errorf("Failed to get update-satus of %v from pod: %v", containerName, testPodWithStatuses)
	}

	testPodWithoutStatuses := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bar",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "foo",
					Image: "busybox",
				},
			},
		},
	}
	containerName = "test"
	status = GetStatusFromAnnotation(testPodWithoutStatuses, containerName)
	if status != nil {
		t.Errorf("Failed to get update-satus of %v from pod: %v", containerName, testPodWithStatuses)
	}
}

func TestGetSpecHashFromAnnotation(t *testing.T) {
	hashStr := "12345678"
	testPodWithSpecHash := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "bar",
			Namespace:   "default",
			Annotations: map[string]string{sigmak8sapi.AnnotationPodSpecHash: hashStr},
		},
	}
	specHashStr, exists := GetSpecHashFromAnnotation(testPodWithSpecHash)
	if !exists {
		t.Errorf("Failed to get signature from testPodWithSignature")
	}
	if specHashStr != hashStr {
		t.Errorf("Failed to get signature from testPodWithSignature")
	}
	testPodWithoutSpecHash := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bar",
			Namespace: "default",
		},
	}
	_, exists = GetSpecHashFromAnnotation(testPodWithoutSpecHash)
	if exists {
		t.Errorf("Failed to get signature from testPodWithSignature")
	}
}

func TestGetContainerDesiredStateFromAnnotation(t *testing.T) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:         "12345678",
			Name:        "foo",
			Namespace:   "new",
			Annotations: make(map[string]string),
		},
	}

	haveContainerStateAnnotation, _, _ := GetContainerDesiredStateFromAnnotation(nil)
	assert.False(t, haveContainerStateAnnotation, "should not have annotation, because pod is nil")

	haveContainerStateAnnotation, _, _ = GetContainerDesiredStateFromAnnotation(pod)
	assert.Falsef(t, haveContainerStateAnnotation, "should not have annotation, because pod have no anotation: %s",
		sigmak8sapi.AnnotationContainerStateSpec)

	pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = "fakeData"
	haveContainerStateAnnotation, _, _ = GetContainerDesiredStateFromAnnotation(pod)
	assert.False(t, haveContainerStateAnnotation, "should not have annotation, because pod annotation is invalid data")

	stateSpec := sigmak8sapi.ContainerStateSpec{
		States: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerState{
			sigmak8sapi.ContainerInfo{Name: "foo1"}: sigmak8sapi.ContainerStateRunning,
		}}
	containerStateSpecByte, err := json.Marshal(stateSpec)
	assert.NoError(t, err)
	pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = string(containerStateSpecByte)
	haveContainerStateAnnotation, containerDesiredState, _ := GetContainerDesiredStateFromAnnotation(pod)

	assert.True(t, haveContainerStateAnnotation, "should have valid annotation")
	assert.True(t, reflect.DeepEqual(stateSpec, containerDesiredState))

	pod.Annotations = nil
	haveContainerStateAnnotation, _, _ = GetContainerDesiredStateFromAnnotation(pod)
	assert.False(t, haveContainerStateAnnotation, "should not have annotation, because pod annotation is nil")
}

func TestGetRebuildContainerIDFromPodAnnotation(t *testing.T) {
	containerID := "7090b74e9300"
	rebuildContainerInfo := sigmak8sapi.RebuildContainerInfo{
		ContainerID: containerID,
		DiskQuotaID: "123",
		AliAdminUID: "567",
	}
	rebuildContainerInfoBytes, err := json.Marshal(rebuildContainerInfo)
	if err != nil {
		t.Errorf("Failed to marshal RebuildContainerInfo %v", rebuildContainerInfo)
	}
	for caseName, testCase := range map[string]struct {
		pod        *v1.Pod
		expectedID string
	}{
		"pod has empty annotation": {
			pod: &v1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:         "12345678",
					Name:        "foo",
					Namespace:   "new",
					Annotations: map[string]string{},
				},
			},
			expectedID: "",
		},
		"pod has right annotation": {
			pod: &v1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:         "12345678",
					Name:        "foo",
					Namespace:   "new",
					Annotations: map[string]string{sigmak8sapi.AnnotationRebuildContainerInfo: string(rebuildContainerInfoBytes)},
				},
			},
			expectedID: containerID,
		},
		"pod has wrong annotation": {
			pod: &v1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:         "12345678",
					Name:        "foo",
					Namespace:   "new",
					Annotations: map[string]string{"test-key": string(rebuildContainerInfoBytes)},
				},
			},
			expectedID: "",
		},
	} {
		buildContainerID := GetRebuildContainerIDFromPodAnnotation(testCase.pod)
		if buildContainerID != testCase.expectedID {
			t.Errorf("Case %s: expect buildContainerID %s bug got %s", caseName, buildContainerID, testCase.expectedID)
		}

	}
}

func TestGetContainerRebuildInfoFromAnnotation(t *testing.T) {
	testCase := []struct {
		name                     string
		annotationValue          *sigmak8sapi.RebuildContainerInfo
		expectError              bool
		withWrongAnnotationValue bool
		withWrongAnnotationKey   bool
	}{
		{
			name:            "annotation is nil, should error",
			expectError:     true,
			annotationValue: nil,
		},
		{
			name:        "every thing is ok",
			expectError: false,
			annotationValue: &sigmak8sapi.RebuildContainerInfo{
				ContainerID: "123-test",
			},
		},
		{
			name:        "with wrong annotation value, so exist error",
			expectError: true,
			annotationValue: &sigmak8sapi.RebuildContainerInfo{
				ContainerID: "123-test",
			},
			withWrongAnnotationValue: true,
		},
		{
			name:        "with wrong annotation key, so exist error",
			expectError: true,
			annotationValue: &sigmak8sapi.RebuildContainerInfo{
				ContainerID: "123-test",
			},
			withWrongAnnotationKey: true,
		},
	}

	for _, cs := range testCase {
		t.Run(cs.name, func(t *testing.T) {
			pod := &v1.Pod{}
			if cs.annotationValue != nil {
				annotationValue, err := json.Marshal(cs.annotationValue)
				assert.NoError(t, err)

				pod.Annotations = map[string]string{
					sigmak8sapi.AnnotationRebuildContainerInfo: string(annotationValue),
				}
				if cs.withWrongAnnotationValue {
					pod.Annotations[sigmak8sapi.AnnotationRebuildContainerInfo] = "test"
				}
				if cs.withWrongAnnotationKey {
					pod.Annotations["testKey"] = "testValue"
					delete(pod.Annotations, sigmak8sapi.AnnotationRebuildContainerInfo)
				}
			}
			rebuildContainerInfo, err := GetContainerRebuildInfoFromAnnotation(pod)
			if cs.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, rebuildContainerInfo, cs.annotationValue)
		})
	}
}

func generateDefaultAllocSpec(containerName string) *sigmak8sapi.AllocSpec {
	return &sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: containerName,
				HostConfig: sigmak8sapi.HostConfigInfo{
					Ulimits: []sigmak8sapi.Ulimit{
						{
							Name: "nofile",
							Soft: 1024,
							Hard: 8196,
						},
					},
				},
			},
		},
	}
}

func TestGetUlimitsFromAnnotation(t *testing.T) {
	testCase := []struct {
		name            string
		annotationValue *sigmak8sapi.AllocSpec
		expectUlimits   []sigmak8sapi.Ulimit
	}{
		{
			name:            "annotation is nil",
			annotationValue: nil,
			expectUlimits:   []sigmak8sapi.Ulimit{},
		},
		{
			name:            "container not exists",
			annotationValue: generateDefaultAllocSpec("foo"),
			expectUlimits:   []sigmak8sapi.Ulimit{},
		},
		{
			name: "no host config info",
			annotationValue: &sigmak8sapi.AllocSpec{
				Containers: []sigmak8sapi.Container{
					{
						Name: "bar",
					},
				},
			},
			expectUlimits: nil,
		},
		{
			name:            "everything is ok",
			annotationValue: generateDefaultAllocSpec("bar"),
			expectUlimits:   []sigmak8sapi.Ulimit{{Name: "nofile", Soft: 1024, Hard: 8196}},
		},
	}

	for _, cs := range testCase {
		t.Run(cs.name, func(t *testing.T) {
			pod := &v1.Pod{}
			if cs.annotationValue != nil {
				annotation, err := json.Marshal(cs.annotationValue)
				assert.NoError(t, err)

				pod.Annotations = map[string]string{
					sigmak8sapi.AnnotationPodAllocSpec: string(annotation),
				}
			}
			container := &v1.Container{Name: "bar"}
			ulimits := GetUlimitsFromAnnotation(container, pod)
			assert.Equal(t, cs.expectUlimits, ulimits)
		})
	}
}

func TestNetworkStatusFromAnnotation(t *testing.T) {
	networkStatus := sigmak8sapi.NetworkStatus{
		VlanID:              "700",
		NetworkPrefixLength: 22,
		Gateway:             "100.81.187.247",
		Ip:                  "100.81.187.21",
	}
	networkStatusStr, err := json.Marshal(networkStatus)
	if err != nil {
		t.Errorf("Failed to marshal %v", networkStatus)
	}
	testPodWithNetworkStatus := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "bar",
			Namespace:   "default",
			Annotations: map[string]string{sigmak8sapi.AnnotationPodNetworkStats: string(networkStatusStr)},
		},
	}
	testPodWithoutNetworkStatus := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bar",
			Namespace: "default",
		},
	}

	for desc, test := range map[string]struct {
		pod                 *v1.Pod
		expectNetworkStatus *sigmak8sapi.NetworkStatus
	}{
		"pod has network status annotation": {
			pod:                 testPodWithNetworkStatus,
			expectNetworkStatus: &networkStatus,
		},
		"pod has no network status annotation": {
			pod:                 testPodWithoutNetworkStatus,
			expectNetworkStatus: nil,
		},
	} {
		networkStatus := GetNetworkStatusFromAnnotation(test.pod)
		assert.Equal(t, test.expectNetworkStatus, networkStatus, desc)
	}
}

func TestGetTimeoutSecondsFromPodAnnotation(t *testing.T) {
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:         "12345678",
			Name:        "foo",
			Namespace:   "new",
			Annotations: map[string]string{},
		},
	}

	for caseName, testCase := range map[string]struct {
		timeoutConfigKey   string
		timeoutConfigValue string
		containerName      string
	}{
		"postStartHook timeout": {
			timeoutConfigKey:   sigmak8sapi.PostStartHookTimeoutSeconds,
			timeoutConfigValue: "10",
			containerName:      "container-test",
		},
		"imagepull timeout": {
			timeoutConfigKey:   sigmak8sapi.ImagePullTimeoutSeconds,
			timeoutConfigValue: "20",
			containerName:      "container-test",
		},
	} {
		containerInfo := sigmak8sapi.ContainerInfo{
			Name: testCase.containerName,
		}
		containerConfig := sigmak8sapi.ContainerConfig{
			testCase.timeoutConfigKey: testCase.timeoutConfigValue,
		}
		extraConfig := sigmak8sapi.ContainerExtraConfig{
			ContainerConfigs: map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerConfig{
				containerInfo: containerConfig,
			},
		}
		extraConfigStr, err := json.Marshal(extraConfig)
		if err != nil {
			t.Errorf("Case %s: Failed marshal extra config into string", caseName)
		}
		pod.Annotations[sigmak8sapi.AnnotationContainerExtraConfig] = string(extraConfigStr)
		timeout := GetTimeoutSecondsFromPodAnnotation(pod, testCase.containerName, testCase.timeoutConfigKey)
		if strconv.Itoa(timeout) != testCase.timeoutConfigValue {
			t.Errorf("Case %s: Failed to get timeout value from pod annotation", caseName)
		}

	}
}

func TestGetHostConfigFromAnnotation(t *testing.T) {
	// container1
	containerName1 := "test1"
	hostConfig1 := sigmak8sapi.HostConfigInfo{
		CgroupParent:     "/kubepods",
		MemorySwap:       12345678,
		MemorySwappiness: 20,
		PidsLimit:        100,
		CPUBvtWarpNs:     2,
		MemoryWmarkRatio: 0.2,
	}

	// container2
	containerName2 := "test2"
	hostConfig2 := sigmak8sapi.HostConfigInfo{
		CgroupParent:     "/kubepods/kubepods",
		MemorySwap:       123456789,
		MemorySwappiness: 200,
		PidsLimit:        1000,
		CPUBvtWarpNs:     20,
		MemoryWmarkRatio: 0.222,
	}

	// cotnainer not exists
	containerName3 := "test3"

	allocSpec := &sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			sigmak8sapi.Container{
				Name:       containerName1,
				HostConfig: hostConfig1,
			},
			sigmak8sapi.Container{
				Name:       containerName2,
				HostConfig: hostConfig2,
			},
		},
	}
	allocSpecBytes, err := json.Marshal(allocSpec)
	if err != nil {
		t.Fatalf("Failed to marshal allocSpec: %v, error: %v", allocSpec, err)
	}
	tests := []struct {
		message          string
		pod              *v1.Pod
		containerName    string
		expectHostConfig *sigmak8sapi.HostConfigInfo
	}{
		{
			message: "get hostconfig from pod for container1",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Namespace:   "foo",
					Annotations: map[string]string{sigmak8sapi.AnnotationPodAllocSpec: string(allocSpecBytes)},
				},
			},
			containerName:    containerName1,
			expectHostConfig: &hostConfig1,
		},
		{
			message: "get another hostconfig from pod for container2",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Namespace:   "foo",
					Annotations: map[string]string{sigmak8sapi.AnnotationPodAllocSpec: string(allocSpecBytes)},
				},
			},
			containerName:    containerName2,
			expectHostConfig: &hostConfig2,
		},
		{
			message: "get another hostconfig from pod for unexisted container",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Namespace:   "foo",
					Annotations: map[string]string{sigmak8sapi.AnnotationPodAllocSpec: string(allocSpecBytes)},
				},
			},
			containerName:    containerName3,
			expectHostConfig: nil,
		},
	}

	for _, test := range tests {
		t.Logf("start to test case: %s", test.message)
		hostConfig := GetHostConfigFromAnnotation(test.pod, test.containerName)
		assert.Equal(t, test.expectHostConfig, hostConfig)
	}
}

func TestGetAllocResourceFromAnnotation(t *testing.T) {
	// container1
	containerName1 := "test1"

	allocResource1 := sigmak8sapi.ResourceRequirements{
		CPU: sigmak8sapi.CPUSpec{
			BindingStrategy: sigmak8sapi.CPUBindStrategyAllCPUs,
			CPUSet: &sigmak8sapi.CPUSetSpec{
				SpreadStrategy: sigmak8sapi.SpreadStrategySpread,
				CPUIDs:         []int{1, 2},
			},
		},
	}

	// container2
	containerName2 := "test2"
	allocResource2 := sigmak8sapi.ResourceRequirements{}

	allocSpec := &sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			sigmak8sapi.Container{
				Name:     containerName1,
				Resource: allocResource1,
			},
			sigmak8sapi.Container{
				Name:     containerName2,
				Resource: allocResource2,
			},
		},
	}
	allocSpecBytes, err := json.Marshal(allocSpec)
	if err != nil {
		t.Fatalf("Failed to marshal allocSpec: %v, error: %v", allocSpec, err)
	}

	tests := []struct {
		message             string
		pod                 *v1.Pod
		containerName       string
		expectAllocResource *sigmak8sapi.ResourceRequirements
	}{
		{
			message: "get alloc resource from pod for container1: resource is defined",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Namespace:   "foo",
					Annotations: map[string]string{sigmak8sapi.AnnotationPodAllocSpec: string(allocSpecBytes)},
				},
			},
			containerName:       containerName1,
			expectAllocResource: &allocResource1,
		},
		{
			message: "get alloc resource from pod for container2: resource is nil",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Namespace:   "foo",
					Annotations: map[string]string{sigmak8sapi.AnnotationPodAllocSpec: string(allocSpecBytes)},
				},
			},
			containerName:       containerName2,
			expectAllocResource: &allocResource2,
		},
		{
			message: "get another hostconfig from pod for unexisted container",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Namespace:   "foo",
					Annotations: map[string]string{sigmak8sapi.AnnotationPodAllocSpec: string(allocSpecBytes)},
				},
			},
			containerName:       "ContainerNotExists",
			expectAllocResource: nil,
		},
		{
			message: "alloc spec is not defined",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "foo",
				},
			},
			containerName:       "containerName2",
			expectAllocResource: nil,
		},
	}

	for _, test := range tests {
		t.Logf("start to test case: %s", test.message)
		allocResource := GetAllocResourceFromAnnotation(test.pod, test.containerName)
		assert.Equal(t, test.expectAllocResource, allocResource)
	}
}

func TestGetDanglingPodsFromNodeAnnotation(t *testing.T) {
	danglingPods := []sigmak8sapi.DanglingPod{
		sigmak8sapi.DanglingPod{
			Name:      "pod1",
			Namespace: "namespace1",
		},
		sigmak8sapi.DanglingPod{
			Name:      "pod2",
			Namespace: "namespace2",
		},
		sigmak8sapi.DanglingPod{
			Name:      "pod2",
			Namespace: "namespace2",
		},
	}

	danglingPodsBytes, err := json.Marshal(danglingPods)
	if err != nil {
		t.Errorf("Failed to marshal danglingPods: %v, error: %v", danglingPods, err)
	}

	for caseName, testCase := range map[string]struct {
		node                 *v1.Node
		expectErrorOccurs    bool
		expectedDanglingPods []sigmak8sapi.DanglingPod
	}{
		"node is nil": {
			node:                 nil,
			expectErrorOccurs:    true,
			expectedDanglingPods: []sigmak8sapi.DanglingPod{},
		},
		"node has empty annotation": {
			node: &v1.Node{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Node",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:         "12345678",
					Name:        "foo",
					Namespace:   "new",
					Annotations: map[string]string{},
				},
			},
			expectErrorOccurs:    false,
			expectedDanglingPods: []sigmak8sapi.DanglingPod{},
		},
		"node has valid annotation": {
			node: &v1.Node{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Node",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "foo",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationDanglingPods: string(danglingPodsBytes),
					},
				},
			},
			expectErrorOccurs:    false,
			expectedDanglingPods: danglingPods,
		},
		"node has invalid annotation": {
			node: &v1.Node{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Node",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "foo",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationDanglingPods: "invalid string",
					},
				},
			},
			expectErrorOccurs:    true,
			expectedDanglingPods: []sigmak8sapi.DanglingPod{},
		},
		"node has other annotation": {
			node: &v1.Node{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Node",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "foo",
					Namespace: "new",
					Annotations: map[string]string{
						"otherAnnotation": "value",
					},
				},
			},
			expectErrorOccurs:    false,
			expectedDanglingPods: []sigmak8sapi.DanglingPod{},
		},
	} {
		actualDanglingPods, err := GetDanglingPodsFromNodeAnnotation(testCase.node)
		if testCase.expectErrorOccurs && err == nil {
			t.Errorf("Case %s: expect error occurs but not", caseName)
		}
		if !reflect.DeepEqual(actualDanglingPods, testCase.expectedDanglingPods) {
			t.Errorf("Case %s: expect danglingPods %v bug got %v", caseName, testCase.expectedDanglingPods, actualDanglingPods)
		}

	}
}

func TestGetPodAnnotationByName(t *testing.T) {
	tests := []struct {
		name            string
		pod             *v1.Pod
		annotationKey   string
		annotationValue string
	}{
		{
			name:            "pod is nil",
			pod:             nil,
			annotationValue: "",
		},
		{
			name:            "pod annotation nil",
			pod:             &v1.Pod{},
			annotationValue: "",
		},
		{
			name: "pod annotation key not exist",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"test-key": "test-value",
					},
				},
			},
			annotationKey:   "test123",
			annotationValue: "",
		},
		{
			name: "everything is ok",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"test-key": "test-value",
					},
				},
			},
			annotationKey:   "test-key",
			annotationValue: "test-value",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := GetPodAnnotationByName(test.pod, test.annotationKey)
			assert.Equal(t, test.annotationValue, value)
		})
	}
}

func TestGetNetPriorityFromAnnotation(t *testing.T) {
	netPriority := 2
	tests := []struct {
		message           string
		pod               *v1.Pod
		expectNetPriority int
	}{
		{
			message: "get NetPriority from pod with net priority definition",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Namespace:   "foo",
					Annotations: map[string]string{sigmak8sapi.AnnotationNetPriority: strconv.Itoa(netPriority)},
				},
			},
			expectNetPriority: netPriority,
		},
		{
			message: "get NetPriority from pod without net priority definition",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Namespace:   "foo",
					Annotations: map[string]string{},
				},
			},
			expectNetPriority: 0,
		},
	}

	for _, test := range tests {
		t.Logf("start to test case: %s", test.message)
		netPriority := GetNetPriorityFromAnnotation(test.pod)
		assert.Equal(t, test.expectNetPriority, netPriority)
	}
}
