package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPodQOSClass(t *testing.T) {
	guaranteedPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"sigma.ali/qos": "SigmaGuaranteed",
			},
		},
	}
	burstablePod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"sigma.ali/qos": "SigmaBurstable",
			},
		},
	}
	bestEffortPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"sigma.ali/qos": "SigmaBestEffort",
			},
		},
	}
	nonQOSPod1 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"sigma.ali/qos": "",
			},
		},
	}
	nonQOSPod2 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{},
	}

	targetQOSClass1 := GetPodQOSClass(guaranteedPod)
	assert.Equal(t, SigmaQOSGuaranteed, targetQOSClass1)

	targetQOSClass2 := GetPodQOSClass(burstablePod)
	assert.Equal(t, SigmaQOSBurstable, targetQOSClass2)

	targetQOSClass3 := GetPodQOSClass(bestEffortPod)
	assert.Equal(t, SigmaQOSBestEffort, targetQOSClass3)

	targetQOSClass4 := GetPodQOSClass(nonQOSPod1)
	assert.Equal(t, SigmaQOSNone, targetQOSClass4)

	targetQOSClass5 := GetPodQOSClass(nonQOSPod2)
	assert.Equal(t, SigmaQOSNone, targetQOSClass5)
}
