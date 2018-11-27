package kubelet

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type UpgradeSpecHashTestCase struct {
	pod        *v1.Pod
	patchData  string
	expectHash string
}

func doUpgradeSpecHashTestCase(f *framework.Framework, testCase *UpgradeSpecHashTestCase) {
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
	By("change container's image")
	upgradedPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(testCase.patchData))
	Expect(err).NotTo(HaveOccurred(), "patch pod err")

	// Step4: Wait for upgrade action finished.
	By("wait until pod is upgraded")
	err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 3*time.Minute, upgradeSuccessStr)
	Expect(err).NotTo(HaveOccurred(), "upgrade pod err")

	// Step5: Check upgraded container
	By("check upgraded pod")
	getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get pod err")

	// Check specHash
	updateStatusStr, exists := getPod.Annotations[sigmak8sapi.AnnotationPodUpdateStatus]
	if !exists {
		framework.Failf("update status doesn't exist")
	}
	framework.Logf("updateStatusStr: %v", updateStatusStr)
	containerStatus := sigmak8sapi.ContainerStateStatus{}
	if err := json.Unmarshal([]byte(updateStatusStr), &containerStatus); err != nil {
		framework.Failf("unmarshal failed")
	}
	for containerInfo, containerStatus := range containerStatus.Statuses {
		if containerInfo.Name == containerName {
			if containerStatus.SpecHash != testCase.expectHash {
				framework.Failf("expect specHash: %s but got: %s", testCase.expectHash, containerStatus.SpecHash)
			}
			break
		}
	}
}

var _ = Describe("[sigma-kubelet][upgrade-specHash] check container's specHash", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("[smoke] check running container's specHash", func() {
		specHash := string(uuid.NewUUID())
		testCase := UpgradeSpecHashTestCase{
			pod:        generateRunningPodWithSpecHash(specHash),
			patchData:  `{"spec":{"containers":[{"name":"pod-base","image":"reg.docker.alibaba-inc.com/sigma-x/mysql:test-v2"}]}}`,
			expectHash: specHash,
		}
		doUpgradeSpecHashTestCase(f, &testCase)
	})
})
