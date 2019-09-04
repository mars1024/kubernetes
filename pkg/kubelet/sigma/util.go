package sigma

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/util/format"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

const (
	// Define protection finalizer naming convention, any finalizer that name matches
	// this will be regard as protection finalizer, and
	// kubelet will not upgrade pod when such finalizer exists.
	FinalizerPodUpgradeProtectionFmt = "protection-upgrade.pod.beta1.sigma.ali/*"

	// Define protection finalizer naming convention, any finalizer that name matches
	// this will be regard as protection finalizer, and
	// kubelet will not delete pod when such finalizer exists.
	FinalizerPodDeleteProtectionFmt = "protection-delete.pod.beta1.sigma.ali/*"
)

var protectionFinalizerRegexp = regexp.MustCompile(sigmak8sapi.FinalizerPodProtectionFmt)

// if pod finalizer match size, it should not upgrade
var protectionUpgradeFinalizerRegexp = regexp.MustCompile(FinalizerPodUpgradeProtectionFmt)

// if pod finalizer match size, it should not delete
var protectionDeleteFinalizerRegexp = regexp.MustCompile(FinalizerPodDeleteProtectionFmt)

// HasProtectionFinalizer returns true if pod has any protection finalizer
func HasProtectionFinalizer(pod *v1.Pod) bool {
	if pod == nil {
		return false
	}
	finalizers := pod.Finalizers
	if len(finalizers) == 0 {
		return false
	}
	for _, finalizer := range finalizers {
		if protectionFinalizerRegexp.MatchString(finalizer) {
			return true
		}
	}
	return false
}

// HasUpgradeProtectionFinalizer returns true if pod has any upgrade protection finalizer
func HasUpgradeProtectionFinalizer(pod *v1.Pod) bool {
	if pod == nil {
		return false
	}
	finalizers := pod.Finalizers
	if len(finalizers) == 0 {
		return false
	}
	for _, finalizer := range finalizers {
		if protectionUpgradeFinalizerRegexp.MatchString(finalizer) {
			return true
		}
	}
	return false
}

// HasDeleteProtectionFinalizer returns true if pod has any delete protection finalizer
func HasDeleteProtectionFinalizer(pod *v1.Pod) bool {
	if pod == nil {
		return false
	}
	finalizers := pod.Finalizers
	if len(finalizers) == 0 {
		return false
	}
	for _, finalizer := range finalizers {
		if protectionDeleteFinalizerRegexp.MatchString(finalizer) {
			return true
		}
	}
	return false
}

// PodShouldNotUpgrade used to judge whether a pod can upgrade
func PodShouldNotUpgrade(pod *v1.Pod) bool {
	return HasProtectionFinalizer(pod) || HasUpgradeProtectionFinalizer(pod)
}

// PodShouldNotDelete used to judge whether a pod can delete
func PodShouldNotDelete(pod *v1.Pod) bool {
	return HasProtectionFinalizer(pod) || HasDeleteProtectionFinalizer(pod)
}

func IsInplaceUpdateAccepted(pod *v1.Pod) bool {
	state, ok := pod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState]
	if ok && state == sigmak8sapi.InplaceUpdateStateAccepted {
		glog.Infof("this inplace update request is accepted for pod (%s)", format.Pod(pod))
		return true
	}

	return false
}

// IsPodHostDNSMode will return true if a pod is HostDNS mode.
// HostDNS mode means user can modify /etc/hosts, /etc/hostname, /etc/resolve.conf as physical machine.
func IsPodHostDNSMode(pod *v1.Pod) bool {
	if pod == nil || len(pod.Labels) == 0 {
		return false
	}

	if pod.Labels[sigmak8sapi.LabelHostDNS] == "true" ||
		pod.Labels[sigmak8sapi.LabelServerType] == sigmak8sapi.PodLabelDockerVM {
		return true
	}

	return false
}

// IsPodDockerVMMode returns whether a pod is DockerVM or not.
func IsPodDockerVMMode(pod *v1.Pod) bool {
	if pod == nil || len(pod.Labels) == 0 {
		return false
	}

	return pod.Labels[sigmak8sapi.LabelServerType] == sigmak8sapi.PodLabelDockerVM
}

// IsPodJob can judge pod is a job or not.
func IsPodJob(pod *v1.Pod) bool {
	if pod == nil || len(pod.Labels) == 0 {
		return false
	}

	if pod.Labels[sigmak8sapi.LabelPodIsJob] == "true" {
		return true
	}

	return false
}

// IsContainerReadyIgnore returns whether ignore this container.
func IsContainerReadyIgnore(container *v1.Container) bool {
	if container == nil || len(container.Env) == 0 {
		return false
	}
	for _, value := range container.Env {
		if strings.EqualFold(value.Name, sigmak8sapi.EnvIgnoreReady) {
			isSidecar, err := strconv.ParseBool(value.Value)
			if err != nil {
				glog.Errorf("container %s env %s parse error %v", container.Name, sigmak8sapi.EnvIgnoreReady, err)
				return false
			}
			return isSidecar
		}
	}
	return false
}

// IsPodDisableServiceLinks returns whether container should igore service envs or not.
// It is only used in 1.12, because EnableServiceLinks field in PodSpec is already implemented in 1.14.
func IsPodDisableServiceLinks(pod *v1.Pod) bool {
	if pod == nil || len(pod.Labels) == 0 {
		return false
	}

	if isPodDisableServiceLinks, err := strconv.ParseBool(pod.Labels[sigmak8sapi.LabelDisableServiceLinks]); err == nil && isPodDisableServiceLinks {
		return true
	}

	return false
}
