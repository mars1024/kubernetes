package kubelet

import (
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/json"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-kubelet]", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	caseName := "[set_container_ulimit]"
	It("[sigma-kubelet]"+caseName, func() {
		podFileName := "pod-base.json"
		containerName := "pod-base"
		checkCommand := "ulimit -a | grep open"

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
						Ulimits: []sigmak8sapi.Ulimit{{Name: "nofile", Soft: 2048, Hard: 4196}},
					},
				},
			},
		}
		allocSpecStr, err := json.Marshal(allocSpec)
		Expect(err).NotTo(HaveOccurred())
		pod.Annotations = map[string]string{sigmak8sapi.AnnotationPodAllocSpec: string(allocSpecStr)}

		testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		defer util.DeletePod(f.ClientSet, testPod)

		// Step2: Wait for container's creation finished.
		By(caseName + "wait until pod running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

		// Step3: check ulimits settings
		By(caseName + "change ulimits settings")
		result := f.ExecShellInContainer(testPod.Name, containerName, checkCommand)
		Expect(result).Should(ContainSubstring("2048"))
	})
})
