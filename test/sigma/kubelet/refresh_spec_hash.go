package kubelet

import (
	"encoding/json"
	"fmt"
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

type RefreshSpecHashTestCase struct {
	pod        *v1.Pod
	patchData  string
	expectHash string
}

func doRefreshSpecHashTestCase(f *framework.Framework, testCase *RefreshSpecHashTestCase) {
	pod := testCase.pod

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

	// Step3: Update pod's specHash.
	By("change spec hash")
	_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(testCase.patchData))
	Expect(err).NotTo(HaveOccurred(), "patch pod err")

	// Wait for 30 seceond to wait kubelet update all containers' spec hash.
	time.Sleep(time.Duration(30) * time.Second)

	// Step5: Get latest pod
	By("get latest pod")
	getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get pod err")

	// Step6: Check containers' sepcHash.
	By("check specHash")
	updateStatusStr, exists := getPod.Annotations[sigmak8sapi.AnnotationPodUpdateStatus]
	if !exists {
		framework.Failf("update status doesn't exist")
	}
	framework.Logf("updateStatusStr: %v", updateStatusStr)
	containerStatus := sigmak8sapi.ContainerStateStatus{}
	if err := json.Unmarshal([]byte(updateStatusStr), &containerStatus); err != nil {
		framework.Failf("unmarshal failed")
	}
	for _, containerStatus := range containerStatus.Statuses {
		if containerStatus.SpecHash != testCase.expectHash {
			framework.Failf("expect specHash: %s, but got: %s", testCase.expectHash, containerStatus.SpecHash)
		}
	}
}

var _ = Describe("[sigma-kubelet][refresh-specHash] refresh all containers' specHash", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("[smoke] refresh all containers' specHash if specHash is changed", func() {
		initialSpecHash := "12345678"
		changedSpecHash := "87654321"
		patchData := fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%s"}}}`, sigmak8sapi.AnnotationPodSpecHash, changedSpecHash)
		pod := generateMultiConRunningPod()
		if len(pod.Annotations) == 0 {
			pod.Annotations = map[string]string{}
		}
		pod.Annotations[sigmak8sapi.AnnotationPodSpecHash] = initialSpecHash
		testCase := RefreshSpecHashTestCase{
			pod:        pod,
			patchData:  patchData,
			expectHash: changedSpecHash,
		}
		doRefreshSpecHashTestCase(f, &testCase)
	})
})
