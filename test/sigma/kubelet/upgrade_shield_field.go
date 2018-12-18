package kubelet

import (
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type UpgradeShieldFieldTestCase struct {
	pod       *v1.Pod
	patchData string
}

func doUpgradeShieldFieldTestCase(f *framework.Framework, testCase *UpgradeShieldFieldTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name
	upgradeSuccessStr := "upgrade container success"
	timeoutStr := "timeout"

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
	err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 1*time.Minute, upgradeSuccessStr, true)
	if err != nil && strings.Contains(err.Error(), timeoutStr) {
		glog.Infof("Timeout, the container is not upgraded as expect")
	} else {
		framework.Failf("Container is upgraded unexpectly")
	}
}

var _ = Describe("[sigma-kubelet][upgrade-shieldField] change shield field won't cause upgrade", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("[smoke] change running container's shield field", func() {
		testCase := UpgradeShieldFieldTestCase{
			pod: generateRunningPod(),
			patchData: `{"spec":{"containers":[{"name":"pod-base","imagePullPolicy":"Never","livenessProbe":{"exec":{"command":["/bin/bash", "-c", "touch /home/helloLiveNess"]}},
                                "lifecycle":{"postStart":{"exec":{"command":["/bin/bash", "-c", "touch /home/helloPostStartHook"]}}}}]}}`,
		}
		doUpgradeShieldFieldTestCase(f, &testCase)
	})
})
