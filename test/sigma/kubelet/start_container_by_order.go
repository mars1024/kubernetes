package kubelet

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-kubelet][start-container-by-order] start containers in pod by order", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	It("[smoke][ant]create and start containers by order", func() {
		postStartHookErr := "No such file or directory"

		// Generate pod.
		// Containers: pod-base, pod-base1, pod-base2
		pod := generateMultiConRunningPod()
		// pod-base1 can't start successfully
		lifecycle := &v1.Lifecycle{
			PostStart: &v1.Handler{
				Exec: &v1.ExecAction{[]string{"/bin/bash", "-c", "cat /home/hello"}},
			},
		}
		pod.Spec.Containers[1].Lifecycle = lifecycle

		// Step1: Create pod
		By("create pod")
		testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		// Step2: Wait for container's creation finished
		By("wait until pod get an error")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, pod.Spec.Containers[1].Name, 3*time.Minute, postStartHookErr, false)
		Expect(err).NotTo(HaveOccurred(), "wait pod's poststarthook error timeout")

		// Step3: Check containers
		By("check containers")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "get pod err")
		// Container pod-base is running.
		containerState := util.GetContainerUpdateStatus(getPod, getPod.Spec.Containers[0].Name)
		Expect(containerState.CurrentState).Should(Equal(sigmak8sapi.ContainerStateRunning))
		// Container pod-base1 is not running.
		containerState1 := util.GetContainerUpdateStatus(getPod, getPod.Spec.Containers[1].Name)
		Expect(containerState1.CurrentState).ShouldNot(Equal(sigmak8sapi.ContainerStateRunning))
		// Container pod-base2 is not created because of pod-base1.
		containerState2 := util.GetContainerUpdateStatus(getPod, getPod.Spec.Containers[2].Name)
		Expect(containerState2).Should(BeNil())

		// Step4: Update container so pod can be in running state.
		patchData := fmt.Sprintf(
			`{"spec":{"containers":[{"name":"%s","lifecycle":{"postStart":{"exec":{"command":["/bin/bash", "-c", "touch /home/hello"]}}}}]}}`,
			pod.Spec.Containers[1].Name)

		By("change container's postStartHook command")
		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(pod.Name, types.StrategicMergePatchType, []byte(patchData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		// Step5: Wait for container's creation finished
		By("wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")
	})

	It("[smoke][ant]start exited containers by order", func() {
		successKillStr := "kill container success"
		postStartHookErr := "No such file or directory"

		// Generate pod.
		// Containers: pod-base, pod-base1, pod-base2
		pod := generateMultiConRunningPod()
		// container1 can start successfully
		lifecycle := &v1.Lifecycle{
			PostStart: &v1.Handler{
				Exec: &v1.ExecAction{[]string{"/bin/bash", "-c", "touch /home/hello"}},
			},
		}
		pod.Spec.Containers[1].Lifecycle = lifecycle

		// Step1: Create pod
		By("create pod")
		testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		// Step2: Wait for container's creation finished
		By("wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

		// Step3: Stop all containers
		By("stop all containers")
		patchData := fmt.Sprintf(
			`{"metadata":{"annotations":{"pod.beta1.sigma.ali/container-state-spec":"{\"states\":{\"%s\":\"%s\",\"%s\":\"%s\",\"%s\":\"%s\"}}"}}}`,
			pod.Spec.Containers[0].Name, sigmak8sapi.ContainerStateExited,
			pod.Spec.Containers[1].Name, sigmak8sapi.ContainerStateExited,
			pod.Spec.Containers[2].Name, sigmak8sapi.ContainerStateExited)

		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(pod.Name, types.StrategicMergePatchType, []byte(patchData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		// Step4: Wait all containers are exited
		By("wait until all containers are exited")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, pod.Spec.Containers[0].Name, 3*time.Minute, successKillStr, true)
		Expect(err).NotTo(HaveOccurred(), "wait container0's termination error")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, pod.Spec.Containers[1].Name, 3*time.Minute, successKillStr, true)
		Expect(err).NotTo(HaveOccurred(), "wait container1's termination error")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, pod.Spec.Containers[2].Name, 3*time.Minute, successKillStr, true)
		Expect(err).NotTo(HaveOccurred(), "wait container2's termination error")

		// Step5: Update container1's postStartHook so container1 can't be started successfully.
		By("change container's postStartHook command")
		patchData = fmt.Sprintf(
			`{"spec":{"containers":[{"name":"%s","lifecycle":{"postStart":{"exec":{"command":["/bin/bash", "-c", "cat /home/helloNotExist"]}}}}]}}`,
			pod.Spec.Containers[1].Name)

		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(pod.Name, types.StrategicMergePatchType, []byte(patchData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		// Step6: Start all containers
		By("start all containers")
		patchData = fmt.Sprintf(
			`{"metadata":{"annotations":{"pod.beta1.sigma.ali/container-state-spec":"{\"states\":{\"%s\":\"%s\",\"%s\":\"%s\",\"%s\":\"%s\"}}"}}}`,
			pod.Spec.Containers[0].Name, sigmak8sapi.ContainerStateRunning,
			pod.Spec.Containers[1].Name, sigmak8sapi.ContainerStateRunning,
			pod.Spec.Containers[2].Name, sigmak8sapi.ContainerStateRunning)

		By("change container's postStartHook command (start failed)")
		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(pod.Name, types.StrategicMergePatchType, []byte(patchData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		// Step7: Wait for container1 get an error
		By("wait until pod get an error")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, pod.Spec.Containers[1].Name, 3*time.Minute, postStartHookErr, false)
		Expect(err).NotTo(HaveOccurred(), "wait pod's poststarthook error")

		// Step8: check containers
		By("check upgraded pod")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "get pod err")
		// Container pod-base is running.
		containerState := util.GetContainerUpdateStatus(getPod, getPod.Spec.Containers[0].Name)
		Expect(containerState.CurrentState).Should(Equal(sigmak8sapi.ContainerStateRunning))
		// Container pod-base1 is not running.
		containerState1 := util.GetContainerUpdateStatus(getPod, getPod.Spec.Containers[1].Name)
		Expect(containerState1.CurrentState).ShouldNot(Equal(sigmak8sapi.ContainerStateRunning))
		// Container pod-base2 is in exited.
		containerState2 := util.GetContainerUpdateStatus(getPod, getPod.Spec.Containers[2].Name)
		Expect(containerState2.CurrentState).Should(Equal(sigmak8sapi.ContainerStateExited))

		// Step9: Update container so pod can start successfully.
		By("change container's postStartHook command (start successfully)")
		patchData = fmt.Sprintf(
			`{"spec":{"containers":[{"name":"%s","lifecycle":{"postStart":{"exec":{"command":["/bin/bash", "-c", "touch /home/hello"]}}}}]}}`,
			pod.Spec.Containers[1].Name)

		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(pod.Name, types.StrategicMergePatchType, []byte(patchData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		// Step10: Wait for container's creation finished
		By("wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")
	})
})
