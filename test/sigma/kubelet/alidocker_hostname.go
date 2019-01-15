package kubelet

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

// Compare hostname with pod name if usePodName is "true",
// else compare with expectedHostname.
type hostnameTestCase struct {
	pod              *v1.Pod
	usePodName       bool
	expectedHostname string
}

func doHostnameTestCase(f *framework.Framework, testCase *hostnameTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name
	expectedHostname := testCase.expectedHostname

	if testCase.usePodName {
		expectedHostname = pod.Name
	}

	// Step1: Create pod
	By("create pod")
	testPod, err := util.CreatePod(f.ClientSet, pod, f.Namespace.Name)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Step3: Get created container
	By("Get created pod")
	getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get pod err")
	hostIP := getPod.Status.HostIP
	hostSn := util.GetHostSnFromHostIp(hostIP)

	// Step4: Check the hostname
	By("Check the hostname")
	result := f.ExecShellInContainer(testPod.Name, containerName, "hostname")
	framework.Logf("Command 'hostname' result:", result)
	Expect(result).Should(Equal(expectedHostname))
	result = f.ExecShellInContainer(testPod.Name, containerName, "cat /etc/hostname")
	framework.Logf("Content of /etc/hostname:", result)
	Expect(result).Should(Equal(expectedHostname))

	// Step5: Stop all cotnainers in pod(include pause container).
	By("Restart all containers")
	stopCommand := fmt.Sprintf("cmd://docker(stop $(docker ps | grep %s | awk '{print $1}'))", string(getPod.UID))
	_, err = util.ResponseFromStarAgentTask(stopCommand, hostIP, hostSn)
	Expect(err).NotTo(HaveOccurred(), "stop container failed")

	// Wait 20 second to get all containers are started.
	time.Sleep(time.Duration(20) * time.Second)

	// Step6: Check the hostname after restart
	By("Check the hostname after restart")
	result = f.ExecShellInContainer(testPod.Name, containerName, "hostname")
	framework.Logf("Command 'hostname' result:", result)
	Expect(result).Should(Equal(expectedHostname))
	result = f.ExecShellInContainer(testPod.Name, containerName, "cat /etc/hostname")
	framework.Logf("Content of /etc/hostname:", result)
	Expect(result).Should(Equal(expectedHostname))

}

var _ = Describe("[sigma-kubelet][alidocker-hostname] check AliDocker's hostname", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	labelHostDNS := "ali.host.dns"
	podHostname := "sigma-slave110.alipay.com"
	It("[smoke][ant]ali.host.dns=true and pod has hostname specified", func() {
		pod := generateRunningPod()
		pod.Labels[labelHostDNS] = "true"
		pod.Annotations[sigmak8sapi.AnnotationPodHostNameTemplate] = podHostname

		testCase := hostnameTestCase{
			pod:              pod,
			expectedHostname: podHostname,
		}
		doHostnameTestCase(f, &testCase)
	})

	It("[ant]ali.host.dns=true and pod has no hostname specified", func() {
		pod := generateRunningPod()
		pod.Labels[labelHostDNS] = "true"

		testCase := hostnameTestCase{
			pod:        pod,
			usePodName: true,
		}
		doHostnameTestCase(f, &testCase)
	})

	It("[ant]ali.host.dns=false and pod has hostname specified", func() {
		pod := generateRunningPod()
		pod.Annotations[sigmak8sapi.AnnotationPodHostNameTemplate] = podHostname

		testCase := hostnameTestCase{
			pod:              pod,
			expectedHostname: podHostname,
		}
		doHostnameTestCase(f, &testCase)
	})

	It("[ant]ali.host.dns=false and pod has no hostname specified", func() {
		pod := generateRunningPod()

		testCase := hostnameTestCase{
			pod:        pod,
			usePodName: true,
		}
		doHostnameTestCase(f, &testCase)
	})
})
