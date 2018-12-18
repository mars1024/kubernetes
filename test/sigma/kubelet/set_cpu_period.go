package kubelet

import (
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-kubelet]", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	caseName := "[set_cpu_period]"
	It("[sigma-kubelet]"+caseName, func() {
		podFileName := "pod-base.json"
		containerName := "pod-base"
		cpuPeriod := 150 * 1000

		// Step1: Create a pod.
		By(caseName + "create a pod from file")
		podFile := filepath.Join(util.TestDataDir, podFileName)
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred())

		// add alloc spec annotation
		allocSpec := sigmak8sapi.AllocSpec{
			Containers: []sigmak8sapi.Container{
				{
					Name: containerName,
					HostConfig: sigmak8sapi.HostConfigInfo{
						CpuPeriod: int64(cpuPeriod),
					},
				},
			},
		}
		allocSpecStr, err := json.Marshal(allocSpec)
		Expect(err).NotTo(HaveOccurred())
		pod.Annotations = map[string]string{sigmak8sapi.AnnotationPodAllocSpec: string(allocSpecStr)}
		pod.Spec.Containers[0].Resources = getResourceRequirements(getResourceList("500m", "128Mi"), getResourceList("500m", "128Mi"))

		testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		defer util.DeletePod(f.ClientSet, testPod)

		// Step2: Wait for container's creation finished.
		By(caseName + "wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(getPod.Status.HostIP).NotTo(BeEmpty(), "status.HostIP should not be empty")
		Expect(getPod.Status.PodIP).NotTo(BeEmpty(), "status.PodIP should not be empty")

		// Step3: check ulimits settings
		By(caseName + "check cpu period")
		// log into slave node and check container cpu period
		realCpuPeriod := getCPUPeriod(getPod)
		framework.Logf("realCpuPeriod: %d", realCpuPeriod)
		Expect(realCpuPeriod).Should(Equal(int64(cpuPeriod)))
	})
})
