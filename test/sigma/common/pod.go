package common

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/swarm"
	"k8s.io/kubernetes/test/sigma/util"
	"k8s.io/apimachinery/pkg/util/wait"
)


func WaitTimeOutForArmoryStatus(pod *v1.Pod, isExist bool) error {
	return wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		nsInfo, err := util.QueryArmory(fmt.Sprintf("dns_ip=='%v'", pod.Status.PodIP))
		if err != nil {
			framework.Logf("query naming service error: %s", err.Error())
			return false, err
		}

		if isExist {
			if len(nsInfo) != 1 {
				framework.Logf("armory info :%v", nsInfo)
				return false, fmt.Errorf("should only have one result in armory, but get %d", len(nsInfo))
			}
			return true, nil
		} else {
			if len(nsInfo) != 0 {
				framework.Logf("armory info :%v", nsInfo)
				return false, fmt.Errorf("should not have any result in armory, but get %d", len(nsInfo))
			}
			return true, nil
		}
	})
}

var _ = Describe("[sigma-common] Pod", func() {
	f := framework.NewDefaultFramework("sigma-common")
	var testPod *v1.Pod

	It("[smoke] create one pod, check the whole link is correct", func() {
		By("create a pod from file, specify sigma-scheduler")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred())

		// name should be unique
		pod.Name = "namingcontrollertest" + time.Now().Format("20160607123450")

		// the following tables are MUST required
		pod.Labels = make(map[string]string, 0)
		pod.Labels["sigma.ali/site"] = "et2sqa"
		pod.Labels["sigma.ali/app-name"] = "common-app"
		pod.Labels["sigma.ali/instance-group"] = "pouch-test_testhost"
		pod.Labels["sigma.alibaba-inc.com/app-unit"] = "CENTER_UNIT.center"
		pod.Labels["sigma.alibaba-inc.com/app-stage"] = "DAILY"

		testPod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		defer util.DeletePod(f.ClientSet, testPod)

		By("check pod is running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(getPod.Status.HostIP).NotTo(BeEmpty(), "status.HostIP should not be empty")
		Expect(getPod.Status.PodIP).NotTo(BeEmpty(), "status.PodIP should not be empty")

		By("sleep 5s to wait for naming controller register pod info in armory")
		time.Sleep(5 * time.Second)

		By("check pod has been registered in armory")
		// could query by both name and ip
		err = WaitTimeOutForArmoryStatus(getPod, true)
		Expect(err).NotTo(HaveOccurred(), "pod is not registered in armory")

		By("delete a pod should success")
		err = util.DeletePod(f.ClientSet, testPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		By("check pod should have been unregistered from armory")
		err = WaitTimeOutForArmoryStatus(getPod, false)
		Expect(err).NotTo(HaveOccurred(), "pod is not unregistered in armory")
	})

})

var _ = Describe("[sigma-common][pod-reconstruct]", func() {
	f := framework.NewDefaultFramework("sigma-common")
	appName := "jianzhan-mock-app"
	deployUnit := "jianzhan-test"
	image := "reg.docker.alibaba-inc.com/ali/os:5u7"

	BeforeEach(func() {
		By(fmt.Sprintf("first make sure no pod exists in namespace %s", appName))
		err := util.DeleteAllPodsInNamespace(f.ClientSet, appName)
		Expect(err).ShouldNot(HaveOccurred(), "delete all pods of test namespace error")
	})

	It("pod reconstruct: first create a 2.0 container, then upgrade to 3.1 pod [Slow][pouch-only]", func() {
		ns := appName
		containerHostName, podSn, container20ID, site, hostIP := createSigma2Container(appName, deployUnit, image)
		defer swarm.DeleteContainer(container20ID)

		By("start this 2.0 container")
		err := swarm.StartContainer(container20ID)
		Expect(err).NotTo(HaveOccurred(), "start 2.0 container error")

		By("sigma2.0: add a user account[sigma3test]")
		out := util.ExecCmdInContainer(hostIP, container20ID, "useradd sigma3test")
		framework.Logf(out)

		By("sigma2.0: get the 2.0 container hostname")
		framework.Logf("container hostname is: %s", containerHostName)

		By("sigma2.0: get the 2.0 container QuotaID")
		container20QuotaID := util.GetContainerQuotaID(hostIP, container20ID)

		By("sigma2.0: get the 2.0 container ali_admin_uid")
		container20AdminUID := util.GetContainerAdminUID(hostIP, container20ID)

		By("sigma2.0: get the 2.0 container cpu set")
		container20CpuSets := util.GetContainerCpusets(hostIP, container20ID)

		// wait 90 seconds util sigma_agent report the 2.0 container info
		framework.Logf("wait 90 seconds util sigma_agent report the 2.0 container info")
		time.Sleep(90 * time.Second)

		// rebuild sigma3.1 pod
		testPod := rebuildSigma3Pod(f, podSn, appName, site)

		By("update pod's annotation: publish with new image")
		oldDesiredSpec := testPod.Annotations["inplaceset.beta1.sigma.ali/desired-spec"]
		newDesiredSpec := strings.Replace(oldDesiredSpec, "5u7", "7u2", 1)
		testPod.Annotations["inplaceset.beta1.sigma.ali/desired-spec"] = newDesiredSpec
		testPod.Annotations["pod.beta1.sigma.ali/pod-spec-hash"] = string(uuid.NewUUID())
		patch, err := json.Marshal(testPod)
		Expect(err).NotTo(HaveOccurred(), "patch 3.1 pod with new image error")
		_, err = f.ClientSet.CoreV1().Pods(ns).Patch(podSn, types.StrategicMergePatchType, patch)
		Expect(err).NotTo(HaveOccurred(), "patch 3.1 pod with new image error")

		By("check pod is running")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")
		testPod, err = f.ClientSet.CoreV1().Pods(ns).Get(podSn, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "can not get the rebuild pod from namespace")
		container31ID := util.GetContainerIDFromPod(testPod)

		By("check pod spec image is updated")
		testPod, err = f.ClientSet.CoreV1().Pods(ns).Get(podSn, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "can not get the rebuild pod from namespace")
		Expect(testPod.Status.ContainerStatuses[0].Image).To(Equal("reg.docker.alibaba-inc.com/ali/os:7u2"), "pod spec image is not updated")

		inplaceSetName := util.GetInplaceSetNameFromPod(testPod)
		defer deleteInplaceSet(inplaceSetName, podSn, ns)

		podInfo, err := json.Marshal(testPod)
		Expect(err).NotTo(HaveOccurred())
		framework.Logf("3.1 reconstruct pod info: %s", string(podInfo))

		By("check 2.0 container's etcd info is removed")
		etcdKey := fmt.Sprintf("/pod/%s/%s/%s", testPod.Labels["sigma.ali/site"], testPod.Labels["sigma.ali/instance-group"], podSn)
		framework.Logf("check etcd key %s", etcdKey)
		val, err := swarm.EtcdGet(etcdKey)
		Expect(val).To(BeNil(), "2.0 container's etcd info is not removed")
		Expect(err).NotTo(HaveOccurred(), "2.0 container's etcd info is not removed")

		By("check 2.0 container is stopped")
		// log into slave node and check container status, container should be stopped
		runOutput := util.GetContainerPsOutPut(hostIP, container20ID)
		if !strings.Contains(runOutput, "Exited") && !strings.Contains(runOutput, "Stopped") {
			Fail("2.0 container status is not Exited or Stopped, but we expect it should be that")
		}

		By("check 3.1 container is up")
		runOutput = util.GetContainerPsOutPut(hostIP, container31ID)
		if !strings.Contains(runOutput, "Up") {
			Fail("3.0 container status is not up, but we expect it should be that")
		}

		By("use container SN to invoke 2.0 exec interface to run a command: e.g. echo hi")
		err = swarm.ExecCommandInContainer(podSn, []string{"/home/admin/.start"})
		Expect(err).NotTo(HaveOccurred(), "invoke 2.0 exec interface to run a command error")

		By("sigma3.1: check container QuotaID remain")
		container31QuotaID := util.GetContainerQuotaID(hostIP, container31ID)
		Expect(container31QuotaID).To(Equal(container20QuotaID), "QuotaID is not maintained after rebuild pod")

		By("sigma3.1: check container ali_admin_uid remain")
		container31AdminUID := util.GetContainerAdminUID(hostIP, container31ID)
		Expect(container31AdminUID).To(Equal(container20AdminUID), "ali_admin_uid is not maintained after rebuild pod")

		By("sigma3.1: check container cpu set remain")
		container31CpuSets := util.GetContainerCpusets(hostIP, container31ID)
		Expect(container31CpuSets).To(Equal(container20CpuSets), "cpu set is not maintained after rebuild pod")

		By("sigma3.1: check container hostname should not change")
		cmd := []string{"hostname"}
		stdout, _, err := f.ExecWithOptions(framework.ExecOptions{
			Command:       cmd,
			Namespace:     ns,
			PodName:       testPod.Name,
			ContainerName: testPod.Status.ContainerStatuses[0].Name,
			CaptureStdout: true,
			CaptureStderr: true,
		})
		Expect(err).NotTo(HaveOccurred(), "check 3.1 pod hostname error")
		Expect(stdout).To(Equal(containerHostName), "3.1 pod hostname is not equal with 2.0 container")

		By("sigma3.1: check user account[sigma3test]")
		cmd = []string{"cat", "/etc/passwd"}
		stdout, _, err = f.ExecWithOptions(framework.ExecOptions{
			Command:       cmd,
			Namespace:     ns,
			PodName:       testPod.Name,
			ContainerName: testPod.Status.ContainerStatuses[0].Name,
			CaptureStdout: true,
			CaptureStderr: true,
		})
		Expect(err).NotTo(HaveOccurred(), "check 3.1 pod user account error")
		if !strings.Contains(stdout, "sigma3test") {
			framework.Logf("cmd output: %s", stdout)
			Fail("sigma3test account is not passed to 3.1 pod")
		}

		By("cleanup: set inplaceset replicas to 0, which means remove the rebuild pods, check both 2.0 and 3.1 containers are deleted")
		framework.Logf("scale inplaceset[%s] replicas to 0", inplaceSetName)
		patchData := fmt.Sprintf("{\"spec\":{\"replicas\":0, \"podsToDelete\":[\"%s\"]}}", podSn)
		cmdOutput := framework.RunKubectlOrDie("patch", "inplaceset.extensions.sigma", inplaceSetName, "-p", patchData, fmt.Sprintf("--namespace=%v", ns))
		framework.Logf("kubectl cmd output: %s", cmdOutput)

		By("cleanup: check pods in inplaceset are deleted")
		err = framework.WaitForPodToDisappear(f.ClientSet, ns, podSn, labels.Everything(), 5*time.Second, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pods in inplaceset are not deleted")

		By("cleanup: check both 2.0 and 3.1 containers are deleted")
		framework.Logf("check 3.1 container is deleted")
		err = util.CheckContainerNotExistInHost(hostIP, container31ID, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "3.1 container is not deleted")
		framework.Logf("check 2.0 container is deleted")
		err = util.CheckContainerNotExistInHost(hostIP, container20ID, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "2.0 container is not deleted")
	})

	It("pod rollback: first create a 2.0 container, then upgrade to 3.1 pod, and then rollback to 2.0 [Slow][pouch-only]", func() {
		ns := appName
		_, podSn, container20ID, site, _ := createSigma2Container(appName, deployUnit, image)
		defer swarm.DeleteContainer(container20ID)

		By("start this 2.0 container")
		err := swarm.StartContainer(container20ID)
		Expect(err).NotTo(HaveOccurred(), "start 2.0 container error")

		// wait 90 seconds util sigma_agent report the 2.0 container info
		framework.Logf("wait 90 seconds util sigma_agent report the 2.0 container info")
		time.Sleep(90 * time.Second)

		// rebuild sigma3.1 pod
		testPod := rebuildSigma3Pod(f, podSn, appName, site)

		By("update pod's annotation: publish with new image")
		oldDesiredSpec := testPod.Annotations["inplaceset.beta1.sigma.ali/desired-spec"]
		newDesiredSpec := strings.Replace(oldDesiredSpec, "5u7", "7u2", 1)
		testPod.Annotations["inplaceset.beta1.sigma.ali/desired-spec"] = newDesiredSpec
		testPod.Annotations["pod.beta1.sigma.ali/pod-spec-hash"] = string(uuid.NewUUID())
		patch, err := json.Marshal(testPod)
		Expect(err).NotTo(HaveOccurred(), "patch 3.1 pod with new image error")
		_, err = f.ClientSet.CoreV1().Pods(ns).Patch(podSn, types.StrategicMergePatchType, patch)
		Expect(err).NotTo(HaveOccurred(), "patch 3.1 pod with new image error")

		By("check pod is running")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")
		testPod, err = f.ClientSet.CoreV1().Pods(ns).Get(podSn, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "can not get the rebuild pod from namespace")
		container31ID := util.GetContainerIDFromPod(testPod)
		hostIP := testPod.Status.HostIP

		By("check pod spec image is updated")
		testPod, err = f.ClientSet.CoreV1().Pods(ns).Get(podSn, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "can not get the rebuild pod from namespace")
		Expect(testPod.Status.ContainerStatuses[0].Image).To(Equal("reg.docker.alibaba-inc.com/ali/os:7u2"), "pod spec image is not updated")

		inplaceSetName := util.GetInplaceSetNameFromPod(testPod)
		defer deleteInplaceSet(inplaceSetName, podSn, ns)

		podInfo, err := json.Marshal(testPod)
		Expect(err).NotTo(HaveOccurred())
		framework.Logf("3.1 reconstruct pod info: %s", string(podInfo))

		By("check 2.0 container's etcd info is removed")
		etcdKey := fmt.Sprintf("/pod/%s/%s/%s", testPod.Labels["sigma.ali/site"], testPod.Labels["sigma.ali/instance-group"], podSn)
		framework.Logf("check etcd key %s", etcdKey)
		val, err := swarm.EtcdGet(etcdKey)
		Expect(val).To(BeNil(), "2.0 container's etcd info is not removed")
		Expect(err).NotTo(HaveOccurred(), "2.0 container's etcd info is not removed")

		By("check 2.0 container is stopped")
		// log into slave node and check container status, container should be stopped
		runOutput := util.GetContainerPsOutPut(hostIP, container20ID)
		if !strings.Contains(runOutput, "Exited") && !strings.Contains(runOutput, "Stopped") {
			Fail("2.0 container status is not Exited or Stopped, but we expect it should be that")
		}

		By("check 3.1 container is up")
		runOutput = util.GetContainerPsOutPut(hostIP, container31ID)
		if !strings.Contains(runOutput, "Up") {
			Fail("3.0 container status is not up, but we expect it should be that")
		}

		// invoke boss api to rollback
		rollbackSigma3Pod(appName, site)

		By("rollback: invoke 2.0 upgrade api")
		requestID, err := swarm.UpgradeContainer(podSn, swarm.ContainerUpgradeOption{
			Image: image,
			Labels: map[string]string{
				"ali.Async": "true",
			},
			Env: []string{
				"foo=bar",
				fmt.Sprintf("appName=%s", appName),
			},
		})
		Expect(err).NotTo(HaveOccurred(), "upgrade 2.0 container error")

		By("rollback: check 2.0 rollback container should be created")
		upgradeContainer, err := swarm.QueryRequestStateWithTimeout(requestID, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "check request of upgrade 2.0 container error")
		Expect(upgradeContainer.ContainerID).NotTo(BeEmpty(), "can not get upgrade container ID")
		framework.Logf("rollback container id is: %s", upgradeContainer.ContainerID)
		defer swarm.DeleteContainer(podSn) // whether rollback success or fail, remember to remove container when test is finished

		rs := []rune(upgradeContainer.ContainerID)
		upgradeContainerID := string(rs[0:6])
		runOutput = util.GetContainerPsOutPut(hostIP, upgradeContainerID)
		if !strings.Contains(runOutput, upgradeContainerID) || !strings.Contains(runOutput, "Up") {
			Fail(fmt.Sprintf("upgrade container[%s] does not exist or is not up", upgradeContainerID))
		}

		By("rollback: check 3.1 pod/container should be deleted")
		framework.Logf("check 3.1 container is deleted")
		err = util.CheckContainerNotExistInHost(hostIP, container31ID, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "3.1 container is not deleted")
		framework.Logf("check 3.1 pod is deleted")
		_, err = f.ClientSet.CoreV1().Pods(ns).Get(podSn, metav1.GetOptions{})
		Expect(err).To(HaveOccurred(), "3.1 pod still exists")
		framework.Logf(err.Error())

		By("check container/pod sn still exists in armory")
		armoryInfo, err := util.QueryArmory(fmt.Sprintf("sn=='%v'", podSn))
		Expect(err).NotTo(HaveOccurred(), "query naming service should pass")
		Expect(armoryInfo).NotTo(BeEmpty(), "naming service info should not be empty")
		Expect(len(armoryInfo)).Should(Equal(1), "should only have one result in armory")

		By("rollback: delete 2.0 rollback container")

	})
})

// createSigma2Container create a sigma2.0 container, if success, return containerHostName, containerSN, containerID, site, hostIP
func createSigma2Container(appName, deployUnit, image string) (string, string, string, string, string) {
	By("create a 2.0 container")
	requestID := string(uuid.NewUUID())
	config := &swarm.ContainerOption{
		Resource: swarm.Resource{
			CPUCount: 1,
			Memory:   2147483648,
			DiskSize: 10737418240,
		},
		Name:      requestID,
		ImageName: image,
		Labels: map[string]string{
			"ali.RequestId":     requestID,
			"ali.RequirementId": requestID,
			"ali.AppDeployUnit": deployUnit,
			"ali.AppName":       appName,
			"ali.InstanceGroup": deployUnit,
			//"ali.SpecifiedNcIps": "100.81.155.62",
		},
	}
	container2_0, err := swarm.CreateContainerWithOption(config)
	Expect(err).NotTo(HaveOccurred(), "create 2.0 container error")

	if container2_0.ID == "" {
		Fail(fmt.Sprintf("container id is empty: %v", container2_0.Warnings))
	}
	framework.Logf("create a 2.0 container, id is %s", container2_0.ID)
	//defer swarm.DeleteContainer(container2_0.ID)

	containerResult := swarm.GetRequestState(requestID)
	site := containerResult.Site
	framework.Logf("container's site is %s", site)
	Expect(site).NotTo(BeEmpty())
	podSn := containerResult.ContainerSN
	framework.Logf("container/pod sn is: %s", podSn)
	rs := []rune(container2_0.ID)
	containerID := string(rs[0:6])
	return containerResult.ContainerHN, podSn, containerID, site, containerResult.HostIP
}

func rebuildSigma3Pod(f *framework.Framework, podSn, appName, site string) *v1.Pod {
	By("invoke sigma boss api to reconstruct pod object")
	taskID, err := util.RebuildSigma3Pod(appName, site)
	framework.Logf("rebuild 3.1 pod, task id is: %v", taskID)
	err = util.QuerySigma3RebuildPodWithTimeout(taskID, appName, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "rebuild 3.1 pod error")

	framework.Logf("shutdown the rebuild process of app[%s]", appName)
	_, err = util.ShutDownRebuildSigma3Pod(appName, site)
	Expect(err).NotTo(HaveOccurred(), "shutdown rebuild 3.1 pod error")

	By("check the 3.1 pod object exists")
	testPod, err := f.ClientSet.CoreV1().Pods(appName).Get(podSn, metav1.GetOptions{IncludeUninitialized: true})
	Expect(err).NotTo(HaveOccurred(), "can not get the rebuild pod from namespace")
	return testPod
}

func rollbackSigma3Pod(appName, site string) {
	rollbackEtcdKey := "/internal/sigma3/graylist/v3_1/rollback"
	By("rollback: update rollback list in etcd")
	err := swarm.EtcdPutString(rollbackEtcdKey, appName)
	Expect(err).NotTo(HaveOccurred(), "update rollback list in etcd error")

	By("rollback: invoke sigma boss api to enable app to be rollback")
	taskID, err := util.OpenStockBuildingForSigma3Pod(appName, site)
	Expect(err).NotTo(HaveOccurred(), "invoke sigma boss api to enable app to be rollback error")
	err = util.QuerySigma3RebuildPodWithTimeout(taskID, appName, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "invoke sigma boss api to enable app to be rollback error")
}

func deleteInplaceSet(inplaceSetName, podSn, ns string) {
	if inplaceSetName != "" {
		patchData := fmt.Sprintf("{\"spec\":{\"replicas\":0, \"podsToDelete\":[\"%s\"]}}", "xxxx-xxxx-xxxx-xxxx")
		cmdOutput := framework.RunKubectlOrDie("patch", "inplaceset.extensions.sigma", inplaceSetName, "-p", patchData, fmt.Sprintf("--namespace=%v", ns))
		framework.Logf("kubectl cmd output: %s", cmdOutput)
		cmdOutput = framework.RunKubectlOrDie("delete", "inplaceset.extensions.sigma", inplaceSetName, fmt.Sprintf("--namespace=%v", ns))
		framework.Logf("kubectl cmd output: %s", cmdOutput)
	}
}
