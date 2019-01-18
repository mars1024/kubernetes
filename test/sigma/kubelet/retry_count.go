package kubelet

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-kubelet][retry-count] RetryCount records sigmalet's retry times", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("[smoke] test retry count", func() {
		pod := generateRunningPodWithPostStartHook([]string{"/bin/bash", "-c", "cat /home/helloPost"})
		containerName := pod.Spec.Containers[0].Name

		// Step1: Create pod
		By("create pod")
		testPod, err := util.CreatePod(f.ClientSet, pod, f.Namespace.Name)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		defer util.DeletePod(f.ClientSet, testPod)

		// Step2: Wait the fail message to start container.
		By("wait until RetryCount is three")
		err = util.WaitTimeoutForContainerUpdateRetryCount(f.ClientSet, testPod, containerName, 5*time.Minute, 3)
		Expect(err).NotTo(HaveOccurred(), "failed to wait pod's retry times is 3")

		// Step3: patch new postStartHook.
		By("change postStartHook")
		// "touch /home/helloPost1" can be executed successfully
		patchPostStartHookData := fmt.Sprintf(`{"spec":{"containers":[{"name":"%s","lifecycle":{"postStart":{"exec":{"command":
                                                ["/bin/bash", "-c", "touch /home/helloPost"]}}}}]}}`, containerName)
		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(patchPostStartHookData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		// Step4: wait pod is running
		By("wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 5*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

		// Step5: Check upgraded container
		// RetryCount is reset to 0
		By("check upgraded pod")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "get pod err")

		containerUpdateStatus := util.GetContainerUpdateStatus(getPod, containerName)
		Expect(containerUpdateStatus).NotTo(BeNil(), "failed to get updateStatus")
		if containerUpdateStatus.RetryCount != 0 {
			framework.Failf("expect RetryCount is 0, but got %d", containerUpdateStatus.RetryCount)
		}
	})
})
