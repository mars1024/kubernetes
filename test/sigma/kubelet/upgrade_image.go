package kubelet

import (
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type UpgradeImageTestCase struct {
	pod            *v1.Pod
	patchData      string
	checkCommand   string
	resultKeywords []string
	// running, exited
	expectState string
}

func doUpgradeImageTestCase(f *framework.Framework, testCase *UpgradeImageTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name
	stopSuccessStr := "kill container success"
	upgradeSuccessStr := "upgrade container success"

	// name should be unique
	pod.Name = "createpodtest" + string(uuid.NewUUID())

	// Step1: Create pod
	testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished.
	switch testCase.expectState {
	case "running":
		By("wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")
	case "exited":
		// Step2.1: Wait for container's creation finished.
		By("wait until pod is stopped after creation")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, containerName, 3*time.Minute, stopSuccessStr)
		Expect(err).NotTo(HaveOccurred(), "start/stop pod err")
	}

	// Step3: Update container to tigger upgrade action.
	By("change container's field")
	upgradedPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(testCase.patchData))
	Expect(err).NotTo(HaveOccurred(), "patch pod err")

	// Step4: Wait for upgrade action finished.
	By("wait until pod is upgraded")
	err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 3*time.Minute, upgradeSuccessStr)
	Expect(err).NotTo(HaveOccurred(), "upgrade pod err")

	// Step5: Check upgraded container
	switch testCase.expectState {
	case "running":
		By("check upgraded pod")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "get pod err")

		result := f.ExecShellInContainer(getPod.Name, containerName, testCase.checkCommand)
		glog.Infof("command resut: %v", result)
		for _, resultKeyword := range testCase.resultKeywords {
			if !strings.Contains(result, resultKeyword) {
				framework.Failf("result doesn't contain keyword: %s", resultKeyword)
			}
		}
	case "exited":
		By("wait until pod is stopped")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 3*time.Minute, stopSuccessStr)
		Expect(err).NotTo(HaveOccurred(), "stop pod err")
	}
}

var _ = Describe("[sigma-kubelet][upgrade-image] upgrade container's image", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("[smoke] upgrade running container", func() {
		testCase := UpgradeImageTestCase{
			pod:            generateRunningPod(),
			patchData:      `{"spec":{"containers":[{"name":"pod-base","image":"reg.docker.alibaba-inc.com/sigma-x/mysql:test-v2"}]}}`,
			checkCommand:   "cat /TAG",
			resultKeywords: []string{"test-v2"},
			expectState:    "running",
		}
		doUpgradeImageTestCase(f, &testCase)
	})
	It("[smoke] upgrade exited container", func() {
		testCase := UpgradeImageTestCase{
			pod:            generateExitedPod(),
			patchData:      `{"spec":{"containers":[{"name":"pod-base","image":"reg.docker.alibaba-inc.com/sigma-x/mysql:test-v2"}]}}`,
			checkCommand:   "cat /TAG",
			resultKeywords: []string{"test-v2"},
			expectState:    "exited",
		}
		doUpgradeImageTestCase(f, &testCase)
	})
})
