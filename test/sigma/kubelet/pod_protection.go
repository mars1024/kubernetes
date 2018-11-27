package kubelet

import (
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-kubelet]", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	caseName := "[pod_protection]"
	It("[sigma-kubelet]"+caseName, func() {
		podFileName := "pod-base.json"
		patchDataAdd := `{"metadata":{"finalizers":["protection.pod.beta1.sigma.ali/test-protection"]}}`

		// Step1: Create a pod.
		By(caseName + "create a pod from file")
		podFile := filepath.Join(util.TestDataDir, podFileName)
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred())

		// name should be unique
		pod.Name = "protectpodtest" + string(uuid.NewUUID())

		testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		defer util.DeletePod(f.ClientSet, testPod)

		// Step2: Wait for container's creation finished.
		By(caseName + "wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

		// Step3: Update container to add protection finalizer as controller did
		By(caseName + "add protection finalizer")
		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(patchDataAdd))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		time.Sleep(2500 * time.Millisecond) // sleep a while before we delete pod

		// Step4: delete pod
		By(caseName + "delete pod with grace period 30s")
		err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Delete(testPod.Name, metav1.NewDeleteOptions(30))
		Expect(err).NotTo(HaveOccurred(), "delete pod err")

		// Step5: sleep to wait grace window expire
		time.Sleep(30 * time.Second)

		// Step6: get latest pod to check pod status
		By(caseName + "check pod still running")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "get pod err")
		Expect(getPod.Status.Phase).To(Equal(v1.PodRunning))

		// Step7: Update container to remove protection finalizer as controller did
		By(caseName + "remove protection finalizer")
		getPod.Finalizers = []string{}
		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Update(getPod) // patch doesn't work on list of simple type, finalizers is a list of string
		Expect(err).NotTo(HaveOccurred(), "update pod err")

		// Step8: use defer to delete pod again after remove protection finalizer
		By(caseName + "delete pod again by defer after remove protection finalizer")
	})
})
