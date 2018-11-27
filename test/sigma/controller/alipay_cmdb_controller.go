package controller_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sApi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipayapis "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"gitlab.alipay-inc.com/sigma/controller-manager/pkg/alipaymeta"
	cmdbClient "gitlab.alipay-inc.com/sigma/controller-manager/pkg/cmdb"

	"k8s.io/apimachinery/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	sigmabvt "k8s.io/kubernetes/test/sigma/ant-sigma-bvt"
	"k8s.io/kubernetes/test/sigma/util"
)

const (
	AnnotationLastSpecHash = "meta.k8s.alipay.com/last-spec-hash"
)

var (
	site      string
	cmdbURL   string
	cmdbUser  string
	cmdbToken string
	cmdbCli   cmdbClient.Client
)

var _ = Describe("[ant][sigma-alipay-controller][cmdb]", func() {
	f := framework.NewDefaultFramework("sigma-ant-controller")
	enableOverQuota := sigmabvt.IsEnableOverQuota()
	framework.Logf("EnableOverQuota:%v", enableOverQuota)
	BeforeEach(func() {
		LoadCMDBInfo()
		cmdbCli = cmdbClient.NewCMDBClient(cmdbURL, cmdbUser, cmdbToken)
		site, err := sigmabvt.GetNodeSite(f.ClientSet)
		Expect(err).To(BeNil(), "get node labels site failed.")
		site = strings.ToLower(site)
	})
	It("[sigma-alipay-controller][cmdb][smoke] test pod cmdb lifecycle with zappinfo un-register, add/get/delete.", func() {
		testPod := CreateCMDBPod(f, false, enableOverQuota)

		defer util.DeletePod(f.ClientSet, testPod)
		By("wait until pod running and have pod/host IP")
		err := util.WaitTimeoutForPodStatus(f.ClientSet, testPod, corev1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(getPod.Status.HostIP).NotTo(BeEmpty(), "status.HostIP should not be empty")
		Expect(getPod.Status.PodIP).NotTo(BeEmpty(), "status.PodIP should not be empty")

		By("wait for cmdb controller reigister pod info in cmdb")
		checkPodCMDBInfo(getPod, cmdbCli)
		By("delete pod.")
		err = util.DeletePod(f.ClientSet, testPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
		// query cmdbInfo
		sn, ok := getPod.Labels[k8sApi.LabelPodSn]
		Expect(ok).To(BeTrue(), "pod sn need specified.")
		err = WaitTimeoutForCMDBInfo(cmdbCli, sn, http.StatusNotFound, 1*time.Minute)
		Expect(err).To(BeNil(), "cmdb Info should be removed.")
	})

	It("[sigma-alipay-controller][cmdb][smoke] test pod cmdb lifecycle with zappinfo registered, add/get/delete.", func() {
		testPod := CreateCMDBPod(f, true, enableOverQuota)

		defer util.DeletePod(f.ClientSet, testPod)
		By("wait until pod running and have pod/host IP")
		err := util.WaitTimeoutForPodStatus(f.ClientSet, testPod, corev1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(getPod.Status.HostIP).NotTo(BeEmpty(), "status.HostIP should not be empty")
		Expect(getPod.Status.PodIP).NotTo(BeEmpty(), "status.PodIP should not be empty")

		By("wait for cmdb controller reigister pod info in cmdb")
		checkPodCMDBInfo(getPod, cmdbCli)

		By("delete pod.")
		err = util.DeletePod(f.ClientSet, testPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
		// query cmdbInfo
		sn, ok := getPod.Labels[k8sApi.LabelPodSn]
		Expect(ok).To(BeTrue(), "pod sn need specified.")
		err = WaitTimeoutForCMDBInfo(cmdbCli, sn, http.StatusNotFound, 1*time.Minute)
		Expect(err).To(BeNil(), "cmdb Info should be removed.")
	})
})

//checkPodCMDBInfo() check pod cmdbinfo and annotation/finalizer.
func checkPodCMDBInfo(getPod *corev1.Pod, cmdbCli cmdbClient.Client) {
	sn, ok := getPod.Labels[k8sApi.LabelPodSn]
	Expect(ok).To(BeTrue(), "pod sn need specified.")
	Expect(sn).NotTo(BeEmpty(), "pod sn should not be empty.")
	framework.Logf("Pod SN:%v", sn)
	err := WaitTimeoutForCMDBInfo(cmdbCli, sn, http.StatusOK, 1*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "cmdb Info should be registered.")
	// query cmdbInfo
	cmdbResp, err := cmdbCli.GetContainerInfo(sn)
	Expect(err).NotTo(HaveOccurred(), "query cmdb Info should pass")
	Expect(cmdbResp).NotTo(BeNil(), "cmdb info should not be empty")
	Expect(cmdbResp.Code).To(Equal(http.StatusOK), "cmdb info should be ok.")
	framework.Logf("cmdbInfo :%+v, PodInfo:%+v", cmdbResp.Data, *getPod)
	cmdbInfo := cmdbResp.Data
	Expect(cmdbInfo.BizName).To(Equal(getPod.Labels["ali.BizName"]), "cmdb BizName should same as pod label ali.BizName")
	Expect(cmdbInfo.AppName).To(Equal(getPod.Labels[k8sApi.LabelAppName]), "cmdb AppName should same as pod label app-name")
	Expect(strings.ToLower(cmdbInfo.NodeSn)).To(Equal(getPod.Spec.NodeName), "cmdb ncSn should same as pod NodeName")
	Expect(cmdbInfo.AllocPlanStatus).To(Equal("allocated"), "cmdb allocplanStatus should equal allocated.")
	Expect(cmdbInfo.InstanceStatus).To(Equal("allocated"), "cmdb instancestatus should equal allocated")
	Expect(cmdbInfo.InstanceType).To(Equal(getPod.Labels["com.alipay.acs.container.server_type"]), "cmdb instacneType default CONTAINER.")
	Expect(cmdbInfo.ContainerId).To(Equal(string(getPod.UID)), "cmdb containerId should same as pod UID")
	Expect(cmdbInfo.ContainerIp).To(Equal(getPod.Status.PodIP), "cmdb ncSn should same as podIp")
	Expect(cmdbInfo.ContainerSn).To(Equal(sn), "cmdb ncSn should same as podsn")
	Expect(cmdbInfo.ContainerHostName).To(Equal(getHostName(getPod)),
		"cmdb containerHostName should same as annotation hostname template.")
	Expect(cmdbInfo.DeployUnit).To(Equal(getPod.Labels[k8sApi.LabelDeployUnit]), "cmdb deployunit should same as pod deployunit")
	Expect(cmdbInfo.PoolSystem).To(Equal("sigma3_1"), "cmdb poolsystem should same as sigma3_1")

	//resources
	memory, disk, cpu := getResources(getPod.Spec.Containers)
	Expect(cmdbInfo.MemorySize).To(Equal(memory), "cmdb memory should same sa pod memory.")
	Expect(cmdbInfo.DiskSize).To(Equal(disk), "cmdb disk should same sa pod disk.")
	Expect(cmdbInfo.CpuNum).To(Equal(cpu), "cmdb cpu should same sa pod cpu num.")
	cpuIds := getCpuIds(getPod)
	Expect(cmdbInfo.CpuIds).To(Equal(cpuIds))
	Expect(getPod.Finalizers).Should(ContainElement(ContainSubstring(alipaymeta.CMDBFinalizer)), "CMDBFinalizer should be register into pod.finalizers")
	lastSpecHash, ok := getPod.Annotations[AnnotationLastSpecHash]
	Expect(ok).To(BeTrue(), "Last-spec-hash should be specified.")
	Expect(lastSpecHash).NotTo(BeEmpty(), "Last-spec-hash should not be empty.")
}

//CreateCMDBPod() return pod for cmdb.
func CreateCMDBPod(f *framework.Framework, register bool, enableOverQuota string) *corev1.Pod {
	name := "cmdb-controller-e2e-" + time.Now().Format("20160607123450")
	pod, err := sigmabvt.LoadAlipayBasePod(name, k8sApi.ContainerStateRunning, enableOverQuota)
	Expect(err).To(BeNil(), "Load create container config failed.")
	pod.Labels[k8sApi.LabelSite] = site
	pod = newCMDBPod(pod, f.Namespace.Name, register)
	pod.Spec.Hostname = pod.Name
	testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
	Expect(err).NotTo(HaveOccurred(), "create pod err")
	framework.Logf("create pod config:%v", pod)
	return testPod
}

//newCMDBPod() Load zappinfo for pods.
func newCMDBPod(pod *corev1.Pod, namespace string, registered bool) *corev1.Pod {
	hostname := "dapanweb-" + string(uuid.NewUUID())
	info := alipayapis.PodZappinfo{
		Spec: &alipayapis.PodZappinfoSpec{
			AppName:    "dapanweb",
			Zone:       "RZ11A",
			ServerType: "DOCKER_VM",
			Fqdn:       fmt.Sprintf("%s.%s.alipay.net", hostname, pod.Labels[k8sApi.LabelSite]),
		},
		Status: &alipayapis.PodZappinfoStatus{
			Registered: registered,
		},
	}
	infoBytes, _ := json.Marshal(info)
	pod.Annotations[alipayapis.AnnotationZappinfo] = string(infoBytes)
	pod.Labels["ali.BizName"] = "cloudprovision"
	pod.Labels[alipayapis.LabelZone] = "ant-sigma-test-zone"
	pod.Namespace = namespace
	return pod
}

//LoadCMDBInfo() load cmdb info.
func LoadCMDBInfo() {
	cmdbURL = GetCMDBEnv("CMDB_URL", "http://vulcanboss.stable.alipay.net")
	cmdbUser = GetCMDBEnv("CMDB_USER", "local-test")
	cmdbToken = GetCMDBEnv("CMDB_TOKEN", "test-token")
}

//GetCMDBEnv() get env, if key doesn't exist, return default.
func GetCMDBEnv(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if value == "" || !ok {
		return defaultValue
	}
	return value
}

//getResources() get pod resource info.
func getResources(containers []corev1.Container) (int64, int64, int64) {
	var memorys, disks, cpus int64
	for _, container := range containers {
		resource := container.Resources.Requests
		memory, _ := resource.Memory().AsInt64()
		disk, _ := resource.StorageEphemeral().AsInt64()
		cpu, _ := resource.Cpu().AsInt64()
		memorys += memory
		disks += disk
		cpus += cpu
	}
	return memorys, disks, cpus
}

//getHostName() get pod hostName.
func getHostName(pod *corev1.Pod) string {
	var hostName string
	if pod.Spec.Hostname != "" {
		hostName = pod.Spec.Hostname
	} else {
		value, ok := pod.Annotations[k8sApi.AnnotationPodHostNameTemplate]
		if ok && value != "" && !strings.Contains(value, "{") {
			hostName = value
		}
	}
	return hostName
}

//GetCpuIds()  get cpuIds.
func getCpuIds(pod *corev1.Pod) string {
	//allocSpec
	allocSpecStr, ok := pod.Annotations[k8sApi.AnnotationPodAllocSpec]
	if !ok || allocSpecStr == "" {
		return ""
	}
	//
	allocSpec := k8sApi.AllocSpec{}
	err := json.Unmarshal([]byte(allocSpecStr), &allocSpec)
	if err != nil {
		return ""
	}
	if len(allocSpec.Containers) == 0 {
		return ""
	}
	var cpuIds string
	for _, containers := range allocSpec.Containers {
		cpuIdStr := make([]string, 0)
		if containers.Resource.CPU.CPUSet != nil {
			cpuIds := containers.Resource.CPU.CPUSet.CPUIDs
			for _, id := range cpuIds {
				cpuIdStr = append(cpuIdStr, strconv.Itoa(id))
			}
		}
		if len(cpuIdStr) != 0 {
			cpuIds += strings.Join(cpuIdStr, ",")
		}
	}
	return cpuIds
}

// WaitTimeoutForCMDBInfo check whether cmdbInfo is registerd within the timeout.
func WaitTimeoutForCMDBInfo(cmdbCli cmdbClient.Client, sn string, code int, timeout time.Duration) error {
	return wait.PollImmediate(5*time.Second, timeout, checkCMDBInfo(cmdbCli, sn, code))
}

// checkCMDBInfo check whether pod status is same as expected status.
func checkCMDBInfo(client cmdbClient.Client, sn string, code int) wait.ConditionFunc {
	return func() (bool, error) {
		cmdbResp, err := client.GetContainerInfo(sn)
		if err != nil {
			return false, err
		}
		framework.Logf("pod[%s] cmdbinfo resp code:%d.", sn, cmdbResp.Code)
		if cmdbResp.Code != code {
			return false, nil
		}
		framework.Logf("pod[%s] cmdbinfo is ok.", sn)
		return true, nil
	}
}
