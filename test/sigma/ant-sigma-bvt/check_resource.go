package ant_sigma_bvt

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/samalba/dockerclient"
	k8sApi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"gitlab.alipay-inc.com/sigma/clients/armory"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/json"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/swarm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//CheckAdapterCreateResource()  check created container info by adapter, cpu/disk/mem/hostname/ip/env/network/armory.
func CheckAdapterCreateResource(f *framework.Framework, testPod *v1.Pod, result *swarm.AllocResult, createConfig *dockerclient.ContainerConfig) {
	By("sigma-adapter: check container hostname should same as pod.")
	cmd := []string{"hostname"}
	stdout, _, err := RetryExec(f, testPod, cmd, "check_hostname", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] check 3.1 pod hostname error")
	Expect(stdout).To(Equal(result.ContainerHn), "[AdapterLifeCycle] 3.1 pod hostname is not equal with input.")

	By("sigma-adapter: check container memory should same as pod.")
	cmd = []string{"cat", "/proc/meminfo"}
	stdout, _, err = RetryExec(f, testPod, cmd, "check_mem", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] get 3.1 pod memory error")
	memory := Atoi64(createConfig.Labels["ali.MemoryHardlimit"], 0)
	isEqual := CompareMemory(memory, stdout)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] check 3.1 pod mem is not equal with input.")
	Expect(isEqual).To(BeTrue(), "[AdapterLifeCycle] check 3.1 pod mem is not equal with input.")

	By("sigma-adapter: check container cpu should same as pod.")
	cmd = []string{"cat", "/proc/cpuinfo"}
	stdout, _, err = RetryExec(f, testPod, cmd, "check_cpu", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] get 3.1 pod cpu error")
	cpu := Atoi64(createConfig.Labels["ali.CpuCount"], 0)
	isEqual = CompareCPU(cpu, stdout)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] check 3.1 pod cpu is not equal with input.")
	Expect(isEqual).To(BeTrue(), "[AdapterLifeCycle] check 3.1 pod cpu is not equal with input.")

	By("sigma-adapter: check container disksize should same as pod.")
	cmd = []string{"df", "-h"}
	stdout, _, err = RetryExec(f, testPod, cmd, "check_disk", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] get 3.1 pod disksize error")
	disk := Quota2Byte(createConfig.Labels["ali.DiskSize"])
	isEqual = CompareDisk(disk, stdout)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] check 3.1 pod disksize is not equal with input.")
	Expect(isEqual).To(BeTrue(), "[AdapterLifeCycle] check 3.1 pod disksize is not equal with input.")

	By("sigma-adapter: check container env should same as pod.")
	cmd = []string{"env"}
	stdout, _, err = RetryExec(f, testPod, cmd, "check_env", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] get 3.1 pod env error")
	isEqual = CompareENV(createConfig.Env, stdout)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] check 3.1 pod env is not equal with input.")
	Expect(isEqual).To(BeTrue(), "[AdapterLifeCycle] check 3.1 pod env is not equal with input.")

	By("sigma-adapter: check container network settings.")
	CheckNewWorkSettings(f, testPod)

	req_net_priority := createConfig.Labels["ali.NetPriority"]
	net_prority := testPod.Annotations[k8sApi.AnnotationNetPriority]
	Expect(req_net_priority).To(Equal(net_prority), "[AdapterLifeCycle] Unexpected net-priority.")

	By("sigma-adapter: check container armory info.")
	CheckArmory(testPod)

	By("sigma-adapter:check container dnsConfig")
	checkDNSPolicy(f, testPod)
}

//CheckAdapterUpgradeResource() check resource upgraded by adapter. check env/ip/network/armory
func CheckAdapterUpgradeResource(f *framework.Framework, testPod *v1.Pod, upgradeConfig *dockerclient.ContainerConfig) () {
	By("sigma-adapter: [upgrade] check container env should same as pod.")
	upPod, err := f.ClientSet.CoreV1().Pods(testPod.Namespace).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] get pod list failed.")
	framework.Logf("newPod env:%v", upPod.Spec.Containers[0].Env)
	cmd := []string{"env"}
	stdout, _, err := RetryExec(f, testPod, cmd, "check_env", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] [upgrade] get 3.1 pod env error")
	isEqual := CompareENV(upgradeConfig.Env, stdout)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] [upgrade] check 3.1 pod env is not equal with input.")
	Expect(isEqual).To(BeTrue(), "[AdapterLifeCycle] [upgrade] check 3.1 pod env is not equal with input.")

	By("sigma-adapter: check container network settings.")
	CheckNewWorkSettings(f, testPod)

	By("sigma-adapter: check container armory info.")
	CheckArmory(testPod)

	By("sigma-adapter:check container dnsConfig")
	checkDNSPolicy(f, testPod)
}

//CheckNewWorkSettings() check network-settings, ip/ping
func CheckNewWorkSettings(f *framework.Framework, testPod *v1.Pod) {
	ip := testPod.Status.PodIP
	cmd := []string{"ifconfig", "eth0"}
	stdout, _, err := RetryExec(f, testPod, cmd, "check_ifconfig", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[AdapterLifeCycle] get 3.1 pod ip error")
	ok := CompareIpAddress(ip, stdout)
	Expect(ok).To(BeTrue(), "[AdapterLifeCycle] get 3.1 pod ip error")

	checkPing(f, testPod)
}

//CheckArmory() check pod ip, same as armory, and pod.Spec.NodeName equals Armory.Vmparent
func CheckArmory(testPod *v1.Pod) {
	framework.Logf("[before armory]Pod status:%#v, annotations:%#v, finalizers:%#v", testPod.Status, testPod.Annotations, testPod.Finalizers)
	err := Retry(func() error {
		armoryClient := armory.NewClient("http://gapi.a.alibaba-inc.com", a.User, a.Key)
		podArmory, err := armoryClient.QueryDevice(fmt.Sprintf("dns_ip=%s", testPod.Status.PodIP))
		if err != nil {
			framework.Logf("[AdapterLifeCycle] get pod armory failed, err:%v.", err)
			return err
		}
		if podArmory == nil || podArmory.Vmparent == "" {
			framework.Logf("[AdapterLifeCycle] get pod armory failed, nil.")
			return fmt.Errorf("pod Armory is nil.")

		}
		nodeArmory, err := armoryClient.QueryDevice(fmt.Sprintf("nodename=%s", podArmory.Vmparent))
		if err != nil {
			framework.Logf("[AdapterLifeCycle] get node armory failed, err:%v.", err)
			return err
		}
		if nodeArmory == nil {
			framework.Logf("[AdapterLifeCycle] get node armory failed, nil.")
			return fmt.Errorf("node Armory is nil.")
		}
		if strings.ToLower(nodeArmory.ServiceTag) == testPod.Spec.NodeName {
			return nil
		}
		return fmt.Errorf("[AdapterLifeCycle] pod hostname should be same as armory info.")
	}, "CheckArmory", 10, 1)
	Expect(err).To(BeNil(), "[AdapterLifeCycle] Check armory info failed.")
}

//checkPing() check ping result, 0% packet loss is expected.
func checkPing(f *framework.Framework, pod *v1.Pod) {
	framework.Logf("[before ping]Pod status:%#v, annotations:%#v", pod.Status, pod.Annotations)
	err := Retry(func() error {
		network, ok := pod.Annotations[k8sApi.AnnotationPodNetworkStats]
		if !ok {
			return fmt.Errorf("pod not ok.")
		}
		networkSettings := &k8sApi.NetworkStatus{}
		err := json.Unmarshal([]byte(network), networkSettings)
		if err != nil {
			return err
		}
		cmd := []string{"ping", "-c", "3", fmt.Sprintf("%s", networkSettings.Gateway)}
		stdout, _, err := GetOptionsUseExec(f, pod, cmd)
		framework.Logf("Ping %v gateway %v stdout:%v", pod.Status.PodIP, networkSettings.Gateway, stdout)
		if err != nil {
			return err
		}
		cmds := exec.Command("sh", "-c", fmt.Sprintf("ping -c 3 %s || echo SIMGA_PING_FAILED", pod.Status.PodIP))
		framework.Logf("Exec command:%v", pod.Status.PodIP)
		out, err := cmds.Output()
		if err != nil {
			framework.Logf("Exec ping failed: %v", err)
			return err
		}
		framework.Logf("Ping result:%v", string(out))
		if err != nil {
			return err
		}
		if strings.Contains(string(out), "SIMGA_PING_FAILED") {
			return fmt.Errorf("Ping result error, out:%v.", string(out))
		}
		return nil
	}, "CheckPing", 10, 1)
	Expect(err).To(BeNil(), "[AdapterLifeCycle] Ping local ip fail.")
}

//check sigma3.1 resource.
//CheckSigmaCreateResource() check created container by sigma3.1, cpu/disk/mem/ip/env/networksettings/armory
func CheckSigmaCreateResource(f *framework.Framework, testPod *v1.Pod) {
	Expect(len(testPod.Spec.Containers) > 0).To(BeTrue(), "[Sigma3.1LifeCycle] check 3.1 pod container error")
	By("sigma 3.1: check container hostname should same as pod.")
	cmd := []string{"hostname"}
	stdout, _, err := RetryExec(f, testPod, cmd, "check_hostname", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] check 3.1 pod hostname error")
	Expect(stdout).To(Equal(testPod.Spec.Hostname), "[Sigma3.1LifeCycle] 3.1 pod hostname is not equal with input.")

	By("sigma 3.1: check container memory should same as pod.")
	cmd = []string{"cat", "/proc/meminfo"}
	stdout, _, err = RetryExec(f, testPod, cmd, "check_mem", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] get 3.1 pod memory error")
	memory, _ := testPod.Spec.Containers[0].Resources.Requests.Memory().AsInt64()
	isEqual := CompareMemory(memory, stdout)
	Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] check 3.1 pod mem is not equal with input.")
	Expect(isEqual).To(BeTrue(), "[Sigma3.1LifeCycle] check 3.1 pod mem is not equal with input.")

	By("sigma 3.1: check container cpu should same as pod.")
	cmd = []string{"cat", "/proc/cpuinfo"}
	stdout, _, err = RetryExec(f, testPod, cmd, "check_cpu", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] get 3.1 pod cpu error")
	cpu, _ := testPod.Spec.Containers[0].Resources.Requests.Cpu().AsInt64()
	isEqual = CompareCPU(cpu, stdout)
	Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] check 3.1 pod cpu is not equal with input.")
	Expect(isEqual).To(BeTrue(), "[Sigma3.1LifeCycle] check 3.1 pod cpu is not equal with input.")

	By("sigma 3.1: check container disksize should same as pod.")
	cmd = []string{"df", "-h"}
	stdout, _, err = RetryExec(f, testPod, cmd, "check_disk", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] get 3.1 pod disksize error")
	disk, _ := testPod.Spec.Containers[0].Resources.Requests.StorageEphemeral().AsInt64()
	isEqual = CompareDisk(disk, stdout)
	Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] check 3.1 pod disksize is not equal with input.")
	Expect(isEqual).To(BeTrue(), "[Sigma3.1LifeCycle] check 3.1 pod disksize is not equal with input.")

	By("sigma 3.1: check container env should same as pod.")
	cmd = []string{"env"}
	stdout, _, err = RetryExec(f, testPod, cmd, "check_env", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] get 3.1 pod env error")
	envs := []string{}
	for _, env := range testPod.Spec.Containers[0].Env {
		envs = append(envs, fmt.Sprintf("%v=%v", env.Name, env.Value))
	}
	isEqual = CompareENV(envs, stdout)
	Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] check 3.1 pod env is not equal with input.")
	Expect(isEqual).To(BeTrue(), "[Sigma3.1LifeCycle] check 3.1 pod env is not equal with input.")

	By("sigma 3.1: check container network settings.")
	CheckNewWorkSettings(f, testPod)

	By("sigma 3.1: check container armory info.")
	CheckArmory(testPod)

	By("sigma 3.1: check container dnsConfig")
	checkDNSPolicy(f, testPod)
}

//CheckSigmaUpgradeResource() check upgraded container resource by simga3.1, check env/network-settings/armory.
func CheckSigmaUpgradeResource(f *framework.Framework, testPod *v1.Pod, upgradePod *v1.Pod) () {
	By("sigma 3.1: [upgrade] check container env should same as pod.")
	cmd := []string{"env"}
	stdout, _, err := RetryExec(f, testPod, cmd, "check_env", 10, 2)
	Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] [upgrade] get 3.1 pod env error")
	envs := []string{}
	for _, env := range upgradePod.Spec.Containers[0].Env {
		envs = append(envs, fmt.Sprintf("%v=%v", env.Name, env.Value))
	}
	isEqual := CompareENV(envs, stdout)
	Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] [upgrade] check 3.1 pod env is not equal with input.")
	Expect(isEqual).To(BeTrue(), "[Sigma3.1LifeCycle] [upgrade] check 3.1 pod env is not equal with input.")

	By("sigma 3.1: check container network settings.")
	CheckNewWorkSettings(f, testPod)

	By("sigma 3.1: check container armory info.")
	CheckArmory(testPod)

	By("sigma 3.1: check container dnsConfig")
	checkDNSPolicy(f, testPod)
}

//RetryExec() retry exec commands if err is not nil.
func RetryExec(f *framework.Framework, pod *v1.Pod, cmd []string, name string, attempts int, retryWaitSeconds int) (stdout string, stderr string, err error) {
	err = Retry(func() error {
		stdout, stderr, err = f.ExecWithOptions(framework.ExecOptions{
			Command:       cmd,
			Namespace:     pod.Namespace,
			PodName:       pod.Name,
			ContainerName: pod.Status.ContainerStatuses[0].Name,
			CaptureStdout: true,
			CaptureStderr: true,
		})
		return err
	}, name, attempts, retryWaitSeconds)
	return
}

//Retry() retry operation if return err.
func Retry(operation func() error, name string, attempts int, retryWaitSeconds int) (err error) {
	return RetryInc(operation, name, attempts, retryWaitSeconds, 0)
}

//RetryInc() core function fro Retry.
func RetryInc(operation func() error, name string, attempts int, retryWaitSeconds int, retryWaitIncSeconds int) (err error) {
	for i := 0; ; i++ {
		err = operation()
		if err == nil {
			if i > 0 {
				framework.Logf("retry #%d %v finally succeed", i, name)
			}
			return nil
		}
		framework.Logf("retry #%d %v, error: %s", i, name, err)

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(time.Second * time.Duration(retryWaitSeconds))
		retryWaitSeconds = retryWaitSeconds + retryWaitIncSeconds
	}
	return err
}
