package apiserver

import (
	"encoding/json"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kubernetes/plugin/pkg/admission/alipay/resourcemutationbe"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type containerMilliCPU struct {
	Request int64
	Limit   int64
}

var _ = Describe("[ant][kube-apiserver][admission][resource-mutation-best-effort-admission]", func() {
	f := framework.NewDefaultFramework("sigma-apiserver")

	It("[ant][smoke][test-resource-mutation-best-effort-admission] "+
		"test best effort pod creation with resource mutation admission [Serial]", func() {
		By("load pod template")
		podFile := filepath.Join(util.TestDataDir, "alipay-best-effort-pod.json")
		podCfg, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "load pod template failed")

		By("create best effort pods with resource should succeed")
		pod := podCfg.DeepCopy()

		containersMilliCPU := make(map[string]containerMilliCPU)
		for _, c := range pod.Spec.Containers {
			cpuRequest := c.Resources.Requests[v1.ResourceCPU]
			cpuRequestValue := cpuRequest.MilliValue()
			cpuLimit := c.Resources.Limits[v1.ResourceCPU]
			cpuLimitValue := cpuLimit.MilliValue()
			containersMilliCPU[c.Name] = containerMilliCPU{
				Request: cpuRequestValue,
				Limit:   cpuLimitValue,
			}
		}

		for name, c := range containersMilliCPU {
			framework.Logf("containersMilliCPU[%s] before admission: %+v", name, c)
		}

		pod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		defer util.DeletePod(f.ClientSet, pod)
		Expect(err).NotTo(HaveOccurred(), "failed to create best effort pod")

		getPod, err := f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "failed to get best effort pod")

		allocSpecStr, ok := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
		Expect(ok).To(BeTrue(), "failed to get alloc spec string from annotation")
		framework.Logf("allocSpecStr: %s", allocSpecStr)

		allocSpec := &sigmak8sapi.AllocSpec{}
		err = json.Unmarshal([]byte(allocSpecStr), allocSpec)
		Expect(err).NotTo(HaveOccurred(), "failed to unmarshal alloc spec from json string")

		for _, c := range getPod.Spec.Containers {
			for name, milliCPU := range containersMilliCPU {
				if name != c.Name {
					continue
				}
				framework.Logf("check containers[%s].resources: %+v", name, c.Resources)
				// Best effort value should be equal to cpu value.
				beRequest := c.Resources.Requests[apis.SigmaBEResourceName]
				framework.Logf("beRequest.MilliValue(): %v", beRequest.MilliValue())
				framework.Logf("milliCPU.Request: %v", milliCPU.Request)
				Expect(beRequest.MilliValue()).To(Equal(milliCPU.Request*1000),
					"best effort resource request should be equal to cpu request")

				beLimit := c.Resources.Limits[apis.SigmaBEResourceName]
				Expect(beLimit.MilliValue()).To(Equal(milliCPU.Limit*1000),
					"best effort limit should be equal to cpu limit")

				// CPU value should be equal to zero.
				cpuRequest := c.Resources.Requests[v1.ResourceCPU]
				cpuRequestValue := cpuRequest.MilliValue()
				Expect(cpuRequestValue).To(Equal(int64(0)),
					"cpu request should be equal to zero")

				cpuLimit := c.Resources.Limits[v1.ResourceCPU]
				cpuLimitValue := cpuLimit.MilliValue()
				Expect(cpuLimitValue).To(Equal(int64(0)),
					"cpu limit should be equal to zero")

				// Check cgroup value in host config.
				for _, ac := range allocSpec.Containers {
					if ac.Name != c.Name {
						continue
					}
					framework.Logf("host config of container[%s]: %+v", ac.Name, ac.HostConfig)
					expectedCPUShares := resourcemutationbe.MilliCPUToShares(milliCPU.Request)
					Expect(ac.HostConfig.CpuShares).To(Equal(expectedCPUShares),
						"cpushares should be equal to expected")

					expectedCPUQuota := resourcemutationbe.MilliCPUToQuota(milliCPU.Request,
						resourcemutationbe.QuotaPeriod)
					Expect(ac.HostConfig.CpuQuota).To(Equal(expectedCPUQuota),
						"cpuquota should be equal to expected")
					Expect(ac.Resource.CPU.BindingStrategy).To(Equal(sigmak8sapi.CPUBindStrategyAllCPUs),
						"cpu binding strategy should be equal to expected")
				}
			}
		}
	})
})
