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
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/armon/circbuf"
	"github.com/golang/glog"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubetypes "k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/kubernetes/pkg/features"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/events"
	sigmautil "k8s.io/kubernetes/pkg/kubelet/sigma"
	"k8s.io/kubernetes/pkg/kubelet/types"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
	"k8s.io/kubernetes/pkg/util/selinux"
	"k8s.io/kubernetes/pkg/util/tail"
)

var (
	ErrCreateContainerConfig = errors.New("CreateContainerConfigError")
	ErrCreateContainer       = errors.New("CreateContainerError")
	ErrPreStartHook          = errors.New("PreStartHookError")
	ErrPostStartHook         = errors.New("PostStartHookError")
)

// recordContainerEvent should be used by the runtime manager for all container related events.
// it has sanity checks to ensure that we do not write events that can abuse our masters.
// in particular, it ensures that a containerID never appears in an event message as that
// is prone to causing a lot of distinct events that do not count well.
// it replaces any reference to a containerID with the containerName which is stable, and is what users know.
func (m *kubeGenericRuntimeManager) recordContainerEvent(pod *v1.Pod, container *v1.Container, containerID, eventType, reason, message string, args ...interface{}) {
	ref, err := kubecontainer.GenerateContainerRef(pod, container)
	if err != nil {
		glog.Errorf("Can't make a ref to pod %q, container %v: %v", format.Pod(pod), container.Name, err)
		return
	}
	eventMessage := message
	if len(args) > 0 {
		eventMessage = fmt.Sprintf(message, args...)
	}
	// this is a hack, but often the error from the runtime includes the containerID
	// which kills our ability to deduplicate events.  this protection makes a huge
	// difference in the number of unique events
	if containerID != "" {
		eventMessage = strings.Replace(eventMessage, containerID, container.Name, -1)
	}
	m.recorder.Event(ref, eventType, reason, eventMessage)
}

// startContainer starts a container and returns a message indicates why it is failed on error.
// It starts the container through the following steps:
// * pull the image
// * create the container
// * start the container
// * run the post start lifecycle hooks (if applicable)
func (m *kubeGenericRuntimeManager) startContainer(podSandboxID string, podSandboxConfig *runtimeapi.PodSandboxConfig, container *v1.Container, pod *v1.Pod, podStatus *kubecontainer.PodStatus, pullSecrets []v1.Secret, podIP string, containerType kubecontainer.ContainerType) (string, error) {
	// Step 1: pull the image.
	imageRef, msg, err := m.imagePuller.EnsureImageExists(pod, container, pullSecrets)
	if err != nil {
		m.recordContainerEvent(pod, container, "", v1.EventTypeWarning, events.FailedToCreateContainer, "Error: %v", grpc.ErrorDesc(err))
		return msg, err
	}

	userinfoBackup := m.userinfoBackup
	rebuildContainerID := sigmautil.GetRebuildContainerIDFromPodAnnotation(pod)

	// Backup userinfo from rebuildContainerID in sigma2 and restore it to new created container.
	if userinfoBackup != nil && rebuildContainerID != "" {
		glog.V(0).Infof("Start to backup userinfo from container %s in sigma2", rebuildContainerID)
		msg, err := userinfoBackup.BackupUserinfo(pod, container, rebuildContainerID)
		if err != nil {
			glog.Errorf("Backup userinfo from container %q failed: %v, %q",
				rebuildContainerID, err, msg)
			return fmt.Sprintf("Failed to backup userinfo from container: %s", rebuildContainerID), err
		}
		glog.V(0).Infof("Backup userinfo from container %q in sigma2 successfully", rebuildContainerID)
	}

	// Step 2: create the container.
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
		return grpc.ErrorDesc(err), ErrCreateContainerConfig
	}

	containerID, err := m.runtimeService.CreateContainer(podSandboxID, containerConfig, podSandboxConfig)
	if err != nil {
		m.recordContainerEvent(pod, container, containerID, v1.EventTypeWarning, events.FailedToCreateContainer, "Error: %v", grpc.ErrorDesc(err))
		return grpc.ErrorDesc(err), ErrCreateContainer
	}
	err = m.internalLifecycle.PreStartContainer(pod, container, containerID)
	if err != nil {
		m.recordContainerEvent(pod, container, containerID, v1.EventTypeWarning, events.FailedToStartContainer, "Internal PreStartContainer hook failed: %v", grpc.ErrorDesc(err))
		return grpc.ErrorDesc(err), ErrPreStartHook
	}
	m.recordContainerEvent(pod, container, containerID, v1.EventTypeNormal, events.CreatedContainer, "Created container")

	if ref != nil {
		m.containerRefManager.SetRef(kubecontainer.ContainerID{
			Type: m.runtimeName,
			ID:   containerID,
		}, ref)
	}

	// Backup userinfo from rebuildContainerID in sigma2 and restore it to new created container.
	// If the new container is created, and userinfo restore procedure or userinfo delete procedure fails
	// this container is called dirty container and will be processed by Upgrade.
	if userinfoBackup != nil && rebuildContainerID != "" {
		if _, err = userinfoBackup.CheckUserinfoExists(pod, container); err == nil {
			// Userinfo dir in the disk should be deleted when we finish the restore procedure.
			glog.V(0).Infof("Start to restore userinfo for container %q(id=%q) in pod %q",
				container.Name, containerID, format.Pod(pod))
			if msg, err := userinfoBackup.RestoreUserinfo(pod, container, containerID); err != nil {
				glog.Errorf("Restore userinfo for container %q(id=%q) in pod %q failed: %v, %s",
					container.Name, containerID, format.Pod(pod), err, msg)
				return fmt.Sprintf("failed to restore userinfo to container %q", containerID), err
			}
			glog.V(0).Infof("Restore userinfo for container %q(id=%q) in pod %q successfully",
				container.Name, containerID, format.Pod(pod))
			// Delete userinfo dir in the disk.
			// Removal of userinfo dir means that the container gets all creating steps finished and can be started now.
			if msg, err := userinfoBackup.DeleteUserinfo(pod, container); err != nil {
				glog.Errorf("Delete userinfo for container %q in pod %q failed: %v, %q",
					container.Name, format.Pod(pod), err, msg)
				return fmt.Sprintf("failed to delete userinfo for pod %q", format.Pod(pod)), err
			}
			glog.V(0).Infof("Delete userinfo for container %q(id=%q) in pod %q successfully",
				container.Name, containerID, format.Pod(pod))
		}
	}

	// Step 3: start the container.
	err = m.runtimeService.StartContainer(containerID)
	if err != nil {
		m.recordContainerEvent(pod, container, containerID, v1.EventTypeWarning, events.FailedToStartContainer, "Error: %v", grpc.ErrorDesc(err))
		return grpc.ErrorDesc(err), kubecontainer.ErrRunContainer
	}
	m.recordContainerEvent(pod, container, containerID, v1.EventTypeNormal, events.StartedContainer, "Started container")

	// Symlink container logs to the legacy container log location for cluster logging
	// support.
	// TODO(random-liu): Remove this after cluster logging supports CRI container log path.
	containerMeta := containerConfig.GetMetadata()
	sandboxMeta := podSandboxConfig.GetMetadata()
	legacySymlink := legacyLogSymlink(containerID, containerMeta.Name, sandboxMeta.Name,
		sandboxMeta.Namespace)
	containerLog := filepath.Join(podSandboxConfig.LogDirectory, containerConfig.LogPath)
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

	// Step 4: execute the post start hook.
	if container.Lifecycle != nil && container.Lifecycle.PostStart != nil {
		timeout := sigmautil.GetTimeoutSecondsFromPodAnnotation(pod, container.Name, sigmak8sapi.PostStartHookTimeoutSeconds)
		kubeContainerID := kubecontainer.ContainerID{
			Type: m.runtimeName,
			ID:   containerID,
		}
		glog.V(4).Infof("Exec PostStartHook:%v in container %s-%s with timeout value: %d",
			container.Lifecycle.PostStart, format.Pod(pod), container.Name, timeout)
		msg, handlerErr := m.runner.Run(kubeContainerID, pod, container, container.Lifecycle.PostStart, time.Duration(timeout)*time.Second)
		if handlerErr != nil {
			m.recordContainerEvent(pod, container, kubeContainerID.ID, v1.EventTypeWarning, events.FailedPostStartHook, msg)
			if err := m.killContainer(pod, kubeContainerID, container.Name, "FailedPostStartHook", nil); err != nil {
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

	return containerID, nil
}

func (m *kubeGenericRuntimeManager) updateContainer(containerID kubecontainer.ContainerID, container *v1.Container, pod *v1.Pod) (string, error) {
	specAnnotations := make(map[string]string)
	resources, err := m.generateLinuxContainerResources(container, pod)
	if err != nil {
		return "", err
	}
	glog.Infof("update container: %s with linux resource: %+v", container.Name, resources)
	containerStatus, err := m.runtimeService.ContainerStatus(containerID.ID)
	if err != nil {
		glog.Errorf("ContainerStatus for %s error: %v", containerID.ID, err)
		return "", err
	}
	if containerStatus != nil {
		if containerStatus.Resources != nil {
			if (resources.MemorySwap != containerStatus.Resources.MemorySwap) && (resources.MemorySwap > 0) {
				specAnnotations[containerPouchMemorySwapAnnotation] = strconv.FormatInt(int64(resources.MemorySwap), 10)
			}
		}
	}

	err = m.runtimeService.UpdateContainerResources(containerID.ID, resources, specAnnotations)
	if err != nil {
		message := fmt.Sprintf("failed to update container %q(id=%q) in pod %q: %v", container.Name, containerID.String(), format.Pod(pod), err)
		m.recordContainerEvent(pod, container, containerID.ID, v1.EventTypeWarning, events.FailedToUpdateContainer, message)
		return message, err
	}
	if err := m.internalLifecycle.PostResizeContainer(pod, container, containerID.ID); err != nil {
		return "", err
	}
	message := fmt.Sprintf("updated container successfully %q(id=%q) in pod %q", container.Name, containerID.String(), format.Pod(pod))
	m.recordContainerEvent(pod, container, containerID.ID, v1.EventTypeNormal, events.UpdatedContainer, message)
	return message, nil
}

// generateContainerConfig generates container config for kubelet runtime v1.
func (m *kubeGenericRuntimeManager) generateContainerConfig(container *v1.Container, pod *v1.Pod, restartCount int, podIP, imageRef string, containerType kubecontainer.ContainerType) (*runtimeapi.ContainerConfig, func(), error) {
	opts, cleanupAction, err := m.runtimeHelper.GenerateRunContainerOptions(pod, container, podIP)
	if err != nil {
		return nil, nil, err
	}

	uid, username, err := m.getImageUser(container.Image)
	if err != nil {
		return nil, cleanupAction, err
	}

	// Verify RunAsNonRoot. Non-root verification only supports numeric user.
	if err := verifyRunAsNonRoot(pod, container, uid, username); err != nil {
		return nil, cleanupAction, err
	}

	command, args := kubecontainer.ExpandContainerCommandAndArgs(container, opts.Envs)
	logDir := BuildContainerLogsDirectory(kubetypes.UID(pod.UID), container.Name)
	err = m.osInterface.MkdirAll(logDir, 0755)
	if err != nil {
		return nil, cleanupAction, fmt.Errorf("create container log directory for container %s failed: %v", container.Name, err)
	}
	containerLogsPath := buildContainerLogsPath(container.Name, restartCount)
	restartCountUint32 := uint32(restartCount)
	config := &runtimeapi.ContainerConfig{
		Metadata: &runtimeapi.ContainerMetadata{
			Name:    container.Name,
			Attempt: restartCountUint32,
		},
		Image:       &runtimeapi.ImageSpec{Image: imageRef},
		Command:     command,
		Args:        args,
		WorkingDir:  container.WorkingDir,
		Labels:      newContainerLabels(container, pod, containerType),
		Annotations: newContainerAnnotations(container, pod, restartCount, opts),
		Devices:     makeDevices(opts),
		Mounts:      m.makeMounts(opts, container),
		LogPath:     containerLogsPath,
		Stdin:       container.Stdin,
		StdinOnce:   container.StdinOnce,
		Tty:         container.TTY,
	}

	// add anonymous volume which from annotation
	// https://lark.alipay.com/sigma.pouch/sigma3.x/ia25vq
	anonymousVolumeFromAnnotation := getAnonymousVolumesMount(pod)
	config.Mounts = append(config.Mounts, anonymousVolumeFromAnnotation...)

	// set platform specific configurations.
	if err := m.applyPlatformSpecificContainerConfig(config, container, pod, uid, username); err != nil {
		return nil, cleanupAction, err
	}
	// Reuse the quota ID if found from pod annotation
	if quotaId := GetDiskQuotaID(pod); quotaId != nil {
		config.QuotaId = *quotaId
	}

	// set environment variables
	envs := make([]*runtimeapi.KeyValue, len(opts.Envs))
	for idx := range opts.Envs {
		e := opts.Envs[idx]
		envs[idx] = &runtimeapi.KeyValue{
			Key:   e.Name,
			Value: e.Value,
		}
	}
	hostName := ""
	if opts.Hostname != "" && opts.HostDomain != "" {
		hostName = fmt.Sprintf("%s.%s", opts.Hostname, opts.HostDomain)
	} else if opts.Hostname != "" {
		hostName = opts.Hostname
	}
	if hostName != "" {
		// add hostname to env
		envs = append(envs, &runtimeapi.KeyValue{
			Key:   "HOSTNAME",
			Value: hostName,
		})
	}

	// Support modifying cgroup in container if privileged is false or undefined.
	if !(container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged == true) {
		config.Labels[containerPouchSupportCgroupLabel] = "true"
	}

	isHostDNS := sigmautil.IsPodHostDNSMode(pod)
	if isHostDNS {
		config.Labels[containerPouchCopyPodHostsLabel] = "true"
	}

	// Add network env on sigmalet side.
	// See: https://yuque.antfin-inc.com/sys/sigma3.x/tgpuxq
	// The network can't be changed in pod's lifecycle.
	if utilfeature.DefaultFeatureGate.Enabled(features.NetworkEnv) && !pod.Spec.HostNetwork {
		newPod, exists := m.podManager.GetPodByUID(pod.UID)
		if !exists {
			return nil, cleanupAction, fmt.Errorf("Failed to get new pod %s from podManager", format.Pod(pod))
		}
		networkEnvs := generateNetworkEnvs(pod, m.podManager)
		if len(networkEnvs) == 0 {
			return nil, cleanupAction, fmt.Errorf("Failed to get network status or invalid network status got from pod: %s in podManager", format.Pod(newPod))
		}
		envs = append(envs, networkEnvs...)

		node, err := m.runtimeHelper.GetNode()
		if err != nil || node == nil {
			return nil, cleanupAction, fmt.Errorf("Failed to get node: %v", err)
		}

		topologyEnvs := generateTopologyEnvs(node)
		envs = append(envs, topologyEnvs...)

		// Get pod ip from envs and add ip to config labels.
		podIP := ""
		labelPodIPKey := "PodIP"
		for _, networkEnv := range networkEnvs {
			if networkEnv.Key == envRequestIP {
				podIP = networkEnv.Value
				break
			}
		}
		if podIP == "" {
			glog.Warningf("Can't get pod ip from network status of pod: %s in podManager", format.Pod(newPod))
		}
		config.Labels[labelPodIPKey] = podIP
	}

	config.Envs = envs
	return config, cleanupAction, nil
}

// returns linux container resources to the container.
func (m *kubeGenericRuntimeManager) generateLinuxContainerResources(container *v1.Container, pod *v1.Pod) (*runtimeapi.LinuxContainerResources, error) {
	// Get image status
	imageSpec := &runtimeapi.ImageSpec{
		Image: container.Image,
	}
	imageStatus, err := m.imageService.ImageStatus(imageSpec)
	if err != nil {
		return nil, err
	}
	if imageStatus == nil {
		msg := fmt.Sprintf("image %s not found", container.Image)
		return nil, errors.New(msg)
	}

	return m.generateLinuxContainerConfig(container, pod, imageStatus, nil, "").Resources, nil
}

// makeDevices generates container devices for kubelet runtime v1.
func makeDevices(opts *kubecontainer.RunContainerOptions) []*runtimeapi.Device {
	devices := make([]*runtimeapi.Device, len(opts.Devices))

	for idx := range opts.Devices {
		device := opts.Devices[idx]
		devices[idx] = &runtimeapi.Device{
			HostPath:      device.PathOnHost,
			ContainerPath: device.PathInContainer,
			Permissions:   device.Permissions,
		}
	}

	return devices
}

// makeMounts generates container volume mounts for kubelet runtime v1.
func (m *kubeGenericRuntimeManager) makeMounts(opts *kubecontainer.RunContainerOptions, container *v1.Container) []*runtimeapi.Mount {
	volumeMounts := []*runtimeapi.Mount{}

	for idx := range opts.Mounts {
		v := opts.Mounts[idx]
		selinuxRelabel := v.SELinuxRelabel && selinux.SELinuxEnabled()
		mount := &runtimeapi.Mount{
			HostPath:       v.HostPath,
			ContainerPath:  v.ContainerPath,
			Readonly:       v.ReadOnly,
			SelinuxRelabel: selinuxRelabel,
			Propagation:    v.Propagation,
		}

		volumeMounts = append(volumeMounts, mount)
	}

	// The reason we create and mount the log file in here (not in kubelet) is because
	// the file's location depends on the ID of the container, and we need to create and
	// mount the file before actually starting the container.
	if opts.PodContainerDir != "" && len(container.TerminationMessagePath) != 0 {
		// Because the PodContainerDir contains pod uid and container name which is unique enough,
		// here we just add a random id to make the path unique for different instances
		// of the same container.
		cid := makeUID()
		containerLogPath := filepath.Join(opts.PodContainerDir, cid)
		fs, err := m.osInterface.Create(containerLogPath)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("error on creating termination-log file %q: %v", containerLogPath, err))
		} else {
			fs.Close()

			// Chmod is needed because ioutil.WriteFile() ends up calling
			// open(2) to create the file, so the final mode used is "mode &
			// ~umask". But we want to make sure the specified mode is used
			// in the file no matter what the umask is.
			if err := m.osInterface.Chmod(containerLogPath, 0666); err != nil {
				utilruntime.HandleError(fmt.Errorf("unable to set termination-log file permissions %q: %v", containerLogPath, err))
			}

			selinuxRelabel := selinux.SELinuxEnabled()
			volumeMounts = append(volumeMounts, &runtimeapi.Mount{
				HostPath:       containerLogPath,
				ContainerPath:  container.TerminationMessagePath,
				SelinuxRelabel: selinuxRelabel,
			})
		}
	}

	return volumeMounts
}

// getKubeletContainers lists containers managed by kubelet.
// The boolean parameter specifies whether returns all containers including
// those already exited and dead containers (used for garbage collection).
func (m *kubeGenericRuntimeManager) getKubeletContainers(allContainers bool) ([]*runtimeapi.Container, error) {
	filter := &runtimeapi.ContainerFilter{}
	if !allContainers {
		filter.State = &runtimeapi.ContainerStateValue{
			State: runtimeapi.ContainerState_CONTAINER_RUNNING,
		}
	}

	containers, err := m.runtimeService.ListContainers(filter)
	if err != nil {
		glog.Errorf("getKubeletContainers failed: %v", err)
		return nil, err
	}

	return containers, nil
}

// makeUID returns a randomly generated string.
func makeUID() string {
	return fmt.Sprintf("%08x", rand.Uint32())
}

// getTerminationMessage looks on the filesystem for the provided termination message path, returning a limited
// amount of those bytes, or returns true if the logs should be checked.
func getTerminationMessage(status *runtimeapi.ContainerStatus, terminationMessagePath string, fallbackToLogs bool) (string, bool) {
	if len(terminationMessagePath) == 0 {
		return "", fallbackToLogs
	}
	for _, mount := range status.Mounts {
		if mount.ContainerPath != terminationMessagePath {
			continue
		}
		path := mount.HostPath
		data, _, err := tail.ReadAtMost(path, kubecontainer.MaxContainerTerminationMessageLength)
		if err != nil {
			if os.IsNotExist(err) {
				return "", fallbackToLogs
			}
			return fmt.Sprintf("Error on reading termination log %s: %v", path, err), false
		}
		return string(data), (fallbackToLogs && len(data) == 0)
	}
	return "", fallbackToLogs
}

// readLastStringFromContainerLogs attempts to read up to the max log length from the end of the CRI log represented
// by path. It reads up to max log lines.
func (m *kubeGenericRuntimeManager) readLastStringFromContainerLogs(path string) string {
	value := int64(kubecontainer.MaxContainerTerminationMessageLogLines)
	buf, _ := circbuf.NewBuffer(kubecontainer.MaxContainerTerminationMessageLogLength)
	if err := m.ReadLogs(context.Background(), path, "", &v1.PodLogOptions{TailLines: &value}, buf, buf); err != nil {
		return fmt.Sprintf("Error on reading termination message from logs: %v", err)
	}
	return buf.String()
}

// getPodContainerStatuses gets all containers' statuses for the pod.
func (m *kubeGenericRuntimeManager) getPodContainerStatuses(uid kubetypes.UID, name, namespace string) ([]*kubecontainer.ContainerStatus, error) {
	// Select all containers of the given pod.
	containers, err := m.runtimeService.ListContainers(&runtimeapi.ContainerFilter{
		LabelSelector: map[string]string{types.KubernetesPodUIDLabel: string(uid)},
	})
	if err != nil {
		glog.Errorf("ListContainers error: %v", err)
		return nil, err
	}

	statuses := make([]*kubecontainer.ContainerStatus, len(containers))
	// TODO: optimization: set maximum number of containers per container name to examine.
	for i, c := range containers {
		status, err := m.runtimeService.ContainerStatus(c.Id)
		if err != nil {
			glog.Errorf("ContainerStatus for %s error: %v", c.Id, err)
			return nil, err
		}
		cStatus := toKubeContainerStatus(status, m.runtimeName)
		if status.State == runtimeapi.ContainerState_CONTAINER_EXITED {
			// Populate the termination message if needed.
			annotatedInfo := getContainerInfoFromAnnotations(status.Annotations)
			labeledInfo := getContainerInfoFromLabels(status.Labels)
			fallbackToLogs := annotatedInfo.TerminationMessagePolicy == v1.TerminationMessageFallbackToLogsOnError && cStatus.ExitCode != 0
			tMessage, checkLogs := getTerminationMessage(status, annotatedInfo.TerminationMessagePath, fallbackToLogs)
			if checkLogs {
				// if dockerLegacyService is populated, we're supposed to use it to fetch logs
				if m.legacyLogProvider != nil {
					tMessage, err = m.legacyLogProvider.GetContainerLogTail(uid, name, namespace, kubecontainer.ContainerID{Type: m.runtimeName, ID: c.Id})
					if err != nil {
						tMessage = fmt.Sprintf("Error reading termination message from logs: %v", err)
					}
				} else {
					path := buildFullContainerLogsPath(uid, labeledInfo.ContainerName, annotatedInfo.RestartCount)
					tMessage = m.readLastStringFromContainerLogs(path)
				}
			}
			// Use the termination message written by the application is not empty
			if len(tMessage) != 0 {
				cStatus.Message = tMessage
			}
		}
		statuses[i] = cStatus
	}

	sort.Sort(containerStatusByCreated(statuses))
	return statuses, nil
}

func toKubeContainerStatus(status *runtimeapi.ContainerStatus, runtimeName string) *kubecontainer.ContainerStatus {
	annotatedInfo := getContainerInfoFromAnnotations(status.Annotations)
	labeledInfo := getContainerInfoFromLabels(status.Labels)
	cStatus := &kubecontainer.ContainerStatus{
		ID: kubecontainer.ContainerID{
			Type: runtimeName,
			ID:   status.Id,
		},
		Name:         labeledInfo.ContainerName,
		Image:        status.Image.Image,
		ImageID:      status.ImageRef,
		Hash:         annotatedInfo.Hash,
		HashVersion:  annotatedInfo.HashVersion,
		RestartCount: annotatedInfo.RestartCount,
		State:        toKubeContainerState(status.State),
		Resources:    status.Resources,
		CreatedAt:    time.Unix(0, status.CreatedAt),
		Envs:         status.Envs,
	}

	if status.State != runtimeapi.ContainerState_CONTAINER_CREATED {
		// If container is not in the created state, we have tried and
		// started the container. Set the StartedAt time.
		cStatus.StartedAt = time.Unix(0, status.StartedAt)
	}
	if status.State == runtimeapi.ContainerState_CONTAINER_EXITED {
		cStatus.Reason = status.Reason
		cStatus.Message = status.Message
		cStatus.ExitCode = int(status.ExitCode)
		cStatus.FinishedAt = time.Unix(0, status.FinishedAt)
	}
	return cStatus
}

// executePreStopHook runs the pre-stop lifecycle hooks if applicable and returns the duration it takes.
func (m *kubeGenericRuntimeManager) executePreStopHook(pod *v1.Pod, containerID kubecontainer.ContainerID, containerSpec *v1.Container, gracePeriod int64) int64 {
	glog.V(3).Infof("Running preStop hook for container %q", containerID.String())

	start := metav1.Now()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer utilruntime.HandleCrash()
		if msg, err := m.runner.Run(containerID, pod, containerSpec, containerSpec.Lifecycle.PreStop, 0); err != nil {
			glog.Errorf("preStop hook for container %q failed: %v", containerSpec.Name, err)
			m.recordContainerEvent(pod, containerSpec, containerID.ID, v1.EventTypeWarning, events.FailedPreStopHook, msg)
		}
	}()

	select {
	case <-time.After(time.Duration(gracePeriod) * time.Second):
		glog.V(2).Infof("preStop hook for container %q did not complete in %d seconds", containerID, gracePeriod)
	case <-done:
		glog.V(3).Infof("preStop hook for container %q completed", containerID)
	}

	return int64(metav1.Now().Sub(start.Time).Seconds())
}

// restoreSpecsFromContainerLabels restores all information needed for killing a container. In some
// case we may not have pod and container spec when killing a container, e.g. pod is deleted during
// kubelet restart.
// To solve this problem, we've already written necessary information into container labels. Here we
// just need to retrieve them from container labels and restore the specs.
// TODO(random-liu): Add a node e2e test to test this behaviour.
// TODO(random-liu): Change the lifecycle handler to just accept information needed, so that we can
// just pass the needed function not create the fake object.
func (m *kubeGenericRuntimeManager) restoreSpecsFromContainerLabels(containerID kubecontainer.ContainerID) (*v1.Pod, *v1.Container, error) {
	var pod *v1.Pod
	var container *v1.Container
	s, err := m.runtimeService.ContainerStatus(containerID.ID)
	if err != nil {
		return nil, nil, err
	}

	l := getContainerInfoFromLabels(s.Labels)
	a := getContainerInfoFromAnnotations(s.Annotations)
	// Notice that the followings are not full spec. The container killing code should not use
	// un-restored fields.
	pod = &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:                        l.PodUID,
			Name:                       l.PodName,
			Namespace:                  l.PodNamespace,
			DeletionGracePeriodSeconds: a.PodDeletionGracePeriod,
		},
		Spec: v1.PodSpec{
			TerminationGracePeriodSeconds: a.PodTerminationGracePeriod,
		},
	}
	container = &v1.Container{
		Name:  l.ContainerName,
		Ports: a.ContainerPorts,
		TerminationMessagePath: a.TerminationMessagePath,
	}
	if a.PreStopHandler != nil {
		container.Lifecycle = &v1.Lifecycle{
			PreStop: a.PreStopHandler,
		}
	}
	return pod, container, nil
}

// killContainer kills a container through the following steps:
// * Run the pre-stop lifecycle hooks (if applicable).
// * Stop the container.
func (m *kubeGenericRuntimeManager) killContainer(pod *v1.Pod, containerID kubecontainer.ContainerID, containerName string, reason string, gracePeriodOverride *int64) error {
	var containerSpec *v1.Container
	if pod != nil {
		if containerSpec = kubecontainer.GetContainerSpec(pod, containerName); containerSpec == nil {
			return fmt.Errorf("failed to get containerSpec %q(id=%q) in pod %q when killing container for reason %q",
				containerName, containerID.String(), format.Pod(pod), reason)
		}
	} else {
		// Restore necessary information if one of the specs is nil.
		restoredPod, restoredContainer, err := m.restoreSpecsFromContainerLabels(containerID)
		if err != nil {
			return err
		}
		pod, containerSpec = restoredPod, restoredContainer
	}

	// From this point , pod and container must be non-nil.
	gracePeriod := int64(minimumGracePeriodInSeconds)
	switch {
	case pod.DeletionGracePeriodSeconds != nil:
		gracePeriod = *pod.DeletionGracePeriodSeconds
	case pod.Spec.TerminationGracePeriodSeconds != nil:
		gracePeriod = *pod.Spec.TerminationGracePeriodSeconds
	}

	glog.V(2).Infof("Killing container %q with %d second grace period", containerID.String(), gracePeriod)

	// Run internal pre-stop lifecycle hook
	if err := m.internalLifecycle.PreStopContainer(containerID.ID); err != nil {
		return err
	}

	timeout := int64(sigmautil.GetTimeoutSecondsFromPodAnnotation(pod, containerName, sigmak8sapi.PreStopHookTimeoutSeconds))
	glog.V(2).Infof("Executing container %q pre-stop hook with %d second grace period for container %s in pod %s", containerID.String(), timeout, containerName, format.Pod(pod))

	// Run the pre-stop lifecycle hooks if applicable and if there is enough time to run it
	if containerSpec.Lifecycle != nil && containerSpec.Lifecycle.PreStop != nil && gracePeriod > 0 {
		if timeout > 0 {
			m.executePreStopHook(pod, containerID, containerSpec, timeout)
			gracePeriod = gracePeriod - timeout
		} else {
			gracePeriod = gracePeriod - m.executePreStopHook(pod, containerID, containerSpec, gracePeriod)
		}
	}
	// always give containers a minimal shutdown window to avoid unnecessary SIGKILLs
	if gracePeriod < minimumGracePeriodInSeconds {
		gracePeriod = minimumGracePeriodInSeconds
	}
	if gracePeriodOverride != nil {
		gracePeriod = *gracePeriodOverride
		glog.V(3).Infof("Killing container %q, but using %d second grace period override", containerID, gracePeriod)
	}

	err := m.runtimeService.StopContainer(containerID.ID, gracePeriod)
	if err != nil {
		glog.Errorf("Container %q termination failed with gracePeriod %d: %v", containerID.String(), gracePeriod, err)
	} else {
		glog.V(3).Infof("Container %q exited normally", containerID.String())
	}

	message := fmt.Sprintf("Killing container with id %s", containerID.String())
	if reason != "" {
		message = fmt.Sprint(message, ":", reason)
	}
	m.recordContainerEvent(pod, containerSpec, containerID.ID, v1.EventTypeNormal, events.KillingContainer, message)
	m.containerRefManager.ClearRef(containerID)

	return err
}

// killContainersWithSyncResult kills all pod's containers with sync results.
func (m *kubeGenericRuntimeManager) killContainersWithSyncResult(pod *v1.Pod, runningPod kubecontainer.Pod, gracePeriodOverride *int64) (syncResults []*kubecontainer.SyncResult) {
	containerResults := make(chan *kubecontainer.SyncResult, len(runningPod.Containers))
	wg := sync.WaitGroup{}

	wg.Add(len(runningPod.Containers))
	for _, container := range runningPod.Containers {
		go func(container *kubecontainer.Container) {
			defer utilruntime.HandleCrash()
			defer wg.Done()

			killContainerResult := kubecontainer.NewSyncResult(kubecontainer.KillContainer, container.Name)
			if err := m.killContainer(pod, container.ID, container.Name, "Need to kill Pod", gracePeriodOverride); err != nil {
				killContainerResult.Fail(kubecontainer.ErrKillContainer, err.Error())
			}
			containerResults <- killContainerResult
		}(container)
	}
	wg.Wait()
	close(containerResults)

	for containerResult := range containerResults {
		syncResults = append(syncResults, containerResult)
	}
	return
}

// pruneInitContainersBeforeStart ensures that before we begin creating init
// containers, we have reduced the number of outstanding init containers still
// present. This reduces load on the container garbage collector by only
// preserving the most recent terminated init container.
func (m *kubeGenericRuntimeManager) pruneInitContainersBeforeStart(pod *v1.Pod, podStatus *kubecontainer.PodStatus) {
	// only the last execution of each init container should be preserved, and only preserve it if it is in the
	// list of init containers to keep.
	initContainerNames := sets.NewString()
	for _, container := range pod.Spec.InitContainers {
		initContainerNames.Insert(container.Name)
	}
	for name := range initContainerNames {
		count := 0
		for _, status := range podStatus.ContainerStatuses {
			if status.Name != name || !initContainerNames.Has(status.Name) || status.State != kubecontainer.ContainerStateExited {
				continue
			}
			count++
			// keep the first init container for this name
			if count == 1 {
				continue
			}
			// prune all other init containers that match this container name
			glog.V(4).Infof("Removing init container %q instance %q %d", status.Name, status.ID.ID, count)
			if err := m.removeContainer(status.ID.ID); err != nil {
				utilruntime.HandleError(fmt.Errorf("failed to remove pod init container %q: %v; Skipping pod %q", status.Name, err, format.Pod(pod)))
				continue
			}

			// remove any references to this container
			if _, ok := m.containerRefManager.GetRef(status.ID); ok {
				m.containerRefManager.ClearRef(status.ID)
			} else {
				glog.Warningf("No ref for container %q", status.ID)
			}
		}
	}
}

// Remove all init containres. Note that this function does not check the state
// of the container because it assumes all init containers have been stopped
// before the call happens.
func (m *kubeGenericRuntimeManager) purgeInitContainers(pod *v1.Pod, podStatus *kubecontainer.PodStatus) {
	initContainerNames := sets.NewString()
	for _, container := range pod.Spec.InitContainers {
		initContainerNames.Insert(container.Name)
	}
	for name := range initContainerNames {
		count := 0
		for _, status := range podStatus.ContainerStatuses {
			if status.Name != name || !initContainerNames.Has(status.Name) {
				continue
			}
			count++
			// Purge all init containers that match this container name
			glog.V(4).Infof("Removing init container %q instance %q %d", status.Name, status.ID.ID, count)
			if err := m.removeContainer(status.ID.ID); err != nil {
				utilruntime.HandleError(fmt.Errorf("failed to remove pod init container %q: %v; Skipping pod %q", status.Name, err, format.Pod(pod)))
				continue
			}
			// Remove any references to this container
			if _, ok := m.containerRefManager.GetRef(status.ID); ok {
				m.containerRefManager.ClearRef(status.ID)
			} else {
				glog.Warningf("No ref for container %q", status.ID)
			}
		}
	}
}

// findNextInitContainerToRun returns the status of the last failed container, the
// next init container to start, or done if there are no further init containers.
// Status is only returned if an init container is failed, in which case next will
// point to the current container.
func findNextInitContainerToRun(pod *v1.Pod, podStatus *kubecontainer.PodStatus) (status *kubecontainer.ContainerStatus, next *v1.Container, done bool) {
	if len(pod.Spec.InitContainers) == 0 {
		return nil, nil, true
	}

	// If there are failed containers, return the status of the last failed one.
	for i := len(pod.Spec.InitContainers) - 1; i >= 0; i-- {
		container := &pod.Spec.InitContainers[i]
		status := podStatus.FindContainerStatusByName(container.Name)
		if status != nil && isContainerFailed(status) {
			return status, container, false
		}
	}

	// There are no failed containers now.
	for i := len(pod.Spec.InitContainers) - 1; i >= 0; i-- {
		container := &pod.Spec.InitContainers[i]
		status := podStatus.FindContainerStatusByName(container.Name)
		if status == nil {
			continue
		}

		// container is still running, return not done.
		if status.State == kubecontainer.ContainerStateRunning {
			return nil, nil, false
		}

		if status.State == kubecontainer.ContainerStateExited {
			// all init containers successful
			if i == (len(pod.Spec.InitContainers) - 1) {
				return nil, nil, true
			}

			// all containers up to i successful, go to i+1
			return nil, &pod.Spec.InitContainers[i+1], false
		}
	}

	return nil, &pod.Spec.InitContainers[0], false
}

// GetContainerLogs returns logs of a specific container.
func (m *kubeGenericRuntimeManager) GetContainerLogs(ctx context.Context, pod *v1.Pod, containerID kubecontainer.ContainerID, logOptions *v1.PodLogOptions, stdout, stderr io.Writer) (err error) {
	status, err := m.runtimeService.ContainerStatus(containerID.ID)
	if err != nil {
		glog.V(4).Infof("failed to get container status for %v: %v", containerID.String(), err)
		return fmt.Errorf("Unable to retrieve container logs for %v", containerID.String())
	}
	return m.ReadLogs(ctx, status.GetLogPath(), containerID.ID, logOptions, stdout, stderr)
}

// GetExec gets the endpoint the runtime will serve the exec request from.
func (m *kubeGenericRuntimeManager) GetExec(id kubecontainer.ContainerID, cmd []string, stdin, stdout, stderr, tty bool) (*url.URL, error) {
	req := &runtimeapi.ExecRequest{
		ContainerId: id.ID,
		Cmd:         cmd,
		Tty:         tty,
		Stdin:       stdin,
		Stdout:      stdout,
		Stderr:      stderr,
	}
	resp, err := m.runtimeService.Exec(req)
	if err != nil {
		return nil, err
	}

	return url.Parse(resp.Url)
}

// GetAttach gets the endpoint the runtime will serve the attach request from.
func (m *kubeGenericRuntimeManager) GetAttach(id kubecontainer.ContainerID, stdin, stdout, stderr, tty bool) (*url.URL, error) {
	req := &runtimeapi.AttachRequest{
		ContainerId: id.ID,
		Stdin:       stdin,
		Stdout:      stdout,
		Stderr:      stderr,
		Tty:         tty,
	}
	resp, err := m.runtimeService.Attach(req)
	if err != nil {
		return nil, err
	}
	return url.Parse(resp.Url)
}

// RunInContainer synchronously executes the command in the container, and returns the output.
func (m *kubeGenericRuntimeManager) RunInContainer(id kubecontainer.ContainerID, cmd []string, timeout time.Duration) ([]byte, error) {
	stdout, stderr, err := m.runtimeService.ExecSync(id.ID, cmd, timeout)
	// NOTE(tallclair): This does not correctly interleave stdout & stderr, but should be sufficient
	// for logging purposes. A combined output option will need to be added to the ExecSyncRequest
	// if more precise output ordering is ever required.
	return append(stdout, stderr...), err
}

// removeContainer removes the container and the container logs.
// Notice that we remove the container logs first, so that container will not be removed if
// container logs are failed to be removed, and kubelet will retry this later. This guarantees
// that container logs to be removed with the container.
// Notice that we assume that the container should only be removed in non-running state, and
// it will not write container logs anymore in that state.
func (m *kubeGenericRuntimeManager) removeContainer(containerID string) error {
	glog.V(4).Infof("Removing container %q", containerID)
	// Call internal container post-stop lifecycle hook.
	if err := m.internalLifecycle.PostStopContainer(containerID); err != nil {
		return err
	}

	// Remove the container log.
	// TODO: Separate log and container lifecycle management.
	if err := m.removeContainerLog(containerID); err != nil {
		return err
	}
	// Remove the container.
	return m.runtimeService.RemoveContainer(containerID)
}

// removeContainerLog removes the container log.
func (m *kubeGenericRuntimeManager) removeContainerLog(containerID string) error {
	// Remove the container log.
	status, err := m.runtimeService.ContainerStatus(containerID)
	if err != nil {
		return fmt.Errorf("failed to get container status %q: %v", containerID, err)
	}
	labeledInfo := getContainerInfoFromLabels(status.Labels)
	annotatedInfo := getContainerInfoFromAnnotations(status.Annotations)
	path := buildFullContainerLogsPath(labeledInfo.PodUID, labeledInfo.ContainerName, annotatedInfo.RestartCount)
	if err := m.osInterface.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove container %q log %q: %v", containerID, path, err)
	}

	// Remove the legacy container log symlink.
	// TODO(random-liu): Remove this after cluster logging supports CRI container log path.
	legacySymlink := legacyLogSymlink(containerID, labeledInfo.ContainerName, labeledInfo.PodName,
		labeledInfo.PodNamespace)
	if err := m.osInterface.Remove(legacySymlink); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove container %q log legacy symbolic link %q: %v",
			containerID, legacySymlink, err)
	}
	return nil
}

// DeleteContainer removes a container.
func (m *kubeGenericRuntimeManager) DeleteContainer(containerID kubecontainer.ContainerID) error {
	return m.removeContainer(containerID.ID)
}
