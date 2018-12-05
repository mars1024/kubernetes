package apiserver

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/plugin/pkg/admission/poddeletionflowcontrol"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[kube-apiserver][admission][pdfc]", func() {
	image := "reg.docker.alibaba-inc.com/k8s-test/nginx:1.15.3"
	f := framework.NewDefaultFramework("sigma-apiserver")

	It("[smoke][test-flow-control] test pod deletion limited by pdfc [Serial]", func() {
		By("load pdfc template")
		pdfcConfigFile := filepath.Join(util.TestDataDir, "pdfc-config.json")
		pdfcConfig, err := util.LoadConfigMapFromFile(pdfcConfigFile)
		Expect(err).NotTo(HaveOccurred(), "load pdfc template failed")

		By("load pod template")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		podCfg, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "load pod template failed")

		By("create pdfc rules")
		pdfcConfig.Data[poddeletionflowcontrol.PdfcConfigRuleKey] =
			`[{"duration":"1m","deleteLimit":10}]`
		_, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Create(pdfcConfig)
		Expect(err).NotTo(HaveOccurred(), "create pdfc failed")
		defer f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Delete(poddeletionflowcontrol.PdfcConfigName, nil)

		By("create 11 pods")
		var pods []*v1.Pod
		podCfg.Spec.Containers[0].Image = image
		for i := 0; i <= 10; i++ {
			pod := podCfg.DeepCopy()
			pod.Name = pod.Name + strconv.Itoa(i)
			pod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
			Expect(err).NotTo(HaveOccurred(), "create pod failed")
			pods = append(pods, pod)
		}

		second := time.Now().Second()
		if second > 55 || second < 5 {
			count := time.Duration((65 - second) % 60)
			time.Sleep(count * time.Second)
		}
		By(fmt.Sprintf("delete time date: %v", time.Now()))

		By("try to delete 11 pods in one minute")
		for i := range pods {
			err := f.ClientSet.CoreV1().Pods(pods[i].Namespace).Delete(pods[i].Name, nil)
			if i < 3 {
				Expect(err).NotTo(HaveOccurred(), "delete pod failed")
			} else if i == 10 {
				Expect(err.Error()).To(ContainSubstring("rejected by flow control"))
			}
		}

		By("wait until next minute")
		expectRecords := fmt.Sprintf(`{"%v":{"deleteCount":10}}`, time.Unix(time.Now().Unix(), 0).Format("200601021504"))
		if time.Now().Second() != 0 {
			sleepSecs := time.Duration(70 - time.Now().Second())
			time.Sleep(sleepSecs * time.Second)
		}

		By("check pod deletion record")
		pdfcCm, err := f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Get(poddeletionflowcontrol.PdfcConfigName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "get pdfc failed")
		framework.Logf("pdfc: %+v", pdfcCm)
		Expect(pdfcCm.Data[poddeletionflowcontrol.PdfcConfigRecordKey]).To(Equal(expectRecords))
	})

	It("[test-counter-reload] test counter reload when apiserver restart [Serial]", func() {
		By("load pdfc template")
		pdfcConfigFile := filepath.Join(util.TestDataDir, "pdfc-config.json")
		pdfcConfig, err := util.LoadConfigMapFromFile(pdfcConfigFile)
		Expect(err).NotTo(HaveOccurred(), "load pdfc template failed")

		By("load pod template")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		podCfg, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "load pod template failed")

		By("create pdfc rules")
		pdfcConfig.Data[poddeletionflowcontrol.PdfcConfigRuleKey] =
			`[{"duration":"2m","deleteLimit":15}]`
		pdfcConfig.Data[poddeletionflowcontrol.PdfcConfigRecordKey] =
			fmt.Sprintf(`{"%v":{"deleteCount":10}}`, time.Unix(time.Now().Add(-time.Minute).Unix(), 0).Format("200601021504"))
		_, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Create(pdfcConfig)
		Expect(err).NotTo(HaveOccurred(), "create pdfc failed")
		defer f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Delete(poddeletionflowcontrol.PdfcConfigName, nil)

		By("create 6 pods")
		var pods []*v1.Pod
		podCfg.Spec.Containers[0].Image = image
		for i := 0; i <= 6; i++ {
			pod := podCfg.DeepCopy()
			pod.Name = pod.Name + strconv.Itoa(i)
			pod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
			Expect(err).NotTo(HaveOccurred(), "create pod failed")
			pods = append(pods, pod)
		}

		By("try to delete 6 pods in one minute")
		for i := range pods {
			err := f.ClientSet.CoreV1().Pods(pods[i].Namespace).Delete(pods[i].Name, nil)
			if i < 5 {
				Expect(err).NotTo(HaveOccurred(), "delete pod failed")
			} else {
				// Todo: fix panic
				Expect(err.Error()).To(ContainSubstring("rejected by flow control"))
			}
		}
	})

	It("[test-rules-change] test rules change when apiserver running [Serial]", func() {
		By("load pdfc template")
		pdfcConfigFile := filepath.Join(util.TestDataDir, "pdfc-config.json")
		pdfcConfig, err := util.LoadConfigMapFromFile(pdfcConfigFile)
		Expect(err).NotTo(HaveOccurred(), "load pdfc template failed")

		By("load pod template")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		podCfg, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "load pod template failed")

		By("create pdfc rules")
		pdfcConfig.Data[poddeletionflowcontrol.PdfcConfigRuleKey] =
			`[{"duration":"5m","deleteLimit":5}]`
		_, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Create(pdfcConfig)
		Expect(err).NotTo(HaveOccurred(), "create pdfc failed")
		defer f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Delete(poddeletionflowcontrol.PdfcConfigName, nil)

		By("create 6 pods")
		var pods []*v1.Pod
		podCfg.Spec.Containers[0].Image = image
		for i := 0; i <= 6; i++ {
			pod := podCfg.DeepCopy()
			pod.Name = pod.Name + strconv.Itoa(i)
			pod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
			Expect(err).NotTo(HaveOccurred(), "create pod failed")
			pods = append(pods, pod)
		}

		By("try to delete 6 pods in one minute")
		for i := range pods {
			err := f.ClientSet.CoreV1().Pods(pods[i].Namespace).Delete(pods[i].Name, nil)
			if i < 5 {
				Expect(err).NotTo(HaveOccurred(), "delete pod failed")
			} else {
				Expect(err.Error()).To(ContainSubstring("rejected by flow control"))
			}
		}

		By("update pdfc rules")
		pdfcCm, err := f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Get(poddeletionflowcontrol.PdfcConfigName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "get pdfc failed")
		pdfcCm.Data[poddeletionflowcontrol.PdfcConfigRuleKey] = `[{"duration":"5m","deleteLimit":15}]`
		_, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Update(pdfcCm)
		Expect(err).NotTo(HaveOccurred(), "update pdfc rules failed")

		By("wait one minute")
		time.Sleep(70 * time.Second)

		By("try to delete last pod")
		err = f.ClientSet.CoreV1().Pods(pods[5].Namespace).Delete(pods[5].Name, nil)
		Expect(err).NotTo(HaveOccurred(), "delete pod failed")
	})
})
