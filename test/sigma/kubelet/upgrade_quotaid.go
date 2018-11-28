package kubelet

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type upgradeQuotaIDTestCase struct {
	pod       *v1.Pod
	patchData string
}

func getQuotaID(pod *v1.Pod) string {
	if pod.Status.HostIP == "" {
		framework.Logf("can't get HostIP from pod: %v", pod)
		return ""
	}
	if len(pod.Status.ContainerStatuses) == 0 {
		framework.Logf("Failed to get ContainerStatuses from pod: %v", pod)
		return ""
	}
	segs := strings.Split(pod.Status.ContainerStatuses[0].ContainerID, "//")
	if len(segs) != 2 {
		framework.Logf("Failed to get ContainerID from pod: %v", pod)
		return ""
	}
	containerID := segs[1]

	runtimeType, _ := util.GetContainerDType(pod.Status.HostIP)
	format := ""
	switch runtimeType {
	case util.ContainerdTypePouch:
		format = "{{.Config.QuotaID}}"
	case util.ContainerdTypeDocker:
		format = "{{.Config.Labels.QuotaId}}"
	}
	// Get QuotaID.
	quotaID, err := util.GetContainerInspectField(pod.Status.HostIP, containerID, format)
	if err != nil {
		framework.Logf("Failed to get quotaID from pod: %s", pod.Status.HostIP)
		return ""
	}

	return quotaID
}

func doUpgradeQuotaIDTestCase(f *framework.Framework, testCase *upgradeQuotaIDTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name
	upgradeSuccessStr := "upgrade container success"

	// Step1: Create pod
	testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Step2: Check created container
	By("check created pod")
	getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get pod err")

	// Step3: Check quotaID after creation
	By("check quotaID after creation")
	quotaIDPre := getQuotaID(getPod)
	if len(quotaIDPre) == 0 {
		framework.Failf("Failed to get quotaID when pod is created")
	}
	framework.Logf("QuotaID is %s when pod is created", quotaIDPre)

	// Step4: Update container to tigger upgrade action.
	By("change container's field")
	upgradedPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(testCase.patchData))
	Expect(err).NotTo(HaveOccurred(), "patch pod err")

	// Step5: Wait for upgrade action finished.
	By("wait until pod is upgraded")
	err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 3*time.Minute, upgradeSuccessStr)
	Expect(err).NotTo(HaveOccurred(), "upgrade pod err")

	// Step6: Check upgraded container
	By("check upgraded pod")
	getPod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get pod err")

	quotaIDPro := getQuotaID(getPod)
	if len(quotaIDPro) == 0 {
		framework.Failf("Failed to get quotaID when pod is upgraded")
	}

	framework.Logf("QuotaID is %s when pod is upgraded", quotaIDPro)

	Expect(quotaIDPre).Should(Equal(quotaIDPro))
}

var _ = Describe("[sigma-kubelet][upgrade-quotaid] check quotaID when upgrade container's image", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("[smoke] check quotaID when upgrade running container", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
			Limits: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		testCase := upgradeQuotaIDTestCase{
			pod:       pod,
			patchData: `{"spec":{"containers":[{"name":"pod-base","image":"reg.docker.alibaba-inc.com/sigma-x/mysql:test-v2"}]}}`,
		}
		doUpgradeQuotaIDTestCase(f, &testCase)
	})
})
