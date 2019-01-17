package api

import (
	"k8s.io/api/core/v1"
)

// SigmaQOSClass defines the supported qos classes of Pods.
type SigmaQOSClass string

const (
	// SigmaQOSGuaranteed is the Guaranteed qos class.
	SigmaQOSGuaranteed SigmaQOSClass = "SigmaGuaranteed"

	// SigmaQOSBurstable is the Burstable qos class.
	SigmaQOSBurstable SigmaQOSClass = "SigmaBurstable"

	// SigmaQOSBestEffort is the BestEffort qos class.
	SigmaQOSBestEffort SigmaQOSClass = "SigmaBestEffort"

	// SigmaQOSNone is the undefined qos class.
	SigmaQOSNone SigmaQOSClass = ""
)

func GetPodQOSClass(pod *v1.Pod) SigmaQOSClass {
	if v, ok := pod.Labels[LabelPodQOSClass]; ok {
		switch SigmaQOSClass(v) {
		case SigmaQOSGuaranteed:
			return SigmaQOSGuaranteed
		case SigmaQOSBurstable:
			return SigmaQOSBurstable
		case SigmaQOSBestEffort:
			return SigmaQOSBestEffort
		default:
			return SigmaQOSNone
		}
	} else {
		return SigmaQOSNone
	}
}
