package algorithm

import (
	cafelabels "gitlab.alipay-inc.com/antstack/cafe-k8s-api/pkg"
	v1 "k8s.io/api/core/v1"
)


// CheckPodMonotype check whether the pod is the expected
// monotype
func CheckPodMonotype(pod *v1.Pod, monotype string) bool {
	if pod == nil {
		return false
	}

	if value, ok := pod.Labels[cafelabels.MonotypeLabelKey]; ok && (value == monotype) {
		return true
	}
	return false
}

func IsPodMonotypeHard(pod *v1.Pod) bool {
	return CheckPodMonotype(pod, cafelabels.MonotypeLabelValueHard)
}

func IsPodMonotypeSoft(pod *v1.Pod) bool {
	return CheckPodMonotype(pod, cafelabels.MonotypeLabelValueSoft)
}