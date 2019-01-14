package kubelet

import (
	"reflect"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	kubepod "k8s.io/kubernetes/pkg/kubelet/pod"
	sigmautil "k8s.io/kubernetes/pkg/kubelet/sigma"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"
)

// getRunningPodByUID can get pod from runtime by uid.
func (kl *Kubelet) getRunningPodByUID(uid string) *kubecontainer.Pod {
	// Get runningPods from runtime.
	runningPods, err := kl.containerRuntime.GetPods(false)
	if err != nil {
		glog.Errorf("[DanglingPod] Error listing containers: %#v", err)
		return nil
	}
	for _, runningPod := range runningPods {
		if string(runningPod.ID) == uid {
			return runningPod
		}
	}
	return nil
}

// getRunningPodByName can get pod from runtime by name and namespace
func (kl *Kubelet) getRunningPodByName(name, namespace string) *kubecontainer.Pod {
	// Get runningPods from runtime.
	runningPods, err := kl.containerRuntime.GetPods(false)
	if err != nil {
		glog.Errorf("[DanglingPod] Error listing containers: %#v", err)
		return nil
	}
	for _, runningPod := range runningPods {
		if runningPod.Name == name && runningPod.Namespace == namespace {
			return runningPod
		}
	}
	return nil
}

// isPodFromAPIServerSource return true if pod's source is apiserver; else return false.
func isPodFromAPIServerSource(pod *v1.Pod) bool {
	if pod.Annotations != nil {
		if source, exists := pod.Annotations[kubetypes.ConfigSourceAnnotationKey]; exists &&
			source == kubetypes.ApiserverSource && !kubepod.IsMirrorPod(pod) {
			return true
		}
	}
	return false
}

func generateDanglingPodKey(name, namespace, uid string) string {
	return name + "_" + namespace + "_" + uid
}

// 1 Get dangling pods from node's annotation.
//   If the dangling pod's SafeToRemove is true，then this dangling pod will be deleted.
// 2 Update current dangling pods into node's annotation.
func (kl *Kubelet) SyncDanglingPods() {
	// If dangling pod's SafeToRemove filed is marked as true, it means that this dangling pod is safe to remove.
	danglingPods, err := sigmautil.GetDanglingPods(kl.kubeClient, string(kl.nodeName))
	if err != nil {
		glog.Infof("[DanglingPod] Failed to get danglingPods from apiserver, error: %v", err)
		return
	}
	// Used to check is there a need to update danglingPods.
	danglingPodsMap := map[string]sigmak8sapi.DanglingPod{}
	// Record the "unSafeToRemove" danglingPods.
	remainingDanglingPods := map[string]sigmak8sapi.DanglingPod{}
	// Record the "SafeToRemove" danglingPods.
	deletingDanglingPods := map[string]sigmak8sapi.DanglingPod{}
	for _, danglingPod := range danglingPods {
		key := generateDanglingPodKey(danglingPod.Name, danglingPod.Namespace, danglingPod.UID)
		danglingPodsMap[key] = danglingPod
		if !danglingPod.SafeToRemove {
			remainingDanglingPods[key] = danglingPod
			continue
		}

		deletingDanglingPods[key] = danglingPod

		// If dangling pod is in podManager, clean related resources.
		if pod, exists := kl.podManager.GetPodByUID(types.UID(danglingPod.UID)); exists {
			kl.podManager.DeletePod(pod)
			kl.probeManager.RemovePod(pod)
		}
		// Delete dangling pod.
		runningPod := kl.getRunningPodByUID(danglingPod.UID)
		if runningPod != nil {
			glog.V(0).Infof("[DanglingPod] Dangling pod %s will be removed", kubecontainer.FormatPod(runningPod))
			kl.podKillingCh <- &kubecontainer.PodPair{APIPod: nil, RunningPod: runningPod}
		}
	}

	// Update the dangling pods.
	activePods, err := kl.containerRuntime.GetPods(false)
	if err != nil {
		glog.Errorf("[DanglingPod] Error listing containers: %#v", err)
		return
	}
	currentDanglingPods := map[string]sigmak8sapi.DanglingPod{}
	// validPods records the pod can find in apiserver.
	validPods := map[string]struct{}{}
	for _, pod := range activePods {
		// If we can get pod from apiserver or the error is not "IsNotFound", just skip.

		getPod, err := kl.kubeClient.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err == nil {
			podKey := generateDanglingPodKey(getPod.Name, getPod.Namespace, string(getPod.UID))
			validPods[podKey] = struct{}{}
			continue
		}

		if !errors.IsNotFound(err) {
			continue
		}

		// Basic information from container, guaranteed.
		danglingPod := sigmak8sapi.DanglingPod{
			Name:         pod.Name,
			Namespace:    pod.Namespace,
			UID:          string(pod.ID),
			SafeToRemove: false,
			Phase:        v1.PodRunning,
		}

		key := generateDanglingPodKey(danglingPod.Name, danglingPod.Namespace, danglingPod.UID)

		// Set SafeToRemove if danglingPod is safe to remove.
		if _, exists := deletingDanglingPods[key]; exists {
			danglingPod.SafeToRemove = true
		}

		// Extra information from podManager, not guaranteed.
		if apiPod, exists := kl.podManager.GetPodByUID(pod.ID); exists {
			danglingPod.CreationTimestamp = apiPod.CreationTimestamp
			podSN, _ := sigmautil.GetSNFromLabel(apiPod)
			danglingPod.SN = podSN

			danglingPod.PodIP = apiPod.Status.PodIP
		} else if previousDangingPod, exists := remainingDanglingPods[key]; exists {
			// Try to get information from remainingDanglingPods.
			danglingPod.CreationTimestamp = previousDangingPod.CreationTimestamp
			danglingPod.SN = previousDangingPod.SN

			danglingPod.PodIP = previousDangingPod.PodIP
		} else if previousDangingPod, exists := deletingDanglingPods[key]; exists {
			// Try to get information from deletingDanglingPods.
			danglingPod.CreationTimestamp = previousDangingPod.CreationTimestamp
			danglingPod.SN = previousDangingPod.SN

			danglingPod.PodIP = previousDangingPod.PodIP
		}

		currentDanglingPods[key] = danglingPod
	}

	// TODO: Find another way to keep dangingPod.
	// Add the "unSafeToRemove" danglingPod to currentDanglingPods.
	//	for key, danglingPod := range remainingDanglingPods {
	//		// Skip the danglingPod can be find in apiserver.
	//		if _, exists := validPods[key]; exists {
	//			continue
	//		}
	//		if _, exists := currentDanglingPods[key]; !exists {
	//			glog.V(0).Infof("[DanglingPod] DanglingPod %s is terminated unexpectly", key)
	//			// Mark danglingPod's phase as Unknown because it isn't in runtime(stopped or deleted).
	//			danglingPod.Phase = v1.PodUnknown
	//			currentDanglingPods[key] = danglingPod
	//		}
	//	}

	// Update danglingPods to node's annotation.
	if !reflect.DeepEqual(currentDanglingPods, danglingPodsMap) {
		updateDanglingPods := []sigmak8sapi.DanglingPod{}
		for _, danglingPod := range currentDanglingPods {
			updateDanglingPods = append(updateDanglingPods, danglingPod)
		}
		sigmautil.UpdateDanglingPods(kl.kubeClient, string(kl.nodeName), updateDanglingPods)
	}

	return
}
