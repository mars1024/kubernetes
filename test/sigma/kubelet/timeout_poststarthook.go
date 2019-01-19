package kubelet

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type TimeoutPostStartHookTestCase struct {
	pod         *v1.Pod
	timeSeconds int
}

func doTimeoutPostStartTestCase(f *framework.Framework, testCase *TimeoutPostStartHookTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name
	timeoutErr := "DeadlineExceeded"

	// Step1: Create pod
	testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished, but get timeout error.
	By("wait until pod running and have pod/host IP, should be timeout")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, time.Duration(testCase.timeSeconds)*time.Second)
	Expect(err).To(HaveOccurred(), "pod status is not running")

	// Step3: Wait for container's postStartHook timeout.
	By("wait until pod's poststarthook timeout")
	err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, containerName, 3*time.Minute, timeoutErr, false)
	Expect(err).NotTo(HaveOccurred(), "pod's poststarthook timeout")
}

var _ = Describe("[sigma-kubelet][timeout-poststarthook] poststarthook timeout", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("default timeout: 2min", func() {
		pod := generateRunningPod()
		lifecycle := &v1.Lifecycle{
			PostStart: &v1.Handler{
				Exec: &v1.ExecAction{[]string{"/bin/bash", "-c", " sleep 1000"}},
			},
		}
		pod.Spec.Containers[0].Lifecycle = lifecycle
		// timeout = max(2min, PostStartHookTimeoutSeconds), so the timeout is 2min in this case.
		// Wait 100s, the container shouldn't in running state. and 20s later, postStartHook timeout error happens.
		pod.Annotations[sigmak8sapi.AnnotationContainerExtraConfig] = `{"containerConfigs":{"pod-base":{"PostStartHookTimeoutSeconds":"10"}}}`
		testCase := TimeoutPostStartHookTestCase{
			pod:         pod,
			timeSeconds: 100,
		}
		doTimeoutPostStartTestCase(f, &testCase)
	})
	It("custom timeout: 2.5min", func() {
		pod := generateRunningPod()
		lifecycle := &v1.Lifecycle{
			PostStart: &v1.Handler{
				Exec: &v1.ExecAction{[]string{"/bin/bash", "-c", " sleep 1000"}},
			},
		}
		pod.Spec.Containers[0].Lifecycle = lifecycle
		// timeout = max(2min, PostStartHookTimeoutSeconds), so the timeout is 2.5min(150 second) in this case.
		// Wait 130s, the container shouldn't in running state. and 20s later, postStartHook timeout error happens.
		pod.Annotations[sigmak8sapi.AnnotationContainerExtraConfig] = `{"containerConfigs":{"pod-base":{"PostStartHookTimeoutSeconds":"150"}}}`
		testCase := TimeoutPostStartHookTestCase{
			pod:         pod,
			timeSeconds: 130,
		}
		doTimeoutPostStartTestCase(f, &testCase)
	})
})
