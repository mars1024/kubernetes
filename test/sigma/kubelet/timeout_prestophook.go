package kubelet

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
	"time"
)

type TimeoutPreStopHookTestCase struct {
	pod         *v1.Pod
	timeSeconds int
}

func doTimeoutPreStopTestCase(f *framework.Framework, testCase *TimeoutPreStopHookTestCase) {
	pod := testCase.pod
	timeSeconds := testCase.timeSeconds

	// Step1: Create pod
	By("Create a pod")
	testPod := f.PodClient().Create(pod)
	defer util.DeletePod(f.ClientSet, testPod)

	By("Waiting for pods to come up.")
	err := framework.WaitForPodRunningInNamespace(f.ClientSet, testPod)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	// Step2: Wait for pod's deletion.
	err = util.DeletePodWithTimeout(f.ClientSet, testPod, time.Duration(timeSeconds)*time.Second)
	Expect(err.Error()).To(HavePrefix("Gave up"), "pod's stop timeout")
}

var _ = Describe("[sigma-kubelet][timeout-prestophook] prestophook timeout", func() {
	f := framework.NewDefaultFramework("e2e-ak8s-kubelet")
	It("[p3]custom timeout: 80s", func() {
		By("generate running pod")
		pod := generateRunningPod()

		By("add pre-stop hook and timeout to the pod")
		var gracePeriodLocal int64 = 10
		pod.DeletionGracePeriodSeconds = &gracePeriodLocal

		lifecycle := &v1.Lifecycle{
			PreStop: &v1.Handler{
				Exec: &v1.ExecAction{[]string{"/bin/bash", "-c", " sleep 1000s"}},
			},
		}
		pod.Spec.Containers[0].Lifecycle = lifecycle
		// Set pre-stop timeout = 80s.
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations[sigmak8sapi.AnnotationContainerExtraConfig] = `{"containerConfigs":{"pod-base":{"PreStopHookTimeoutSeconds":"100"}}}`

		testCase := TimeoutPreStopHookTestCase{
			pod:         pod,
			timeSeconds: 80,
		}
		doTimeoutPreStopTestCase(f, &testCase)
	})
})
