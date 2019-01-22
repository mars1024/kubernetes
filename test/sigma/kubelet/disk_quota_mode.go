package kubelet

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type diskQuotaModeTestCase struct {
	pod                   *v1.Pod
	expectDockerDiskQuota string
	expectPouchDiskQuota  string
}

func getDiskQuota(pod *v1.Pod) string {
	if pod.Status.HostIP == "" {
		framework.Logf("can't get HostIP from pod: %v", pod)
		return ""
	}
	if len(pod.Status.ContainerStatuses) == 0 {
		framework.Logf("Failed to get ContainerStatuses from pod: %v", pod)
		return ""
	}
	segs := strings.Split(pod.Status.ContainerStatuses[0].ContainerID, "//")
	if len(segs) != 2 {
		framework.Logf("Failed to get ContainerID from pod: %v", pod)
		return ""
	}
	containerID := segs[1]

	runtimeType, _ := util.GetContainerDType(pod.Status.HostIP)
	format := ""
	switch runtimeType {
	case util.ContainerdTypePouch:
		format = "{{.Config.DiskQuota}}"
	case util.ContainerdTypeDocker:
		format = "{{.Config.Labels.DiskQuota}}"
	}
	// Get DiskQuota.
	DiskQuota, err := util.GetContainerInspectField(pod.Status.HostIP, containerID, format)
	if err != nil {
		framework.Logf("Failed to get quotaID from pod: %s", pod.Status.HostIP)
		return ""
	}

	return DiskQuota
}

func doDiskQuotaModeTestCase(f *framework.Framework, testCase *diskQuotaModeTestCase) {
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

	// Step3: Check created container.
	By("get created pod")
	getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get pod err")

	// Step4: Get DiskQuota.
	By("get created pod")
	diskQuota := getDiskQuota(getPod)

	// Step5: Check DiskQuota.
	By("check created pod")
	runtimeType, err := util.GetContainerDType(getPod.Status.HostIP)
	Expect(err).NotTo(HaveOccurred(), "get runtime type err")
	switch runtimeType {
	case util.ContainerdTypeDocker:
		checkResult(checkMethodContain, diskQuota, []string{testCase.expectDockerDiskQuota})
	case util.ContainerdTypePouch:
		checkResult(checkMethodContain, diskQuota, []string{testCase.expectPouchDiskQuota})
	}
}

var _ = Describe("[sigma-kubelet][disk-quota-mode] check disk quota", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	It("[smoke] check '.*' mode", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
			Limits: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = fmt.Sprintf(`{"containers":[{"name":"%s","hostConfig":{"diskQuotaMode":".*"}}]}`, pod.Spec.Containers[0].Name)
		testCase := diskQuotaModeTestCase{
			pod: pod,
			expectDockerDiskQuota: "2g",
			expectPouchDiskQuota:  ".*:2g",
		}
		doDiskQuotaModeTestCase(f, &testCase)
	})
	It("[smoke] check '/' mode", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
			Limits: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = fmt.Sprintf(`{"containers":[{"name":"%s","hostConfig":{"diskQuotaMode":"/"}}]}`, pod.Spec.Containers[0].Name)
		testCase := diskQuotaModeTestCase{
			pod: pod,
			expectDockerDiskQuota: "/=2g",
			expectPouchDiskQuota:  "/:2g",
		}
		doDiskQuotaModeTestCase(f, &testCase)
	})
	It("[smoke] check default mode", func() {
		pod := generateRunningPod()
		resources := v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
			Limits: v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		}
		pod.Spec.Containers[0].Resources = resources
		testCase := diskQuotaModeTestCase{
			pod: pod,
			expectDockerDiskQuota: "2g",
			expectPouchDiskQuota:  ".*:2g",
		}
		doDiskQuotaModeTestCase(f, &testCase)
	})
})
