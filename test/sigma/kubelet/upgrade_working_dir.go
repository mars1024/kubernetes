package kubelet

import (
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

type UpgradeWorkingDirTestCase struct {
	pod              *v1.Pod
	patchData        string
	checkCommand     string
	expectWorkingDir string
	patchType        string
}

func doUpgradeWorkingDirTestCase(f *framework.Framework, testCase *UpgradeWorkingDirTestCase) {
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
	var upgradedPod *v1.Pod
	switch testCase.patchType {
	case "remove":
		By("remove container's workingdir")
		newPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.JSONPatchType, []byte(testCase.patchData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")
		upgradedPod = newPod
	default:
		By("change container's workingdir")
		newPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(testCase.patchData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")
		upgradedPod = newPod
	}

	// Step4: Wait for upgrade action finished.
	By("wait until pod is upgraded")
	err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 3*time.Minute, upgradeSuccessStr, true)
	Expect(err).NotTo(HaveOccurred(), "upgrade pod err")

	// Step5: Check upgraded container
	By("check upgraded pod")
	getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get pod err")

	// Check command's result.
	result := f.ExecShellInContainer(getPod.Name, containerName, testCase.checkCommand)
	glog.Infof("command resut: %v", result)
	if result != testCase.expectWorkingDir {
		framework.Failf("expect working dir %s, but got %s", testCase.expectWorkingDir, result)
	}

}

var _ = Describe("[sigma-kubelet][upgrade-workingDir] upgrade container's working dir", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("[smoke] add container's working dir", func() {
		testCase := UpgradeWorkingDirTestCase{
			pod:              generateRunningPod(),
			patchData:        `{"spec":{"containers":[{"name":"pod-base","workingDir":"/home"}]}}`,
			checkCommand:     "pwd",
			expectWorkingDir: "/home",
		}
		doUpgradeWorkingDirTestCase(f, &testCase)
	})
	It("change container's working dir", func() {
		workingDir := "/home"
		testCase := UpgradeWorkingDirTestCase{
			pod:              generateRunningPodWithWorkingDir(workingDir),
			patchData:        `{"spec":{"containers":[{"name":"pod-base","workingDir":"/var/log"}]}}`,
			checkCommand:     "pwd",
			expectWorkingDir: "/var/log",
		}
		doUpgradeWorkingDirTestCase(f, &testCase)
	})
	It("set container's working dir to empty value", func() {
		workingDir := "/home"
		testCase := UpgradeWorkingDirTestCase{
			pod:              generateRunningPodWithWorkingDir(workingDir),
			patchData:        `{"spec":{"containers":[{"name":"pod-base","workingDir":""}]}}`,
			checkCommand:     "pwd",
			expectWorkingDir: "/",
		}
		doUpgradeWorkingDirTestCase(f, &testCase)
	})
	It("delete container's working dir", func() {
		workingDir := "/home"
		testCase := UpgradeWorkingDirTestCase{
			pod:              generateRunningPodWithWorkingDir(workingDir),
			patchData:        `[{"op":"remove","path":"/spec/containers/0/workingDir", "value": "workingDir"}]`,
			checkCommand:     "pwd",
			expectWorkingDir: "/",
			patchType:        "remove",
		}
		doUpgradeWorkingDirTestCase(f, &testCase)
	})
})
