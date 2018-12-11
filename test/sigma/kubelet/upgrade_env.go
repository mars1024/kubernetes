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

type UpgradeEnvTestCase struct {
	pod            *v1.Pod
	patchData      string
	checkCommand   string
	resultKeywords []string
	checkMethod    string
}

func doUpgradeEnvTestCase(f *framework.Framework, testCase *UpgradeEnvTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name
	upgradeSuccessStr := "upgrade container success"

	// name should be unique
	pod.Name = "createpodtest" + string(uuid.NewUUID())

	// Step1: Create pod
	testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Step3: Update container to tigger upgrade action.
	By("change container's field")
	upgradedPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(testCase.patchData))
	Expect(err).NotTo(HaveOccurred(), "patch pod err")

	// Step4: Wait for upgrade action finished.
	By("wait until pod is upgraded")
	err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 3*time.Minute, upgradeSuccessStr, true)
	Expect(err).NotTo(HaveOccurred(), "upgrade pod err")

	// Step5: Check upgraded container
	By("check upgraded pod")
	getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get pod err")

	result := f.ExecShellInContainer(getPod.Name, containerName, testCase.checkCommand)
	glog.Infof("command resut: %v", result)
	switch testCase.checkMethod {
	case checkMethodContain:
		for _, resultKeyword := range testCase.resultKeywords {
			if !strings.Contains(result, resultKeyword) {
				framework.Failf("result doesn't contain keyword: %s", resultKeyword)
			}
		}
	case checkMethodNotContain:
		for _, resultKeyword := range testCase.resultKeywords {
			if strings.Contains(result, resultKeyword) {
				framework.Failf("result doesn't contain keyword: %s", resultKeyword)
			}
		}
	}
}

var _ = Describe("[sigma-kubelet][upgrade-env] upgrade container's env", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("[smoke] add env", func() {
		testCase := UpgradeEnvTestCase{
			pod:            generateRunningPod(),
			patchData:      `{"spec":{"containers":[{"name":"pod-base","env":[{"name":"DEMO_GREETING","value":"This is a test"}]}]}}`,
			checkCommand:   "env",
			resultKeywords: []string{"DEMO_GREETING", "This is a test"},
			checkMethod:    checkMethodContain,
		}
		doUpgradeEnvTestCase(f, &testCase)
	})
	It("change env", func() {
		testCase := UpgradeEnvTestCase{
			pod:            generateRunningPodWithEnv(map[string]string{"DEMO_GREETING": "This is a test"}),
			patchData:      `{"spec":{"containers":[{"name":"pod-base","env":[{"name":"DEMO_GREETING","value":"hello world"}]}]}}`,
			checkCommand:   "env",
			resultKeywords: []string{"DEMO_GREETING", "hello world"},
			checkMethod:    checkMethodContain,
		}
		doUpgradeEnvTestCase(f, &testCase)
	})
})
