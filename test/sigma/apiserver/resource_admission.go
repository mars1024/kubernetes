package apiserver

import (
	"fmt"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[ant][kube-apiserver][admission][resource]", func() {
	image := "reg.docker.alibaba-inc.com/k8s-test/nginx:1.15.3"
	f := framework.NewDefaultFramework("sigma-apiserver")

	It("[ant][smoke][test-resource-admission] test pod creation without resource admission [Serial]", func() {
		By("load pod template")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		podCfg, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "load pod template failed")

		By("create default pods should fail, because it does not have resource setting")
		podCfg.Spec.Containers[0].Image = image
		pod := podCfg.DeepCopy()
		pod.Name = pod.Name + "-without-resource"
		pod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		defer util.DeletePod(f.ClientSet, pod)
		Expect(err).To(HaveOccurred(), "pod without resource should not be created")
	})

	It("[ant][smoke][test-resource-admission] test pod creation without resource admission [Serial]", func() {
		By("load pod template")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		podCfg, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "load pod template failed")

		By("create pods with resource should succeed")
		podCfg.Spec.Containers[0].Image = image
		pod := podCfg.DeepCopy()
		pod.Name = pod.Name + "-with-resource"
		for i := range pod.Spec.Containers {
			pod.Spec.Containers[i].Resources = v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU:              *resource.NewQuantity(1000, resource.DecimalSI),
					v1.ResourceMemory:           *resource.NewQuantity(1000*10000*10000, resource.BinarySI),
					v1.ResourceEphemeralStorage: *resource.NewQuantity(1000*10000*1000, resource.BinarySI),
				},
				Requests: v1.ResourceList{
					v1.ResourceCPU:              *resource.NewQuantity(1000, resource.DecimalSI),
					v1.ResourceMemory:           *resource.NewQuantity(1000*10000*10000, resource.BinarySI),
					v1.ResourceEphemeralStorage: *resource.NewQuantity(1000*10000*1000, resource.BinarySI),
				},
			}
		}
		pod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		defer util.DeletePod(f.ClientSet, pod)
		Expect(err).NotTo(HaveOccurred(), "create pod failed")
	})

	It("[ant][smoke][test-resource-admission] test pod creation with different resource scenarios [Serial]", func() {
		By("load pod template")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		podCfg, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "load pod template failed")

		testcases := []struct {
			cpuLimit       int64
			cpuRequest     int64
			memoryLimit    int64
			memoryRequest  int64
			storageLimit   int64
			storageRequest int64
			isValid        bool
		}{
			{
				cpuLimit:       0,
				cpuRequest:     0,
				memoryLimit:    0,
				memoryRequest:  0,
				storageLimit:   0,
				storageRequest: 0,
				isValid:        false,
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
		}

		podCfg.Spec.Containers[0].Image = image

		for index, testcase := range testcases {
			By(fmt.Sprintf("running apiserver admission testcase %d", index))
			pod := podCfg.DeepCopy()
			pod.Name = pod.Name + strconv.Itoa(index)

			for i := range pod.Spec.Containers {
				pod.Spec.Containers[i].Resources = v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceCPU:              *resource.NewQuantity(testcase.cpuLimit, resource.DecimalSI),
						v1.ResourceMemory:           *resource.NewQuantity(testcase.memoryLimit, resource.BinarySI),
						v1.ResourceEphemeralStorage: *resource.NewQuantity(testcase.storageLimit, resource.BinarySI),
					},
					Requests: v1.ResourceList{
						v1.ResourceCPU:              *resource.NewQuantity(testcase.cpuRequest, resource.DecimalSI),
						v1.ResourceMemory:           *resource.NewQuantity(testcase.memoryRequest, resource.BinarySI),
						v1.ResourceEphemeralStorage: *resource.NewQuantity(testcase.storageRequest, resource.BinarySI),
					},
				}
			}
			pod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
			defer util.DeletePod(f.ClientSet, pod)

			if testcase.isValid {
				Expect(err).NotTo(HaveOccurred(), "create pod with valid resource should succeed")
			} else {
				Expect(err).To(HaveOccurred(), "create pod with invalid resource should fail")
			}
		}
	})
})
