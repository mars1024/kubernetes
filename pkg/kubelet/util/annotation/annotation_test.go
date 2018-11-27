package annotation

import (
	"encoding/json"
	"strconv"
	"testing"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
