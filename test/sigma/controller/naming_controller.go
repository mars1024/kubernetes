package controller_test

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-controller][naming]", func() {
	f := framework.NewDefaultFramework("sigma-controller")

	It("[sigma-controller][naming][smoke] create a pod, should register in armory", func() {
		By("create a pod from file")
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

		testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")
		glog.Info("create pod config:%v", pod)

		defer util.DeletePod(f.ClientSet, testPod)

		By("wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(getPod.Status.HostIP).NotTo(BeEmpty(), "status.HostIP should not be empty")
		Expect(getPod.Status.PodIP).NotTo(BeEmpty(), "status.PodIP should not be empty")

		By("sleep 1s to wait for naming controller register pod info in armory")
		time.Sleep(3 * time.Second)

		By("check pod has been registered in armory")
		// could query by both name and ip
		nsInfo, err := util.QueryArmory(fmt.Sprintf("dns_ip=='%v'", getPod.Status.PodIP))
		Expect(err).NotTo(HaveOccurred(), "query naming service should pass")
		Expect(nsInfo).NotTo(BeEmpty(), "naming service info should not be empty")
		Expect(len(nsInfo)).Should(Equal(1), "should only have one result in armory")
		glog.Info("nsinfo :%v", nsInfo)

		By("delete a pod should success")
		err = util.DeletePod(f.ClientSet, testPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		By("check pod should have been unrigistered from armory")
		newNsInfo, _ := util.QueryArmory(fmt.Sprintf("dns_ip=='%v'", pod.Name))
		Expect(newNsInfo).To(BeEmpty(), "naming service info should not be empty")
	})

})
