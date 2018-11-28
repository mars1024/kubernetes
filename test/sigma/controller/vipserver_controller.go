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
	"k8s.io/kubernetes/test/sigma/util/vipserver"
)

var (
	images = []string{
		"reg.docker.alibaba-inc.com/k8s-test/nginx:1.15.3",
		"reg.docker.alibaba-inc.com/k8s-test/nginx:1.15.3-2",
	}
)

var _ = Describe("[sigma-controller][vipserver]", func() {
	f := framework.NewDefaultFramework("sigma-controller")

	It("[test-create1] Create a vipserver without domain indicated [Slow]", func() {
		By("create a client pod with client and dns-f container")
		clientPodFile := filepath.Join(util.TestDataDir, "vipserver-client-pod.json")
		clientPodCfg, err := util.LoadPodFromFile(clientPodFile)
		Expect(err).NotTo(HaveOccurred())
		clientPodCfg.Spec.Containers[0].Command = []string{"sleep", "999d"}
		clientPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(clientPodCfg)
		Expect(err).NotTo(HaveOccurred(), "create client pod failed")
		defer f.ClientSet.CoreV1().Pods(f.Namespace.Name).Delete(clientPod.Name, nil)

		By("create a vipserver service")
		serviceFile := filepath.Join(util.TestDataDir, "vipserver-service.json")
		serviceCfg, err := util.LoadServiceFromFile(serviceFile)
		Expect(err).NotTo(HaveOccurred(), "load vipserver service failed")

		uniqLabel := fmt.Sprintf("vipserver-test-%v", uuid.NewUUID())
		serviceCfg.Spec.Selector = map[string]string{"usage": uniqLabel}
		vipserverSvc, err := f.ClientSet.CoreV1().Services(f.Namespace.Name).Create(serviceCfg)
		Expect(err).NotTo(HaveOccurred(), "create vipserver service failed")
		defer util.WaitUntilServiceDeleted(f.ClientSet, f.Namespace.Name, vipserverSvc.Name)

		By("create app deployment")
		deployFile := filepath.Join(util.TestDataDir, "deploy-base.json")
		deployCfg, err := util.LoadDeploymentFromFile(deployFile)
		Expect(err).NotTo(HaveOccurred())

		deployCfg.Spec.Selector.MatchLabels["usage"] = uniqLabel
		deployCfg.Spec.Template.Labels["usage"] = uniqLabel
		deploy, err := f.ClientSet.ExtensionsV1beta1().Deployments(f.Namespace.Name).Create(deployCfg)
		Expect(err).NotTo(HaveOccurred(), "create app deployment failed")
		defer util.DeleteDeploymentPods(f.ClientSet, deploy.Namespace, deploy.Name)

		// the server url = servicename.namespace.vipserver:port
		domain := fmt.Sprintf("%v.%v.vipserver", vipserverSvc.Name, f.Namespace.Name)
		defer vipserverutil.RemoveDomain(domain)
		srvUrl := domain + ":80"
		framework.Logf("service domain: %v", srvUrl)

		By("wait until all pods running")
		err = util.WaitDeploymentPodsRunning(f.ClientSet, deploy, 1)
		Expect(err).NotTo(HaveOccurred(), "wait for deployment pods running failed")

		By("wait until client pod running")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, clientPod, v1.PodRunning, 5*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait for client pod running failed")

		for i := 0; i < 30; i++ {
			stdout, _, _ := f.ExecCommandInContainerWithFullOutput(
				clientPod.Name, clientPod.Spec.Containers[0].Name, "curl", "--connect-timeout", "10", srvUrl)
			if strings.Contains(stdout, "nginx!") {
				return
			}
			framework.Logf("wait until dns-f works")
			time.Sleep(10 * time.Second)
		}

		ginkgowrapper.Fail("wait for dns-f works timeout")
	})

	It("[test-create2] Create a vipserver with domain indicated [Slow]", func() {
		By("create a client pod with client and dns-f container")
		clientPodFile := filepath.Join(util.TestDataDir, "vipserver-client-pod.json")
		clientPodCfg, err := util.LoadPodFromFile(clientPodFile)
		Expect(err).NotTo(HaveOccurred())
		clientPodCfg.Spec.Containers[0].Command = []string{"sleep", "999d"}
		clientPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(clientPodCfg)
		Expect(err).NotTo(HaveOccurred(), "create client pod failed")
		defer f.ClientSet.CoreV1().Pods(f.Namespace.Name).Delete(clientPod.Name, nil)

		By("create a vipserver service")
		serviceFile := filepath.Join(util.TestDataDir, "vipserver-service.json")
		serviceCfg, err := util.LoadServiceFromFile(serviceFile)
		Expect(err).NotTo(HaveOccurred(), "load vipserver service failed")

		domain := fmt.Sprintf("%v-one-port", f.Namespace.Name)
		defer vipserverutil.RemoveDomain(domain)
		uniqLabel := fmt.Sprintf("vipserver-test-%v", uuid.NewUUID())
		serviceCfg.Spec.Selector = map[string]string{"usage": uniqLabel}
		serviceCfg.Annotations = map[string]string{"sigma.ali/vipserver-domain": domain}
		vipserverSvc, err := f.ClientSet.CoreV1().Services(f.Namespace.Name).Create(serviceCfg)
		Expect(err).NotTo(HaveOccurred(), "create vipserver service failed")
		defer util.WaitUntilServiceDeleted(f.ClientSet, f.Namespace.Name, vipserverSvc.Name)

		By("create app deployment")
		deployFile := filepath.Join(util.TestDataDir, "deploy-base.json")
		deployCfg, err := util.LoadDeploymentFromFile(deployFile)
		Expect(err).NotTo(HaveOccurred())

		deployCfg.Spec.Selector.MatchLabels["usage"] = uniqLabel
		deployCfg.Spec.Template.Labels["usage"] = uniqLabel
		deploy, err := f.ClientSet.ExtensionsV1beta1().Deployments(f.Namespace.Name).Create(deployCfg)
		Expect(err).NotTo(HaveOccurred(), "create app deployment failed")
		defer util.DeleteDeploymentPods(f.ClientSet, deploy.Namespace, deploy.Name)

		By("wait until all pods running")
		err = util.WaitDeploymentPodsRunning(f.ClientSet, deploy, 1)
		Expect(err).NotTo(HaveOccurred(), "wait for deployment pods running failed")

		By("wait until client pod running")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, clientPod, v1.PodRunning, 5*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait for client pod running failed")

		srvUrl := domain + ":80"
		for i := 0; i < 30; i++ {
			stdout, _, _ := f.ExecCommandInContainerWithFullOutput(
				clientPod.Name, clientPod.Spec.Containers[0].Name, "curl", "--connect-timeout", "10", srvUrl)
			if strings.Contains(stdout, "nginx!") {
				return
			}
			framework.Logf("wait until dns-f works")
			time.Sleep(10 * time.Second)
		}

		ginkgowrapper.Fail("wait for dns-f works timeout")
	})

	It("[smoke][test-create3] Create a vipserver with multiple ports [Slow]", func() {
		By("create a client pod with client and dns-f container")
		clientPodFile := filepath.Join(util.TestDataDir, "vipserver-client-pod.json")
		clientPodCfg, err := util.LoadPodFromFile(clientPodFile)
		Expect(err).NotTo(HaveOccurred())
		clientPodCfg.Spec.Containers[0].Command = []string{"sleep", "999d"}
		clientPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(clientPodCfg)
		Expect(err).NotTo(HaveOccurred(), "create client pod failed")
		defer f.ClientSet.CoreV1().Pods(f.Namespace.Name).Delete(clientPod.Name, nil)

		By("create a vipserver service")
		serviceFile := filepath.Join(util.TestDataDir, "vipserver-service.json")
		serviceCfg, err := util.LoadServiceFromFile(serviceFile)
		Expect(err).NotTo(HaveOccurred(), "load vipserver service failed")

		domain := fmt.Sprintf("%v-multiple-ports", f.Namespace.Name)
		defer vipserverutil.RemoveDomain(domain)
		uniqLabel := fmt.Sprintf("vipserver-test-%v", uuid.NewUUID())
		serviceCfg.Spec.Selector = map[string]string{"usage": uniqLabel}
		serviceCfg.Annotations = map[string]string{"sigma.ali/vipserver-domain": domain}
		serviceCfg.Spec.Ports = []v1.ServicePort{
			{Name: "web1", Port: 1234, TargetPort: intstr.FromInt(80)},
			{Name: "web2", Port: 4321, TargetPort: intstr.FromInt(8080)},
		}
		vipserverSvc, err := f.ClientSet.CoreV1().Services(f.Namespace.Name).Create(serviceCfg)
		Expect(err).NotTo(HaveOccurred(), "create vipserver service failed")
		defer util.WaitUntilServiceDeleted(f.ClientSet, f.Namespace.Name, vipserverSvc.Name)

		By("create app deployment")
		deployFile := filepath.Join(util.TestDataDir, "deploy-base.json")
		deployCfg, err := util.LoadDeploymentFromFile(deployFile)
		Expect(err).NotTo(HaveOccurred())

		deployCfg.Spec.Selector.MatchLabels["usage"] = uniqLabel
		deployCfg.Spec.Template.Labels["usage"] = uniqLabel
		deployCfg.Spec.Template.Spec.Containers[0].Image = images[1]
		deploy, err := f.ClientSet.ExtensionsV1beta1().Deployments(f.Namespace.Name).Create(deployCfg)
		Expect(err).NotTo(HaveOccurred(), "create app deployment failed")
		defer util.DeleteDeploymentPods(f.ClientSet, deploy.Namespace, deploy.Name)

		By("wait until all pods running")
		err = util.WaitDeploymentPodsRunning(f.ClientSet, deploy, 1)
		Expect(err).NotTo(HaveOccurred(), "wait for deployment pods running failed")

		By("wait until client pod running")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, clientPod, v1.PodRunning, 5*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait for client pod running failed")

		srvUrls := []string{domain + ":80", domain + ":8080"}
		for i := 0; i < 30; i++ {
			stdout, _, _ := f.ExecCommandInContainerWithFullOutput(
				clientPod.Name, clientPod.Spec.Containers[0].Name, "curl", "--connect-timeout", "10", srvUrls[0])
			if strings.Contains(stdout, "nginx:80!") {
				break
			}
			framework.Logf("wait until dns-f works")
			time.Sleep(10 * time.Second)
		}
		for i := 0; i < 30; i++ {
			stdout, _, _ := f.ExecCommandInContainerWithFullOutput(
				clientPod.Name, clientPod.Spec.Containers[0].Name, "curl", "--connect-timeout", "10", srvUrls[1])
			if strings.Contains(stdout, "nginx:8080!") {
				return
			}
			framework.Logf("wait until dns-f works")
			time.Sleep(10 * time.Second)
		}

		ginkgowrapper.Fail("wait for dns-f works timeout")
	})

	It("[smoke][test-sync1] Test viperver sync with pods number [Slow]", func() {
		By("create a vipserver service")
		serviceFile := filepath.Join(util.TestDataDir, "vipserver-service.json")
		serviceCfg, err := util.LoadServiceFromFile(serviceFile)
		Expect(err).NotTo(HaveOccurred(), "load vipserver service failed")

		domain := fmt.Sprintf("%v-sync-pods-number", f.Namespace.Name)
		defer vipserverutil.RemoveDomain(domain)
		uniqLabel := fmt.Sprintf("vipserver-test-%v", uuid.NewUUID())
		serviceCfg.Spec.Selector = map[string]string{"usage": uniqLabel}
		serviceCfg.Annotations = map[string]string{"sigma.ali/vipserver-domain": domain}
		vipserverSvc, err := f.ClientSet.CoreV1().Services(f.Namespace.Name).Create(serviceCfg)
		Expect(err).NotTo(HaveOccurred(), "create vipserver service failed")
		defer util.WaitUntilServiceDeleted(f.ClientSet, f.Namespace.Name, vipserverSvc.Name)

		By("create app deployment")
		deployFile := filepath.Join(util.TestDataDir, "deploy-base.json")
		deployCfg, err := util.LoadDeploymentFromFile(deployFile)
		Expect(err).NotTo(HaveOccurred())

		deployCfg.Spec.Selector.MatchLabels["usage"] = uniqLabel
		deployCfg.Spec.Template.Labels["usage"] = uniqLabel
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
		err = vipserverutil.WaitUntilBackendsCorrect(pods, domain, time.Second, 5*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait until backends correct failed")

		By("scale deployment replicas to 5")
		err = util.UpdateDeploymentReplicas(f.ClientSet, deploy, 5)
		Expect(err).NotTo(HaveOccurred(), "update deployment replicas failed")

		By("wait until all pods running")
		err = util.WaitDeploymentPodsRunning(f.ClientSet, deploy, 5)
		Expect(err).NotTo(HaveOccurred(), "wait for deployment pods running failed")

		By("check is backends correct")
		pods, err = util.ListDeploymentPods(f.ClientSet, deploy)
		Expect(err).NotTo(HaveOccurred(), "get deployment pods failed")
		err = vipserverutil.WaitUntilBackendsCorrect(pods, domain, time.Second, 5*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait until backends correct failed")
	})

	It("[smoke][test-sync2] Test viperver sync with pods status [Slow]", func() {
		By("create a vipserver service")
		serviceFile := filepath.Join(util.TestDataDir, "vipserver-service.json")
		serviceCfg, err := util.LoadServiceFromFile(serviceFile)
		Expect(err).NotTo(HaveOccurred(), "load vipserver service failed")

		domain := fmt.Sprintf("%v-sync-pods-status", f.Namespace.Name)
		defer vipserverutil.RemoveDomain(domain)
		uniqLabel := fmt.Sprintf("vipserver-test-%v", uuid.NewUUID())
		serviceCfg.Spec.Selector = map[string]string{"usage": uniqLabel}
		serviceCfg.Annotations = map[string]string{"sigma.ali/vipserver-domain": domain}
		vipserverSvc, err := f.ClientSet.CoreV1().Services(f.Namespace.Name).Create(serviceCfg)
		Expect(err).NotTo(HaveOccurred(), "create vipserver service failed")
		defer util.WaitUntilServiceDeleted(f.ClientSet, f.Namespace.Name, vipserverSvc.Name)

		By("create app deployment")
		deployFile := filepath.Join(util.TestDataDir, "deploy-base.json")
		deployCfg, err := util.LoadDeploymentFromFile(deployFile)
		Expect(err).NotTo(HaveOccurred())

		replicas := int32(5)
		deployCfg.Spec.Selector.MatchLabels["usage"] = uniqLabel
		deployCfg.Spec.Template.Labels["usage"] = uniqLabel
		deployCfg.Spec.Replicas = &replicas
		deployCfg.Spec.Template.Spec.Containers[0].ReadinessProbe = &v1.Probe{
			Handler: v1.Handler{
				Exec: &v1.ExecAction{
					Command: []string{"cat", "/tmp/healthy"},
				},
			},
			PeriodSeconds: 1,
		}
		deployCfg.Spec.Template.Spec.Containers[0].Command = []string{
			"/bin/bash", "-c", "touch /tmp/healthy;  sleep 1d"}
		deploy, err := f.ClientSet.ExtensionsV1beta1().Deployments(f.Namespace.Name).Create(deployCfg)
		Expect(err).NotTo(HaveOccurred(), "create app deployment failed")
		defer util.DeleteDeploymentPods(f.ClientSet, deploy.Namespace, deploy.Name)

		By("wait until all pods running")
		err = util.WaitDeploymentPodsRunning(f.ClientSet, deploy, 5)
		Expect(err).NotTo(HaveOccurred(), "wait for deployment pods running failed")

		By("wait until all pods container status ready")
		pods, err := util.ListDeploymentPods(f.ClientSet, deploy)
		Expect(err).NotTo(HaveOccurred(), "get deployment pods failed")
		for i := 0; i < len(pods.Items); i++ {
			util.WaitTimeoutForPodContainerStatusReady(f.ClientSet, &pods.Items[i], 5*time.Minute)
		}

		By("check is backends correct")
		pods, err = util.ListDeploymentPods(f.ClientSet, deploy)
		err = vipserverutil.WaitUntilBackendsCorrect(pods, domain, time.Second, 5*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait until backends correct failed")

		cmd := []string{"rm", "/tmp/healthy"}

		for i := 0; i < len(pods.Items); i++ {
			_, _, err := f.ExecWithOptions(framework.ExecOptions{
				Command:       cmd,
				Namespace:     f.Namespace.Name,
				PodName:       pods.Items[i].Name,
				ContainerName: pods.Items[i].Status.ContainerStatuses[0].Name,
				CaptureStdout: true,
				CaptureStderr: true,
			})
			Expect(err).NotTo(HaveOccurred(), "exec rm /tmp/healthy in pod failed")
		}

		By("check is backends correct again")
		err = vipserverutil.WaitUntilBackendsCorrect(nil, domain, time.Second, 5*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait until backends correct failed")
	})

	It("[smoke][test-sync3] Test viperver pods cleanup [Slow][Serial]", func() {
		By("create a vipserver service")
		serviceFile := filepath.Join(util.TestDataDir, "vipserver-service.json")
		serviceCfg, err := util.LoadServiceFromFile(serviceFile)
		Expect(err).NotTo(HaveOccurred(), "load vipserver service failed")

		domain := fmt.Sprintf("%v-sync-pods-cleanup", f.Namespace.Name)
		defer vipserverutil.RemoveDomain(domain)
		uniqLabel := fmt.Sprintf("vipserver-test-%v", uuid.NewUUID())
		serviceCfg.Spec.Selector = map[string]string{"usage": uniqLabel}
		serviceCfg.Annotations = map[string]string{"sigma.ali/vipserver-domain": domain}
		vipserverSvc, err := f.ClientSet.CoreV1().Services(f.Namespace.Name).Create(serviceCfg)
		Expect(err).NotTo(HaveOccurred(), "create vipserver service failed")
		defer util.WaitUntilServiceDeleted(f.ClientSet, f.Namespace.Name, vipserverSvc.Name)

		By("create app deployment")
		deployFile := filepath.Join(util.TestDataDir, "deploy-base.json")
		deployCfg, err := util.LoadDeploymentFromFile(deployFile)
		Expect(err).NotTo(HaveOccurred())

		replicas := int32(10)
		deployCfg.Spec.Replicas = &replicas
		deployCfg.Spec.Selector.MatchLabels["usage"] = uniqLabel
		deployCfg.Spec.Template.Labels["usage"] = uniqLabel
		deploy, err := f.ClientSet.ExtensionsV1beta1().Deployments(f.Namespace.Name).Create(deployCfg)
		Expect(err).NotTo(HaveOccurred(), "create app deployment failed")
		defer util.DeleteDeploymentPods(f.ClientSet, deploy.Namespace, deploy.Name)

		By("wait until all pods running")
		err = util.WaitDeploymentPodsRunning(f.ClientSet, deploy, int(replicas))
		Expect(err).NotTo(HaveOccurred(), "wait for deployment pods running failed")

		By("check is backends correct")
		pods, err := util.ListDeploymentPods(f.ClientSet, deploy)
		Expect(err).NotTo(HaveOccurred(), "get deployment pods failed")
		err = vipserverutil.WaitUntilBackendsCorrect(pods, domain, time.Second, 5*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait until backends correct failed")

		By("scale deployment replicas to 0")
		err = util.UpdateDeploymentReplicas(f.ClientSet, deploy, 0)
		Expect(err).NotTo(HaveOccurred(), "update deployment replicas failed")
		err = util.WaitTimeoutForPodReplicas(f.ClientSet, deploy, 0, 5*time.Second, 5*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait deployment replicas failed")

		By("check is backends correct")
		err = vipserverutil.WaitUntilBackendsCorrect(nil, domain, time.Second, 5*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "wait until backends correct failed")
	})
})
