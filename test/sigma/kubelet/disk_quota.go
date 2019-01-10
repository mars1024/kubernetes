package kubelet

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type diskQuotaTestCase struct {
	pod            *v1.Pod
	checkCommand   string
	resultKeywords []string
	checkMethod    string
}

func doDiskQuotaTestCase(f *framework.Framework, testCase *diskQuotaTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name

	// Step1: Create pod
	By("create pod")
	testPod, err := util.CreatePod(f.ClientSet, pod, f.Namespace.Name)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Step3: Check command's result.
	result := f.ExecShellInContainer(testPod.Name, containerName, testCase.checkCommand)
	framework.Logf("command resut: %v", result)
	checkResult(testCase.checkMethod, result, testCase.resultKeywords)
}

var _ = Describe("[sigma-kubelet][disk-quota] check disk quota", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	// RequestEphemeralStorage is defined, container has diskquota.
	It("[smoke] check disk quota: RequestEphemeralStorage is defined", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		testCase := diskQuotaTestCase{
			pod:            pod,
			checkCommand:   "df -h | grep '/$' | awk '{print $2}'",
			resultKeywords: []string{"2.0G"},
			checkMethod:    checkMethodEqual,
		}
		doDiskQuotaTestCase(f, &testCase)
	})
	// LimitEphemeralStorage is defined, container has diskquota.
	It("check disk quota: LimitEphemeralStorage is defined", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		testCase := diskQuotaTestCase{
			pod:            pod,
			checkCommand:   "df -h | grep '/$' | awk '{print $2}'",
			resultKeywords: []string{"2.0G"},
			checkMethod:    checkMethodEqual,
		}
		doDiskQuotaTestCase(f, &testCase)
	})
	// RequestEphemeralStorage and LimitEphemeralStorage are defined, container's diskquota is equal to RequestEphemeralStorage.
	It("check disk quota: RequestEphemeralStorage and LimitEphemeralStorage are defined", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
			},
			Limits: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		testCase := diskQuotaTestCase{
			pod:            pod,
			checkCommand:   "df -h | grep '/$' | awk '{print $2}'",
			resultKeywords: []string{"1.0G"},
			checkMethod:    checkMethodEqual,
		}
		doDiskQuotaTestCase(f, &testCase)
	})
	// RequestEphemeralStorage and LimitEphemeralStorage are not defined, container has no diskquota.
	It("check disk quota: RequestEphemeralStorage and LimitEphemeralStorage are not defined.", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{}
		pod.Spec.Containers[0].Resources = resources
		testCase := diskQuotaTestCase{
			pod:            pod,
			checkCommand:   "df -h | grep '/$' | awk '{print $2}'",
			resultKeywords: []string{"1.0G", "2.0G"},
			checkMethod:    checkMethodNotEqual,
		}
		doDiskQuotaTestCase(f, &testCase)
	})
})
