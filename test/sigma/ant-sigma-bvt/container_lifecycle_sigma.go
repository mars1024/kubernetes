package ant_sigma_bvt

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	k8sApi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var site string
var timeOut time.Duration

var _ = Describe("[ant][sigma-alipay-bvt]", func() {
	f := framework.NewDefaultFramework("sigma-ant-bvt")
	enableOverQuota := IsEnableOverQuota()
	timeOut = GetOperatingTimeOut()
	framework.Logf("EnableOverQuota:%v, timeOut:%v, Default:%v, concurrent:%v", enableOverQuota, timeOut, timeOut*time.Minute, GetConcurrentNum())
	BeforeEach(func() {
		CheckAdapterParameters()
		By(fmt.Sprintf("first make sure no pod exists in namespace %s", AppName))
		err := CheckPodNameSpace(f.ClientSet, AppName)
		Expect(err).ShouldNot(HaveOccurred(), "check namespace error")
		err = util.DeleteAllPodsInNamespace(f.ClientSet, AppName)
		Expect(err).ShouldNot(HaveOccurred(), "delete all pods of test namespace error")
		site, err = GetNodeSite(f.ClientSet)
		Expect(err).To(BeNil(), "get node labels site failed.")
		site = strings.ToLower(site)
	})
	It("[ant][sigma-alipay-bvt][smoke][adapter] test pod lifecycle use adapter.", func() {
		configFile := filepath.Join(util.TestDataDir, "alipay-adapter-create-container.json")
		framework.Logf("TestDir:%v", util.TestDataDir)
		createConfig, err := LoadBaseCreateFile(configFile)
		Expect(err).To(BeNil(), "Load create container config failed.")
		createConfig.Labels["ali.Site"] = site
		if enableOverQuota == "true" {
			createConfig.Labels["ali.EnableOverQuota"] = enableOverQuota
		}
		By("Create container.")
		//create
		pod, result := MustCreatePod(s, f.ClientSet, createConfig)
		defer util.DeletePod(f.ClientSet, &pod)
		framework.Logf("Pod Info: %#v", DumpJson(pod))
		By("Check container info.")
		CheckAdapterCreateResource(f, &pod, result, createConfig)
		//stop pod
		By("Stop pod.")
		MustOperatePod(s, f.ClientSet, result.ContainerSn, &pod, "stop", v1.PodPending)
		//start pod
		By("Start pod.")
		MustOperatePod(s, f.ClientSet, result.ContainerSn, &pod, "start", v1.PodRunning)
		By("check pod dnsPolicy")

		// update pod
		By("Update Pod, decrease resources.")
		updateConfig := LoadUpdateConfig(1, 1073741824, "1G")
		MustUpdate(s, f.ClientSet, &pod, updateConfig, timeOut*time.Minute)
		CheckAdapterUpdatedResource(f, &pod, updateConfig)

		CheckDNSPolicy(f, &pod)
		//restart pod
		By("Restart pod.")
		MustOperatePod(s, f.ClientSet, result.ContainerSn, &pod, "restart", v1.PodRunning)

		//upgrade pod
		By("Upgrade pod.")
		requestId := string(uuid.NewRandom().String())
		framework.Logf("upgrade pod %#v, requestId:%v", pod, requestId)
		upgradeConfig := NewUpgradeConfig("FOO=bar")
		MustUpgradeContainer(s, result.ContainerSn, requestId, false, upgradeConfig)
		//check status
		err = util.WaitTimeoutForPodStatus(f.ClientSet, &pod, v1.PodRunning, 1*time.Minute)
		Expect(err).To(BeNil(), "[AdapterLifeCycle] [2] upgrade container expect running failed.")

		CheckAdapterUpgradeResource(f, &pod, upgradeConfig)
		//start pod
		By("Start pod.")
		MustOperatePod(s, f.ClientSet, result.ContainerSn, &pod, "start", v1.PodRunning)

		// update pod
		By("Update Pod, increase resources.")
		updateConfig = LoadUpdateConfig(2, 2147483648, "2G")
		MustUpdate(s, f.ClientSet, &pod, updateConfig, timeOut*time.Minute)
		CheckAdapterUpdatedResource(f, &pod, updateConfig)

		//restart pod
		By("Restart pod.")
		MustOperatePod(s, f.ClientSet, result.ContainerSn, &pod, "restart", v1.PodRunning)

		//upgrade pod
		By("Upgrade pod second time.")
		requestId = string(uuid.NewRandom().String())
		framework.Logf("requestId:%v", requestId)
		upgradeConfig2 := NewUpgradeConfig("FOO2=bar2")
		framework.Logf("upgrade pod %#v, requestId:%v", pod, requestId)
		MustUpgradeContainer(s, result.ContainerSn, requestId, true, upgradeConfig2)

		//check status
		err = util.WaitTimeoutForPodStatus(f.ClientSet, &pod, v1.PodPending, 1*time.Minute)
		Expect(err).To(BeNil(), "[AdapterLifeCycle] [2] upgrade container expect exited failed.")

		MustOperatePod(s, f.ClientSet, result.ContainerSn, &pod, "start", v1.PodRunning)

		CheckAdapterUpgradeResource(f, &pod, NewUpgradeConfig("FOO2=bar2"))
		//delete pod
		By("Delete pod.")
		resp, err := s.DeleteContainer(result.ContainerSn, true)
		Expect(err).To(BeNil(), "[AdapterLifeCycle] Delete container failed.")
		Expect(resp).To(BeEmpty(), "[AdapterLifeCycle] Delete container failed with response.")

		//check exist
		err = checkPodDelete(f.ClientSet, &pod)
		Expect(err).To(BeNil(), "[AdapterLifeCycle] Delete container failed.")
	})

	It("[ant][sigma-alipay-bvt][smoke][sigma3.1] test sigma3.1 pod lifecycle use sigma3.1.", func() {
		framework.Logf("TestDir:%v", util.TestDataDir)
		name := "simga-bvt-test-" + time.Now().Format("20160607123450")
		pod, err := LoadAlipayBasePod(name, k8sApi.ContainerStateRunning, enableOverQuota)
		Expect(err).To(BeNil(), "Load create container config failed.")
		pod.Labels[k8sApi.LabelSite] = site
		By("Create sigma3.1 pod.")
		err = CreateSigmaPod(f.ClientSet, pod)
		Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] Create sigma3.1 pod failed.")
		defer util.DeletePod(f.ClientSet, pod)
		//check resource
		newPod, err := f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] get created sigma3.1 pod failed.")
		Expect(newPod).NotTo(BeNil(), "[Sigma3.1LifeCycle] get created sigma3.1 pod nil.")
		framework.Logf("Pod Info: %#v", DumpJson(newPod))

		CheckSigmaCreateResource(f, newPod)
		CheckDNSPolicy(f, newPod)

		//stop pod
		By("Stop sigma3.1 pod.")
		err = StopOrStartSigmaPod(f.ClientSet, newPod, k8sApi.ContainerStateExited)
		Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] Stop sigma3.1 pod failed.")
		//start pod
		By("Start sigma3.1 pod.")
		err = StopOrStartSigmaPod(f.ClientSet, newPod, k8sApi.ContainerStateRunning)
		Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] Start sigma3.1 pod failed.")
		CheckDNSPolicy(f, newPod)

		//upgrade pod.
		By("Upgrade sigma3.1 pod, expect exited.")
		err = UpgradeSigmaPod(f.ClientSet, newPod, NewUpgradePod(upgradeEnv), k8sApi.ContainerStateExited)
		Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] Upgrade created sigma3.1 pod failed.")

		//start pod
		By("start upgraded sigma3.1 pod.")
		err = StopOrStartSigmaPod(f.ClientSet, newPod, k8sApi.ContainerStateRunning)
		Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] Start sigma3.1 pod failed after upgrade.")
		CheckSigmaUpgradeResource(f, newPod, NewUpgradePod(upgradeEnv))

		//upgrade pod.
		By("Upgrade sigma3.1 pod, expect running.")
		err = UpgradeSigmaPod(f.ClientSet, newPod, NewUpgradePod(upgradeEnv2), k8sApi.ContainerStateRunning)
		Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] Upgrade created sigma3.1 expect running pod failed.")
		CheckSigmaUpgradeResource(f, newPod, NewUpgradePod(upgradeEnv2))
		CheckDNSPolicy(f, newPod)

		// update pod.
		By("Update sigma 3.1 pod,  decrease resource, expect running.")
		err = UpdateSigmaPod(f.ClientSet, newPod, NewUpdatePod(updateResource2), k8sApi.ContainerStateRunning)
		Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] increase resource sigma 3.1 expect running pod failed.")

		// check resource increase.
		newPod, err = f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] get updated sigma 3.1 pod failed.")
		Expect(newPod).NotTo(BeNil(), "[Sigma3.1LifeCycle] get updated sigma 3.1 pod nil.")
		CheckSigmaCreateResource(f, newPod)

		// restart pod
		By("restart sigma 3.1 pod.")
		err = StopOrStartSigmaPod(f.ClientSet, newPod, k8sApi.ContainerStateRunning)
		Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] Start sigma 3.1 pod failed after update.")
		CheckSigmaCreateResource(f, newPod)
		// decrease resource.
		By("Update sigma 3.1 pod, increase resource, expect running.")
		err = UpdateSigmaPod(f.ClientSet, newPod, NewUpdatePod(updateResource1), k8sApi.ContainerStateRunning)
		Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] decrease resource sigma 3.1 expect running pod failed.")
		// check resource decrease.
		newPod, err = f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] get updated sigma 3.1 pod failed.")
		Expect(newPod).NotTo(BeNil(), "[Sigma3.1LifeCycle] get updated sigma 3.1 pod nil.")
		CheckSigmaCreateResource(f, newPod)

		//delete pod
		By("Delete sigma3.1 pod.")
		err = util.DeletePod(f.ClientSet, newPod)
		Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] Delete created sigma3.1 pod failed.")
	})
	It("[ant][sigma-alipay-bvt][adapter-concurrency] test adapter pod lifecycle use adapter with concurrency.", func() {
		var wg sync.WaitGroup
		var lock sync.Mutex
		count := GetConcurrentNum()
		var num int
		for i := 0; i < count; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer GinkgoRecover()
				configFile := filepath.Join(util.TestDataDir, "alipay-adapter-create-container.json")
				framework.Logf("TestDir:%v", util.TestDataDir)
				createConfig, err := LoadBaseCreateFile(configFile)
				Expect(err).To(BeNil(), "Load create container config failed.")
				createConfig.Labels["ali.Site"] = site
				if enableOverQuota == "true" {
					createConfig.Labels["ali.EnableOverQuota"] = enableOverQuota
				}
				By("Create container.")
				//create
				pod, result := MustCreatePod(s, f.ClientSet, createConfig)
				defer util.DeletePod(f.ClientSet, &pod)
				framework.Logf("Pod Info: %#v", DumpJson(pod))
				By("Check container info.")
				CheckAdapterCreateResource(f, &pod, result, createConfig)

				//delete pod
				By("Delete pod.")
				resp, err := s.DeleteContainer(result.ContainerSn, true)
				Expect(err).To(BeNil(), "[AdapterLifeCycle] Delete container failed.")
				Expect(resp).To(BeEmpty(), "[AdapterLifeCycle] Delete container failed with response.")
				//check exist
				err = checkPodDelete(f.ClientSet, &pod)
				Expect(err).To(BeNil(), "[AdapterLifeCycle] Delete container failed.")
				lock.Lock()
				num += 1
				lock.Unlock()
			}()
		}
		wg.Wait()
		Expect(num).To(Equal(count), "[AdapterLifeCycle] all create action should be succeed.")
	})

	It("[ant][sigma-alipay-bvt][sigma3-concurrency] test sigma3 pod lifecycle use sigma3.1 with concurrency.", func() {
		var wg sync.WaitGroup
		var lock sync.Mutex
		count := GetConcurrentNum()
		var num int
		for i := 0; i < count; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer GinkgoRecover()
				framework.Logf("TestDir:%v", util.TestDataDir)
				name := fmt.Sprintf("simga-bvt-test-%d", time.Now().UnixNano())
				pod, err := LoadAlipayBasePod(name, k8sApi.ContainerStateRunning, enableOverQuota)
				Expect(err).To(BeNil(), "Load create container config failed.")
				pod.Labels[k8sApi.LabelSite] = site
				By("Create sigma3.1 pod.")
				err = CreateSigmaPod(f.ClientSet, pod)
				Expect(err).NotTo(HaveOccurred(), "[Sigma3.1LifeCycle] Create sigma3.1 pod failed.")
				defer util.DeletePod(f.ClientSet, pod)
				framework.Logf("Pod Info: %#v", DumpJson(pod))
				//check resource
				newPod, err := f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
				Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] get created sigma3.1 pod failed.")
				Expect(newPod).NotTo(BeNil(), "[Sigma3.1LifeCycle] get created sigma3.1 pod nil.")
				CheckSigmaCreateResource(f, newPod)
				//delete pod
				By("Delete sigma3.1 pod.")
				err = util.DeletePod(f.ClientSet, pod)
				Expect(err).To(BeNil(), "[Sigma3.1LifeCycle] Delete created sigma3.1 pod failed.")
				lock.Lock()
				num += 1
				lock.Unlock()
			}()
		}
		wg.Wait()
		Expect(num).To(Equal(count), "[Sigma3.1LifeCycle] all create action should be succeed.")
	})

	It("[ant][sigma-alipay-bvt][smoke][adapter][mosn] test mosn use adapter.", func() {
		// Step1: Load pod file.
		configFile := filepath.Join(util.TestDataDir, "alipay-adapter-create-container.json")
		framework.Logf("TestDir:%v", util.TestDataDir)
		createConfig, err := LoadBaseCreateFile(configFile)
		Expect(err).To(BeNil(), "Load create container config failed.")
		createConfig.Labels["ali.Site"] = site
		if enableOverQuota == "true" {
			createConfig.Labels["ali.EnableOverQuota"] = enableOverQuota
		}
		// Add mosn image and zone.
		createConfig.Labels["sidecar:mosn.image"] = "reg.docker.alibaba-inc.com/antmesh/mosn-dev:1.4.3-0b5be970-dev"
		createConfig.Labels["com.alipay.cloudprovision.zone"] = "GZ00B"
		framework.Logf("createConfig: %v", createConfig)

		// Step2: Create pod by adaptor.
		By("Create container.")
		pod, result := MustCreatePod(s, f.ClientSet, createConfig)
		defer util.DeletePod(f.ClientSet, &pod)

		framework.Logf("[mosn] Create pod Info: %#v", DumpJson(pod))

		// Mosn container injection failed if pod has only one container.
		if len(pod.Spec.Containers) == 1 {
			framework.Failf("Failed to inject mosn container, pod:  %#v", pod)
		}

		// Check pod after creation.
		By("Check container after creation.")
		CheckAdapterCreateResource(f, &pod, result, createConfig)
		CheckDNSPolicy(f, &pod)

		// Step3: Stop container.
		By("Stop pod.")
		MustOperatePod(s, f.ClientSet, result.ContainerSn, &pod, "stop", v1.PodPending)

		// Step4: Start container.
		By("Start pod.")
		MustOperatePod(s, f.ClientSet, result.ContainerSn, &pod, "start", v1.PodRunning)

		// Step5: Update container.
		By("Update Pod, decrease resources.")
		updateConfig := LoadUpdateConfig(1, 1073741824, "1G")
		MustUpdate(s, f.ClientSet, &pod, updateConfig, timeOut*time.Minute)

		// Check pod after update.
		By("Check container after update.")
		CheckAdapterUpdatedResource(f, &pod, updateConfig)
		CheckDNSPolicy(f, &pod)

		// Step7: Upgrade container.
		By("Upgrade pod.")
		requestId := string(uuid.NewRandom().String())
		framework.Logf("[mosn] upgrade pod %#v, requestId:%v", pod, requestId)
		upgradeConfig := NewUpgradeConfig("FOO=bar")
		MustUpgradeContainer(s, result.ContainerSn, requestId, false, upgradeConfig)
		//check status
		err = util.WaitTimeoutForPodStatus(f.ClientSet, &pod, v1.PodRunning, 3*time.Minute)
		Expect(err).To(BeNil(), "[mosn] Wait pod is running, timeout.")

		// Check pod after upgrade.
		By("Check container after upgrade.")
		CheckAdapterUpgradeResource(f, &pod, upgradeConfig)
		CheckDNSPolicy(f, &pod)

		// TODO: Step8: Smooth upgrade mosn container.

		// TODO: Step9: Upgrade mosn container.

		// Step10: Delete container.
		By("Delete pod.")
		_, err = s.DeleteContainer(result.ContainerSn, true)
		Expect(err).To(BeNil(), "[mosn] Delete container failed.")
	})
})
