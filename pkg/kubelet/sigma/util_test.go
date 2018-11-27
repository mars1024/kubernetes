package sigma

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHasProtectionFinalizer(t *testing.T) {
	tests := []struct {
		pod                    *v1.Pod
		hasProtectionFinalizer bool
	}{
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"protection.pod.beta1.sigma.ali/vip-removed", "pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			hasProtectionFinalizer: true,
		},
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			hasProtectionFinalizer: false,
		},
	}

	for _, test := range tests {
		assert.Equal(t, HasProtectionFinalizer(test.pod), test.hasProtectionFinalizer)
	}
}
