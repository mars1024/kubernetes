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

var _ = Describe("[sigma-kubelet]", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	caseName := "[upgrade_pod_with_protection_finalizer]"
	It("[sigma-kubelet]"+caseName, func() {
		patchData := `{"spec":{"containers":[{"name":"pod-base","image":"reg.docker.alibaba-inc.com/sigma-x/mysql:test-v2"}]}}`
		upgradeSuccessStr := "upgrade container success"

		// Step1: Create a pod.
		By(caseName + "create a pod from file")
		pod := generateRunningPod()
		pod.Finalizers = []string{"protection.pod.beta1.sigma.ali/test-protection"}
		containerName := pod.Spec.Containers[0].Name
		testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		defer util.DeletePod(f.ClientSet, testPod)

		// Step2: Wait for pod's creation finished.
		By(caseName + "wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

		// Step4: Upgrade pod
		By(caseName + "update container image to trigger upgrade")
		upgradedPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(patchData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		// Step5: Wait for upgrade action finished.
		By("wait until pod is upgraded")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 3*time.Minute, upgradeSuccessStr)
		Expect(err).NotTo(HaveOccurred(), "upgrade pod err")

		// Step6: Get latest pod to check pod status
		By(caseName + "check pod still running")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "get pod err")
		Expect(getPod.Status.Phase).To(Equal(v1.PodRunning))

		// Step7: Update pod to remove protection finalizer
		By(caseName + "remove protection finalizer")
		getPod.Finalizers = []string{}
		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Update(getPod) // patch doesn't work on list of simple type, finalizers is a list of string
		Expect(err).NotTo(HaveOccurred(), "update pod err")

		// Step8: use defer to delete pod again after remove protection finalizer
		By(caseName + "delete pod again by defer after remove protection finalizer")
	})
})
