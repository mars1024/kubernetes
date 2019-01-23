package ant_migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/samalba/dockerclient"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	antsigma "k8s.io/kubernetes/test/sigma/ant-sigma-bvt"
	"k8s.io/kubernetes/test/sigma/swarm"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[ant][migrate-container]", func() {
	f := framework.NewDefaultFramework("ant-migrate-container")
	appName := "ant-migrate-container"
	BeforeEach(func() {
		By(fmt.Sprintf("first make sure no pod exists in namespace %s", appName))
		err := util.DeleteAllPodsInNamespace(f.ClientSet, appName)
		Expect(err).ShouldNot(HaveOccurred(), "delete all pods of test namespace error")
		antsigma.CheckArmoryParameters()
	})

	It("[ant][migrate-container][single] RebuildContainer: first create a 2.0 container, then migrate to 3.1 pod", func() {
		RebuildContainer20ToSigma31Pod(f, appName, true)
	})

	It("[ant][migrate-container][multi] RebuildContainer: first create a 2.0 container, then migrate to 3.1 pod.", func() {
		var wg sync.WaitGroup
		var lock sync.Mutex
		count := 5
		var num int
		for i := 0; i < count; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer GinkgoRecover()
				RebuildContainer20ToSigma31Pod(f, appName, false)
				lock.Lock()
				num += 1
				lock.Unlock()
			}()
		}
		wg.Wait()
		Expect(num).To(Equal(count), "[Multi Rebuild] all create action should be succeed.")
	})
})

// RebuildContainer20ToSigma31Pod() create sigma2.0 container and migrate to sigma3.1 pod, then check resources.
func RebuildContainer20ToSigma31Pod(f *framework.Framework, appName string, lifeCyle bool) {
	createConfig := GetCreateConfig(appName)
	ns := appName
	containerHostName, podSn, container20ID, site, hostIP := CreateSigma2Container(createConfig)
	defer swarm.DeleteContainer(container20ID)

	By("start 2.0 container")
	err := swarm.StartContainer(container20ID)
	Expect(err).NotTo(HaveOccurred(), "start 2.0 container error")

	By("get 2.0 container info.")
	containerJson, err := swarm.InspectContainer(container20ID)
	Expect(err).NotTo(HaveOccurred(), "inspect 2.0 container failed.")
	By("sigma2.0: add a user account[sigma3test]")
	out := util.ExecCmdInContainer(hostIP, container20ID, "useradd sigma3test")
	framework.Logf("User Add Result:%v", out)

	By("sigma2.0: inspect container info.")

	By("sigma2.0: get container adminUID/ Container20ID, quotaId")
	container20QuotaID, container20AdminUID, container20CpuSets := GetSigmaContainerInfo(hostIP, container20ID)

	// rebuild sigma3.1 pod
	By("sigma3.1: rebuild 3.1 pod.")
	testPod := rebuildSigma3Pod(f, podSn, appName)
	defer util.DeletePod(f.ClientSet, testPod)

	By("check pod is running， rebuild finish.")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")
	testPod, err = f.ClientSet.CoreV1().Pods(ns).Get(podSn, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "can not get the rebuild pod from namespace")
	container31ID := util.GetContainerIDFromPod(testPod)

	podInfo, err := json.Marshal(testPod)
	Expect(err).NotTo(HaveOccurred())
	framework.Logf("3.1 reconstruct pod info: %s", string(podInfo))

	By("sigma2.0/3.1: check 2.0 container stopped/3.1 running.")
	CheckContainerStatus(hostIP, container20ID, container31ID)

	By("sigma3.1: check 3.1 pod resource.")
	CheckSigma31Resouce(f, testPod, containerHostName, containerJson)

	By("sigma3.1: check container QuotaID/ali_admin_uid/cpu set remain")
	container31QuotaID, container31AdminUID, container31CpuSets := GetSigmaContainerInfo(hostIP, container31ID)
	Expect(container31QuotaID).To(Equal(container20QuotaID), "QuotaID is not maintained after rebuild pod")
	Expect(container31AdminUID).To(Equal(container20AdminUID), "ali_admin_uid is not maintained after rebuild pod")
	Expect(container31CpuSets).To(Equal(container20CpuSets), "cpu set is not maintained after rebuild pod")

	if lifeCyle {
		// lifecycle test
		By("sigma3.1: container life cycle test.")
		ContainerLifeCycle(f, testPod)
	}

	By("sigma2.0: check etcd resource.")
	CheckSigma20ResourceReomoved(f, testPod, site)

	//delete pod
	By("Delete sigma3.1 pod.")
	err = util.DeletePod(f.ClientSet, testPod)
	Expect(err).To(BeNil(), "Delete rebuild sigma3.1 pod failed.")

	By("Delete sigma2.0 container.")
	swarm.DeleteContainer(container20ID)

	By("cleanup: check both 2.0 and 3.1 containers are deleted")
	framework.Logf("check 3.1 container is deleted")
	err = util.CheckContainerNotExistInHost(hostIP, container31ID, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "3.1 container is not deleted")

	// sigma2.0 cluster has only one instance, process zombie works on slave node, delete again if u want to clean container.
	By("Delete sigma2.0 container again.")
	swarm.DeleteContainer(container20ID)

	framework.Logf("check 2.0 container is deleted")
	err = util.CheckContainerNotExistInHost(hostIP, container20ID, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "2.0 container is not deleted")
}

// CreateSigma2Container() create a sigma2.0 container, if success, return containerHostName, containerSN, containerID, site, hostIP
func CreateSigma2Container(containerConfig *dockerclient.ContainerConfig) (string, string, string, string, string) {
	By("create a 2.0 container")
	requestID := string(uuid.NewUUID())
	containerConfig.Labels["ali.RequestId"] = requestID
	container2_0, err := swarm.CreateContainerWithAliapayParameters(containerConfig)
	Expect(err).NotTo(HaveOccurred(), "create 2.0 container error")

	if container2_0.ID == "" {
		Fail(fmt.Sprintf("container id is empty: %v", container2_0.Warnings))
	}
	framework.Logf("create a 2.0 container, id is %s", container2_0.ID)
	//defer swarm.DeleteContainer(container2_0.ID)
	containerResult, err := swarm.QueryRequestStateWithTimeout(requestID, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "Query sigma2.0 container create result failed.")
	framework.Logf("containerResult:%v", containerResult)
	site := containerResult.Site
	framework.Logf("container's site is %s", site)
	Expect(site).NotTo(BeEmpty())
	podSn := containerResult.ContainerSN
	framework.Logf("container/pod sn is: %s", podSn)
	rs := []rune(containerResult.ContainerID)
	containerID := string(rs[0:12])
	return containerResult.ContainerHN, podSn, containerID, site, containerResult.HostIP
}

// rebuildSigma3Pod() rebuild sigma3.1 pod.
func rebuildSigma3Pod(f *framework.Framework, podSn, appName string) *v1.Pod {
	By("use swarm rebuild API reconstruct 3.1 pod.")
	reqInfo := swarm.ContainerUpgradeOption{}
	// rebuild sigma3.1 pod
	requestId, err := swarm.RebuildContainer(podSn, reqInfo)
	framework.Logf("rebuild container %v, requestId:%v, err:%v", podSn, requestId, err)
	Expect(err).NotTo(HaveOccurred(), "rebuild 3.1 pod error.")
	Expect(requestId).NotTo(BeEmpty(), "unexpected rebuild requestId.")

	By("check the 3.1 pod object exists")
	testPod, err := f.ClientSet.CoreV1().Pods(appName).Get(podSn, metav1.GetOptions{IncludeUninitialized: true})
	Expect(err).NotTo(HaveOccurred(), "can not get the rebuild pod from namespace")
	rebuildResult, err := swarm.QueryRequestStateWithTimeout(requestId, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "Query sigma3.1 container create result failed.")
	framework.Logf("container rebuild Result:%v", rebuildResult)
	return testPod
}

// GetCreateConfig() load sigma2.0 container config.
func GetCreateConfig(appName string) *dockerclient.ContainerConfig {
	site := os.Getenv("SIGMA_SITE")
	configFile := filepath.Join(util.TestDataDir, "alipay-adapter-create-container.json")
	framework.Logf("TestDir:%v", util.TestDataDir)
	createConfig, err := antsigma.LoadBaseCreateFile(configFile)
	Expect(err).To(BeNil(), "Load create container config failed.")
	createConfig.HostConfig.Binds = []string{"/tmp1:/tmp2"}
	createConfig.Labels["ali.Site"] = site
	Expect(site).NotTo(BeEmpty(), "site must be specified.")
	createConfig.Image = "reg.docker.alibaba-inc.com/ali/os:7u2"
	createConfig.Labels["ali.AppName"] = appName
	createConfig.Labels["com.alipay.acs.container.server_type"] = "DOCKER_VM"
	return createConfig
}
