package ant_sigma_bvt

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	k8sApi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
	"github.com/samalba/dockerclient"

)

var _ = Describe("[ant][sigma-alipay-bvt]", func() {
	f := framework.NewDefaultFramework("sigma-ant-bvt")
	appName := "alipay-test-bvt-container"
	enableOverQuota := IsEnableOverQuota()
	framework.Logf("EnableOverQuota:%v", enableOverQuota)
	BeforeEach(func() {
		CheckAdapterParameters()
		By(fmt.Sprintf("first make sure no pod exists in namespace %s", appName))
		err := CheckPodNameSpace(f.ClientSet, appName)
		Expect(err).ShouldNot(HaveOccurred(), "check namespace error")
		err = util.DeleteAllPodsInNamespace(f.ClientSet, appName)
		Expect(err).ShouldNot(HaveOccurred(), "delete all pods of test namespace error")
		site, err = GetNodeSite(f.ClientSet)
		Expect(err).To(BeNil(), "get node labels site failed.")
		site = strings.ToLower(site)
	})
	It("[ant][sigma-alipay-bvt][adapter][dns-policy-default] test container default dnsPolicy with nameServer specified use adapter.", func() {
		configFile := filepath.Join(util.TestDataDir, "alipay-adapter-create-container.json")
		framework.Logf("TestDir:%v", util.TestDataDir)
		createConfig, err := LoadBaseCreateFile(configFile)
		Expect(err).To(BeNil(), "Load create container config failed.")
		createConfig.HostConfig.Dns = []string{"8.8.8.8"}
		createConfig.Labels["ali.Site"] = site
		if enableOverQuota == "true" {
			createConfig.Labels["ali.EnableOverQuota"] = enableOverQuota
		}
		//start
		CheckAdapterDNSPolicy(f, createConfig)
	})
	It("[ant][sigma-alipay-bvt][adapter][dns-policy-none] test container dnsPolicy=none with all dnsConfig specified use adapter.", func() {
		configFile := filepath.Join(util.TestDataDir, "alipay-adapter-create-container.json")
		framework.Logf("TestDir:%v", util.TestDataDir)
		createConfig, err := LoadBaseCreateFile(configFile)
		Expect(err).To(BeNil(), "Load create container config failed.")
		createConfig.HostConfig.Dns = []string{"8.8.8.8"}
		createConfig.HostConfig.DnsSearch = []string{"test.alipay.net"}
		createConfig.HostConfig.DNSOptions = []string{"test-timeout:2"}
		createConfig.Labels["ali.Site"] = site
		if enableOverQuota == "true" {
			createConfig.Labels["ali.EnableOverQuota"] = enableOverQuota
		}
		//start
		CheckAdapterDNSPolicy(f, createConfig)
	})

	It("[ant][sigma-alipay-bvt][sigma3.1][dns-policy-default] test container default dnsPolicy with nameServer specified use sigma3.1.", func() {
		framework.Logf("TestDir:%v", util.TestDataDir)
		name := "simga-bvt-test-" + time.Now().Format("20160607123450")
		pod, err := LoadAlipayBasePod(name, k8sApi.ContainerStateRunning, enableOverQuota)
		Expect(err).To(BeNil(), "Load create container config failed.")
		pod.Namespace = appName
		pod.Labels[k8sApi.LabelSite] = site
		pod.Spec.DNSConfig.Nameservers = []string{"8.8.8.8"}
		//start
		CheckSigma31DNSPolicy(f, pod)
	})

	It("[ant][sigma-alipay-bvt][sigma3.1][dns-policy-none] test container default dnsPolicy with all dnsConfig specified use sigma3.1.", func() {
		framework.Logf("TestDir:%v", util.TestDataDir)
		name := "simga-bvt-test-" + time.Now().Format("20160607123450")
		pod, err := LoadAlipayBasePod(name, k8sApi.ContainerStateRunning, enableOverQuota)
		Expect(err).To(BeNil(), "Load create container config failed.")
		pod.Namespace = appName
		pod.Labels[k8sApi.LabelSite] = site
		pod.Spec.DNSPolicy = v1.DNSNone
		pod.Spec.DNSConfig = &v1.PodDNSConfig{
			Nameservers: []string{"8.8.8.8"},
			Searches:    []string{"test.alipay.net"},
			Options:     []v1.PodDNSConfigOption{
				{
					Name:  "test-timeout:2",
				},
			},
		}
		//start
		CheckSigma31DNSPolicy(f, pod)
	})
})


//CheckAdapterDNSPolicy()
func CheckAdapterDNSPolicy(f * framework.Framework, createConfig *dockerclient.ContainerConfig) {
	By("Create container.")
	//create
	pod, result := MustCreatePod(s, f.ClientSet, createConfig)
	defer util.DeletePod(f.ClientSet, &pod)

	checkDNSPolicy(f, &pod)
	//stop pod
	By("Stop pod.")
	MustOperatePod(s, f.ClientSet, result.ContainerSn, &pod, "stop", v1.PodPending)
	//start pod
	By("Start pod.")
	MustOperatePod(s, f.ClientSet, result.ContainerSn, &pod, "start", v1.PodRunning)
	By("check pod dnsPolicy")

	checkDNSPolicy(f, &pod)
	//restart pod
	By("Restart pod.")
	MustOperatePod(s, f.ClientSet, result.ContainerSn, &pod, "restart", v1.PodRunning)

	//upgrade pod
	By("Upgrade pod.")
	requestId := string(uuid.NewRandom().String())
	framework.Logf("upgrade pod %#v, requestId:%v", pod, requestId)
	upgradeConfig2 := NewUpgradeConfig("FOO2=bar2")
	MustUpgradeContainer(s, result.ContainerSn, requestId, true, upgradeConfig2)

	//check status
	err := util.WaitTimeoutForPodStatus(f.ClientSet, &pod, v1.PodPending, 1*time.Minute)
	Expect(err).To(BeNil(), "[AdapterLifeCycle] [2] upgrade container expect exited failed.")

	MustOperatePod(s, f.ClientSet, result.ContainerSn, &pod, "start", v1.PodRunning)
	checkDNSPolicy(f, &pod)

	//delete pod
	By("Delete pod.")
	resp, err := s.DeleteContainer(result.ContainerSn, true)
	Expect(err).To(BeNil(), "[AdapterLifeCycle] Delete container failed.")
	Expect(resp).To(BeEmpty(), "[AdapterLifeCycle] Delete container failed with response.")

	//check exist
	err = checkPodDelete(f.ClientSet, &pod)
	Expect(err).To(BeNil(), "[AdapterLifeCycle] Delete container failed.")
}


func CheckSigma31DNSPolicy(f *framework.Framework, pod *v1.Pod) {
	By("Create sigma3.1 pod.")
	err := CreateSigmaPod(f.ClientSet, pod)
	Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] Create sigma3.1 pod failed.")
	defer util.DeletePod(f.ClientSet, pod)
	//check resource
	newPod, err := f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] get created sigma3.1 pod failed.")
	Expect(newPod).NotTo(BeNil(), "[Sigma3.1LifeCycle] get created sigma3.1 pod nil.")
	checkDNSPolicy(f, newPod)

	//stop pod
	By("Stop sigma3.1 pod.")
	err = StopOrStartSigmaPod(f.ClientSet, newPod, k8sApi.ContainerStateExited)
	Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] Stop sigma3.1 pod failed.")
	//start pod
	By("Start sigma3.1 pod.")
	err = StopOrStartSigmaPod(f.ClientSet, newPod, k8sApi.ContainerStateRunning)
	Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] Start sigma3.1 pod failed.")
	checkDNSPolicy(f, newPod)

	//upgrade pod.
	By("Upgrade sigma3.1 pod, expect running.")
	err = UpgradeSigmaPod(f.ClientSet, newPod, NewUpgradePod(upgradeEnv2), k8sApi.ContainerStateRunning)
	Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] Upgrade created sigma3.1 expect running pod failed.")
	checkDNSPolicy(f, newPod)

	//delete pod
	By("Delete sigma3.1 pod.")
	err = util.DeletePod(f.ClientSet, newPod)
	Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] Delete created sigma3.1 pod failed.")
}
