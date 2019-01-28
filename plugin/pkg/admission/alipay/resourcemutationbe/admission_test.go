package resourcemutationbe

import (
	"encoding/json"
	"testing"

	log "github.com/golang/glog"
	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
)

func TestResourceMutationBE(t *testing.T) {
	testcases := []struct {
		cpuRequest        int64
		cpuLimit          int64
		expectedCPUShares int64
		expectedCPUQuota  int64
	}{
		{
			cpuRequest:        0,
			cpuLimit:          0,
			expectedCPUShares: 2,
			expectedCPUQuota:  0,
		},
		{
			cpuRequest:        1,
			cpuLimit:          1,
			expectedCPUShares: 2,
			expectedCPUQuota:  MinQuotaPeriod,
		},
		{
			cpuRequest:        2,
			cpuLimit:          2,
			expectedCPUShares: 2,
			expectedCPUQuota:  MinQuotaPeriod,
		},
		{
			cpuRequest:        3,
			cpuLimit:          3,
			expectedCPUShares: 3,
			expectedCPUQuota:  MinQuotaPeriod,
		},
		{
			cpuRequest:        10,
			cpuLimit:          20,
			expectedCPUShares: 10,
			expectedCPUQuota:  1500,
		},
		{
			cpuRequest:        100,
			cpuLimit:          200,
			expectedCPUShares: 102,
			expectedCPUQuota:  15000,
		},
		{
			cpuRequest:        1000,
			cpuLimit:          2000,
			expectedCPUShares: 1024,
			expectedCPUQuota:  150000,
		},
		{
			cpuRequest:        2000,
			cpuLimit:          2000,
			expectedCPUShares: 2048,
			expectedCPUQuota:  2 * 150000,
		},
	}

	for i, tcase := range testcases {
		pod := newPodWithResource(tcase.cpuRequest, tcase.cpuLimit)
		attr := admission.NewAttributesRecord(pod, nil,
			core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
			core.Resource("pods").WithVersion("version"), "",
			admission.Create, false, nil)
		handler := newAlipayResourceMutationBestEffort()
		err := handler.Admit(attr)
		assert.Nil(t, err)
		allocSpec, err := podAllocSpec(pod)
		log.Infof("case: %d", i)

		for _, c := range pod.Spec.Containers {
			// Best effort value should be equal to cpu value.
			beRequest := c.Resources.Requests[apis.SigmaBEResourceName]
			assert.Equal(t, beRequest.MilliValue(), tcase.cpuRequest,
				"best effort request should be equal to cpu request")
			beLimit := c.Resources.Limits[apis.SigmaBEResourceName]
			assert.Equal(t, beLimit.MilliValue(), tcase.cpuLimit,
				"best effort limit should be equal to cpu limit")

			// CPU value should be equal to zero.
			cpuRequest := c.Resources.Requests[v1.ResourceCPU]
			cpuRequestValue := cpuRequest.MilliValue()
			assert.Equal(t, int64(0), cpuRequestValue,
				"cpu request should be equal to zero")
			cpuLimit := c.Resources.Limits[v1.ResourceCPU]
			cpuLimitValue := cpuLimit.MilliValue()
			assert.Equal(t, int64(0), cpuLimitValue,
				"cpu limit should be equal to zero")

			// Check cgroup value in host config.
			for _, ac := range allocSpec.Containers {
				if ac.Name == c.Name {
					continue
				}

				log.Infof("host config of container[%s]: %+v", ac.Name, ac.HostConfig)
				assert.Equal(t, tcase.expectedCPUShares, ac.HostConfig.CpuShares,
					"cpushares should be equal to expected")
				assert.Equal(t, tcase.expectedCPUQuota, ac.HostConfig.CpuQuota,
					"cpuquota should be equal to expected")
			}
		}
	}
}

func newPodWithResource(cpuRequest, cpuLimit int64) *v1.Pod {
	pod := newPod()
	allocSpec := sigmak8sapi.AllocSpec{}
	for i, c := range pod.Spec.Containers {
		pod.Spec.Containers[i].Resources.Limits = map[v1.ResourceName]resource.Quantity{}
		pod.Spec.Containers[i].Resources.Requests = map[v1.ResourceName]resource.Quantity{}

		pod.Spec.Containers[i].Resources.Limits[v1.ResourceCPU] =
			*resource.NewMilliQuantity(cpuLimit, resource.DecimalSI)
		pod.Spec.Containers[i].Resources.Requests[v1.ResourceCPU] =
			*resource.NewMilliQuantity(cpuRequest, resource.DecimalSI)
		allocSpec.Containers = append(allocSpec.Containers, newAllocSpecContainer(c.Name))
	}

	data, _ := json.Marshal(&allocSpec)
	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)

	return pod
}

func newPod() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-resource-mutation-best-effort-pod",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				sigmak8sapi.LabelPodQOSClass: string(sigmak8sapi.SigmaQOSBestEffort),
			},
			Annotations: map[string]string{},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "container-1",
					Image: "image:1.0",
				},
				{
					Name:  "container-2",
					Image: "image:2.0",
				},
			},
		},
	}
}

func newAllocSpecContainer(name string) sigmak8sapi.Container {
	return sigmak8sapi.Container{
		Name: name,
		Resource: sigmak8sapi.ResourceRequirements{
			// GPU.ShareMode is validated in admission controller 'sigmascheduling'
			GPU: sigmak8sapi.GPUSpec{ShareMode: sigmak8sapi.GPUShareModeExclusive},
		},
	}
}
