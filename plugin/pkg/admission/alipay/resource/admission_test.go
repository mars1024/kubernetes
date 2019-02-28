package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"

	"k8s.io/kubernetes/pkg/apis/core"
)

func TestResourceValidate(t *testing.T) {
	assert := assert.New(t)

	testcases := []struct {
		cpuLimit       int64
		cpuRequest     int64
		memoryLimit    int64
		memoryRequest  int64
		storageLimit   int64
		storageRequest int64
		isValid        bool
		isSigmaBE      bool
	}{
		{
			cpuLimit:       0,
			cpuRequest:     0,
			memoryLimit:    0,
			memoryRequest:  0,
			storageLimit:   0,
			storageRequest: 0,
			isValid:        false,
			isSigmaBE:      false,
		},
		// request can equal to limit
		{
			cpuLimit:       1000,                    // 1core
			cpuRequest:     1000,                    // 1core
			memoryLimit:    1 * 1024 * 1024 * 1024,  // 1G
			memoryRequest:  1 * 1024 * 1024 * 1024,  // 1G
			storageLimit:   10 * 1024 * 1024 * 1024, // 10G
			storageRequest: 10 * 1024 * 1024 * 1024, // 10G
			isValid:        true,
		},
		// cpu request can smaller than limit
		{
			cpuLimit:       2000,                    // 2core
			cpuRequest:     1000,                    // 1core
			memoryLimit:    1 * 1024 * 1024 * 1024,  // 1G
			memoryRequest:  1 * 1024 * 1024 * 1024,  // 1G
			storageLimit:   10 * 1024 * 1024 * 1024, // 10G
			storageRequest: 10 * 1024 * 1024 * 1024, // 10G
			isValid:        true,
		},
		// memory request MUST equal to limit
		{
			cpuLimit:       2000,                    // 2core
			cpuRequest:     1000,                    // 1core
			memoryLimit:    2 * 1024 * 1024 * 1024,  // 2G
			memoryRequest:  1 * 1024 * 1024 * 1024,  // 1G
			storageLimit:   10 * 1024 * 1024 * 1024, // 10G
			storageRequest: 10 * 1024 * 1024 * 1024, // 10G
			isValid:        false,
		},
		// memory limit and request can NOT be zero
		{
			cpuLimit:       2000,                    // 2core
			cpuRequest:     1000,                    // 1core
			memoryLimit:    0 * 1024 * 1024 * 1024,  // 0G
			memoryRequest:  0 * 1024 * 1024 * 1024,  // 0G
			storageLimit:   10 * 1024 * 1024 * 1024, // 10G
			storageRequest: 10 * 1024 * 1024 * 1024, // 10G
			isValid:        false,
		},
		// memory request MUST greater than zero
		{
			cpuLimit:       2000,                    // 2core
			cpuRequest:     1000,                    // 1core
			memoryLimit:    1 * 1024 * 1024 * 1024,  // 0G
			memoryRequest:  0 * 1024 * 1024 * 1024,  // 0G
			storageLimit:   10 * 1024 * 1024 * 1024, // 10G
			storageRequest: 10 * 1024 * 1024 * 1024, // 10G
			isValid:        false,
		},
		// cpu limit can not be zero
		{
			cpuLimit:       0,
			cpuRequest:     0,
			memoryLimit:    1 * 1024 * 1024 * 1024,  // 1G
			memoryRequest:  1 * 1024 * 1024 * 1024,  // 1G
			storageLimit:   10 * 1024 * 1024 * 1024, // 10G
			storageRequest: 10 * 1024 * 1024 * 1024, // 10G
			isValid:        false,
			isSigmaBE:      false,
		},
		// cpu request can be zero
		{
			cpuLimit:       1000,
			cpuRequest:     0,
			memoryLimit:    1 * 1024 * 1024 * 1024,  // 1G
			memoryRequest:  1 * 1024 * 1024 * 1024,  // 1G
			storageLimit:   10 * 1024 * 1024 * 1024, // 10G
			storageRequest: 10 * 1024 * 1024 * 1024, // 10G
			isValid:        true,
		},
		// storage request can not be zero
		{
			cpuLimit:       1000,                   // 1core
			cpuRequest:     1000,                   // 1core
			memoryLimit:    1 * 1024 * 1024 * 1024, // 1G
			memoryRequest:  1 * 1024 * 1024 * 1024, // 1G
			storageLimit:   0,                      // 10G
			storageRequest: 0,                      // 10G
			isValid:        false,
		},
		// storage request must equal to limit
		{
			cpuLimit:       1000,                    // 1core
			cpuRequest:     1000,                    // 1core
			memoryLimit:    1 * 1024 * 1024 * 1024,  // 1G
			memoryRequest:  1 * 1024 * 1024 * 1024,  // 1G
			storageLimit:   10 * 1024 * 1024 * 1024, // 10G
			storageRequest: 5 * 1024 * 1024 * 1024,  // 5G
			isValid:        false,
		},
		// cpu limit can be zero if this is a sigma best effort pod
		{
			cpuLimit:       0,                       // 0 core
			cpuRequest:     0,                       // 0 core
			memoryLimit:    1 * 1024 * 1024 * 1024,  // 1G
			memoryRequest:  1 * 1024 * 1024 * 1024,  // 1G
			storageLimit:   10 * 1024 * 1024 * 1024, // 10G
			storageRequest: 10 * 1024 * 1024 * 1024, // 10G
			isValid:        true,
			isSigmaBE:      true,
		},
	}

	for _, testcase := range testcases {
		pod := newPodWithResource(testcase.cpuRequest, testcase.cpuLimit, testcase.memoryRequest, testcase.memoryLimit, testcase.storageRequest, testcase.storageLimit)

		if testcase.isSigmaBE {
			pod.Labels[sigmak8sapi.LabelPodQOSClass] = string(sigmak8sapi.SigmaQOSBestEffort)
		}

		attr := admission.NewAttributesRecord(pod, nil, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Create, false, nil)
		handler := newAlipayResourceAdmission()
		err := handler.Validate(attr)

		if !testcase.isValid {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
	}
}

func newPodWithResource(cpuRequest, cpuLimit, memoryRequest, memoryLimit, storageRequest, storageLimit int64) *core.Pod {
	pod := newPod()

	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].Resources.Limits = map[core.ResourceName]resource.Quantity{}
		pod.Spec.Containers[i].Resources.Requests = map[core.ResourceName]resource.Quantity{}

		pod.Spec.Containers[i].Resources.Limits[core.ResourceCPU] = *resource.NewQuantity(cpuLimit, resource.DecimalSI)
		pod.Spec.Containers[i].Resources.Requests[core.ResourceCPU] = *resource.NewQuantity(cpuRequest, resource.DecimalSI)

		pod.Spec.Containers[i].Resources.Limits[core.ResourceMemory] = *resource.NewQuantity(memoryLimit, resource.BinarySI)
		pod.Spec.Containers[i].Resources.Requests[core.ResourceMemory] = *resource.NewQuantity(memoryRequest, resource.BinarySI)

		pod.Spec.Containers[i].Resources.Limits[core.ResourceEphemeralStorage] = *resource.NewQuantity(storageLimit, resource.BinarySI)
		pod.Spec.Containers[i].Resources.Requests[core.ResourceEphemeralStorage] = *resource.NewQuantity(storageRequest, resource.BinarySI)
	}

	return pod
}

func newPod() *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-setdefault-pod",
			Namespace:   metav1.NamespaceDefault,
			Annotations: map[string]string{},
			Labels:      map[string]string{},
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
