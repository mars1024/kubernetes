package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"

	"k8s.io/kubernetes/pkg/apis/core"
)

func TestResourceValidate(t *testing.T) {
	assert := assert.New(t)

	testcases := []struct {
		cpuLimit      int64
		cpuRequest    int64
		memoryLimit   int64
		memoryRequest int64
		isValid       bool
	}{
		{
			cpuLimit:      0,
			cpuRequest:    0,
			memoryLimit:   0,
			memoryRequest: 0,
			isValid:       false,
		},
		// request can equal to limit
		{
			cpuLimit:      1000,                   // 1core
			cpuRequest:    1000,                   // 1core
			memoryLimit:   1 * 1024 * 1024 * 1024, // 1G
			memoryRequest: 1 * 1024 * 1024 * 1024, // 1G
			isValid:       true,
		},
		// cpu request can smaller than limit
		{
			cpuLimit:      2000,                   // 2core
			cpuRequest:    1000,                   // 1core
			memoryLimit:   1 * 1024 * 1024 * 1024, // 1G
			memoryRequest: 1 * 1024 * 1024 * 1024, // 1G
			isValid:       true,
		},
		// memory request MUST equal to limit
		{
			cpuLimit:      2000,                   // 2core
			cpuRequest:    1000,                   // 1core
			memoryLimit:   2 * 1024 * 1024 * 1024, // 2G
			memoryRequest: 1 * 1024 * 1024 * 1024, // 1G
			isValid:       false,
		},
		// memory limit and request can NOT be zero
		{
			cpuLimit:      2000,                   // 2core
			cpuRequest:    1000,                   // 1core
			memoryLimit:   0 * 1024 * 1024 * 1024, // 0G
			memoryRequest: 0 * 1024 * 1024 * 1024, // 0G
			isValid:       false,
		},
		// memory request MUST greater than zero
		{
			cpuLimit:      2000,                   // 2core
			cpuRequest:    1000,                   // 1core
			memoryLimit:   1 * 1024 * 1024 * 1024, // 0G
			memoryRequest: 0 * 1024 * 1024 * 1024, // 0G
			isValid:       false,
		},
		// cpu limit can not be zero
		{
			cpuLimit:      0,
			cpuRequest:    0,
			memoryLimit:   1 * 1024 * 1024 * 1024, // 1G
			memoryRequest: 1 * 1024 * 1024 * 1024, // 1G
			isValid:       false,
		},
		// cpu request can be zero
		{
			cpuLimit:      1000,
			cpuRequest:    0,
			memoryLimit:   1 * 1024 * 1024 * 1024, // 1G
			memoryRequest: 1 * 1024 * 1024 * 1024, // 1G
			isValid:       true,
		},
	}

	for _, testcase := range testcases {
		pod := newPodWithResource(testcase.cpuRequest, testcase.cpuLimit, testcase.memoryRequest, testcase.memoryLimit)
		attr := admission.NewAttributesRecord(pod, nil, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Create, nil)
		handler := newAlipayResourceAdmission()
		err := handler.Validate(attr)

		if !testcase.isValid {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
	}
}

func newPodWithResource(cpuRequest, cpuLimit, memoryRequest, memoryLimit int64) *core.Pod {
	pod := newPod()

	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].Resources.Limits = map[core.ResourceName]resource.Quantity{}
		pod.Spec.Containers[i].Resources.Requests = map[core.ResourceName]resource.Quantity{}

		pod.Spec.Containers[i].Resources.Limits[core.ResourceCPU] = *resource.NewQuantity(cpuLimit, resource.DecimalSI)
		pod.Spec.Containers[i].Resources.Requests[core.ResourceCPU] = *resource.NewQuantity(cpuRequest, resource.DecimalSI)

		pod.Spec.Containers[i].Resources.Limits[core.ResourceMemory] = *resource.NewQuantity(memoryLimit, resource.BinarySI)
		pod.Spec.Containers[i].Resources.Requests[core.ResourceMemory] = *resource.NewQuantity(memoryRequest, resource.BinarySI)
	}

	return pod
}

func newPod() *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-setdefault-pod",
			Namespace:   metav1.NamespaceDefault,
			Annotations: map[string]string{},
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
}
