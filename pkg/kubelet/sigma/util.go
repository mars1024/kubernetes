package sigma

import (
	"regexp"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/util/format"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

var protectionFinalizerRegexp = regexp.MustCompile(sigmak8sapi.FinalizerPodProtectionFmt)

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

func IsInplaceUpdateAccepted(pod *v1.Pod) bool {
	state, ok := pod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState]
	if ok && state == sigmak8sapi.InplaceUpdateStateAccepted {
		glog.Infof("this inplace update request is accepted for pod (%s)", format.Pod(pod))
		return true
	}

	return false
}
