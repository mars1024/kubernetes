package apiserver

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"path/filepath"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[kube-apiserver][admission][newinjection]", func() {
	image := "reg.docker.alibaba-inc.com/k8s-test/nginx:1.15.3"
	f := framework.NewDefaultFramework("sigma-apiserver")

	It("[smoke][new-injection-pre-schedule] test new ali pod injection pre scheduler [Serial]", func() {
		By("load app and global template")
		injectionAppConfigFile := filepath.Join(util.TestDataDir, "new-injection-app-configmap.json")
		injectionGlobalConfigFile := filepath.Join(util.TestDataDir, "new-injection-global-configmap.json")
		injectionGrayScaleConfigFile := filepath.Join(util.TestDataDir, "new-injection-grayscale-configmap.json")

		appConfig, err := util.LoadConfigMapFromFile(injectionAppConfigFile)
		Expect(err).NotTo(HaveOccurred(), "load app rules template failed")
		globalConfig, err := util.LoadConfigMapFromFile(injectionGlobalConfigFile)
		Expect(err).NotTo(HaveOccurred(), "load global rules template failed")
		graySacleConfig, err := util.LoadConfigMapFromFile(injectionGrayScaleConfigFile)
		Expect(err).NotTo(HaveOccurred(), "load gary scale rules template failed")

		By("load pod template")
		podFile := filepath.Join(util.TestDataDir, "new-injection-pod-base.json")
		podCfg, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "load pod template failed")

		By("create new injection rules")
		_, err = f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Create(appConfig)
		Expect(err).NotTo(HaveOccurred(), "create app rules failed")

		_, err = f.ClientSet.CoreV1().ConfigMaps("kube-system").Create(globalConfig)
		//Expect(err).NotTo(HaveOccurred(), "create global failed")

		_, err = f.ClientSet.CoreV1().ConfigMaps("kube-system").Create(graySacleConfig)
		//Expect(err).NotTo(HaveOccurred(), "create grayscale failed")

		defer f.ClientSet.CoreV1().ConfigMaps(f.Namespace.Name).Delete(appConfig.Name, nil)
		defer f.ClientSet.CoreV1().ConfigMaps("kube-system").Delete(globalConfig.Name, nil)
		defer f.ClientSet.CoreV1().ConfigMaps("kube-system").Delete(graySacleConfig.Name, nil)

		By("create pods")
		podCfg.Spec.Containers[0].Image = image
		pod, err := util.CreatePod(f.ClientSet, podCfg, f.Namespace.Name)
		Expect(err).NotTo(HaveOccurred(), "create pod failed")
		defer util.DeletePod(f.ClientSet, pod)

		By("wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, pod, v1.PodRunning, 1*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	})
})
