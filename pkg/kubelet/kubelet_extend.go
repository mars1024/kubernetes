/*
Copyright 2018 The Kubernetes Authors.

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

package kubelet

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
)

func (kl *Kubelet) UpdatePodCgroup(pod *v1.Pod) error {
	return kl.containerManager.NewPodContainerManager().Update(pod)
}

func (kl *Kubelet) UpdatePodStatusCache(pod *v1.Pod) error {
	timestamp := kl.clock.Now()
	// GetPodStatus(pod *kubecontainer.Pod) so that Docker can avoid listing
	// all containers again.
	podStatus, err := kl.containerRuntime.GetPodStatus(pod.UID, pod.Name, pod.Namespace)
	glog.V(4).Infof("Write status for %s/%s: %#v (err: %v)", pod.Name, pod.Namespace, podStatus, err)
	if err == nil {
		// Preserve the pod IP across cache updates if the new IP is empty.
		// When a pod is torn down, kubelet may race with PLEG and retrieve
		// a pod status after network teardown, but the kubernetes API expects
		// the completed pod's IP to be available after the pod is dead.
		podStatus.IP = kl.getPodIP(pod.UID, podStatus)
	}

	kl.podCache.Set(pod.UID, podStatus, err, timestamp)
	return err
}

// getPodIP preserves an older cached status' pod IP if the new status has no pod IP
// and its sandboxes have exited
func (kl *Kubelet) getPodIP(pid types.UID, status *kubecontainer.PodStatus) string {
	if status.IP != "" {
		return status.IP
	}

	oldStatus, err := kl.podCache.Get(pid)
	if err != nil || oldStatus.IP == "" {
		return ""
	}

	for _, sandboxStatus := range status.SandboxStatuses {
		// If at least one sandbox is ready, then use this status update's pod IP
		if sandboxStatus.State == runtimeapi.PodSandboxState_SANDBOX_READY {
			return status.IP
		}
	}

	if len(status.SandboxStatuses) == 0 {
		// Without sandboxes (which built-in runtimes like rkt don't report)
		// look at all the container statuses, and if any containers are
		// running then use the new pod IP
		for _, containerStatus := range status.ContainerStatuses {
			if containerStatus.State == kubecontainer.ContainerStateCreated || containerStatus.State == kubecontainer.ContainerStateRunning {
				return status.IP
			}
		}
	}

	// For pods with no ready containers or sandboxes (like exited pods)
	// use the old status' pod IP
	return oldStatus.IP
}

// skipAdmit judge pod whether skip admit.
// * if pod is terminated, return true
// * if pod have rebuild-container-info annotation  which represent container 2.0 to 3.1, return true
// * if pod have cni-allocated finalizer which represent pod was created in this node, return true
// * if pod have update status  annotation which represent pod was created in this node, return true
// In theory, if a pod have a cni-allocated finalizer,it must have update status annotation,
// but exist scenarios where kubelet restart after sandbox created before container created,
// pod only have cni-allocated finalizer
func (kl *Kubelet) skipAdmit(pod *v1.Pod) bool {
	if kl.podIsTerminated(pod) {
		glog.V(2).Infof("pod %s is terminated pod, skip admit", format.Pod(pod))
		return true
	}

	if PodHaveRebuildAnnotation(pod) {
		glog.V(2).Infof("pod %s have rebuild annotation,skip admit", format.Pod(pod))
		return true
	}

	if PodHaveCNIAllocatedFinalizer(pod) {
		glog.V(2).Infof("pod %s have cni allocated finalizer, skip admit", format.Pod(pod))
		return true
	}

	if PodHaveUpdateStatus(pod) {
		glog.V(2).Infof("pod %s have update status annotation which represent pod already exist in node,"+
			"should skip admit", format.Pod(pod))
		return true
	}
	glog.V(2).Infof("pod %s should admit", format.Pod(pod))
	return false
}

// PodHaveCNIAllocatedFinalizer judge pod have cni allocated finalizer.
func PodHaveUpdateStatus(pod *v1.Pod) bool {
	if pod == nil || len(pod.Annotations) == 0 {
		glog.V(4).Infof("pod is %v nil, or pod annotation  len is zero", pod)
		return false
	}

	_, ok := pod.Annotations[sigmak8sapi.AnnotationPodUpdateStatus]
	glog.V(4).Infof("pod %s have %s annotation %t", format.Pod(pod), sigmak8sapi.AnnotationPodUpdateStatus, ok)
	return ok
}

// PodHaveRebuildAnnotation judge pod have rebuild container info annotation
func PodHaveRebuildAnnotation(pod *v1.Pod) bool {
	if pod == nil || len(pod.Annotations) == 0 {
		glog.V(4).Infof("pod is %v nil, or pod annotation  len is zero", pod)
		return false
	}
	_, exist := pod.Annotations[sigmak8sapi.AnnotationRebuildContainerInfo]
	if exist {
		glog.V(4).Infof("pod %s have %s annotation %t", format.Pod(pod), sigmak8sapi.AnnotationRebuildContainerInfo)
		return true
	}
	return false
}

// updateStateStatus will update pod's state status annotation with new container's state status.
func (kl *Kubelet) updateStateStatus(pod *v1.Pod, result kubecontainer.PodSyncResult, retry int, triesBeforeBackOff int, backOffPeriod time.Duration) error {
	// Get latest previous StateStatus from pod and the new StateStatus from result.
	podAnnotationValue := getAnnotationValue(pod.GetAnnotations(), sigmak8sapi.AnnotationPodUpdateStatus)
	latestStateStatus :=
		sigmak8sapi.ContainerStateStatus{Statuses: make(map[sigmak8sapi.ContainerInfo]sigmak8sapi.ContainerStatus)}
	resultStateStatus := result.StateStatus
	if err := json.Unmarshal([]byte(podAnnotationValue), &latestStateStatus); err != nil {
		glog.Warningf("Failed to unmarshal state status %s for pod %s, err: %v. Generate new state status instead.",
			podAnnotationValue, format.Pod(pod), err)
	}

	// Merge new statuses to previous StateStatus.
	updated := false
	for containerInfo, stateStatus := range resultStateStatus.Statuses {
		oldStateStatus, exists := latestStateStatus.Statuses[containerInfo]
		if !exists || stateStatus.FinishTimestamp.After(oldStateStatus.FinishTimestamp) {
			latestStateStatus.Statuses[containerInfo] = stateStatus
			updated = true
		}
	}

	// If not updated, just return.
	if !updated {
		return nil
	}
	// Generate patchData
	updateAnnotationBytes, err := json.Marshal(latestStateStatus)
	updateAnnotationValue := string(updateAnnotationBytes)
	if err != nil {
		glog.Errorf("Failed to marshal pod %s update status: %v", format.Pod(pod), updateAnnotationValue)
		return err
	}
	patchData := fmt.Sprintf(
		`{"metadata":{"annotations":{"%s":%q}}}`, sigmak8sapi.AnnotationPodUpdateStatus, updateAnnotationValue)
	// Update pod's StateStatus annotation.
	for i := 0; i < retry; i++ {
		if i > triesBeforeBackOff {
			time.Sleep(backOffPeriod)
		}
		_, err = kl.kubeClient.CoreV1().Pods(pod.GetNamespace()).Patch(pod.GetName(), types.StrategicMergePatchType, []byte(patchData))
		if err != nil && errors.IsConflict(err) {
			glog.Warningf("pod annotation change ,update pod ：%s, conflict err", format.Pod(pod))
			continue
		}
		glog.V(4).Infof("pod %s annotation changed, before syncPod value: %q, after syncPod value: %q",
			format.Pod(pod), podAnnotationValue, updateAnnotationValue)
		break
	}

	return err
}

// getAnnotationValue gets the value corresponding to the specified key.
func getAnnotationValue(annotation map[string]string, key string) string {
	if len(annotation) <= 0 {
		return ""
	}
	value, _ := annotation[key]
	return value
}