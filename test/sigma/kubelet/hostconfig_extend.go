package kubelet

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type hostConfigTestCase struct {
	pod        *v1.Pod
	hostConfig *sigmak8sapi.HostConfigInfo
}

func checkExtendConfigFields(pod *v1.Pod, containerID string, testCase *hostConfigTestCase) {
	hostIP := pod.Status.HostIP
	format := "{{.HostConfig.MemorySwappiness}}/{{.HostConfig.MemorySwap}}/{{.HostConfig.CPUBvtWarpNs}}"
	hostConfigStr, err := util.GetContainerInspectField(hostIP, containerID, format)
	Expect(err).NotTo(HaveOccurred(), "failed get host config from container inspect")
	hostConfigStr = strings.Replace(hostConfigStr, "\n", "", -1)
	// items: MemorySwappiness, MemorySwap, CPUBvtWarpNs
	items := strings.Split(hostConfigStr, "/")
	Expect(len(items)).Should(Equal(3))
	framework.Logf("check MemorySwappiness")
	Expect(items[0]).Should(Equal(strconv.Itoa(int(testCase.hostConfig.MemorySwappiness))))
	framework.Logf("check MemorySwap")
	Expect(items[1]).Should(Equal(strconv.Itoa(int(testCase.hostConfig.MemorySwap))))
	framework.Logf("check CPUBvtWarpNs")
	Expect(items[2]).Should(Equal(strconv.Itoa(int(testCase.hostConfig.CPUBvtWarpNs))))
}

func doHostConfigTestCase(f *framework.Framework, testCase *hostConfigTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name
	upgradeSuccessStr := "upgrade container success"

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

	segs := strings.Split(getPod.Status.ContainerStatuses[0].ContainerID, "//")
	if len(segs) != 2 {
		framework.Failf("failed to get ContainerID from pod: %v", pod)
	}
	containerID := segs[1]

	// Step4: Get and check extended hostconfig fields.
	By("Get and check extended hostconfig fields.")
	checkExtendConfigFields(getPod, containerID, testCase)

	// Stpe5: Upgrade pod
	By("change container's field")
	patchData := `{"spec":{"containers":[{"name":"pod-base","image":"reg.docker.alibaba-inc.com/sigma-x/mysql:test-v2"}]}}`
	upgradedPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(patchData))
	Expect(err).NotTo(HaveOccurred(), "patch pod err")

	// Step6: Wait for upgrade action finished.
	By("wait until pod is upgraded")
	err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 3*time.Minute, upgradeSuccessStr, true)
	Expect(err).NotTo(HaveOccurred(), "upgrade pod err")
	// Wait for new status
	time.Sleep(time.Second * 20)

	// Step7: Get created container
	By("Get upgraded pod")
	getPod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get pod err")

	// Step8: Check containerID changed
	segs = strings.Split(getPod.Status.ContainerStatuses[0].ContainerID, "//")
	if len(segs) != 2 {
		framework.Failf("failed to get ContainerID from pod: %v", pod)
	}
	upgradedContainerID := segs[1]

	Expect(containerID).ShouldNot(Equal(upgradedContainerID))

	// Step9: Get and check extended hostconfig fields.
	By("Get and check extended hostconfig fields after upgrade.")
	checkExtendConfigFields(getPod, upgradedContainerID, testCase)
}

var _ = Describe("[sigma-kubelet][alidocker-hostconfig] check AliDocker's extended hostconfig fields", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	It("[smoke][ant] extended hostconfig fields are defined", func() {
		containerName := "pod-base"
		pod := generateRunningPod()

		// Set alloc spec annotation
		hostConfig := sigmak8sapi.HostConfigInfo{
			MemorySwap:       2048000000,
			MemorySwappiness: 20,
			PidsLimit:        1000,
			CPUBvtWarpNs:     2,
		}

		allocSpec := &sigmak8sapi.AllocSpec{
			Containers: []sigmak8sapi.Container{
				sigmak8sapi.Container{
					Name:       containerName,
					HostConfig: hostConfig,
				},
			},
		}

		allocSpecBytes, err := json.Marshal(allocSpec)
		Expect(err).NotTo(HaveOccurred(), "failed to marshal allocSpec")
		pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(allocSpecBytes)

		// Set resources
		container := &pod.Spec.Containers[0]
		container.Resources = v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			Limits: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
		}

		testCase := hostConfigTestCase{
			pod:        pod,
			hostConfig: &hostConfig,
		}
		doHostConfigTestCase(f, &testCase)
	})
})
