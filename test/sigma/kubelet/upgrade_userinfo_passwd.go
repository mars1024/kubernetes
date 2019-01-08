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

type UpgradeUserInfoTestCase struct {
	pod                *v1.Pod
	initCommand        string
	patchData          string
	checkCommand       string
	resultKeyword      string
	getImageVersion    string
	expectImageVersion string
	upgradeSuccessStr  string
	isExited           bool
}

func doUpgradeUserInfoTestCase(f *framework.Framework, testCase *UpgradeUserInfoTestCase) {
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

	// Step3: Wait for stop action finished if necessary.
	if testCase.isExited {
		By("stop the container")
		err := util.StopContainer(f.ClientSet, testPod, f.Namespace.Name, containerName)
		Expect(err).NotTo(HaveOccurred(), "stop container err")
	}

	// Step4: Update container to tigger upgrade action.
	By("change container's image")
	upgradedPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(testCase.patchData))
	Expect(err).NotTo(HaveOccurred(), "patch container err")

	// Step5: Wait for upgrade action finished.
	By("wait until pod is upgraded")
	err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 3*time.Minute, testCase.upgradeSuccessStr, true)
	Expect(err).NotTo(HaveOccurred(), "upgrade container err")

	// Step6: Wait for start action finished if necessary.
	if testCase.isExited {
		err := util.StartContainer(f.ClientSet, testPod, f.Namespace.Name, containerName)
		Expect(err).NotTo(HaveOccurred(), "start container err")
	}

	// Step7: Check upgraded container
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
	if result != testCase.resultKeyword {
		framework.Failf("result doesn't equal keyword: %s", testCase.resultKeyword)
	}
}

var _ = Describe("[sigma-kubelet][upgrade-userinfo] check userinfo when upgrade container's image", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	It("[smoke] check userinfo when upgrade running container", func() {
		pod := generateRunningPod()
		pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"
		testCase := UpgradeUserInfoTestCase{
			pod:                pod,
			initCommand:        "echo 'www-data:x:33:33:www-data:/var/www:/usr/sbin/nologin' >> /etc/passwd",
			patchData:          `{"spec":{"containers":[{"name":"pod-base","image":"reg.docker.alibaba-inc.com/sigma-x/mysql:test-v2"}]}}`,
			checkCommand:       "cat /etc/passwd | grep 'www-data:x:33:33:www-data:/var/www:/usr/sbin/nologin' | wc -l",
			resultKeyword:      "2",
			getImageVersion:    "cat /TAG",
			expectImageVersion: "test-v2",
			upgradeSuccessStr:  "upgrade container success",
			isExited:           false,
		}
		doUpgradeUserInfoTestCase(f, &testCase)
	})

	It("[smoke] check userinfo when upgrade exited container", func() {
		pod := generateRunningPod()
		pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"
		testCase := UpgradeUserInfoTestCase{
			pod:                pod,
			initCommand:        "echo 'www-data:x:33:33:www-data:/var/www:/usr/sbin/nologin' >> /etc/passwd",
			patchData:          `{"spec":{"containers":[{"name":"pod-base","image":"reg.docker.alibaba-inc.com/sigma-x/mysql:test-v2"}]}}`,
			checkCommand:       "cat /etc/passwd | grep 'www-data:x:33:33:www-data:/var/www:/usr/sbin/nologin' | wc -l",
			resultKeyword:      "2",
			getImageVersion:    "cat /TAG",
			expectImageVersion: "test-v2",
			upgradeSuccessStr:  "upgrade container success",
			isExited:           true,
		}
		doUpgradeUserInfoTestCase(f, &testCase)
	})
})
