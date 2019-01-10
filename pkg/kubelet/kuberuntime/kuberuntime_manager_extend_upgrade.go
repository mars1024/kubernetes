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
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/events"
	"k8s.io/kubernetes/pkg/kubelet/images"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
)

var (
	// ErrUpgradeContainer indicates the failure to upgrade a container
	ErrUpgradeContainer = errors.New("UpgradeContainerError")
	// CreateContainerSuccess indicates the success to upgrade a container
	CreateContainerSuccess = "create container success"
)

const (
	// AliAdminUID is  ali admin uid
	AliAdminUID = "ali_admin_uid"
)

// CmdExecuter is an interface to exec a command
type CmdExecuter interface {
	ExecCommand(s string) (string, error)
}

// ScriptExecuter is a CmdExecuter to exec a command on host
type ScriptExecuter struct {
	Script string
}

// ExecCommand can exec a command on host
func (e *ScriptExecuter) ExecCommand(s string) (string, error) {
	command := e.Script + " " + s
	cmd := exec.Command("/bin/bash", "-c", command)
	glog.V(4).Infof("command: %v", cmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

// UserinfoBackup can backup or restore a container's userinfo via an CmdExecuter.
type UserinfoBackup struct {
	Executer CmdExecuter
}

func (u *UserinfoBackup) execShell(s string) (string, error) {
	msg, err := u.Executer.ExecCommand(s)
	return msg, err
}

// generateUniqueContainerName generates a unique name for a certain container
func (u *UserinfoBackup) generateUniqueContainerName(pod *v1.Pod, container *v1.Container) string {
	uniqueName := pod.Name + "_" + string(pod.UID) + "_" + container.Name
	return uniqueName
}

// BackupUserinfo can copy userinfo from container to disk
// script command: <script> backup <container-id> <unique-container-name>
// Script should exit with 0 if there is no error, else exit with a non-0 value
func (u *UserinfoBackup) BackupUserinfo(pod *v1.Pod, container *v1.Container, containerID string) (string, error) {
	containerName := u.generateUniqueContainerName(pod, container)
	command := "backup " + containerID + " " + containerName
	glog.V(4).Infof("Backup userinfo of container %s with command: %s", containerID, command)
	msg, err := u.execShell(command)
	return msg, err
}

// RestoreUserinfo can copy userinfo from disk to container
// script command: <script> restore <container-id> <unique-container-name>
// Script should exit with 0 if there is no error, else exit with a non-0 value
func (u *UserinfoBackup) RestoreUserinfo(pod *v1.Pod, container *v1.Container, containerID string) (string, error) {
	containerName := u.generateUniqueContainerName(pod, container)
	command := "restore " + containerID + " " + containerName
	glog.V(4).Infof("Restore userinfo for container %s with command: %s", containerID, command)
	msg, err := u.execShell(command)
	return msg, err
}

// CheckUserinfoExists can check whether userinfo exists on disk or not
// script command: <script> check <unique-container-name>
// Script should exit with 0 if userinfo is on the disk, else exit with a non-0 value
func (u *UserinfoBackup) CheckUserinfoExists(pod *v1.Pod, container *v1.Container) (string, error) {
	containerName := u.generateUniqueContainerName(pod, container)
	command := "check " + containerName
	glog.V(4).Infof("Check userinfo for container %s on disk with command: %s", containerName, command)
	msg, err := u.execShell(command)
	return msg, err
}

// DeleteUserinfo can delete userinfo on disk
// script command: <script> delete <unique-container-name>
// Script should exit with 0 if there is no error, else exit with a non-0 value
func (u *UserinfoBackup) DeleteUserinfo(pod *v1.Pod, container *v1.Container) (string, error) {
	containerName := u.generateUniqueContainerName(pod, container)
	command := "delete " + containerName
	glog.V(4).Infof("Delete userinfo for container %s on disk with command: %s", containerName, command)
	msg, err := u.execShell(command)
	return msg, err
}

// NewUserinfoBackup can create a userinfoBackup.
func NewUserinfoBackup(userinfoScript string) *UserinfoBackup {
	if userinfoScript == "" {
		glog.V(0).Infof("Ignore to initialize UserinfoBackup because userinfoScript is empty")
		return nil
	}
	if _, err := os.Stat(userinfoScript); err != nil {
		glog.V(0).Infof("Ignore to initialize UserinfoBackup because userinfo script %s doesn't exist", userinfoScript)
		return nil
	}
	glog.V(0).Infof("UserinfoBackup will be initialized with script: %s", userinfoScript)
	userinfoBackup := &UserinfoBackup{
		Executer: &ScriptExecuter{userinfoScript},
	}
	return userinfoBackup
}

// ContainerInfo contains information of a new container.
type ContainerInfo struct {
	Config *runtimeapi.ContainerConfig
	ID     string
}

// Two conditions can trigger upgrade:
// 1. container spec hash is changed
// 2. container is a dirty container: container's state is Created, and userinfo is on the disk.
func (m *kubeGenericRuntimeManager) isContainerNeedUpgrade(pod *v1.Pod, container *v1.Container, containerStatus *kubecontainer.ContainerStatus) bool {
	if containerStatus != nil {
		if expectedHash, actualHash, needToRestart, _ := m.containerChanged(container, containerStatus, pod); needToRestart {
			glog.V(4).Infof("Container spec hash changed (%d vs %d).", actualHash, expectedHash)
			return true
		}
	}

	if containerStatus != nil && m.isDirtyContainer(pod, container, containerStatus) {
		glog.V(4).Infof("Upgrade: Container %s is a dirty container.", container.Name)
		return true
	}

	return false
}

// Check whether a container is dirty container or not
// If container's state is Created, and container's userinfo is on the disk, we call this container as dirty container
func (m *kubeGenericRuntimeManager) isDirtyContainer(pod *v1.Pod, container *v1.Container, containerStatus *kubecontainer.ContainerStatus) bool {
	if containerStatus.State == kubecontainer.ContainerStateCreated {
		if m.userinfoBackup != nil {
			if _, err := m.userinfoBackup.CheckUserinfoExists(pod, container); err == nil {
				return true
			}
		}
	}

	return false
}

// getDiskSize convert disk size such as "1Gi" to "1g".
func getDiskSize(s string) string {
	if strings.HasSuffix(s, "Gi") {
		s = strings.Replace(s, "Gi", "g", -1)
	}
	if strings.HasSuffix(s, "G") {
		s = strings.Replace(s, "G", "g", -1)
	}
	if strings.HasSuffix(s, "Mi") {
		s = strings.Replace(s, "Mi", "m", -1)
	}
	if strings.HasSuffix(s, "M") {
		s = strings.Replace(s, "M", "m", -1)
	}
	if strings.HasSuffix(s, "Ki") {
		s = strings.Replace(s, "Ki", "k", -1)
	}
	if strings.HasSuffix(s, "K") {
		s = strings.Replace(s, "K", "k", -1)
	}
	return s
}

// createContainerExtension creates a container and returns a message indicates why it is failed on error.
// Steps:
// * create the container
// * run the post PreStart lifecycle hooks (if applicable)
func (m *kubeGenericRuntimeManager) createContainerExtension(podSandboxID string,
	podSandboxConfig *runtimeapi.PodSandboxConfig,
	container *v1.Container,
	pod *v1.Pod,
	podStatus *kubecontainer.PodStatus,
	parentContainerStatus *runtimeapi.ContainerStatus,
	podIP string,
	imageRef string,
	containerType kubecontainer.ContainerType,
	anonymousVolumes map[string]string) (*ContainerInfo, string, error) {

	// Step 1: create the container.
	ref, err := kubecontainer.GenerateContainerRef(pod, container)
	if err != nil {
		glog.Errorf("Can't make a ref to pod %q, container %v: %v", format.Pod(pod), container.Name, err)
	}
	glog.V(4).Infof("Generating ref for container %s: %#v", container.Name, ref)

	// For a new container, the RestartCount should be 0
	restartCount := 0
	containerStatus := podStatus.FindContainerStatusByName(container.Name)
	if containerStatus != nil {
		restartCount = containerStatus.RestartCount + 1
	}

	containerConfig, cleanupAction, err := m.generateContainerConfig(container, pod, restartCount, podIP, imageRef, containerType)
	if cleanupAction != nil {
		defer cleanupAction()
	}
	if err != nil {
		m.recordContainerEvent(pod, container, "", v1.EventTypeWarning, events.FailedToCreateContainer, "Error: %v", grpc.ErrorDesc(err))
		return nil, grpc.ErrorDesc(err), ErrCreateContainerConfig
	}
	//merge ali_admin_uid in envs
	if parentContainerStatus != nil && containerConfig != nil {
		containerConfig.Envs = mergeAliAdminUIDEnv(parentContainerStatus.Envs, containerConfig.Envs)
	}

	if containerConfig.Linux != nil {
		if containerConfig.Linux.Resources == nil {
			containerConfig.Linux.Resources = &runtimeapi.LinuxContainerResources{}
		}
		if parentContainerStatus != nil && parentContainerStatus.Resources != nil {
			containerConfig.QuotaId = parentContainerStatus.QuotaId
			containerConfig.Linux.Resources.DiskQuota = parentContainerStatus.Resources.DiskQuota
		} else {
			limitEphemeralStorage, limitESExists := container.Resources.Limits[v1.ResourceEphemeralStorage]
			requestEphemeralStorage, requestESExists := container.Resources.Requests[v1.ResourceEphemeralStorage]
			if limitESExists && requestESExists && !limitEphemeralStorage.IsZero() && limitEphemeralStorage.Cmp(requestEphemeralStorage) == 0 {
				// 2.0 container to 3.1 container, quota id not empty
				if containerConfig.QuotaId == "" {
					// Set QuotaId as -1 to generate a new quotaid.
					containerConfig.QuotaId = "-1"
				}
				containerConfig.Linux.Resources.DiskQuota = map[string]string{".*": getDiskSize(limitEphemeralStorage.String())}
			}
		}
	}

	glog.V(4).Infof("The new config of container %q in %q is %v", container.Name, format.Pod(pod), *containerConfig)

	// Deal with anonymousVolume
	if len(anonymousVolumes) > 0 {
		glog.V(4).Infof("AnoymousVolume %v will be set to %q of %q", anonymousVolumes, container.Name, format.Pod(pod))
		for containerPath, volume := range anonymousVolumes {
			mount := &runtimeapi.Mount{
				HostPath:      volume,
				ContainerPath: containerPath,
			}
			containerConfig.Mounts = append(containerConfig.Mounts, mount)
		}
	}

	// Create container
	containerID, err := m.runtimeService.CreateContainer(podSandboxID, containerConfig, podSandboxConfig)
	if err != nil {
		m.recordContainerEvent(pod, container, containerID, v1.EventTypeWarning, events.FailedToCreateContainer, "Error: %v", grpc.ErrorDesc(err))
		return nil, grpc.ErrorDesc(err), ErrCreateContainer
	}

	// Step 2: Do PreStart hook
	err = m.internalLifecycle.PreStartContainer(pod, container, containerID)
	if err != nil {
		m.recorder.Eventf(ref, v1.EventTypeWarning, events.FailedToStartContainer, "Internal PreStartContainer hook failed: %v", err)
		return nil, "Internal PreStartContainer hook failed", err
	}
	m.recordContainerEvent(pod, container, containerID, v1.EventTypeNormal, events.CreatedContainer, "Created container")

	if ref != nil {
		m.containerRefManager.SetRef(kubecontainer.ContainerID{
			Type: m.runtimeName,
			ID:   containerID,
		}, ref)
	}

	containerCreateResult := &ContainerInfo{
		Config: containerConfig,
		ID:     containerID,
	}

	return containerCreateResult, CreateContainerSuccess, nil
}

// mergeAliAdminUIDEnv merge ali admin uid env in parent env and now env
func mergeAliAdminUIDEnv(currentEnv []*runtimeapi.KeyValue, specEnv []*runtimeapi.KeyValue) []*runtimeapi.KeyValue {
	adminUID := ""
	for _, kv := range currentEnv {
		if kv.Key != AliAdminUID {
			continue
		}
		adminUID = kv.Value
	}
	if adminUID == "" {
		return specEnv
	}

	find := false
	for _, kv := range specEnv {
		if kv.Key != AliAdminUID {
			continue
		}
		kv.Value = adminUID
		find = true
		break
	}
	if !find {
		specEnv = append(specEnv, &runtimeapi.KeyValue{
			Key:   AliAdminUID,
			Value: adminUID,
		})
	}

	return specEnv
}

// upgradeContainer can upgrade running container, exited container or a dirty container.
// Attention: The condition that the return value "ContainerInfo" is nil should be concerned when call upgradeContainer().
func (m *kubeGenericRuntimeManager) upgradeContainer(containerStatus *kubecontainer.ContainerStatus,
	podSandboxID string,
	podSandboxConfig *runtimeapi.PodSandboxConfig,
	pod *v1.Pod,
	podStatus *kubecontainer.PodStatus,
	pullSecrets []v1.Secret,
	podIP string,
	container *v1.Container) (*ContainerInfo, string, error) {
	upgradedContainer := &ContainerInfo{}
	// There are three kinds of containers can be upgraded: running container, exited container and dirty container.
	switch containerStatus.State {
	case kubecontainer.ContainerStateRunning:
		glog.V(0).Infof("Start to upgrade running container %q in pod %q", container.Name, format.Pod(pod))
		upgradeResult, msg, err := m.upgradeContainerToRunningState(containerStatus, podSandboxID, podSandboxConfig, pod, podStatus,
			pullSecrets, podIP, container)
		if err != nil {
			glog.Errorf("Failed to upgrade running container %q: %v", container.Name, container)
			return upgradeResult, msg, err
		}
		glog.V(0).Infof("Upgrade running container %q in pod %q successfully", container.Name, format.Pod(pod))
		upgradedContainer = upgradeResult
	case kubecontainer.ContainerStateExited:
		glog.V(0).Infof("Start to upgrade exited container %q in pod %q", container.Name, format.Pod(pod))
		upgradeResult, msg, err := m.upgradeContainerToRunningState(containerStatus, podSandboxID, podSandboxConfig, pod, podStatus,
			pullSecrets, podIP, container)
		if err != nil {
			glog.Errorf("Failed to upgrade exited container %q: %v", container.Name, container)
			return upgradeResult, msg, err
		}
		glog.V(0).Infof("Upgrade exited container %q in pod %q successfully", container.Name, format.Pod(pod))
		upgradedContainer = upgradeResult
	case kubecontainer.ContainerStateCreated:
		// Dirty container should be reupgraded
		if m.isDirtyContainer(pod, container, containerStatus) {
			glog.V(0).Infof("Start to upgrade dirty container %q in pod %q", container.Name, format.Pod(pod))
			upgradeResult, msg, err := m.upgradeContainerToRunningState(containerStatus, podSandboxID, podSandboxConfig, pod, podStatus,
				pullSecrets, podIP, container)
			if err != nil {
				glog.Errorf("Failed to upgrade dirty container %q: %v", container.Name, container)
				return upgradeResult, msg, err
			}
			glog.V(0).Infof("Upgrade dirty container %q in pod %q successfully", container.Name, format.Pod(pod))
			upgradedContainer = upgradeResult
		}
		//TODO: upgrade Created container.
	default:
		// The code should never be executed
		return nil, "", fmt.Errorf("Upgrade: Not supported yet")

	}
	return upgradedContainer, "", nil
}

func (m *kubeGenericRuntimeManager) upgradeContainerCommon(containerStatus *kubecontainer.ContainerStatus,
	podSandboxID string,
	podSandboxConfig *runtimeapi.PodSandboxConfig,
	pod *v1.Pod,
	podStatus *kubecontainer.PodStatus,
	pullSecrets []v1.Secret,
	podIP string,
	container *v1.Container) (*ContainerInfo, string, error) {

	imageRef, msg, err := m.imagePuller.EnsureImageExists(pod, container, pullSecrets)
	if err != nil {
		m.recordContainerEvent(pod, container, "", v1.EventTypeWarning, events.FailedToCreateContainer, "Error: %v", grpc.ErrorDesc(err))
		return nil, msg, err
	}

	userinfoBackup := m.userinfoBackup
	// Get current container's status
	currentContainerStatus, err := m.runtimeService.ContainerStatus(containerStatus.ID.ID)
	if err != nil {
		return nil, err.Error(), ErrUpgradeContainer
	}

	// Delete anonymousVolume if new image does't contain this anonymousVolume
	imageSpec := &runtimeapi.ImageSpec{
		Image: container.Image,
	}
	image, err := m.imageService.ImageStatus(imageSpec)
	if err != nil {
		return nil, fmt.Sprintf("Failed to inspect image %s", container.Image), err
	}
	if image == nil {
		msg := fmt.Sprintf("image %s not found", container.Image)
		return nil, msg, errors.New(msg)
	}

	mergedAnonymousVolumes := MergeAnonymousVolumesWithContainerMounts(image, container)
	anonymousVolumes := GetAnonymousVolumesFromContainerStatus(mergedAnonymousVolumes, currentContainerStatus)

	containerState := currentContainerStatus.State

	// Step1: stop current container.
	// If there is something wrong between "old container stopped" and "new container started", we will upgrade an exited container in next syncloop.
	// However, expect-state will correct the container's state finally.
	if containerState == runtimeapi.ContainerState_CONTAINER_RUNNING {
		if err := m.killContainer(pod, containerStatus.ID, containerStatus.Name, "kill container for upgrade", nil); err != nil {
			glog.Errorf("Kill container %q(id=%q) in pod %q failed: %v", containerStatus.Name, containerStatus.ID, format.Pod(pod), err)
			return nil, err.Error(), ErrUpgradeContainer
		}
		glog.V(4).Infof("Kill container %q(id=%q) in pod %q successfully", containerStatus.Name, containerStatus.ID, format.Pod(pod))
	}

	// Step2: Backup userinfo
	// TODO: Created container's userinfo should be backuped when supporting Created container
	if userinfoBackup != nil && (containerState == runtimeapi.ContainerState_CONTAINER_RUNNING || containerState == runtimeapi.ContainerState_CONTAINER_EXITED) {
		// Only when we backup userinfo successfully, we can stop current container.
		// So we can get userinfo in the condition that the stopped container is deleted by GC.
		msg, err := userinfoBackup.BackupUserinfo(pod, container, containerStatus.ID.ID)
		if err != nil {
			glog.Errorf("Backup userinfo for container %q(id=%q) in pod %q failed: %v, %q",
				container.Name, containerStatus.ID, format.Pod(pod), err, msg)
			return nil, err.Error(), ErrUpgradeContainer
		}
		glog.V(4).Infof("Backup userinfo for container %q(id=%q) in pod %q successfully", container.Name, containerStatus.ID, format.Pod(pod))
	}

	// Step3: Create new container.
	// When new container is created, the runtime manager can't see old container any more and old container is waiting to be deleted by GC.
	upgradedContainer, msg, err := m.createContainerExtension(podSandboxID, podSandboxConfig, container, pod,
		podStatus, currentContainerStatus, podIP, imageRef, kubecontainer.ContainerTypeRegular, anonymousVolumes)
	if err != nil {
		glog.Errorf("Create container %q in pod %q failed: %v, %s", container.Name, format.Pod(pod), err, msg)
		return nil, err.Error() + ":" + msg, ErrUpgradeContainer
	}
	glog.V(4).Infof("Create container %q(id=%q) in pod %q successfully", container.Name, upgradedContainer.ID, format.Pod(pod))

	// Step4: Restore userinfo
	// If userinfo exists on the disk, we should copy it to the new created container and then delete it.
	if userinfoBackup != nil {
		if _, err := userinfoBackup.CheckUserinfoExists(pod, container); err == nil {
			// Userinfo dir in the disk should be deleted when we finish the restore procedure.
			if msg, err := userinfoBackup.RestoreUserinfo(pod, container, upgradedContainer.ID); err != nil {
				glog.Errorf("Restore userinfo for container %q(id=%q) in pod %q failed: %v, %s",
					container.Name, upgradedContainer.ID, format.Pod(pod), err, msg)
				return upgradedContainer, err.Error(), ErrUpgradeContainer
			}
			glog.V(4).Infof("Restore userinfo for container %q(id=%q) in pod %q successfully",
				container.Name, upgradedContainer.ID, format.Pod(pod))
			// Delete userinfo dir in the disk.
			// Removal of userinfo dir means that the container gets all creating steps finished and can be started now.
			if msg, err := userinfoBackup.DeleteUserinfo(pod, container); err != nil {
				glog.Errorf("Delete userinfo for container %q in pod %q failed: %v, %q",
					container.Name, format.Pod(pod), err, msg)
				return upgradedContainer, err.Error(), ErrUpgradeContainer
			}
			glog.V(4).Infof("Delete userinfo for container %q(id=%q) in pod %q successfully",
				container.Name, upgradedContainer.ID, format.Pod(pod))
		}
	}

	return upgradedContainer, "", nil
}

// upgradeContainerToRunningState can upgrade a container, and the new container's state is running.
func (m *kubeGenericRuntimeManager) upgradeContainerToRunningState(containerStatus *kubecontainer.ContainerStatus,
	podSandboxID string,
	podSandboxConfig *runtimeapi.PodSandboxConfig,
	pod *v1.Pod,
	podStatus *kubecontainer.PodStatus,
	pullSecrets []v1.Secret,
	podIP string,
	container *v1.Container) (*ContainerInfo, string, error) {
	upgradedContainer, msg, err := m.upgradeContainerCommon(containerStatus, podSandboxID, podSandboxConfig, pod, podStatus,
		pullSecrets, podIP, container)
	if err != nil {
		// known errors that are logged in other places are logged at higher levels here to avoid
		// repetitive log spam
		switch {
		case err == images.ErrImagePullBackOff:
			glog.V(3).Infof("container start failed: %v: %s", err, msg)
		default:
			utilruntime.HandleError(fmt.Errorf("container upgrade failed: %v: %s", err, msg))
		}
		return upgradedContainer, msg, err
	}
	// Start new container.
	// If startContainerWithOutPullImage fails, this container will be started directly in next SyncPod.
	if msg, err := m.startContainerWithOutPullImage(podSandboxConfig, container, upgradedContainer.ID, pod, podStatus); err != nil {
		glog.Errorf("Start container %q(id=%q) in pod %q failed: %v, %q", container.Name, upgradedContainer.ID, format.Pod(pod), err, msg)
		return upgradedContainer, err.Error(), ErrUpgradeContainer
	}
	glog.V(4).Infof("Start container %q(id=%q) in pod %q successfully", container.Name, upgradedContainer.ID, format.Pod(pod))
	return upgradedContainer, "", nil
}

// MergeAnonymousVolumesWithContainerMounts can remove the image's anonymousVolumes whose containerPath is already used by volumeMounts in container spec.
func MergeAnonymousVolumesWithContainerMounts(image *runtimeapi.Image, container *v1.Container) map[string]*runtimeapi.Volume {
	mergedVolumes := map[string]*runtimeapi.Volume{}
	if image == nil || image.Volumes == nil {
		return mergedVolumes
	}
	// Convert VolumeMounts to a map
	volumeMounts := make(map[string]struct{}, len(container.VolumeMounts))
	for i := range container.VolumeMounts {
		volumeMounts[container.VolumeMounts[i].MountPath] = struct{}{}
	}

	// If containerPath is in volumesMounts, then ignore this anonymous volume.
	for containerPath := range image.Volumes {
		_, exists := volumeMounts[containerPath]
		if !exists {
			mergedVolumes[containerPath] = &runtimeapi.Volume{}
		}
	}
	return mergedVolumes
}

// GetAnonymousVolumesFromContainerStatus can get anonymousVolume from container status.
// The result is a map: key is the path in the container, and value is the volume name.
func GetAnonymousVolumesFromContainerStatus(mergedVolumes map[string]*runtimeapi.Volume, containerStatus *runtimeapi.ContainerStatus) map[string]string {
	anonymousVolumes := map[string]string{}

	// Convert Mounts to a map
	volumeMounts := make(map[string]*runtimeapi.Mount, len(containerStatus.Mounts))
	for i := range containerStatus.Mounts {
		volumeMounts[containerStatus.Mounts[i].ContainerPath] = containerStatus.Mounts[i]
	}

	for containerPath := range mergedVolumes {
		mount, exists := volumeMounts[containerPath]
		if exists {
			if mount.Name == "" {
				glog.Warningf("Anonymous volume %s is conflict with bind %v, ignore", containerPath, mount)
				continue
			}
			// Set volume name.
			anonymousVolumes[containerPath] = mount.Name
		}
	}
	return anonymousVolumes
}
