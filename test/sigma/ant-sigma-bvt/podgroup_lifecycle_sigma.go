package ant_sigma_bvt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"github.com/samalba/dockerclient"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"gitlab.alipay-inc.com/sigma/clientset/kubernetes"
	schedulingextensionsv1 "gitlab.alipay-inc.com/sigma/clientset/pkg/apis/schedulingextensions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/swarm"
	"k8s.io/kubernetes/test/sigma/util"
)

const PodGroupName = "alipay-test-pg-bvt"

var ClientSet kubernetes.Interface

var podgroupSite = ""

var _ = Describe("[ant][sigma-alipay-podgroup-bvt]", func() {
	f := framework.NewDefaultFramework("sigma-ant-podgroup-bvt")
	enableOverQuota := IsEnableOverQuota()
	timeOut = GetOperatingTimeOut()
	framework.Logf("EnableOverQuota:%v, timeOut:%v, Default:%v, concurrent:%v", enableOverQuota, timeOut, timeOut*time.Minute, GetConcurrentNum())

	BeforeEach(func() {
		config, err := framework.LoadConfig()
		Expect(err).ShouldNot(HaveOccurred(), "load kube config error")
		config.QPS = 20
		config.Burst = 50
		testDesc := CurrentGinkgoTestDescription()
		if len(testDesc.ComponentTexts) > 0 {
			componentTexts := strings.Join(testDesc.ComponentTexts, " ")
			config.UserAgent = fmt.Sprintf(
				"%v -- %v",
				rest.DefaultKubernetesUserAgent(),
				componentTexts)
		}
		if framework.TestContext.KubeAPIContentType != "" {
			config.ContentType = framework.TestContext.KubeAPIContentType
		}
		ClientSet, err = kubernetes.NewForConfig(config)
		Expect(err).To(BeNil(), "create podgroup clientset error")

		CheckAdapterParameters()
		By(fmt.Sprintf("first make sure no pod and podgroup exists in namespace %s", PodGroupName))
		err = CheckPodNameSpace(f.ClientSet, PodGroupName)
		Expect(err).ShouldNot(HaveOccurred(), "check namespace error")
		err = DeleteAllPodGroupsInNamespace(ClientSet, PodGroupName)
		Expect(err).ShouldNot(HaveOccurred(), "delete all podgroups of test namespace error")
		podgroupSite, err = GetNodeSite(f.ClientSet)
		Expect(err).To(BeNil(), "get node labels site failed.")
		podgroupSite = strings.ToLower(podgroupSite)

	})

	It("[ant][sigma-alipay-podgroup-bvt][smoke][adapter] test podgroup lifecycle use adapter.", func() {
		configFile := filepath.Join(util.TestDataDir, "alipay-adapter-create-podgroup.json")
		framework.Logf("TestDir:%v", util.TestDataDir)
		createConfig, err := LoadPodGroupCreateFile(configFile)
		Expect(err).To(BeNil(), "Load create podgroup config failed.")
		createConfig.Name = PodGroupName
		SetPodGroupLabel(createConfig, "ali.Site", podgroupSite)
		if enableOverQuota == "true" {
			SetPodGroupLabel(createConfig, "ali.EnableOverQuota", enableOverQuota)
		}
		By("Create podgroup.")
		pg, pods, result := MustCreatePodGroup(s, ClientSet, createConfig)
		defer MustDeletePodGroup(ClientSet, pg)
		framework.Logf("PodGroup Info: %s", DumpJson(pg))
		framework.Logf("PodGroup Pod Info: %s", DumpJson(pods))
		framework.Logf("PodGroup Result: %s", DumpJson(result))
		Expect(len(pg.Status.Bundles)).To(Equal(1), "[AdapterPodGroupLifeCycle]PodGroup should only have 1 bundle ")
		Expect(len(pg.Status.Bundles)).To(Equal(len(pg.Spec.Bundles)), "[AdapterPodGroupLifeCycle]PodGroup should have same bundle in spec and status")
		bundle := pg.Status.Bundles[0]
		Expect(len(pods)).To(Equal(int(4)), "[AdapterPodGroupLifeCycle]PodGroup has wrong pods")
		Expect(len(pods)).To(Equal(int(bundle.TotalPods)), "[AdapterPodGroupLifeCycle]PodGroup has wrong pods")
		Expect(bundle.ScheduledPods).To(Equal(bundle.TotalPods), "[AdapterPodGroupLifeCycle]PodGroup has wrong scheduled pods")
		Expect(bundle.RunningPods).To(Equal(bundle.TotalPods), "[AdapterPodGroupLifeCycle]PodGroup has wrong running pods")
		pod := pods[0]
		podSN := pod.Labels[sigmak8sapi.LabelPodSn]
		Expect(podSN).NotTo(BeEmpty(), "[AdapterPodGroupLifeCycle]PodGroup pod has no sn")

		//stop pod
		By("Stop pod.")
		MustOperatePod(s, f.ClientSet, podSN, &pod, "stop", corev1.PodPending)
		//start pod
		By("Start pod.")
		MustOperatePod(s, f.ClientSet, podSN, &pod, "start", corev1.PodRunning)
		By("check pod dnsPolicy")

		CheckDNSPolicy(f, &pod)
		//restart pod
		By("Restart pod.")
		MustOperatePod(s, f.ClientSet, podSN, &pod, "restart", corev1.PodRunning)

		//upgrade pod
		By("Upgrade pod.")
		requestId := string(uuid.NewRandom().String())
		framework.Logf("upgrade pod %#v, requestId:%v", pod, requestId)
		upgradeConfig := NewUpgradeConfig("FOO=bar")
		MustUpgradeContainer(s, podSN, requestId, false, upgradeConfig)
		//check status
		err = util.WaitTimeoutForPodStatus(f.ClientSet, &pod, corev1.PodRunning, 1*time.Minute)
		Expect(err).To(BeNil(), "[AdapterLifeCycle] [2] upgrade container expect running failed.")

		CheckAdapterUpgradeResource(f, &pod, upgradeConfig)
		//start pod
		By("Start pod.")
		MustOperatePod(s, f.ClientSet, podSN, &pod, "start", corev1.PodRunning)

		//restart pod
		By("Restart pod.")
		MustOperatePod(s, f.ClientSet, podSN, &pod, "restart", corev1.PodRunning)

		//upgrade pod
		By("Upgrade pod second time.")
		requestId = string(uuid.NewRandom().String())
		framework.Logf("requestId:%v", requestId)
		upgradeConfig2 := NewUpgradeConfig("FOO2=bar2")
		framework.Logf("upgrade pod %#v, requestId:%v", pod, requestId)
		MustUpgradeContainer(s, podSN, requestId, true, upgradeConfig2)

		//check status
		err = util.WaitTimeoutForPodStatus(f.ClientSet, &pod, corev1.PodPending, 1*time.Minute)
		Expect(err).To(BeNil(), "[AdapterLifeCycle] [2] upgrade container expect exited failed.")

		MustOperatePod(s, f.ClientSet, podSN, &pod, "start", corev1.PodRunning)

		CheckAdapterUpgradeResource(f, &pod, NewUpgradeConfig("FOO2=bar2"))
	})
})

type PodGroupConfig struct {
	Name string
	dockerclient.ContainerConfig
	ContainerConfigList []dockerclient.ContainerConfig
}

// LoadPodGroupCreateFile get base create config for sigma-adapter.
func LoadPodGroupCreateFile(file string) (*PodGroupConfig, error) {
	config := &PodGroupConfig{}
	content, err := ioutil.ReadFile(file)
	if err != nil {
		framework.Logf("Read sigma2.0 create config failed, path:%v, err: %+v", file, err)
		return nil, err
	}
	err = json.Unmarshal(content, config)
	if err != nil {
		framework.Logf("Unmarshal sigma2.0 content failed, %+v", err)
		return nil, err
	}
	config.Labels["ali.AppName"] = PodGroupName
	return config, nil
}

// SetPodGroupLabel set a label to podgroup config and all subconfigs.
func SetPodGroupLabel(config *PodGroupConfig, key, value string) {
	config.Labels[key] = value
	for _, c := range config.ContainerConfigList {
		c.Labels[key] = value
	}
}

// MustCreatePodGroup create a podgroup and wait until pods is ready.
func MustCreatePodGroup(s *AdapterServer, client kubernetes.Interface, c *PodGroupConfig) (*schedulingextensionsv1.PodGroup, []corev1.Pod, *swarm.AllocResult) {
	reqInfo, err := json.Marshal(c)
	Expect(err).NotTo(HaveOccurred(), "[AdapterPodGroupLifeCycle]marshal ReqInfo failed.")
	createResp, message, err := s.CreatePodUnit(reqInfo)
	framework.Logf("Create podgroup resp, message:%v, err:%v, site:%v", message, err, podgroupSite)
	Expect(err).NotTo(HaveOccurred(), "[AdapterPodGroupLifeCycle]Create podgroup error.")
	Expect(message).To(Equal(""), "[AdapterPodGroupLifeCycle]create podgroup failed.")
	Expect(createResp).NotTo(BeNil(), "[AdapterPodGroupLifeCycle]get create response failed.")
	Expect(createResp.Id).NotTo(BeEmpty(), "[AdapterPodGroupLifeCycle]get requestId failed.")
	By("Get sigma-adapter create podgroup async response.")
	result, err := GetCreateResultWithTimeOutWithExpectState(client, createResp.Id, 3*time.Minute, c.Labels["ali.AppName"], "running")
	Expect(err).NotTo(HaveOccurred(), "[AdapterPodGroupLifeCycle] get async response failed.")
	Expect(result).NotTo(BeNil(), "[AdapterPodGroupLifeCycle] result should not be nil.")
	By("Get created podgroup.")
	pg, err := client.SchedulingextensionsV1().PodGroups(PodGroupName).Get(PodGroupName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "[AdapterPodGroupLifeCycle] get podgroup failed.")
	pods, err := client.CoreV1().Pods(PodGroupName).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			sigmak8sapi.LabelPodGroupName: pg.Name,
		}).String(),
	})
	Expect(err).NotTo(HaveOccurred(), "[AdapterPodGroupLifeCycle] get pod failed.")
	return pg, pods.Items, result
}

func MustDeletePodGroup(client kubernetes.Interface, pg *schedulingextensionsv1.PodGroup) {
	err := client.SchedulingextensionsV1().PodGroups(pg.Namespace).Delete(pg.Name, nil)
	Expect(err).NotTo(HaveOccurred(), "[AdapterPodGroupLifeCycle] delete podgroup failed.")

	timeout := 5 * time.Minute
	t := time.Now()
	for {
		pods, err := client.CoreV1().Pods(PodGroupName).List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(labels.Set{
				sigmak8sapi.LabelPodGroupName: pg.Name,
			}).String(),
		})
		Expect(err).NotTo(HaveOccurred(), "[AdapterPodGroupLifeCycle] get deleting podgroup failed.")
		if len(pods.Items) <= 0 {
			framework.Logf("Gave up waiting for podgroup %s is removed after %v seconds", pg.Name, time.Since(t).Seconds())
			return
		}
		if time.Since(t) >= timeout {
			framework.Failf("Gave up waiting for podgroup %s is removed after %v seconds", pg.Name, time.Since(t).Seconds())
			return
		}
		framework.Logf("Retrying to check whether podgroup %s is removed", pg.Name)
		time.Sleep(5 * time.Second)
	}
}

// DeleteAllPodGroupsInNamespace delete all podgroups in a namespace
func DeleteAllPodGroupsInNamespace(client kubernetes.Interface, ns string) error {
	pgList, err := client.SchedulingextensionsV1().PodGroups(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pg := range pgList.Items {
		err := client.SchedulingextensionsV1().PodGroups(pg.Namespace).Delete(pg.Name, &metav1.DeleteOptions{})
		framework.Logf("delete podgroup[%s] in namespace %s", pg.Name, pg.Namespace)
		if err != nil {
			return err
		}
	}
	timeout := 5 * time.Minute
	t := time.Now()
	for {
		podList, err := client.CoreV1().Pods(ns).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		if len(podList.Items) == 0 {
			framework.Logf("all pods in namespace[%s] are removed", ns)
			return nil
		}
		if time.Since(t) >= timeout {
			return fmt.Errorf("gave up waiting for all pod in namespace %s are removed after %v seconds",
				ns, time.Since(t).Seconds())
		}
		framework.Logf("Retrying to check whether all pod in namespace %s are removed", ns)
		time.Sleep(5 * time.Second)
	}
}
