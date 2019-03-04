package kubelet

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type cgroupRWTestCase struct {
	pod *v1.Pod
}

func doCgroupRWTestCase(f *framework.Framework, testCase *cgroupRWTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name

	// Step1: Create pod
	By("create pod")
	testPod, err := util.CreatePod(f.ClientSet, pod, f.Namespace.Name)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Step3: Check Cgroup by /proc/mounts.
	By("check cgroup")
	checkCommand := `cat /proc/mounts | grep "cgroup ro" | wc -l`
	result := f.ExecShellInContainer(testPod.Name, containerName, checkCommand)
	result = strings.Replace(result, "\n", "", -1)

	// There are 0 "cgroup ro" in /proc/mounts.
	// tmpfs /sys/fs/cgroup tmpfs rw,nosuid,nodev,noexec,relatime,mode=755 0 0
	// cgroup /sys/fs/cgroup/cpuset,cpu,cpuacct cgroup rw,nosuid,nodev,noexec,relatime,cpuacct,cpu,cpuset 0 0
	// cgroup /sys/fs/cgroup/net_cls cgroup rw,nosuid,nodev,noexec,relatime,net_cls 0 0
	// cgroup /sys/fs/cgroup/memory cgroup rw,nosuid,nodev,noexec,relatime,memory 0 0
	// cgroup /sys/fs/cgroup/freezer cgroup rw,nosuid,nodev,noexec,relatime,freezer 0 0
	// cgroup /sys/fs/cgroup/perf_event cgroup rw,nosuid,nodev,noexec,relatime,perf_event 0 0
	// cgroup /sys/fs/cgroup/devices cgroup rw,nosuid,nodev,noexec,relatime,devices 0 0
	// cgroup /sys/fs/cgroup/blkio cgroup rw,nosuid,nodev,noexec,relatime,blkio 0 0
	// cgroup /sys/fs/cgroup/hugetlb cgroup rw,nosuid,nodev,noexec,relatime,hugetlb 0 0
	expectResult := "0"
	Expect(result).Should(Equal(expectResult))
}

var _ = Describe("[sigma-kubelet][cgroup-rw] check cgroup RW if privileged is false", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	It("[smoke] we can write cgroup if privileged is false", func() {
		privigedFalse := false
		pod := generateRunningPod()
		pod.Spec.Containers[0].SecurityContext = &v1.SecurityContext{Privileged: &privigedFalse}

		testCase := cgroupRWTestCase{
			pod: pod,
		}
		doCgroupRWTestCase(f, &testCase)
	})

	It("we can write cgroup if privileged is nil", func() {
		pod := generateRunningPod()
		pod.Spec.Containers[0].SecurityContext = &v1.SecurityContext{}

		testCase := cgroupRWTestCase{
			pod: pod,
		}
		doCgroupRWTestCase(f, &testCase)
	})

	It("we can write cgroup if securityContext is nil", func() {
		pod := generateRunningPod()

		testCase := cgroupRWTestCase{
			pod: pod,
		}
		doCgroupRWTestCase(f, &testCase)
	})

	It("we can write cgroup if privileged is true", func() {
		privigedTrue := true
		pod := generateRunningPod()
		pod.Spec.Containers[0].SecurityContext = &v1.SecurityContext{Privileged: &privigedTrue}

		testCase := cgroupRWTestCase{
			pod: pod,
		}
		doCgroupRWTestCase(f, &testCase)
	})
})
