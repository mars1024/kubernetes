package kubelet

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"path/filepath"
	"strings"

	"time"

	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-kubelet] Sandbox restart check", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	var testPod *v1.Pod
	It("[smoke] Create a pod and restart sandbox, pod shouldn't recreate, should just restart", func() {
		By("Load a pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "fail to load pod from file")

		By("Create a pod")
		testPod = f.PodClient().Create(pod)
		defer util.DeletePod(f.ClientSet, testPod)

		By("Waiting for pods to come up.")
		err = framework.WaitForPodRunningInNamespace(f.ClientSet, testPod)
		framework.ExpectNoError(err, "waiting for server pod to start")

		By("Query container ID")
		podInfo, err := f.PodClient().Get(pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "fail query sandboxId and container ID")

		By("Parse containerID")
		containerInfoMap := make(map[string]string, len(podInfo.Status.ContainerStatuses))
		for _, containerInfo := range podInfo.Status.ContainerStatuses {
			containerInfoMap[containerInfo.Name] = containerInfo.ContainerID
		}
		Expect(len(containerInfoMap)).Should(Equal(len(pod.Spec.Containers)))

		By("Parse pod ip")
		podIP := podInfo.Status.PodIP
		framework.Logf("pod ip is %q", podIP)

		By("Get sandbox containerID")
		containerList, err := util.ListContainers(podInfo.Status.HostIP, false)
		Expect(err).NotTo(HaveOccurred())
		var containerID []string
		// container name pkg/kubelet/dockershim/naming.go func  makeSandboxName()
		expectContainer := fmt.Sprintf("%s_%s_%s", podInfo.Name, podInfo.Namespace, string(podInfo.GetObjectMeta().GetUID()))
		framework.Logf("expect container name %s", expectContainer)

		containerType, err := util.GetContainerDType(podInfo.Status.HostIP)
		Expect(err).NotTo(HaveOccurred())

		for _, value := range containerList {
			if !strings.Contains(value, expectContainer) {
				continue
			}
			switch containerType {
			case util.ContainerdTypePouch:
				containerID = append(containerID, strings.Fields(value)[1])
			case util.ContainerdTypeDocker:
				containerID = append(containerID, strings.Fields(value)[0])
			default:
				framework.Failf("container type can't identify")
			}
		}
		Expect(containerID).NotTo(BeEmpty())
		framework.Logf("containerID  is %s", strings.Join(containerID, " "))

		By("Stop container to simulate host reboot")
		podStartTime := time.Now()
		success, err := util.ContainerStop(podInfo.Status.HostIP, strings.Join(containerID, " "))
		Expect(err).NotTo(HaveOccurred())
		Expect(success).Should(BeTrue())
		framework.Logf("container stop time is %s", podStartTime.String())

		By("Waiting for pods to come up.")
		err = framework.WaitForPodRunningInNamespace(f.ClientSet, testPod)
		framework.ExpectNoError(err, "waiting for server pod to start")

		By("Waiting for pods container to restart.")
		err = util.WaitForPodContainerRestartInNamespace(f.ClientSet, testPod, podStartTime)
		framework.ExpectNoError(err)

		By("Query and parse container ID")
		podInfo, err = f.PodClient().Get(pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "fail query sandboxId and container ID")

		containerInfoMapAfterStart := make(map[string]string, len(podInfo.Status.ContainerStatuses))
		for _, containerInfo := range podInfo.Status.ContainerStatuses {
			containerInfoMapAfterStart[containerInfo.Name] = containerInfo.ContainerID
		}
		Expect(len(containerInfoMapAfterStart)).Should(Equal(len(pod.Spec.Containers)))

		By("container check and podIP check")
		Expect(podIP).Should(Equal(podInfo.Status.PodIP))
	})
})
