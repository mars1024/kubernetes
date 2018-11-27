package controller_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/ginkgowrapper"
	"k8s.io/kubernetes/test/sigma/util"
	"k8s.io/kubernetes/test/sigma/util/vip"
)

var (
	vipImages = []string{
		"reg.docker.alibaba-inc.com/k8s-test/nginx:1.15.3",
		"reg.docker.alibaba-inc.com/k8s-test/nginx:1.15.3-2",
	}
)

var _ = Describe("[sigma-controller][vip]", func() {
	f := framework.NewDefaultFramework("sigma-controller")

	It("[test-vip-create1] Create a one port vip service [Serial][Slow]", func() {
		By("create a client pod")
		clientPodFile := filepath.Join(util.TestDataDir, "vip-client-pod.json")
		clientPodCfg, err := util.LoadPodFromFile(clientPodFile)
		Expect(err).NotTo(HaveOccurred())
		clientPodCfg.Spec.Containers[0].Command = []string{"sleep", "999d"}
		clientPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(clientPodCfg)
		Expect(err).NotTo(HaveOccurred(), "create client pod failed")
		defer f.ClientSet.CoreV1().Pods(f.Namespace.Name).Delete(clientPod.Name, nil)

		By("create a vip service")
		serviceFile := filepath.Join(util.TestDataDir, "vip-service.json")
		serviceCfg, err := util.LoadServiceFromFile(serviceFile)
		Expect(err).NotTo(HaveOccurred(), "load vip service failed")

		uniqLabel := fmt.Sprintf("vip-test-%v", uuid.NewUUID())
		serviceCfg.Spec.Selector = map[string]string{"usage": uniqLabel}
		vipSvc, err := f.ClientSet.CoreV1().Services(f.Namespace.Name).Create(serviceCfg)
		Expect(err).NotTo(HaveOccurred(), "create vip service failed")
		defer util.WaitUntilServiceDeleted(f.ClientSet, f.Namespace.Name, vipSvc.Name)

		By("check vip status, if not empty, then clean it")
		vip, vport, vprotocol := serviceCfg.Annotations["sigma.ali/vip"], 80, "TCP"
		viputil.CleanupVip(vip, vport, vprotocol)

		By("create app deployment")
		deployFile := filepath.Join(util.TestDataDir, "deploy-base.json")
		deployCfg, err := util.LoadDeploymentFromFile(deployFile)
		Expect(err).NotTo(HaveOccurred())

		deployCfg.Spec.Selector.MatchLabels["usage"] = uniqLabel
		deployCfg.Spec.Template.Labels["usage"] = uniqLabel
		deployCfg.Name = "vipcontrollertest" // don't modify, to enable naming controller
		deployCfg.Spec.Template.Spec.Containers[0].Image = vipImages[0]
		initLables(deployCfg.Spec.Template.Labels)
		deploy, err := f.ClientSet.ExtensionsV1beta1().Deployments(f.Namespace.Name).Create(deployCfg)
		Expect(err).NotTo(HaveOccurred(), "create app deployment failed")
		defer util.DeleteDeploymentPods(f.ClientSet, deploy.Namespace, deploy.Name)

		By("wait until all pods running")
		err = util.WaitDeploymentPodsRunning(f.ClientSet, deploy, 1)
		Expect(err).NotTo(HaveOccurred(), "wait for deployment pods running failed")

		By("wait until client pod running")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, clientPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait for client pod running failed")

		for i := 0; i < 30; i++ {
			stdout, _, _ := f.ExecCommandInContainerWithFullOutput(
				clientPod.Name, clientPod.Spec.Containers[0].Name, "curl", "--connect-timeout", "10", vip+":80")
			if strings.Contains(stdout, "nginx!") {
				return
			}
			framework.Logf("wait until vip works")
			time.Sleep(time.Second)
		}

		ginkgowrapper.Fail("wait for vip works timeout")
	})

	It("[test-vip-create2] Create a multiple ports vip service [Serial][Slow]", func() {
		By("create a client pod")
		clientPodFile := filepath.Join(util.TestDataDir, "vip-client-pod.json")
		clientPodCfg, err := util.LoadPodFromFile(clientPodFile)
		Expect(err).NotTo(HaveOccurred())
		clientPodCfg.Spec.Containers[0].Command = []string{"sleep", "999d"}
		clientPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(clientPodCfg)
		Expect(err).NotTo(HaveOccurred(), "create client pod failed")
		defer f.ClientSet.CoreV1().Pods(f.Namespace.Name).Delete(clientPod.Name, nil)

		By("create a vip service")
		serviceFile := filepath.Join(util.TestDataDir, "vip-service.json")
		serviceCfg, err := util.LoadServiceFromFile(serviceFile)
		Expect(err).NotTo(HaveOccurred(), "load vip service failed")

		uniqLabel := fmt.Sprintf("vip-test-%v", uuid.NewUUID())
		serviceCfg.Spec.Selector = map[string]string{"usage": uniqLabel}
		serviceCfg.Spec.Ports = []v1.ServicePort{
			{Name: "web1", Port: 80, TargetPort: intstr.FromInt(80)},
			{Name: "web2", Port: 8080, TargetPort: intstr.FromInt(8080)},
		}
		vipSvc, err := f.ClientSet.CoreV1().Services(f.Namespace.Name).Create(serviceCfg)
		Expect(err).NotTo(HaveOccurred(), "create vip service failed")
		defer util.WaitUntilServiceDeleted(f.ClientSet, f.Namespace.Name, vipSvc.Name)

		By("check vip status, if not empty, then clean it")
		vip, vport, vprotocol := serviceCfg.Annotations["sigma.ali/vip"], 80, "TCP"
		viputil.CleanupVip(vip, vport, vprotocol)

		By("create app deployment")
		deployFile := filepath.Join(util.TestDataDir, "deploy-base.json")
		deployCfg, err := util.LoadDeploymentFromFile(deployFile)
		Expect(err).NotTo(HaveOccurred())

		deployCfg.Spec.Selector.MatchLabels["usage"] = uniqLabel
		deployCfg.Spec.Template.Labels["usage"] = uniqLabel
		deployCfg.Name = "vipcontrollertest" // don't modify, to enable naming controller
		deployCfg.Spec.Template.Spec.Containers[0].Image = vipImages[1]
		initLables(deployCfg.Spec.Template.Labels)
		deploy, err := f.ClientSet.ExtensionsV1beta1().Deployments(f.Namespace.Name).Create(deployCfg)
		Expect(err).NotTo(HaveOccurred(), "create app deployment failed")
		defer util.DeleteDeploymentPods(f.ClientSet, deploy.Namespace, deploy.Name)

		By("wait until all pods running")
		err = util.WaitDeploymentPodsRunning(f.ClientSet, deploy, 1)
		Expect(err).NotTo(HaveOccurred(), "wait for deployment pods running failed")

		By("wait until client pod running")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, clientPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait for client pod running failed")

		var i int
		for i = 0; i < 30; i++ {
			stdout, _, _ := f.ExecCommandInContainerWithFullOutput(
				clientPod.Name, clientPod.Spec.Containers[0].Name, "curl", "--connect-timeout", "10", vip+":80")
			if strings.Contains(stdout, "nginx:80!") {
				break
			}
			framework.Logf("wait until vip works")
			time.Sleep(time.Second)
		}
		if i == 30 {
			ginkgowrapper.Fail("wait for vip works timeout")
		}
		for i = 0; i < 30; i++ {
			stdout, _, _ := f.ExecCommandInContainerWithFullOutput(
				clientPod.Name, clientPod.Spec.Containers[0].Name, "curl", "--connect-timeout", "10", vip+":8080")
			if strings.Contains(stdout, "nginx:8080!") {
				return
			}
			framework.Logf("wait until vip works")
			time.Sleep(time.Second)
		}

		ginkgowrapper.Fail("wait for vip works timeout")
	})

	It("[smoke][test-vip-sync1] Test vip sync with pods number [Serial][Slow]", func() {
		By("create a vip service")
		serviceFile := filepath.Join(util.TestDataDir, "vip-service.json")
		serviceCfg, err := util.LoadServiceFromFile(serviceFile)
		Expect(err).NotTo(HaveOccurred(), "load vip service failed")

		uniqLabel := fmt.Sprintf("vip-test-%v", uuid.NewUUID())
		serviceCfg.Spec.Selector = map[string]string{"usage": uniqLabel}
		vipSvc, err := f.ClientSet.CoreV1().Services(f.Namespace.Name).Create(serviceCfg)
		Expect(err).NotTo(HaveOccurred(), "create vip service failed")
		defer util.WaitUntilServiceDeleted(f.ClientSet, f.Namespace.Name, vipSvc.Name)

		By("check vip status, if not empty, then clean it")
		vip, vport, vprotocol := serviceCfg.Annotations["sigma.ali/vip"], 80, "TCP"
		viputil.CleanupVip(vip, vport, vprotocol)

		By("create app deployment")
		deployFile := filepath.Join(util.TestDataDir, "deploy-base.json")
		deployCfg, err := util.LoadDeploymentFromFile(deployFile)
		Expect(err).NotTo(HaveOccurred())

		deployCfg.Spec.Selector.MatchLabels["usage"] = uniqLabel
		deployCfg.Spec.Template.Labels["usage"] = uniqLabel
		deployCfg.Name = "vipcontrollertest" // don't modify, to enable naming controller
		deployCfg.Spec.Template.Spec.Containers[0].Image = vipImages[0]
		initLables(deployCfg.Spec.Template.Labels)
		deploy, err := f.ClientSet.ExtensionsV1beta1().Deployments(f.Namespace.Name).Create(deployCfg)
		Expect(err).NotTo(HaveOccurred(), "create app deployment failed")
		defer util.DeleteDeploymentPods(f.ClientSet, deploy.Namespace, deploy.Name)

		By("wait until all pods running")
		err = util.WaitDeploymentPodsRunning(f.ClientSet, deploy, 1)
		Expect(err).NotTo(HaveOccurred(), "wait for deployment pods running failed")

		By("scale deployment replicas to 9")
		err = util.UpdateDeploymentReplicas(f.ClientSet, deploy, 9)
		Expect(err).NotTo(HaveOccurred(), "update deployment replicas failed")

		By("wait until all pods running")
		err = util.WaitDeploymentPodsRunning(f.ClientSet, deploy, 9)
		Expect(err).NotTo(HaveOccurred(), "wait for deployment pods running failed")

		By("check is backends correct")
		pods, err := util.ListDeploymentPods(f.ClientSet, deploy)
		Expect(err).NotTo(HaveOccurred(), "get deployment pods failed")
		err = viputil.WaitUntilBackendsCorrect(pods, vip, vport, vprotocol, 10*time.Second, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait until backends correct failed")

		By("scale deployment replicas to 5")
		err = util.UpdateDeploymentReplicas(f.ClientSet, deploy, 5)
		Expect(err).NotTo(HaveOccurred(), "update deployment replicas failed")

		By("wait until all pods running")
		err = util.WaitDeploymentPodsRunning(f.ClientSet, deploy, 5)
		Expect(err).NotTo(HaveOccurred(), "wait for deployment pods running failed")

		By("check is backends again")
		pods, err = util.ListDeploymentPods(f.ClientSet, deploy)
		Expect(err).NotTo(HaveOccurred(), "get deployment pods failed")
		err = viputil.WaitUntilBackendsCorrect(pods, vip, vport, vprotocol, 10*time.Second, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait until backends correct failed")
	})

	It("[test-vip-sync2] Test vip pods cleanup [Serial][Slow]", func() {
		By("create a vip service")
		serviceFile := filepath.Join(util.TestDataDir, "vip-service.json")
		serviceCfg, err := util.LoadServiceFromFile(serviceFile)
		Expect(err).NotTo(HaveOccurred(), "load vip service failed")

		uniqLabel := fmt.Sprintf("vip-test-%v", uuid.NewUUID())
		serviceCfg.Spec.Selector = map[string]string{"usage": uniqLabel}
		vipSvc, err := f.ClientSet.CoreV1().Services(f.Namespace.Name).Create(serviceCfg)
		Expect(err).NotTo(HaveOccurred(), "create vip service failed")
		defer util.WaitUntilServiceDeleted(f.ClientSet, f.Namespace.Name, vipSvc.Name)

		By("check vip status, if not empty, then clean it")
		vip, vport, vprotocol := serviceCfg.Annotations["sigma.ali/vip"], 80, "TCP"
		viputil.CleanupVip(vip, vport, vprotocol)

		By("create app deployment")
		deployFile := filepath.Join(util.TestDataDir, "deploy-base.json")
		deployCfg, err := util.LoadDeploymentFromFile(deployFile)
		Expect(err).NotTo(HaveOccurred())

		replicas := int32(10)
		deployCfg.Spec.Replicas = &replicas
		deployCfg.Spec.Selector.MatchLabels["usage"] = uniqLabel
		deployCfg.Spec.Template.Labels["usage"] = uniqLabel
		deployCfg.Name = "vipcontrollertest" // don't modify, to enable naming controller
		deployCfg.Spec.Template.Spec.Containers[0].Image = vipImages[0]
		initLables(deployCfg.Spec.Template.Labels)
		deploy, err := f.ClientSet.ExtensionsV1beta1().Deployments(f.Namespace.Name).Create(deployCfg)
		Expect(err).NotTo(HaveOccurred(), "create app deployment failed")
		defer util.DeleteDeploymentPods(f.ClientSet, deploy.Namespace, deploy.Name)

		By("wait until all pods running")
		err = util.WaitDeploymentPodsRunning(f.ClientSet, deploy, 10)
		Expect(err).NotTo(HaveOccurred(), "wait for deployment pods running failed")

		By("scale deployment replicas to 0")
		err = util.UpdateDeploymentReplicas(f.ClientSet, deploy, 0)
		Expect(err).NotTo(HaveOccurred(), "update deployment replicas failed")
		err = util.WaitTimeoutForPodReplicas(f.ClientSet, deploy, 0, 5*time.Second, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait deployment replicas failed")

		By("check is backends correct")
		err = viputil.WaitUntilBackendsCorrect(nil, vip, vport, vprotocol, 10*time.Second, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait until backends correct failed")
	})
})

func initLables(labels map[string]string) {
	labels["sigma.ali/site"] = "et2sqa"
	labels["sigma.ali/app-name"] = "sigma-k8s-apiserver"
	labels["sigma.ali/instance-group"] = "sigma-k8s-apiserver"
	labels["sigma.alibaba-inc.com/app-stage"] = "DAILY"
	labels["sigma.alibaba-inc.com/app-unit"] = "CENTER_UNIT.center"
}
