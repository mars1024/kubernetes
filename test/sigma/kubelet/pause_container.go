package kubelet

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-kubelet][pause-container] pause the container", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("[smoke] pause the container when desireState is running", func() {
		pod := generateRunningPodWithPostStartHook([]string{"/bin/bash", "-c", "cat /home/helloPost"})
		containerName := pod.Spec.Containers[0].Name

		postStartHookErr := "No such file or directory"
		killSuccessStr := "kill container success"
		startSuccessStr := "start container success"

		// Step1:
		By("create pod")
		testPod, err := util.CreatePod(f.ClientSet, pod, f.Namespace.Name)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		defer util.DeletePod(f.ClientSet, testPod)

		// Step2: Wait the fail message to start container.
		By("wait until pod is starting and failed")
		err = util.WaitTimeoutForContainerUpdateMessage(f.ClientSet, testPod, containerName, 3*time.Minute, postStartHookErr)
		Expect(err).NotTo(HaveOccurred(), "failed to wait pod with postStartHook err")

		// Step3: pause the container.
		By("pause container")
		err = util.PauseContainer(f.ClientSet, testPod, f.Namespace.Name, containerName)
		Expect(err).NotTo(HaveOccurred(), "pause container err")

		// Step5: patch new postStartHook.
		By("change postStartHook")
		// "touch /home/helloPost1" can be executed successfully
		patchPostStartHookData := fmt.Sprintf(`{"spec":{"containers":[{"name":"%s","lifecycle":{"postStart":{"exec":{"command":
                                                ["/bin/bash", "-c", "touch /home/helloPost1"]}}}}]}}`, containerName)
		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(patchPostStartHookData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		// Step6: Unpause container by set desired state as running.
		By("start container")
		patchRunningData, err := util.GenerateContainerStatePatchData(containerName, sigmak8sapi.ContainerStateRunning)
		Expect(err).NotTo(HaveOccurred(), "generate patch data err")

		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(patchRunningData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		// Step7: The unpaused container should be stopped first. So wait container is exited.
		By("wait until container is stopped")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, containerName, 3*time.Minute, killSuccessStr)
		Expect(err).NotTo(HaveOccurred(), "failed to wait container to be exited")

		// Step8: The unpaused container will be started because desire state is running.
		By("wait until container is starting")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, containerName, 3*time.Minute, startSuccessStr)
		Expect(err).NotTo(HaveOccurred(), "failed to wait container to be running")

	})
})
