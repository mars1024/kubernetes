package kubelet

import (
	"encoding/json"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type defaultCPUSetTestCase struct {
	description       string
	pod               *v1.Pod
	expectedCPUShares string
}

func doDefaultCPUSetTestCase(f *framework.Framework, testCase *defaultCPUSetTestCase) {
	framework.Logf("Start to test case %q", testCase.description)
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name

	// Step1: Create pod
	testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Step3: Get the running pod.
	By("get created pod")
	getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	framework.Logf("getPod: %v", getPod.Annotations)
	Expect(err).NotTo(HaveOccurred(), "get pod err")

	// Step5: Get CPUShares
	command := "cat /sys/fs/cgroup/cpu/cpu.shares"
	cpuSharesStr := f.ExecShellInContainer(pod.Name, containerName, command)
	cpuShares := strings.Replace(cpuSharesStr, "\n", "", -1)

	Expect(cpuShares).To(Equal(testCase.expectedCPUShares), "Bad cpu shares")
}

var _ = Describe("[sigma-kubelet][default-cpushares] DefaultCPUShares test", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	It("[smoke] DefaultCPUShares: no DefaultCPUShares defined", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceCPU:              *resource.NewMilliQuantity(0, resource.DecimalSI),
				v1.ResourceMemory:           resource.MustParse("512Mi"),
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
			Limits: v1.ResourceList{
				v1.ResourceCPU:              *resource.NewMilliQuantity(3000, resource.DecimalSI),
				v1.ResourceMemory:           resource.MustParse("512Mi"),
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		testCase := &defaultCPUSetTestCase{
			description:       "no DefaultCPUShares defined",
			pod:               pod,
			expectedCPUShares: "2",
		}
		doDefaultCPUSetTestCase(f, testCase)
	})

	// 待其他组件更新vendor后，把[ant]删除！！！
	It("[smoke][ant] DefaultCPUShares: DefaultCPUShares defined", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceCPU:              *resource.NewMilliQuantity(0, resource.DecimalSI),
				v1.ResourceMemory:           resource.MustParse("512Mi"),
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
			Limits: v1.ResourceList{
				v1.ResourceCPU:              *resource.NewMilliQuantity(3000, resource.DecimalSI),
				v1.ResourceMemory:           resource.MustParse("512Mi"),
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		}

		containerName := pod.Spec.Containers[0].Name
		pod.Spec.Containers[0].Resources = resources

		// Set alloc spec annotation
		cpuShares := int64(512)
		hostConfig := sigmak8sapi.HostConfigInfo{
			DefaultCpuShares: &cpuShares,
		}

		allocSpec := &sigmak8sapi.AllocSpec{
			Containers: []sigmak8sapi.Container{
				sigmak8sapi.Container{
					Name:       containerName,
					HostConfig: hostConfig,
				},
			},
		}

		allocSpecBytes, err := json.Marshal(allocSpec)
		Expect(err).NotTo(HaveOccurred(), "failed to marshal allocSpec")
		pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(allocSpecBytes)

		testCase := &defaultCPUSetTestCase{
			description:       "DefaultCPUShares defined",
			pod:               pod,
			expectedCPUShares: "512",
		}
		doDefaultCPUSetTestCase(f, testCase)
	})
})
