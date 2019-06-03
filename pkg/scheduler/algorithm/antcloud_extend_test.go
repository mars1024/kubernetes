package algorithm

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	cafelabels "gitlab.alipay-inc.com/antstack/cafe-k8s-api/pkg"

)

func TestIsPodMonotypeHard(t *testing.T) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				cafelabels.MonotypeLabelKey: cafelabels.MonotypeLabelValueHard,
			},
		},
		Spec: v1.PodSpec{

		},
	}
	r := IsPodMonotypeHard(pod)
	if r == false {
		t.Errorf("pod should container label: monotype=hard")
	}

}
