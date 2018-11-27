package kubelet

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type UpgradeAnonymousVolumeTestCase struct {
	pod                *v1.Pod
	initCommand        string
	patchData          string
	checkCommand       string
	resultKeywords     []string
	checkMethod        string
	getImageVersion    string
	expectImageVersion string
	upgradeSuccessStr  string
}

func doUpgradeAnonymousVolumeTestCase(f *framework.Framework, testCase *UpgradeAnonymousVolumeTestCase) {
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

	// Do init work such as create a file in anonymous volume
	f.ExecShellInContainer(testPod.Name, containerName, testCase.initCommand)

	// Step3: Update container to tigger upgrade action.
	By("change container's image")
	upgradedPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(testCase.patchData))
	Expect(err).NotTo(HaveOccurred(), "patch pod err")

	// Step4: Wait for upgrade action finished.
	By("wait until pod is upgraded")
	err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 3*time.Minute, testCase.upgradeSuccessStr)
	Expect(err).NotTo(HaveOccurred(), "upgrade pod err")

	// Step5: Check upgraded container
	By("check upgraded pod")
	getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get pod err")

	// Check image version
	imageVersion := f.ExecShellInContainer(getPod.Name, containerName, testCase.getImageVersion)
	if imageVersion != testCase.expectImageVersion {
		framework.Failf("wrong image version, expect %s, bug got %s", testCase.expectImageVersion, imageVersion)
	}

	// Check command's result.
	result := f.ExecShellInContainer(getPod.Name, containerName, testCase.checkCommand)
	framework.Logf("command resut: %v", result)
	checkResult(testCase.checkMethod, result, testCase.resultKeywords)
}

// image mysql has /TAG file that indicates images' tag.
var _ = Describe("[sigma-kubelet][upgrade_anonymousVolume] check image anonymous volume in upgrade", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	// image mysql:test-v1 has volume /var/lib/mysql
	// image mysql:test-v2 has volume /var/lib/mysql
	It("[smoke] upgrade image: have volume to have volume", func() {
		pod := generateRunningPod()
		pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"
		testCase := UpgradeAnonymousVolumeTestCase{
			pod:                pod,
			initCommand:        "echo 'This is a test' > /var/lib/mysql/test",
			patchData:          `{"spec":{"containers":[{"name":"pod-base","image":"reg.docker.alibaba-inc.com/sigma-x/mysql:test-v2"}]}}`,
			checkCommand:       "cat /var/lib/mysql/test",
			resultKeywords:     []string{"This is a test"},
			checkMethod:        checkMethodContain,
			getImageVersion:    "cat /TAG",
			expectImageVersion: "test-v2",
			upgradeSuccessStr:  "upgrade container success",
		}
		doUpgradeAnonymousVolumeTestCase(f, &testCase)
	})

	// image mysql:test-v1 has volume /var/lib/mysql
	// image mysql:test-v3 has no volume
	It("upgrade image: have volume to have no volume", func() {
		pod := generateRunningPod()
		pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"
		testCase := UpgradeAnonymousVolumeTestCase{
			pod:                pod,
			initCommand:        "echo 'This is a test' > /var/lib/mysql/test",
			patchData:          `{"spec":{"containers":[{"name":"pod-base","image":"reg.docker.alibaba-inc.com/sigma-x/mysql:test-v3"}]}}`,
			checkCommand:       "touch /var/lib/mysql/test && cat /var/lib/mysql/test",
			resultKeywords:     []string{"This is a test"},
			checkMethod:        checkMethodNotContain,
			getImageVersion:    "cat /TAG",
			expectImageVersion: "test-v3",
			upgradeSuccessStr:  "upgrade container success",
		}
		doUpgradeAnonymousVolumeTestCase(f, &testCase)
	})

	// image mysql:test-v3 has no volume
	// image mysql:test-v1 has volume /var/lib/mysql
	It("upgrade image: have no volume to have volume", func() {
		pod := generateRunningPod()
		pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v3"
		testCase := UpgradeAnonymousVolumeTestCase{
			pod:                pod,
			initCommand:        "echo 'This is a test' > /var/lib/mysql/test",
			patchData:          `{"spec":{"containers":[{"name":"pod-base","image":"reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"}]}}`,
			checkCommand:       "touch /var/lib/mysql/test && cat /var/lib/mysql/test",
			resultKeywords:     []string{"This is a test"},
			checkMethod:        checkMethodNotContain,
			getImageVersion:    "cat /TAG",
			expectImageVersion: "test-v1",
			upgradeSuccessStr:  "upgrade container success",
		}
		doUpgradeAnonymousVolumeTestCase(f, &testCase)
	})
})
