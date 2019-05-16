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

type dnsTestCase struct {
	pod                   *v1.Pod
	checkCommand          string
	isCheckWithHostResolv bool
	resultKeywords        []string
	checkMethod           string
	modifiedKeep          bool
}

func doDNSTestCase(f *framework.Framework, testCase *dnsTestCase) {
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

	// Step4: Check command's result.
	By("Check created pod's resolv.conf")
	result := f.ExecShellInContainer(testPod.Name, containerName, testCase.checkCommand)
	framework.Logf("command result: %v", result)

	if !testCase.isCheckWithHostResolv {
		checkResult(testCase.checkMethod, result, testCase.resultKeywords)
	} else {
		// Get resolv.conf of physical server.
		content := getResolvFromHost(hostIP)
		if content == "" {
			framework.Failf("Failed to get reslov.conf from %s", hostIP)
		}
		// Remove '/n' in the result.
		content = strings.Replace(content, "\n", " ", -1)
		// Split content and check the result.
		checkResult(checkMethodContain, result, strings.Split(content, " "))
	}

	// Step5: Do restart test
	// Restart test will restart all containers and check the modification in resolv.conf.
	By("Do restart test")
	// Modify resolv.conf in container.
	keyword := "nameserver 8.81.8.81"
	modifiedCommand := "echo " + keyword + " >> /etc/resolv.conf"
	result = f.ExecShellInContainer(testPod.Name, containerName, modifiedCommand)

	// Stop all cotnainers in pod(include pause container).
	runtimeType, err := util.GetContainerDType(hostIP)
	Expect(err).NotTo(HaveOccurred(), "get runtime type error")

	stopCommand := ""
	if runtimeType == util.ContainerdTypeDocker {
		stopCommand = fmt.Sprintf("cmd://docker(stop $(docker ps | grep %s | awk '{print $1}'))", string(getPod.UID))
	} else {
		stopCommand = fmt.Sprintf("cmd://pouch(stop $(pouch ps | grep %s | awk '{print $1}'))", string(getPod.UID))
	}
	hostSn := util.GetHostSnFromHostIp(hostIP)
	_, err = util.ResponseFromStarAgentTask(stopCommand, hostIP, hostSn)
	Expect(err).NotTo(HaveOccurred(), "stop container failed")

	// Wait 90 second to get all containers are started.
	// TODO: find a better way to check.
	time.Sleep(time.Duration(90) * time.Second)

	// Get container's resolv.conf.
	result = f.ExecShellInContainer(testPod.Name, containerName, testCase.checkCommand)
	framework.Logf("command result after restart: %v", result)

	// Do check after container is restarted.
	if testCase.modifiedKeep {
		checkResult(checkMethodContain, result, []string{keyword})
	}
}

// getResolvFromHost gets resolv.conf from physical server by staragent.
func getResolvFromHost(hostIP string) string {
	hostSn := util.GetHostSnFromHostIp(hostIP)
	cmd := "cmd://cat(/etc/resolv.conf)"
	resp, err := util.ResponseFromStarAgentTask(cmd, hostIP, hostSn)
	if err != nil {
		return ""
	}
	return resp
}

var _ = Describe("[sigma-kubelet][alidocker-dns] check AliDocker's resolv.conf", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	labelHostDNS := "ali.host.dns"
	// TODO: Remove [ant] when new version pouch suppport this.
	It("[smoke][ant]ali.host.dns=true and pod has dnsConfig", func() {
		pod := generateRunningPod()
		pod.Labels[labelHostDNS] = "true"
		pod.Labels[sigmak8sapi.LabelServerType] = sigmak8sapi.PodLabelDockerVM
		pod.Spec.DNSPolicy = v1.DNSNone
		valueStr := "2"
		pod.Spec.DNSConfig = &v1.PodDNSConfig{
			Nameservers: []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"},
			Searches:    []string{"stable.alipay.net"},
			Options: []v1.PodDNSConfigOption{
				v1.PodDNSConfigOption{
					Name:  "timeout",
					Value: &valueStr,
				},
				v1.PodDNSConfigOption{
					Name:  "attempts",
					Value: &valueStr,
				},
				v1.PodDNSConfigOption{
					Name: "rotate",
				},
			},
		}
		testCase := dnsTestCase{
			pod:                   pod,
			checkCommand:          "cat /etc/resolv.conf",
			isCheckWithHostResolv: false,
			resultKeywords:        []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "stable.alipay.net", "timeout:2", "attempts:2", "rotate"},
			checkMethod:           checkMethodContain,
			modifiedKeep:          true,
		}
		doDNSTestCase(f, &testCase)
	})

	It("ali.host.dns=true and pod has no dnsConfig", func() {
		pod := generateRunningPod()
		pod.Labels[labelHostDNS] = "true"

		testCase := dnsTestCase{
			pod:                   pod,
			checkCommand:          "cat /etc/resolv.conf",
			isCheckWithHostResolv: true,
			modifiedKeep:          true,
		}
		doDNSTestCase(f, &testCase)
	})

	It("ali.host.dns=false and pod has dnsConfig", func() {
		pod := generateRunningPod()
		pod.Labels[labelHostDNS] = "false"
		pod.Spec.DNSPolicy = v1.DNSNone
		valueStr := "2"
		pod.Spec.DNSConfig = &v1.PodDNSConfig{
			Nameservers: []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"},
			Searches:    []string{"stable.alipay.net"},
			Options: []v1.PodDNSConfigOption{
				v1.PodDNSConfigOption{
					Name:  "timeout",
					Value: &valueStr,
				},
				v1.PodDNSConfigOption{
					Name:  "attempts",
					Value: &valueStr,
				},
				v1.PodDNSConfigOption{
					Name: "rotate",
				},
			},
		}
		testCase := dnsTestCase{
			pod:                   pod,
			checkCommand:          "cat /etc/resolv.conf",
			isCheckWithHostResolv: false,
			resultKeywords:        []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "stable.alipay.net", "timeout:2", "attempts:2", "rotate"},
			checkMethod:           checkMethodContain,
			modifiedKeep:          false,
		}
		doDNSTestCase(f, &testCase)
	})

	It("ali.host.dns=false and pod has no dnsConfig", func() {
		pod := generateRunningPod()
		pod.Labels[labelHostDNS] = "false"

		testCase := dnsTestCase{
			pod:                   pod,
			checkCommand:          "cat /etc/resolv.conf",
			isCheckWithHostResolv: true,
			modifiedKeep:          false,
		}
		doDNSTestCase(f, &testCase)
	})
})
