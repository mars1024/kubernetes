package common

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"encoding/json"
	"strings"
	"fmt"
	"time"
)

var _ = Describe("[sigma-common][stock-pod][Serial] Pod", func() {
	f := framework.NewDefaultFramework("sigma-common")

	//namespace do-not-delete-ns and 3 pods are created before this test
	It("check stock pods running, upgrade one pod, check pods running", func() {
		By("check stock pods running")
		stockNs:="do-not-delete-ns"
		exceptCnt:=3
		pods,err:=f.ClientSet.CoreV1().Pods(stockNs).List(metav1.ListOptions{})

		Expect(err).NotTo(HaveOccurred(),"pods get error")
		for _, pod := range pods.Items {
			util.CheckPodStatus(f.ClientSet,pod.Name,stockNs,v1.PodRunning)
		}
		//check pod cnt right by check configmap
		By("check pod cnt")
		Expect(len(pods.Items)).To(Equal(exceptCnt))

		testPod,err:=f.ClientSet.CoreV1().Pods(stockNs).Get(pods.Items[0].Name,metav1.GetOptions{})
		By("upgrade one pod")
		oldDesiredSpec := testPod.Annotations["inplaceset.beta1.sigma.ali/desired-spec"]
		newDesiredSpec := oldDesiredSpec
		expectImage:="reg.docker.alibaba-inc.com/ali/os:"
		if strings.Contains(oldDesiredSpec,"5u7") {
			newDesiredSpec = strings.Replace(oldDesiredSpec, "5u7", "7u2", 1)
			expectImage+="7u2"
		}else if strings.Contains(oldDesiredSpec,"7u2") {
			newDesiredSpec = strings.Replace(oldDesiredSpec, "7u2", "5u7", 1)
			expectImage+="5u7"
		}else {
			Fail(fmt.Sprintf("image get error,%v",oldDesiredSpec))
		}
		testPod.Annotations["inplaceset.beta1.sigma.ali/desired-spec"] = newDesiredSpec
		testPod.Annotations["pod.beta1.sigma.ali/pod-spec-hash"] = string(uuid.NewUUID())
		patch, err := json.Marshal(testPod)
		Expect(err).NotTo(HaveOccurred(), "patch 3.1 pod with new image error")
		_, err = f.ClientSet.CoreV1().Pods(stockNs).Patch(pods.Items[0].Name, types.StrategicMergePatchType, patch)
		Expect(err).NotTo(HaveOccurred(), "patch 3.1 pod with new image error")

		By("check pods running again")
		pods,err=f.ClientSet.CoreV1().Pods(stockNs).List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred(),"pods get error")
		for _, pod := range pods.Items {
			err = util.WaitTimeoutForPodStatus(f.ClientSet, &pod, v1.PodRunning, 3*time.Minute)
		}

		By("check pod spec image is updated")
		testPod, err = f.ClientSet.CoreV1().Pods(stockNs).Get(testPod.Name, metav1.GetOptions{})
		cnt:=0
		for cnt<180 && !strings.Contains(testPod.Status.ContainerStatuses[0].Image,expectImage) {
			cnt++
			time.Sleep(1 * time.Second)
			testPod, err = f.ClientSet.CoreV1().Pods(stockNs).Get(testPod.Name, metav1.GetOptions{})
		}
		Expect(err).NotTo(HaveOccurred(), "image not updated within 3min")
		Expect(testPod.Status.ContainerStatuses[0].Image).To(Equal(expectImage),"pod image check err")

		//check pod cnt right by check configmap after patch image
		By("check pod cnt after patch pod")
		pods,err=f.ClientSet.CoreV1().Pods(stockNs).List(metav1.ListOptions{})
		Expect(len(pods.Items)).To(Equal(exceptCnt))

	})


})



