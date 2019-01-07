package kuberuntime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc/status"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/pkg/kubelet/events"
	"k8s.io/kubernetes/pkg/kubelet/util/format"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	sigmautil "k8s.io/kubernetes/pkg/kubelet/sigma"
)

// ContainerAction describe what action should do.
type ContainerAction string

const (
	// ContainerStart start container.
	ContainerStart ContainerAction = "CONTAINER_START"
	// ContainerStop stop container.
	ContainerStop ContainerAction = "CONTAINER_STOP"
	// ContainerUpdate update container.
	ContainerUpdate ContainerAction = "CONTAINER_UPDATE"
	// ContainerDoNothing do nothing.
	ContainerDoNothing ContainerAction = "CONTAINER_DO_NOTHING"
)

const (
	// StartContainerSuccess start container success message
	StartContainerSuccess = "start container success"
	// KillContainerSuccess kill container success message
	KillContainerSuccess = "kill container success"
	// UpgradeContainerSuccess upgrade container success message
	UpgradeContainerSuccess = "upgrade container success"
	// UpdateContainerSuccess update container success message
	UpdateContainerSuccess = "update container success"
	// CreateStartAndPostStartContainerSuccess create container success, start container success, and post start success message
	CreateStartAndPostStartContainerSuccess = "create start and post start success"
	// PauseContainerSuccess pause container success message
	PauseContainerSuccess = "pause container success"
)

// containerOperationInfo contains necessary information about the operation to the container.
type containerOperationInfo struct {
	// The spec of the container.
	container *v1.Container
	// The name of the container.
	name string
	// The message indicates why the container will do the operation.
	message string
}

// SyncPodExtension extension sync pod function, satisfy user’s requirement that operation to the container,
// like start, stop a container.
//  syncs the container into the desired status by executing following steps:
//  1. stop container which need to stop.
//  2. start container which need to start.
//  3. upgrade container which need to upgrade.
//  4. update container which need to update.
//  5. start container without doing postStartHook which need to pause.
//  6. clean container state status which not exist
//  7. update pod annotation.
func (m *kubeGenericRuntimeManager) SyncPodExtension(podSandboxConfig *runtimeapi.PodSandboxConfig, pod *v1.Pod,
	podStatus *kubecontainer.PodStatus, pullSecrets []v1.Secret, podIP string, result *kubecontainer.PodSyncResult,
	changes podActions, backOff *flowcontrol.Backoff) {

	podSandboxID := changes.SandboxID

	// step 1:  stop container which need to stop.
	for containerID, containerInfo := range changes.ContainersToKillBecauseDesireState {
		glog.V(3).Infof("Killing unwanted container %q(id=%q) for pod %q",
			containerInfo.name, containerID, format.Pod(pod))
		killContainerResult := kubecontainer.NewSyncResult(kubecontainer.KillContainer, containerInfo.name)
		result.AddSyncResult(killContainerResult)

		containerStatus := createContainerStatus(podStatus, sigmak8sapi.StopContainerAction, containerInfo.name, pod)

		if err := m.killContainer(pod, containerID, containerInfo.name,
			containerInfo.message, nil); err != nil {
			killContainerResult.Fail(kubecontainer.ErrKillContainer, err.Error())
			msg := fmt.Sprintf("killContainer %q(id=%q) for pod %q failed: %v",
				containerInfo.name, containerID, format.Pod(pod), err)
			glog.Errorf(msg)
			m.updateContainerStateStatus(containerStatus, containerInfo.name, containerID.ID, result.StateStatus, false, msg)
		} else {
			m.updateContainerStateStatus(containerStatus, containerInfo.name, containerID.ID, result.StateStatus, true, KillContainerSuccess)
		}
	}

	// step 2: start container which need to start.
	// step 2: start container which need to start.
	for _, idx := range changes.ContainersToStartBecauseDesireState {
		container := &pod.Spec.Containers[idx]
		status := podStatus.FindContainerStatusByName(container.Name)
		if status == nil {
			glog.V(4).Infof("Failed to get status of %s in pod %s, start container in next loop",
				container.Name, format.Pod(pod))

			if utilfeature.DefaultFeatureGate.Enabled(features.StartContainerByOrder) {
				break
			}
			continue
		}
		containerID := status.ID

		startContainerResult := kubecontainer.NewSyncResult(kubecontainer.StartContainer, container.Name)
		result.AddSyncResult(startContainerResult)

		containerStatus := createContainerStatus(podStatus, sigmak8sapi.StartContainerAction, container.Name, pod)

		glog.V(4).Infof("start container %+v in pod %v", container, format.Pod(pod))

		if msg, err := m.startContainerWithOutPullImage(podSandboxConfig, container, containerID.ID,
			pod, podStatus); err != nil {
			startContainerResult.Fail(err, msg)
			m.updateContainerStateStatus(containerStatus, container.Name, containerID.ID, result.StateStatus, false, msg)

			// known errors that are logged in other places are logged at higher levels here to avoid
			// repetitive log spam
			utilruntime.HandleError(fmt.Errorf("container start failed: %v: %s", err, msg))
			// Break if containers should be started in order
			if utilfeature.DefaultFeatureGate.Enabled(features.StartContainerByOrder) {
				break
			}
		} else {
			m.updateContainerStateStatus(containerStatus, container.Name, containerID.ID, result.StateStatus, true, StartContainerSuccess)
		}
	}

	// step 3: upgrade container which need to be upgraded.
	for containerID, containerInfo := range changes.ContainersToUpgrade {
		container := containerInfo.container
		upgradeContainerResult := kubecontainer.NewSyncResult(kubecontainer.SyncAction("UpgradeContainer"), container.Name)
		result.AddSyncResult(upgradeContainerResult)

		// If previous upgrade failed, doBackoff check should be done.
		previousResult := sigmautil.GetStatusFromAnnotation(pod, container.Name)
		if previousResult != nil && !previousResult.Success {
			isInBackOff, msg, err := m.doBackOffExtension(pod, container, podStatus, backOff)
			if isInBackOff {
				upgradeContainerResult.Fail(err, msg)
				glog.V(4).Infof("Backing Off upgrading container %+v in pod %v", container, format.Pod(pod))
				return
			}
		}

		containerStatusFromCache := podStatus.FindContainerStatusByName(container.Name)
		containerStatus := createContainerStatus(podStatus, sigmak8sapi.UpgradeContainerAction, containerInfo.name, pod)
		containerUpgradeResult, msg, err := m.upgradeContainer(containerStatusFromCache, podSandboxID, podSandboxConfig, pod, podStatus, pullSecrets, podIP, container)
		success := false
		statusMsg := ""
		if err != nil {
			upgradeContainerResult.Fail(err, msg)
			containerStatus.Message = msg
			success = false
			statusMsg = msg

			// known errors that are logged in other places are logged at higher levels here to avoid
			// repetitive log spam
			utilruntime.HandleError(fmt.Errorf("container start failed: %v: %s", err, msg))
		} else {
			success = true
			statusMsg = UpgradeContainerSuccess
		}
		var currentContainerID string
		if containerUpgradeResult != nil {
			currentContainerID = containerUpgradeResult.ID
		} else {
			currentContainerID = containerID.ID
		}
		m.updateContainerStateStatus(containerStatus, containerInfo.name, currentContainerID, result.StateStatus, success, statusMsg)
	}

	// step 4: update container which need to be updated.
	// Ignore update request if inplace update state is not accepted.
	if len(changes.ContainersToUpdate) > 0 && sigmautil.IsInplaceUpdateAccepted(pod) {
		updatePodResult := kubecontainer.NewSyncResult(kubecontainer.UpdateContainer, format.Pod(pod))
		currentPodCPUQuota := int64(0)
		newPodCPUQuota := int64(0)

		for _, container := range pod.Spec.Containers {
			containerStatus := podStatus.FindContainerStatusByName(container.Name)
			if containerStatus.Resources != nil {
				currentPodCPUQuota += containerStatus.Resources.CpuQuota
			}

			newLC := m.generateLinuxContainerResources(&container, pod)
			newPodCPUQuota += newLC.CpuQuota
		}

		// If the total amount of CPUQuota allocated to containers increases by resizing,
		// the CPUQuota at pod-level should be updated before updating container-level CPUQuota.
		// Refer https://lwn.net/Articles/434985/ for more detailed info.
		if m.cpuCFSQuota && (newPodCPUQuota > currentPodCPUQuota) {
			if err := m.runtimeHelper.UpdatePodCgroup(pod); err != nil {
				result.AddSyncResult(updatePodResult)
				errMsg := fmt.Sprintf("update cgroup of pod(%s) failed", format.Pod(pod))
				updatePodResult.Fail(err, errMsg)
				for containerID, containerInfo := range changes.ContainersToUpdate {
					containerStatus := createContainerStatus(podStatus, sigmak8sapi.UpdateContainerAction, containerInfo.name, pod)
					containerStatus.Message = errMsg
					containerStatus.Success = false
					m.updateContainerStateStatus(containerStatus, containerInfo.name, containerID.ID,
						result.StateStatus, false, errMsg)
				}
				return
			}
		}

		for containerID, containerInfo := range changes.ContainersToUpdate {
			container := containerInfo.container
			updateContainerResult := kubecontainer.NewSyncResult(kubecontainer.UpdateContainer, containerInfo.name)
			result.AddSyncResult(updateContainerResult)

			// If previous update failed, call doBackOffExtension.
			previousResult := sigmautil.GetStatusFromAnnotation(pod, container.Name)
			if previousResult != nil && !previousResult.Success {
				isInBackOff, msg, err := m.doBackOffExtension(pod, container, podStatus, backOff)
				if isInBackOff {
					updateContainerResult.Fail(err, msg)
					glog.V(4).Infof("backing off updating container %+v in pod %v", container, format.Pod(pod))
					return
				}
			}

			containerStatus := createContainerStatus(podStatus, sigmak8sapi.UpdateContainerAction, containerInfo.name, pod)
			if msg, err := m.updateContainer(containerID, containerInfo.container, pod); err != nil {
				m.updateContainerStateStatus(containerStatus, containerInfo.name, containerID.ID, result.StateStatus, false, msg)
				updateContainerResult.Fail(err, msg)
			} else {
				// After updateContainer, needs to updates PodStatusCache for the pod resized here.
				// The current PLEG that is responsible for updating PodCache can't detect container resizing by docker update.
				// since the docker currently doesn't care about docker update in the terms of the container status.
				// cf. See g.relist in pkg/kubelet/pleg/generic.go
				if err := m.runtimeHelper.UpdatePodStatusCache(pod); err != nil {
					glog.Errorf("UpdatePodStatusCache failed for pod %s/%s %v", pod.Namespace, pod.Name, err)
				}
				m.updateContainerStateStatus(containerStatus, containerInfo.name, containerID.ID, result.StateStatus, true, UpdateContainerSuccess)
			}
		}
	}

	// step 5: pause container if needed.
	for containerID, containerInfo := range changes.ContainersToStartBecausePause {
		container := containerInfo.container
		pauseContainerResult := kubecontainer.NewSyncResult(kubecontainer.SyncAction("PauseContainer"), container.Name)
		result.AddSyncResult(pauseContainerResult)

		containerStatusFromCache := podStatus.FindContainerStatusByName(container.Name)
		containerStatus := createContainerStatus(podStatus, sigmak8sapi.PauseContainerAction, containerInfo.name, pod)
		if containerStatusFromCache.State != kubecontainer.ContainerStateRunning {
			ref, err := kubecontainer.GenerateContainerRef(pod, container)
			if err != nil {
				glog.Errorf("Couldn't make a ref to pod %q: '%v'", format.Pod(pod), err)
			}
			err = m.runtimeService.StartContainer(containerID.ID)
			if err != nil {
				m.recorder.Eventf(ref, v1.EventTypeWarning, events.FailedToStartContainer, "Failed to start container(paused)")
				glog.V(0).Infof("Failed to start the paused container %s in %s: %s",
					container.Name, format.Pod(pod), err.Error())
				m.updateContainerStateStatus(containerStatus, containerInfo.name, containerID.ID, result.StateStatus, false, err.Error())
			} else {
				m.recorder.Eventf(ref, v1.EventTypeNormal, events.StartedContainer, "Success to start container(paused)")

				containerStatus.CurrentState = sigmak8sapi.ContainerStatePaused
				m.updateContainerStateStatus(containerStatus, containerInfo.name, containerID.ID, result.StateStatus, true, PauseContainerSuccess)
			}
		}
	}
}

// If a container is still in backoff, the function will return a brief backoff error and
// a detailed error message.
func (m *kubeGenericRuntimeManager) doBackOffExtension(pod *v1.Pod, container *v1.Container, podStatus *kubecontainer.PodStatus, backOff *flowcontrol.Backoff) (bool, string, error) {
	glog.Infof("checking backoff for container %q in pod %q", container.Name, format.Pod(pod))
	// Use the FinishTimestamp in update-status as the start point to calculate whether to do back-off or not.
	containerStatus := sigmautil.GetStatusFromAnnotation(pod, container.Name)

	if containerStatus == nil {
		return false, "", nil
	}
	ts := containerStatus.FinishTimestamp
	// backOff requires a unique key to identify the container.
	key := getStableKey(pod, container)
	if backOff.IsInBackOffSince(key, ts) {
		if ref, err := kubecontainer.GenerateContainerRef(pod, container); err == nil {
			m.recorder.Eventf(ref, v1.EventTypeWarning, events.BackOffStartContainer, "Back-off restarting failed container")
		}
		err := fmt.Errorf("Back-off %s upgrading failed container=%s pod=%s", backOff.Get(key), container.Name, format.Pod(pod))
		glog.Infof("%s", err.Error())
		return true, err.Error(), kubecontainer.ErrCrashLoopBackOff
	}

	backOff.Next(key, ts)
	return false, "", nil
}

// computeContainerAction analysis what action should do for a container by compare expect state with current state.
func computeContainerAction(
	expectState sigmak8sapi.ContainerState, currentState kubecontainer.ContainerState) ContainerAction {
	action := ContainerDoNothing
	switch expectState {
	case sigmak8sapi.ContainerStateRunning:
		if currentState == kubecontainer.ContainerStateCreated ||
			currentState == kubecontainer.ContainerStateExited ||
			currentState == kubecontainer.ContainerStateUnknown {
			action = ContainerStart
		}
	case sigmak8sapi.ContainerStateExited:
		if currentState == kubecontainer.ContainerStateCreated ||
			currentState == kubecontainer.ContainerStateRunning ||
			currentState == kubecontainer.ContainerStateUnknown {
			action = ContainerStop
		}
	default:
	}
	return action
}

// startContainer starts a container and returns a message indicates why it is failed on error.
// It starts the container through the following steps:
// * start the container
// * run the post start lifecycle hooks (if applicable)
// it different startContainer function because it doesn't pull image.
func (m *kubeGenericRuntimeManager) startContainerWithOutPullImage(podSandboxConfig *runtimeapi.PodSandboxConfig,
	container *v1.Container, containerID string, pod *v1.Pod, podStatus *kubecontainer.PodStatus) (string, error) {
	// Step 1: start the container.
	err := m.runtimeService.StartContainer(containerID)
	if err != nil {
		errMsg := func(err error) string {
			if s, ok := status.FromError(err); ok {
				return s.Message()
			}
			return err.Error()
		}(err)
		m.recordContainerEvent(pod, container, containerID,
			v1.EventTypeWarning, events.FailedToStartContainer, "Error: %v", errMsg)
		return errMsg, kubecontainer.ErrRunContainer
	}
	m.recordContainerEvent(pod, container, containerID, v1.EventTypeNormal,
		events.StartedContainer, "Started container")

	// For a new container, the RestartCount should be 0
	restartCount := 0
	containerStatus := podStatus.FindContainerStatusByName(container.Name)
	if containerStatus != nil {
		restartCount = containerStatus.RestartCount + 1
	}

	// Symlink container logs to the legacy container log location for cluster logging
	// support.
	// TODO(random-liu): Remove this after cluster logging supports CRI container log path.
	sandboxMeta := podSandboxConfig.GetMetadata()
	legacySymlink := legacyLogSymlink(containerID, container.Name, sandboxMeta.Name,
		sandboxMeta.Namespace)
	logPath := buildContainerLogsPath(container.Name, restartCount)
	containerLog := filepath.Join(podSandboxConfig.LogDirectory, logPath)
	// only create legacy symlink if containerLog path exists (or the error is not IsNotExist).
	// Because if containerLog path does not exist, only dandling legacySymlink is created.
	// This dangling legacySymlink is later removed by container gc, so it does not make sense
	// to create it in the first place. it happens when journald logging driver is used with docker.
	if _, err := m.osInterface.Stat(containerLog); !os.IsNotExist(err) {
		if err := m.osInterface.Symlink(containerLog, legacySymlink); err != nil {
			glog.Errorf("Failed to create legacy symbolic link %q to container %q log %q: %v",
				legacySymlink, containerID, containerLog, err)
		}
	}

	// Step 2: execute the post start hook.
	if container.Lifecycle != nil && container.Lifecycle.PostStart != nil {
		// Get postStartHook timeout from pod annotation.
		timeout := sigmautil.GetTimeoutSecondsFromPodAnnotation(pod, container.Name, sigmak8sapi.PostStartHookTimeoutSeconds)
		kubeContainerID := kubecontainer.ContainerID{
			Type: m.runtimeName,
			ID:   containerID,
		}
		glog.V(4).Infof("Exec PostStartHook: %v in container %s-%s with timeout value: %d",
			container.Lifecycle.PostStart, format.Pod(pod), container.Name, timeout)
		msg, handlerErr := m.runner.Run(kubeContainerID, pod, container, container.Lifecycle.PostStart, time.Duration(timeout)*time.Second)
		if handlerErr != nil {
			m.recordContainerEvent(pod, container, kubeContainerID.ID,
				v1.EventTypeWarning, events.FailedPostStartHook, msg)
			if err := m.killContainer(pod, kubeContainerID, container.Name,
				"FailedPostStartHook", nil); err != nil {
				glog.Errorf("Failed to kill container %q(id=%q) in pod %q: %v, %v",
					container.Name, kubeContainerID.String(), format.Pod(pod), ErrPostStartHook, err)
			}
			return msg, fmt.Errorf("%s: %v", ErrPostStartHook, handlerErr)
		}
		m.recordContainerEvent(pod, container, containerID, v1.EventTypeNormal, events.SucceedPostStartHook,
			fmt.Sprintf("Container %s execute poststart hook success", container.Name))
	} else {
		m.recordContainerEvent(pod, container, containerID, v1.EventTypeNormal, events.WithOutPostStartHook,
			fmt.Sprintf("Container %s with out poststart hook", container.Name))
	}
	return "", nil
}

// containerStateConvert containerState type convert from kubecontainer.containerState.
func containerStateConvertFromKubeContainer(state kubecontainer.ContainerState) sigmak8sapi.ContainerState {
	switch state {
	case kubecontainer.ContainerStateCreated:
		return sigmak8sapi.ContainerStateCreated
	case kubecontainer.ContainerStateExited:
		return sigmak8sapi.ContainerStateExited
	case kubecontainer.ContainerStateRunning:
		return sigmak8sapi.ContainerStateRunning
	}
	return sigmak8sapi.ContainerStateUnknown
}

// containerStateConvert containerState type convert from runtimeApi.containerState.
func containerStateConvertFromRunTimeAPI(state runtimeapi.ContainerState) sigmak8sapi.ContainerState {
	switch state {
	case runtimeapi.ContainerState_CONTAINER_CREATED:
		return sigmak8sapi.ContainerStateCreated
	case runtimeapi.ContainerState_CONTAINER_EXITED:
		return sigmak8sapi.ContainerStateExited
	case runtimeapi.ContainerState_CONTAINER_RUNNING:
		return sigmak8sapi.ContainerStateRunning
	}
	return sigmak8sapi.ContainerStateUnknown
}

// createContainerStatus create a new containerStatus.
func createContainerStatus(podStatus *kubecontainer.PodStatus, action sigmak8sapi.ContainerAction,
	containerName string, pod *v1.Pod) sigmak8sapi.ContainerStatus {
	lastState := sigmak8sapi.ContainerStateUnknown
	containerStatus := podStatus.FindContainerStatusByName(containerName)
	if containerStatus != nil {
		lastState = containerStateConvertFromKubeContainer(containerStatus.State)
	}

	cs := sigmak8sapi.ContainerStatus{
		Action:            action,
		CreationTimestamp: time.Now(),
		LastState:         lastState,
	}
	// Use user hash as spec hash
	hashStr, specHashExists := sigmautil.GetSpecHashFromAnnotation(pod)
	if specHashExists {
		cs.SpecHash = hashStr
	}
	return cs
}

// validateMessage will filter invalid messages.
// Now, we only skip Back-off status for retryCount
func validateMessage(message string) bool {
	backOffKeyStr := "Back-off"
	return !strings.Contains(message, backOffKeyStr)
}

// updateContainerStateStatus update container state status.
func (m *kubeGenericRuntimeManager) updateContainerStateStatus(status sigmak8sapi.ContainerStatus,
	containerName, containerID string, stateStatus sigmak8sapi.ContainerStateStatus, success bool, message string) {
	status.FinishTimestamp = time.Now()
	status.Success = success
	status.Message = message

	previousStateStatus, _ := stateStatus.Statuses[sigmak8sapi.ContainerInfo{Name: containerName}]
	// 1. keep retryCount if the status is invalid
	// 2. increase retryCount if the status is valid and success is false
	// 3. reset retryCount to 0 if success is true
	if !validateMessage(message) {
		status.RetryCount = previousStateStatus.RetryCount
	} else if !success {
		status.RetryCount = previousStateStatus.RetryCount + 1
	} else {
		status.RetryCount = 0
	}

	if status.CurrentState == sigmak8sapi.ContainerStatePaused {
		stateStatus.Statuses[sigmak8sapi.ContainerInfo{Name: containerName}] = status
		return
	}

	var state runtimeapi.ContainerState
	containerStatusFromRunTime, err := m.runtimeService.ContainerStatus(containerID)
	if err != nil {
		glog.Errorf("get container %s runtime status err: %s", containerName, err.Error())
		state = runtimeapi.ContainerState_CONTAINER_UNKNOWN
	} else {
		state = containerStatusFromRunTime.State
	}

	status.CurrentState = containerStateConvertFromRunTimeAPI(state)
	stateStatus.Statuses[sigmak8sapi.ContainerInfo{Name: containerName}] = status
}

// CompareCurrentStateAndDesiredState compare whether the container exist in desired state spec, and get action
func CompareCurrentStateAndDesiredState(containerDesiredState sigmak8sapi.ContainerStateSpec,
	containerStatus *kubecontainer.ContainerStatus, containerName string) (containerExistInDesire bool, action ContainerAction) {
	if containerStatus == nil {
		glog.V(2).Infof("container %s have no status, so ignore container operation", containerName)
		return false, ContainerDoNothing
	}

	for containerInfo, state := range containerDesiredState.States {
		if !strings.EqualFold(containerInfo.Name, containerName) {
			continue
		}
		action := computeContainerAction(state, containerStatus.State)
		return true, action
	}
	return false, ContainerDoNothing
}

func isContainerPaused(containerDesiredState sigmak8sapi.ContainerStateSpec, containerName string) bool {
	for containerInfo, state := range containerDesiredState.States {
		if !strings.EqualFold(containerInfo.Name, containerName) {
			continue
		}
		return state == sigmak8sapi.ContainerStatePaused
	}
	return false
}
