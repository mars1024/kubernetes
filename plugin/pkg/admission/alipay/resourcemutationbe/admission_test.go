package resourcemutationbe

import (
	"encoding/json"
	"testing"

	log "github.com/golang/glog"
	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
)

type testContainer struct {
	cpuRequest        int64
	cpuLimit          int64
	expectedCPUShares int64
	expectedCPUQuota  int64
}

func TestResourceMutationBE(t *testing.T) {
	type testCase struct {
		containers map[string]testContainer
	}

	testCases := []testCase{
		{
			containers: map[string]testContainer{
				"c1": testContainer{
					cpuRequest:        0,
					cpuLimit:          0,
					expectedCPUShares: 2,
					expectedCPUQuota:  0,
				},
				"c2": testContainer{
					cpuRequest:        1,
					cpuLimit:          1,
					expectedCPUShares: 2,
					expectedCPUQuota:  MinQuotaPeriod,
				},
			},
		},
		{
			containers: map[string]testContainer{
				"c1": testContainer{
					cpuRequest:        1,
					cpuLimit:          1,
					expectedCPUShares: 2,
					expectedCPUQuota:  MinQuotaPeriod,
				},
				"c2": testContainer{
					cpuRequest:        1,
					cpuLimit:          1,
					expectedCPUShares: 2,
					expectedCPUQuota:  MinQuotaPeriod,
				},
				"c3": testContainer{
					cpuRequest:        2,
					cpuLimit:          2,
					expectedCPUShares: 2,
					expectedCPUQuota:  MinQuotaPeriod,
				},
			},
		},
		{
			containers: map[string]testContainer{
				"c1": testContainer{
					cpuRequest:        3,
					cpuLimit:          3,
					expectedCPUShares: 3,
					expectedCPUQuota:  MinQuotaPeriod,
				},
			},
		},
		{
			containers: map[string]testContainer{
				"c1": testContainer{
					cpuRequest:        10,
					cpuLimit:          20,
					expectedCPUShares: 10,
					expectedCPUQuota:  1500,
				},
				"c2": testContainer{
					cpuRequest:        1,
					cpuLimit:          1,
					expectedCPUShares: 2,
					expectedCPUQuota:  MinQuotaPeriod,
				},
			},
		},
		{
			containers: map[string]testContainer{
				"c1": testContainer{
					cpuRequest:        1000,
					cpuLimit:          2000,
					expectedCPUShares: 1024,
					expectedCPUQuota:  150000,
				},
				"c2": testContainer{
					cpuRequest:        100,
					cpuLimit:          200,
					expectedCPUShares: 102,
					expectedCPUQuota:  15000,
				},
			},
		},
		{
			containers: map[string]testContainer{
				"c1": testContainer{
					cpuRequest:        1000,
					cpuLimit:          2000,
					expectedCPUShares: 1024,
					expectedCPUQuota:  150000,
				},
				"c2": testContainer{
					cpuRequest:        2000,
					cpuLimit:          2000,
					expectedCPUShares: 2048,
					expectedCPUQuota:  2 * 150000,
				},
				"c3": testContainer{
					cpuRequest:        10,
					cpuLimit:          20,
					expectedCPUShares: 10,
					expectedCPUQuota:  1500,
				},
			},
		},
	}

	for i, tcase := range testCases {
		pod := newPodWithResource(tcase.containers)
		attr := admission.NewAttributesRecord(pod, nil,
			core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
			core.Resource("pods").WithVersion("version"), "",
			admission.Create, false, nil)
		handler := newAlipayResourceMutationBestEffort()
		err := handler.Admit(attr)
		assert.Nil(t, err)
		allocSpec, err := podAllocSpec(pod)
		for _, c := range pod.Spec.Containers {
			for name, tc := range tcase.containers {
				if name != c.Name {
					continue
				}
				log.Infof("case[%d] check container: %s", i, name)
				// Best effort value should be equal to cpu value.
				beRequest := c.Resources.Requests[apis.SigmaBEResourceName]
				assert.Equal(t, beRequest.MilliValue(), tc.cpuRequest*1000,
					"best effort request should be equal to cpu request")
				beLimit := c.Resources.Limits[apis.SigmaBEResourceName]
				assert.Equal(t, beLimit.MilliValue(), tc.cpuLimit*1000,
					"best effort limit should be equal to cpu limit")

				// CPU value should be equal to zero.
				cpuRequest := c.Resources.Requests[core.ResourceCPU]
				cpuRequestValue := cpuRequest.MilliValue()
				assert.Equal(t, int64(0), cpuRequestValue,
					"cpu request should be equal to zero")
				cpuLimit := c.Resources.Limits[core.ResourceCPU]
				cpuLimitValue := cpuLimit.MilliValue()
				assert.Equal(t, int64(0), cpuLimitValue,
					"cpu limit should be equal to zero")
				// Check cgroup value in host config.
				for _, ac := range allocSpec.Containers {
					if ac.Name != c.Name {
						continue
					}

					log.Infof("host config of container[%s]: %+v", ac.Name, ac.HostConfig)
					assert.Equal(t, tc.expectedCPUShares, ac.HostConfig.CpuShares,
						"cpushares should be equal to expected")
					assert.Equal(t, tc.expectedCPUQuota, ac.HostConfig.CpuQuota,
						"cpuquota should be equal to expected")
				}
			}
		}
	}
}

func newPodWithResource(containers map[string]testContainer) *core.Pod {
	pod := newPod()
	allocSpec := sigmak8sapi.AllocSpec{}

	for name, c := range containers {
		pod.Spec.Containers = append(pod.Spec.Containers, core.Container{
			Name: name,
			Resources: core.ResourceRequirements{
				Requests: map[core.ResourceName]resource.Quantity{
					core.ResourceCPU: *resource.NewMilliQuantity(c.cpuRequest, resource.DecimalSI),
				},
				Limits: map[core.ResourceName]resource.Quantity{
					core.ResourceCPU: *resource.NewMilliQuantity(c.cpuLimit, resource.DecimalSI),
				},
			},
		})
		allocSpec.Containers = append(allocSpec.Containers, newAllocSpecContainer(name))
	}

	data, _ := json.Marshal(&allocSpec)
	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)

	return pod
}

func newPod() *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-resource-mutation-best-effort-pod",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				sigmak8sapi.LabelPodQOSClass: string(sigmak8sapi.SigmaQOSBestEffort),
			},
			Annotations: map[string]string{},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{},
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
