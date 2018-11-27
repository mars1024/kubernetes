package kubelet

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("[sigma-kubelet] Generate hostname check", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	var testPod *v1.Pod
	It("Create a pod without hostname template annotation", func() {
		By("Load a pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "fail to load pod from file")

		By("Create a pod")
		testPod = f.PodClient().Create(pod)
		defer util.DeletePod(f.ClientSet, testPod)

		By("Waiting for pods to come up.")
		err = framework.WaitForPodRunningInNamespace(f.ClientSet, testPod)
		framework.ExpectNoError(err, "waiting for server pod to start")

		By("Get hostname by exec cmd")
		out, err := framework.RunKubectl("exec", fmt.Sprintf("--namespace=%s", f.Namespace.Name), pod.GetName(), "hostname")
		framework.Logf("hostname get by container exec is : %s", out)
		Expect(err).NotTo(HaveOccurred(), "get hostname by exec err")
		Expect(pod.GetName()).Should(Equal(strings.Trim(out, "\n")))
	})

	It("Create a pod with hostname template, generate by ip, but it needn't subDomain. Apply to ant need [pouch-only]", func() {
		By("Load a pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "fail to load pod from file")

		By("Update pod annotation")
		hostName := "sigma-slave0001"
		if len(pod.GetAnnotations()) == 0 {
			pod.Annotations = make(map[string]string, 1)
		}
		pod.Annotations[sigmak8sapi.AnnotationPodHostNameTemplate] = hostName

		By("Update container env, assign it to docker vm")
		if len(pod.Spec.Containers[0].Env) == 0 {
			pod.Spec.Containers[0].Env = make([]v1.EnvVar, 0, 1)
		}
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, v1.EnvVar{
			Name:  "ali_run_mode",
			Value: "common_vm",
		})

		By("Create a pod")
		testPod = f.PodClient().Create(pod)
		defer util.DeletePod(f.ClientSet, testPod)

		By("Waiting for pods to come up.")
		err = framework.WaitForPodRunningInNamespace(f.ClientSet, testPod)
		framework.ExpectNoError(err, "waiting for server pod to start")

		By("Get hostname by exec cmd")
		out, err := framework.RunKubectl("exec", fmt.Sprintf("--namespace=%s", f.Namespace.Name), pod.GetName(), "hostname")
		framework.Logf("hostname get by container exec is : %s", out)
		Expect(err).NotTo(HaveOccurred(), "get hostname by exec err")
		Expect(hostName).Should(Equal(strings.Trim(out, "\n")))
	})

	It("[smoke] Create a pod with hostname template, generate by ip, but it need subDomain. Apply to alibaba need [pouch-only]", func() {
		By("Load a pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "fail to load pod from file")

		By("Update pod annotation")
		hostNameTemplate := "sigma-slave{{.IpAddress}}.unsz.su18"
		if len(pod.GetAnnotations()) == 0 {
			pod.Annotations = make(map[string]string, 1)
		}
		pod.Annotations[sigmak8sapi.AnnotationPodHostNameTemplate] = hostNameTemplate

		By("Update container env, assign it to docker vm")
		if len(pod.Spec.Containers[0].Env) == 0 {
			pod.Spec.Containers[0].Env = make([]v1.EnvVar, 0, 1)
		}
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, v1.EnvVar{
			Name:  "ali_run_mode",
			Value: "common_vm",
		})

		By("Create a pod")
		testPod = f.PodClient().Create(pod)
		defer util.DeletePod(f.ClientSet, testPod)

		By("Waiting for pods to come up.")
		err = framework.WaitForPodRunningInNamespace(f.ClientSet, testPod)
		framework.ExpectNoError(err, "waiting for server pod to start")

		By("Query pod info, get pod Ip")
		podWithIP, err := f.PodClient().Get(pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "fail to query pod Ip")
		podIP := podWithIP.Status.PodIP

		By("Generate host name by podIP and hostName template")
		hostname, hostDomain, success, err := kubelet.GeneratePodHostNameAndDomainByHostNameTemplate(pod, podIP)
		Expect(err).NotTo(HaveOccurred(), "fail to generate pod hostName and domain by hostName template")
		Expect(success).Should(BeTrue())
		Expect(hostDomain).NotTo(BeEmpty())
		Expect(hostname).NotTo(BeEmpty())
		hostName := fmt.Sprintf("%s.%s", hostname, hostDomain)

		By("Get hostname by exec cmd")
		out, err := framework.RunKubectl("exec", fmt.Sprintf("--namespace=%s", f.Namespace.Name), pod.GetName(), "hostname")
		framework.Logf("hostname get by container exec is : %s", out)
		Expect(err).NotTo(HaveOccurred(), "get hostname by exec err")
		Expect(hostName).Should(Equal(strings.Trim(out, "\n")))
	})
})
