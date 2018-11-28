package controller_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
	"k8s.io/apimachinery/pkg/api/resource"
	corev1 "k8s.io/api/core/v1"
	alipayapis "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	zappinfoclient "gitlab.alipay-inc.com/sigma/controller-manager/pkg/zappinfo"
	"gitlab.alipay-inc.com/sigma/controller-manager/pkg/alipaymeta"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func newPod(namespace string, name string, nodeName string, registered bool) *corev1.Pod {
	hostname := "dapanweb-" + string(uuid.NewUUID())
	info := alipayapis.PodZappinfo{
		Spec: &alipayapis.PodZappinfoSpec{
			AppName:    "dapanweb",
			Zone:       "RZ11A",
			ServerType: "DOCKER_VM",
			Fqdn:       fmt.Sprintf("%s.eu95.alipay.net", hostname),
		},
		Status: &alipayapis.PodZappinfoStatus{
			Registered: registered,
		},
	}

	infoBytes, _ := json.Marshal(info)
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				alipayapis.AnnotationZappinfo: string(infoBytes),
			},
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
			Hostname: hostname,
			Containers: []corev1.Container{
				{
					Name:  "c1",
					Image: "reg.docker.alibaba-inc.com/sigma-api-bvt/nginx:1.13.5-alpine",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:              resource.MustParse("2"),
							corev1.ResourceMemory:           resource.MustParse("2Gi"),
							corev1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
						},
					},
				},
			},
			SchedulerName: "default-scheduler",
		},
	}
}

var (
	zappinfoUrl   = "http://zappinfo.stable.alipay.net"
	zappinfoToken = "230cd56e4f9145f39fb65adf56e4fcb8"
)

var _ = Describe("[ant][sigma-alipay-controller][zappinfo]", func() {
	f := framework.NewDefaultFramework("sigma-ant-controller")
	zc := zappinfoclient.NewZappinfoClient(zappinfoUrl, zappinfoToken)
	It("[sigma-alipay-controller][zappinfo][smoke] create a pod with zappinfo unregistered, should register in zappinfo", func() {
		By("create a pod with zappinfo unregistered")
		pod := newPod(f.Namespace.Name, "e2e-pod", "", false)

		// name should be unique
		pod.Name = "zappinfo-controller-e2e-" + time.Now().Format("20160607123450")
		pod.Spec.Hostname = pod.Name

		// the following tables are MUST required
		pod.Labels = make(map[string]string, 0)
		pod.Labels["sigma.ali/site"] = "eu95"
		pod.Labels["sigma.ali/app-name"] = "dapanweb"
		pod.Labels["sigma.ali/instance-group"] = "sigma-alipay-test"
		pod.Labels["sigma.alibaba-inc.com/app-unit"] = "CENTER_UNIT.center"
		pod.Labels["sigma.alibaba-inc.com/app-stage"] = "DAILY"

		testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")
		glog.Infof("create pod config:%v", pod)

		defer util.DeletePod(f.ClientSet, testPod)

		By("wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(getPod.Status.HostIP).NotTo(BeEmpty(), "status.HostIP should not be empty")
		Expect(getPod.Status.PodIP).NotTo(BeEmpty(), "status.PodIP should not be empty")

		By("sleep 1s to wait for zappinfo controller register pod info in zappinfo")
		time.Sleep(3 * time.Second)

		By("check pod has been registered in zappinfo")

		// could query by both name and ip
		zappinfo, err := zc.GetServerByIp(getPod.Status.PodIP)
		glog.Info("zappinfo :%v", zappinfo)
		Expect(err).NotTo(HaveOccurred(), "query zappinfo service should pass")
		Expect(zappinfo).NotTo(BeNil(), "zappinfo service info should not be empty")
		Expect(zappinfo.Ip).Should(Equal(getPod.Status.PodIP), "ip should be same with pod.status.podip in zappinfo")
		Expect(zappinfo.Hostname).Should(Equal(getPod.Spec.Hostname), "zappinfo hostname should be same with pod.spec.hostname")
		Expect(zappinfo.ParentSn).Should(Equal(getPod.Spec.NodeName), "zappinfo ParentSn should be same with pod.spec.nodeName")
		Expect(getPod.Finalizers).Should(ContainElement(ContainSubstring(alipaymeta.ZappinfoFinalizer)), "ZappinfoFinalizer should be register into pod.finalizers")

		By("check annotation alipayapis.AnnotationZappinfo")
		data, _ := getPod.Annotations[alipayapis.AnnotationZappinfo]
		var z alipayapis.PodZappinfo
		err = json.Unmarshal([]byte(data), &z)
		Expect(err).Should(BeNil(), "Unmarshal alipayapis.AnnotationZappinfo should be okay")
		Expect(z.Status.Registered).Should(Equal(true), "zappinfo status should be registered")

		By("delete a pod should success")
		err = util.DeletePod(f.ClientSet, testPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		By("check pod should have been unregistered from zappinfo")
		newZappinfo, _ := zc.GetServerByIp(getPod.Status.PodIP)
		Expect(newZappinfo).To(BeNil(), "zappinfo service info should be empty")
	})
	It("[sigma-controller][zappinfo][smoke] create a pod with zappinfo registered, should not register in zappinfo", func() {
		By("create a pod with zappinfo registered")
		pod := newPod(f.Namespace.Name, "e2e-pod", "", true)

		// name should be unique
		pod.Name = "zappinfo-controller-e2e-" + time.Now().Format("20160607123450")
		pod.Spec.Hostname = pod.Name

		// the following tables are MUST required
		pod.Labels = make(map[string]string, 0)
		pod.Labels["sigma.ali/site"] = "eu95"
		pod.Labels["sigma.ali/app-name"] = "dapanweb"
		pod.Labels["sigma.ali/instance-group"] = "sigma-alipay-test"
		pod.Labels["sigma.alibaba-inc.com/app-unit"] = "CENTER_UNIT.center"
		pod.Labels["sigma.alibaba-inc.com/app-stage"] = "DAILY"

		testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")
		glog.Infof("create pod config:%v", pod)

		defer util.DeletePod(f.ClientSet, testPod)

		By("wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(getPod.Status.HostIP).NotTo(BeEmpty(), "status.HostIP should not be empty")
		Expect(getPod.Status.PodIP).NotTo(BeEmpty(), "status.PodIP should not be empty")

		By("sleep 1s to wait for zappinfo controller register pod info in zappinfo")
		time.Sleep(3 * time.Second)

		By("check pod has not been registered in zappinfo")

		// could query by both name and ip
		zappinfo, err := zc.GetServerByIp(getPod.Status.PodIP)
		Expect(err).NotTo(HaveOccurred(), "query zappinfo service should pass")
		Expect(zappinfo).Should(BeNil(), "zappinfo service info should be empty")
		glog.Info("zappinfo :%v", zappinfo)

		By("delete a pod should success")
		err = util.DeletePod(f.ClientSet, testPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
	})
})
