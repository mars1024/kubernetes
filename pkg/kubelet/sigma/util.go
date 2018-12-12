package sigma

import (
	"regexp"

	"k8s.io/api/core/v1"

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
