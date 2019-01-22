package kubelet

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type hostsTestCase struct {
	pod            *v1.Pod
	checkCommand   string
	isHostDNS      bool
	resultKeywords []string
	checkMethod    string
	modifiedKeep   bool
}

func doHostsTestCase(f *framework.Framework, testCase *hostsTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name

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

	hostname := getHostnameFromHost(hostIP)

	// Step4: Check command's result.
	By("Check created pod's hosts")
	result := f.ExecShellInContainer(testPod.Name, containerName, testCase.checkCommand)
	framework.Logf("command result: %v", result)

	// Always  check with keywords
	checkResult(testCase.checkMethod, result, testCase.resultKeywords)
	// Check with host's hosts file if needed.
	if testCase.isHostDNS {
		// Get hosts of physical server.
		content := getHostsFromHost(hostIP)
		if content == "" {
			framework.Failf("Failed to get reslov.conf from %s", hostIP)
		}
		// Remove '/n' in the result.
		content = strings.Replace(content, "\n", " ", -1)
		keywords := strings.Split(content, " ")
		keywords = append(keywords, hostname)
		// Split content and check the result.
		checkResult(checkMethodContain, result, keywords)
	}

	// Step5: Do restart test
	// Restart test will restart all containers and check the modification in hosts file.
	By("Do restart test")
	// Modify resolv.conf in container.
	keyword := "8.81.8.81 localhost.localdomain99"
	modifiedCommand := "echo " + keyword + " >> /etc/hosts"
	result = f.ExecShellInContainer(testPod.Name, containerName, modifiedCommand)

	// Stop all cotnainers in pod(include pause container).
	stopCommand := fmt.Sprintf("cmd://docker(stop $(docker ps | grep %s | awk '{print $1}'))", string(getPod.UID))
	hostSn := util.GetHostSnFromHostIp(hostIP)
	_, err = util.ResponseFromStarAgentTask(stopCommand, hostIP, hostSn)
	Expect(err).NotTo(HaveOccurred(), "stop container failed")

	// Wait 20 second to get all containers are started.
	time.Sleep(time.Duration(20) * time.Second)

	// Get container's hosts.
	result = f.ExecShellInContainer(testPod.Name, containerName, testCase.checkCommand)
	framework.Logf("command result after restart: %v", result)

	// Do check after container is restarted.
	if testCase.modifiedKeep {
		checkResult(checkMethodContain, result, strings.Split(keyword, " "))
	} else {
		checkResult(checkMethodNotContain, result, strings.Split(keyword, " "))
	}
}

// getHostnameFromHost gets resolv.conf from physical server by staragent.
func getHostnameFromHost(hostIP string) string {
	hostSn := util.GetHostSnFromHostIp(hostIP)
	cmd := "cmd://hostname"
	resp, err := util.ResponseFromStarAgentTask(cmd, hostIP, hostSn)
	if err != nil {
		return ""
	}
	resp = strings.Replace(resp, "\n", "", -1)
	return resp
}

// getHostsFromHost gets resolv.conf from physical server by staragent.
func getHostsFromHost(hostIP string) string {
	hostSn := util.GetHostSnFromHostIp(hostIP)
	cmd := "cmd://cat(/etc/hosts)"
	resp, err := util.ResponseFromStarAgentTask(cmd, hostIP, hostSn)
	if err != nil {
		return ""
	}
	resp = strings.Replace(resp, "\n", " ", -1)
	return resp
}

var _ = Describe("[sigma-kubelet][alidocker-hosts] check AliDocker's HostAliases", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	labelHostDNS := "ali.host.dns"
	It("[smoke][ant]ali.host.dns=true and pod has HostAliases", func() {
		pod := generateRunningPod()
		pod.Labels[labelHostDNS] = "true"
		pod.Spec.HostAliases = []v1.HostAlias{
			v1.HostAlias{
				Hostnames: []string{"localhost.localdomain11", "localhost.localdomain22"},
				IP:        "1.1.1.1",
			},
		}
		testCase := hostsTestCase{
			pod:            pod,
			checkCommand:   "cat /etc/hosts",
			isHostDNS:      true,
			resultKeywords: []string{"localhost.localdomain11", "localhost.localdomain22", "1.1.1.1"},
			checkMethod:    checkMethodContain,
			modifiedKeep:   true,
		}
		doHostsTestCase(f, &testCase)
	})

	It("[smoke][ant]ali.host.dns=true, pod has HostAliases and hostDomainName", func() {
		pod := generateRunningPod()
		pod.Labels[labelHostDNS] = "true"
		pod.Annotations[sigmak8sapi.AnnotationPodHostNameTemplate] = "sigmaslave110.alipay.com"
		pod.Spec.HostAliases = []v1.HostAlias{
			v1.HostAlias{
				Hostnames: []string{"localhost.localdomain11", "localhost.localdomain22"},
				IP:        "1.1.1.1",
			},
		}
		testCase := hostsTestCase{
			pod:            pod,
			checkCommand:   "cat /etc/hosts",
			isHostDNS:      true,
			resultKeywords: []string{"localhost.localdomain11", "localhost.localdomain22", "1.1.1.1"},
			checkMethod:    checkMethodContain,
			modifiedKeep:   true,
		}
		doHostsTestCase(f, &testCase)
	})

	It("[ant]ali.host.dns=true and pod has no HostAliases", func() {
		pod := generateRunningPod()
		pod.Labels[labelHostDNS] = "true"
		testCase := hostsTestCase{
			pod:            pod,
			checkCommand:   "cat /etc/hosts",
			isHostDNS:      true,
			resultKeywords: []string{"localhost.localdomain11", "localhost.localdomain22", "1.1.1.1"},
			checkMethod:    checkMethodNotContain,
			modifiedKeep:   true,
		}
		doHostsTestCase(f, &testCase)
	})

	It("[ant]ali.host.dns=false and pod has HostAliases", func() {
		pod := generateRunningPod()
		pod.Labels[labelHostDNS] = "false"
		pod.Spec.HostAliases = []v1.HostAlias{
			v1.HostAlias{
				Hostnames: []string{"localhost.localdomain11", "localhost.localdomain22"},
				IP:        "1.1.1.1",
			},
		}
		testCase := hostsTestCase{
			pod:            pod,
			checkCommand:   "cat /etc/hosts",
			isHostDNS:      false,
			resultKeywords: []string{"localhost.localdomain11", "localhost.localdomain22", "1.1.1.1"},
			checkMethod:    checkMethodContain,
			modifiedKeep:   true,
		}
		doHostsTestCase(f, &testCase)
	})

	It("[ant]ali.host.dns=false and pod has no HostAliases", func() {
		pod := generateRunningPod()
		pod.Labels[labelHostDNS] = "false"
		testCase := hostsTestCase{
			pod:            pod,
			checkCommand:   "cat /etc/hosts",
			isHostDNS:      false,
			resultKeywords: []string{"localhost.localdomain11", "localhost.localdomain22", "1.1.1.1"},
			checkMethod:    checkMethodNotContain,
			modifiedKeep:   true,
		}
		doHostsTestCase(f, &testCase)
	})
})
