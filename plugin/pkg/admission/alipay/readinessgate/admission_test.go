package readinessgate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	antapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"

	"k8s.io/kubernetes/pkg/apis/core"
)

func TestRegister(t *testing.T) {
	assert := assert.New(t)

	plugins := admission.NewPlugins()
	Register(plugins)
	registered := plugins.Registered()

	assert.Equal(len(registered), 1, "plugin should be registered")
	assert.Equal(registered[0], PluginName, "plugin should be registered")
}

func TestHandles(t *testing.T) {
	assert := assert.New(t)

	testCases := map[admission.Operation]bool{
		admission.Create:  true,
		admission.Update:  false,
		admission.Connect: false,
		admission.Delete:  false,
	}

	for op, shouldHandle := range testCases {
		handler := NewReadinessGate()
		assert.Equal(shouldHandle, handler.Handles(op))
	}
}

func TestAdmit(t *testing.T) {
	assert := assert.New(t)

	type TestCase struct {
		name               string
		promotionType      string
		existReadinessGate bool
		hasReadinessGate   bool
	}

	handler := NewReadinessGate()

	tcs := []*TestCase{
		{
			name:               "promotionType none, expect no readiness Gate.",
			promotionType:      string(antapi.PodPromotionTypeNone),
			existReadinessGate: false,
			hasReadinessGate:   false,
		},
		{
			name:               "promotionType none, exist readiness Gate, expect has readiness gate.",
			promotionType:      string(antapi.PodPromotionTypeNone),
			existReadinessGate: true,
			hasReadinessGate:   true,
		},
		{
			name:               "promotionType antMember, expect readiness Gate.",
			promotionType:      string(antapi.PodPromotionTypeAntMember),
			existReadinessGate: false,
			hasReadinessGate:   true,
		},
		{
			name:               "promotionType share, expect readiness Gate.",
			promotionType:      string(antapi.PodPromotionTypeShare),
			existReadinessGate: false,
			hasReadinessGate:   false,
		},
		{
			name:               "promotionType taobao, expect readiness Gate.",
			promotionType:      string(antapi.PodPromotionTypeTaobao),
			existReadinessGate: false,
			hasReadinessGate:   true,
		},
	}
	for _, tc := range tcs {
		t.Logf("test case: %v", tc.name)
		pod := newPod(tc.promotionType, tc.existReadinessGate)

		attr := admission.NewAttributesRecord(pod, nil, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Create, false, nil)
		err := handler.Admit(attr)
		assert.Nil(err)
		if tc.hasReadinessGate {
			assert.Equal(ReadinessGateExists(pod.Spec.ReadinessGates, antapi.TimeShareSchedulingReadinessGate), true)
		} else {
			assert.Equal(ReadinessGateExists(pod.Spec.ReadinessGates, antapi.TimeShareSchedulingReadinessGate), false)
		}
		t.Logf("Admitted Pod:%#v", pod.Spec.ReadinessGates)
	}

}

func newPod(promotionTye string, existTimeShare bool) *core.Pod {
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-setdefault-pod",
			Namespace:   metav1.NamespaceDefault,
			Annotations: map[string]string{},
			Labels: map[string]string{
				antapi.LabelPodPromotionType: promotionTye,
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:  "javaweb",
					Image: "pause:2.0",
				},
				{
					Name:  "sidecar",
					Image: "pause:2.0",
				},
			},
		},
	}
	if existTimeShare {
		pod.Spec.ReadinessGates = []core.PodReadinessGate{
			{
				ConditionType: core.PodConditionType(antapi.TimeShareSchedulingReadinessGate),
			},
		}
	}
	return pod
}
