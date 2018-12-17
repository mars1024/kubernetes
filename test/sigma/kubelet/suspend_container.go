package kubelet

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
	"k8s.io/api/core/v1"
)

var _ = Describe("[sigma-kubelet][suspend-container] suspend/unsuspend the container", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("suspend the container...", func() {
		pod := generateRunningPod()
		pod.Annotations = map[string]string{}
		containerName := pod.Spec.Containers[0].Name

		By("creating a pod")
		testPod, err := util.CreatePod(f.ClientSet, pod, f.Namespace.Name)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		defer util.DeletePod(f.ClientSet, testPod)

		By("waiting until pod is running")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

		By("suspend a container")
		err = util.SuspendContainer(f.ClientSet, testPod, f.Namespace.Name, containerName)
		Expect(err).NotTo(HaveOccurred(), "suspend container err")

		By("unsuspend a container")
		err = util.UnsuspendContainer(f.ClientSet, testPod, f.Namespace.Name, containerName)
		Expect(err).NotTo(HaveOccurred(), "unsuspend container err")
	})
})
