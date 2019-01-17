/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kuberuntime

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
)

func TestScriptExecuter(t *testing.T) {
	executer := ScriptExecuter{"date"}
	_, err := executer.ExecCommand("")
	if err != nil {
		t.Errorf("Failed to test ScriptExecuter:%v", err)
	}
}

type FakeExecuter struct {
	ActionMap map[string]int
}

func (e *FakeExecuter) ExecCommand(s string) (string, error) {
	action := strings.Split(s, " ")[0]
	if code, exists := e.ActionMap[action]; exists {
		if code == 0 {
			return "Excute successfully", nil
		}
		return "Error occurs", fmt.Errorf("get non-0 code")
	}

	return "error occurs", fmt.Errorf("Unkown action for FakeExecuter")
}

// Test UserinfoBackup
func TestUserinfoBackup(t *testing.T) {
	fakePod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bar",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "foo",
					Image:           "busybox",
					ImagePullPolicy: v1.PullIfNotPresent,
					Command:         []string{"testCommand"},
					WorkingDir:      "testWorkingDir",
				},
			},
		},
	}
	fakeContainer := &fakePod.Spec.Containers[0]
	for scriptName, actionMap := range map[string]map[string]int{
		"scriptNoError":      map[string]int{"restore": 0, "backup": 0, "check": 0, "delete": 0},
		"scriptRestoreError": map[string]int{"restore": 1, "backup": 0, "check": 0, "delete": 0},
		"scriptBackupError":  map[string]int{"restore": 0, "backup": 1, "check": 0, "delete": 0},
		"scriptCheckError":   map[string]int{"restore": 0, "backup": 0, "check": 1, "delete": 0},
		"scriptDeleteError":  map[string]int{"restore": 0, "backup": 0, "check": 0, "delete": 1},
	} {
		userInfoBackup := UserinfoBackup{&FakeExecuter{actionMap}}
		_, errorRestore := userInfoBackup.RestoreUserinfo(fakePod, fakeContainer, "container-id")
		_, errorBackup := userInfoBackup.BackupUserinfo(fakePod, fakeContainer, "container-id")
		_, errorCheck := userInfoBackup.CheckUserinfoExists(fakePod, fakeContainer)
		_, errorDelete := userInfoBackup.DeleteUserinfo(fakePod, fakeContainer)

		if scriptName == "scriptNoError" && (errorRestore != nil || errorBackup != nil ||
			errorCheck != nil || errorDelete != nil) {
			t.Errorf("Failed to test UserInfoBackup with %q", actionMap)
		}
		if scriptName == "scriptRestoreError" && (errorRestore == nil || errorBackup != nil ||
			errorCheck != nil || errorDelete != nil) {
			t.Errorf("Failed to test UserInfoBackup with %q", actionMap)
		}
		if scriptName == "scriptBackupError" && (errorRestore != nil || errorBackup == nil ||
			errorCheck != nil || errorDelete != nil) {
			t.Errorf("Failed to test UserInfoBackup with %q", actionMap)
		}
		if scriptName == "scriptCheckError" && (errorRestore != nil || errorBackup != nil ||
			errorCheck == nil || errorDelete != nil) {
			t.Errorf("Failed to test UserInfoBackup with %q", actionMap)
		}
		if scriptName == "scriptDeleteError" && (errorRestore != nil || errorBackup != nil ||
			errorCheck != nil || errorDelete == nil) {
			t.Errorf("Failed to test UserInfoBackup with %q", actionMap)
		}
	}
}

func TestNewUserinfoBackup(t *testing.T) {
	userinfo := NewUserinfoBackup("")
	if userinfo != nil {
		t.Errorf("Failed to get userinfo from annotation")
	}

	userinfo = NewUserinfoBackup("/tmp/userinfo.sh")
	if userinfo != nil {
		t.Errorf("Wrong userinfo got")
	}
}

func GenerateTestUserinfoBackup() *UserinfoBackup {
	actionMap := map[string]int{"restore": 0, "backup": 0, "check": 0, "delete": 0}
	return &UserinfoBackup{&FakeExecuter{actionMap}}
}

func GenerateTestPod() *v1.Pod {
	testPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bar",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "foo",
					Image:           "busybox",
					ImagePullPolicy: v1.PullIfNotPresent,
					Command:         []string{"testCommand"},
					WorkingDir:      "testWorkingDir",
				},
			},
		},
	}
	return testPod
}

func GeneratePodStatus(container *v1.Container) *kubecontainer.PodStatus {
	containerStatus := &kubecontainer.PodStatus{
		ContainerStatuses: []*kubecontainer.ContainerStatus{
			{
				ID: kubecontainer.ContainerID{
					Type: "docker",
					ID:   container.Name,
				},
				Name:      container.Name,
				State:     kubecontainer.ContainerStateRunning,
				CreatedAt: time.Unix(0, time.Now().Unix()),
			},
		},
	}
	return containerStatus
}

func TestUpgradeRunningContainer(t *testing.T) {
	fakeRuntime, fakeImageService, m, _ := createTestRuntimeManager()
	testPod := GenerateTestPod()
	userinfoBackup := GenerateTestUserinfoBackup()
	m.userinfoBackup = userinfoBackup
	image := testPod.Spec.Containers[0].Image
	fakeImageService.Images = map[string]*runtimeapi.Image{
		image: {Id: "123", Size_: ext4MaxFileNameLen},
	}

	// Fake all the things you need before trying to create a container
	fakeSandBox, _ := makeAndSetFakePod(t, m, fakeRuntime, testPod)
	fakeSandBoxConfig, _ := m.generatePodSandboxConfig(testPod, 0)
	testContainer := &testPod.Spec.Containers[0]
	fakePodStatus := GeneratePodStatus(testContainer)
	// Try to create a container
	createResult, _, err := m.createContainerExtension(fakeSandBox.Id, fakeSandBoxConfig, testContainer, testPod, fakePodStatus, nil, "", "", kubecontainer.ContainerTypeRegular, nil)
	if err != nil {
		t.Errorf("createContainer error =%v", err)
	}
	t.Logf("%v", createResult.ID)

	containerID := createResult.ID
	fakePodStatus.ContainerStatuses[0].ID.ID = containerID
	m.runtimeService.StartContainer(containerID)

	containerStatus := fakePodStatus.ContainerStatuses[0]

	testContainer.Command = []string{"testCommandChange"}

	// Try to upgrade a container
	_, _, err = m.upgradeContainerToRunningState(containerStatus, fakeSandBox.Id, fakeSandBoxConfig, testPod, fakePodStatus, nil, "", testContainer)
	if err != nil {
		t.Errorf("upgradeContainer error = %v", err)
	}
}

func TestUpgradeContainerCaseRunning(t *testing.T) {
	fakeRuntime, fakeImageService, m, _ := createTestRuntimeManager()
	testPod := GenerateTestPod()
	userinfoBackup := GenerateTestUserinfoBackup()
	m.userinfoBackup = userinfoBackup
	image := testPod.Spec.Containers[0].Image
	fakeImageService.Images = map[string]*runtimeapi.Image{
		image: {Id: "123", Size_: ext4MaxFileNameLen},
	}
	// Fake all the things you need before trying to create a container
	fakeSandBox, _ := makeAndSetFakePod(t, m, fakeRuntime, testPod)
	fakeSandBoxConfig, _ := m.generatePodSandboxConfig(testPod, 0)
	testContainer := &testPod.Spec.Containers[0]
	fakePodStatus := GeneratePodStatus(testContainer)
	// Try to create a container
	createResult, _, err := m.createContainerExtension(fakeSandBox.Id, fakeSandBoxConfig, testContainer, testPod, fakePodStatus, nil, "", "", kubecontainer.ContainerTypeRegular, nil)
	if err != nil {
		t.Errorf("createContainer error =%v", err)
	}
	t.Logf("%v", createResult.ID)

	containerID := createResult.ID
	fakePodStatus.ContainerStatuses[0].ID.ID = containerID
	m.runtimeService.StartContainer(containerID)

	containerStatus := fakePodStatus.ContainerStatuses[0]

	testContainer.Command = []string{"testCommandChange"}

	// Try to upgrade a container
	_, _, err = m.upgradeContainer(containerStatus, fakeSandBox.Id, fakeSandBoxConfig, testPod, fakePodStatus, nil, "", testContainer)
	if err != nil {
		t.Errorf("upgradeContainer error = %v", err)
	}
}

func TestUpgradeContainerCaseExited(t *testing.T) {
	fakeRuntime, fakeImageService, m, _ := createTestRuntimeManager()
	testPod := GenerateTestPod()
	userinfoBackup := GenerateTestUserinfoBackup()
	m.userinfoBackup = userinfoBackup
	image := testPod.Spec.Containers[0].Image
	fakeImageService.Images = map[string]*runtimeapi.Image{
		image: {Id: "123", Size_: ext4MaxFileNameLen},
	}
	// Fake all the things you need before trying to create a container
	fakeSandBox, _ := makeAndSetFakePod(t, m, fakeRuntime, testPod)
	fakeSandBoxConfig, _ := m.generatePodSandboxConfig(testPod, 0)
	testContainer := &testPod.Spec.Containers[0]
	fakePodStatus := GeneratePodStatus(testContainer)
	// Try to create a container
	createResult, _, err := m.createContainerExtension(fakeSandBox.Id, fakeSandBoxConfig, testContainer, testPod, fakePodStatus, nil, "", "", kubecontainer.ContainerTypeRegular, nil)
	if err != nil {
		t.Errorf("createContainer error =%v", err)
	}
	t.Logf("%v", createResult.ID)

	containerID := createResult.ID
	fakePodStatus.ContainerStatuses[0].ID.ID = containerID
	m.runtimeService.StartContainer(containerID)
	m.runtimeService.StopContainer(containerID, 30)

	containerStatus := fakePodStatus.ContainerStatuses[0]

	testContainer.Command = []string{"testCommandChange"}

	// Try to upgrade a container
	_, _, err = m.upgradeContainer(containerStatus, fakeSandBox.Id, fakeSandBoxConfig, testPod, fakePodStatus, nil, "", testContainer)
	if err != nil {
		t.Errorf("upgradeContainer error = %v", err)
	}
}

func TestMergeAnonymousVolumesWithContainerMounts(t *testing.T) {
	for _, testItem := range []struct {
		image                  *runtimeapi.Image
		container              *v1.Container
		expectAnonymousVolumes map[string]*runtimeapi.Volume
		message                string
	}{
		{
			image: &runtimeapi.Image{
				Id:      "id",
				Volumes: map[string]*runtimeapi.Volume{"/containerPath": &runtimeapi.Volume{}},
			},
			container: &v1.Container{
				Name:  "foo",
				Image: "busybox",
				VolumeMounts: []v1.VolumeMount{
					v1.VolumeMount{
						Name:      "disk",
						MountPath: "/containerPath",
					},
				},
			},
			expectAnonymousVolumes: map[string]*runtimeapi.Volume{},
			message:                "volume is in VolumeMounts",
		},
		{
			image: &runtimeapi.Image{
				Id:      "id",
				Volumes: map[string]*runtimeapi.Volume{"/containerPath": &runtimeapi.Volume{}},
			},
			container: &v1.Container{
				Name:  "foo",
				Image: "busybox",
				VolumeMounts: []v1.VolumeMount{
					v1.VolumeMount{
						Name:      "disk",
						MountPath: "/otherContainerPath",
					},
				},
			},
			expectAnonymousVolumes: map[string]*runtimeapi.Volume{"/containerPath": &runtimeapi.Volume{}},
			message:                "volume is not in VolumeMounts",
		},
	} {
		actualAnonymousVolumes := MergeAnonymousVolumesWithContainerMounts(testItem.image, testItem.container)
		if !isAnonymousVolumeContainerPathEqual(actualAnonymousVolumes, testItem.expectAnonymousVolumes) {
			t.Errorf("Failed to test GetAnonymousVolumesFromContainerStatus with case %s expect: %v but actual: %v",
				testItem.message, testItem.expectAnonymousVolumes, actualAnonymousVolumes)
		}
	}
}

func isAnonymousVolumeContainerPathEqual(anonymousVolumes1 map[string]*runtimeapi.Volume, anonymousVolumes2 map[string]*runtimeapi.Volume) bool {
	if len(anonymousVolumes1) != len(anonymousVolumes2) {
		return false
	}

	for containerPath1 := range anonymousVolumes1 {
		_, exists := anonymousVolumes2[containerPath1]
		if !exists {
			return false
		}
	}

	return true
}

func TestGetAnonymousVolumesFromContainerStatus(t *testing.T) {
	for _, testItem := range []struct {
		containerStatus        *runtimeapi.ContainerStatus
		anonymousVolumes       map[string]*runtimeapi.Volume
		expectAnonymousVolumes map[string]string
		message                string
	}{
		{
			containerStatus: &runtimeapi.ContainerStatus{
				Id: "id",
				Mounts: []*runtimeapi.Mount{
					&runtimeapi.Mount{
						Name:          "volume1",
						ContainerPath: "/containerPath1",
						HostPath:      "/hostpath/volume1/_data",
					},
				},
			},
			anonymousVolumes:       map[string]*runtimeapi.Volume{"/containerPath1": &runtimeapi.Volume{}},
			expectAnonymousVolumes: map[string]string{"/containerPath1": "volume1"},
			message:                "Docker: exists in mounts, exists in anonymousVolumes",
		},
		{
			containerStatus: &runtimeapi.ContainerStatus{
				Id: "id",
				Mounts: []*runtimeapi.Mount{
					&runtimeapi.Mount{
						Name:          "volume1",
						ContainerPath: "/containerPath1",
						HostPath:      "/hostpath/volume1/_data",
					},
				},
			},
			anonymousVolumes:       map[string]*runtimeapi.Volume{"/containerPath2": &runtimeapi.Volume{}},
			expectAnonymousVolumes: map[string]string{},
			message:                "Docker: only exists in mounts",
		},
		{
			containerStatus: &runtimeapi.ContainerStatus{
				Id: "id",
				Mounts: []*runtimeapi.Mount{
					&runtimeapi.Mount{
						Name:          "volume1",
						ContainerPath: "/containerPath1",
						HostPath:      "/hostpath/volume1/_data",
					},
				},
			},
			anonymousVolumes:       map[string]*runtimeapi.Volume{},
			expectAnonymousVolumes: map[string]string{},
			message:                "Docker: only exists in mounts, empty anonymousVolumes",
		},
		{
			containerStatus: &runtimeapi.ContainerStatus{
				Id: "id",
				Mounts: []*runtimeapi.Mount{
					&runtimeapi.Mount{
						Name:          "volume1",
						ContainerPath: "/containerPath1",
						HostPath:      "/hostpath/volume1",
					},
				},
			},
			anonymousVolumes:       map[string]*runtimeapi.Volume{"/containerPath1": &runtimeapi.Volume{}},
			expectAnonymousVolumes: map[string]string{"/containerPath1": "volume1"},
			message:                "Pouch: exists in mounts, exists in anonymousVolumes",
		},
		{
			containerStatus: &runtimeapi.ContainerStatus{
				Id: "id",
				Mounts: []*runtimeapi.Mount{
					&runtimeapi.Mount{
						Name:          "volume1",
						ContainerPath: "/containerPath1",
						HostPath:      "/hostpath/volume1",
					},
				},
			},
			anonymousVolumes:       map[string]*runtimeapi.Volume{"/containerPath2": &runtimeapi.Volume{}},
			expectAnonymousVolumes: map[string]string{},
			message:                "Pouch: only exists in mounts",
		},
		{
			containerStatus: &runtimeapi.ContainerStatus{
				Id: "id",
				Mounts: []*runtimeapi.Mount{
					&runtimeapi.Mount{
						Name:          "volume1",
						ContainerPath: "/containerPath1",
						HostPath:      "/hostpath/volume1",
					},
				},
			},
			anonymousVolumes:       map[string]*runtimeapi.Volume{},
			expectAnonymousVolumes: map[string]string{},
			message:                "Pouch: only exists in mounts, empty anonymousVolumes",
		},
	} {
		actualAnonymousVolumes := GetAnonymousVolumesFromContainerStatus(testItem.anonymousVolumes, testItem.containerStatus)
		if !isAnonymousVolumeEqual(actualAnonymousVolumes, testItem.expectAnonymousVolumes) {
			t.Errorf("Failed to test GetAnonymousVolumesFromContainerStatus with case %s expect: %v but actual: %v",
				testItem.message, testItem.expectAnonymousVolumes, actualAnonymousVolumes)
		}
	}
}

func isAnonymousVolumeEqual(anonymousVolumes1 map[string]string, anonymousVolumes2 map[string]string) bool {
	if len(anonymousVolumes1) != len(anonymousVolumes2) {
		return false
	}

	for containerPath1, volumeName1 := range anonymousVolumes1 {
		volumeName2, exists := anonymousVolumes2[containerPath1]
		if !exists || volumeName1 != volumeName2 {
			return false
		}
	}

	return true
}

func TestMergeAliAdminUID(t *testing.T) {
	testCase := []struct {
		name      string
		parentEnv []*runtimeapi.KeyValue
		nowEnv    []*runtimeapi.KeyValue
		expectEnv []*runtimeapi.KeyValue
	}{
		{
			name: "both have ali admin uid,should parent value ",
			parentEnv: []*runtimeapi.KeyValue{{
				Key:   AliAdminUID,
				Value: "123",
			}},
			nowEnv: []*runtimeapi.KeyValue{{
				Key:   AliAdminUID,
				Value: "321",
			}},
			expectEnv: []*runtimeapi.KeyValue{{
				Key:   AliAdminUID,
				Value: "123",
			}},
		},
		{
			name:      "if parent not have, should now value",
			parentEnv: []*runtimeapi.KeyValue{},
			nowEnv: []*runtimeapi.KeyValue{{
				Key:   AliAdminUID,
				Value: "321",
			}},
			expectEnv: []*runtimeapi.KeyValue{{
				Key:   AliAdminUID,
				Value: "321",
			}},
		},
		{
			name: "if parent have, should parent value  ",
			parentEnv: []*runtimeapi.KeyValue{
				{
					Key:   AliAdminUID,
					Value: "123",
				},
				{
					Key:   makeUID(),
					Value: "123",
				},
			},
			nowEnv: nil,
			expectEnv: []*runtimeapi.KeyValue{{
				Key:   AliAdminUID,
				Value: "123",
			}},
		},
		{
			name: "now have admin uid,but have other ",
			parentEnv: []*runtimeapi.KeyValue{
				{
					Key:   AliAdminUID,
					Value: "123",
				},
			},
			nowEnv: []*runtimeapi.KeyValue{{
				Key:   AliAdminUID + "1",
				Value: "321",
			}},
			expectEnv: []*runtimeapi.KeyValue{
				{
					Key:   AliAdminUID + "1",
					Value: "321",
				},
				{
					Key:   AliAdminUID,
					Value: "123",
				},
			},
		},
	}
	for _, cs := range testCase {
		t.Run(cs.name, func(t *testing.T) {
			env := mergeAliAdminUIDEnv(cs.parentEnv, cs.nowEnv)
			assert.Equal(t, cs.expectEnv, env)
		})
	}

}
