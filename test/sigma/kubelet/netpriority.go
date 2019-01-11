package kubelet

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type netPriorityTestCase struct {
	pod               *v1.Pod
	expectNetPriority int
}

func doNetPriorityTestCase(f *framework.Framework, testCase *netPriorityTestCase) {
	pod := testCase.pod

	// Step1: Create pod
	By("create pod")
	testPod, err := util.CreatePod(f.ClientSet, pod, f.Namespace.Name)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Step3: Get created container
	By("Get created pod")
	getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get pod err")
	fmt.Println("getPod:", getPod)
	hostIP := getPod.Status.HostIP

	// Step4: Get and check netpriority.
	By("Get and check netpriority")
	segs := strings.Split(getPod.Status.ContainerStatuses[0].ContainerID, "//")
	if len(segs) != 2 {
		framework.Failf("failed to get ContainerID from pod: %v", getPod)
	}
	containerID := segs[1]
	format := "{{.Config.NetPriority}}"
	netPriorityStr, err := util.GetContainerInspectField(hostIP, containerID, format)
	Expect(err).NotTo(HaveOccurred(), "failed get netpriority from container inspect")
	netPriorityStr = strings.Replace(netPriorityStr, "\n", "", -1)

	Expect(netPriorityStr).Should(Equal(strconv.Itoa(testCase.expectNetPriority)))
}

var _ = Describe("[sigma-kubelet][netpriority] check netpriority", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	It("[smoke][ant] netpriority defined", func() {
		netPriority := 2
		pod := generateRunningPod()
		pod.Annotations[sigmak8sapi.AnnotationNetPriority] = strconv.Itoa(netPriority)
		testCase := netPriorityTestCase{
			pod:               pod,
			expectNetPriority: netPriority,
		}
		doNetPriorityTestCase(f, &testCase)
	})
})
