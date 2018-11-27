package kubelet

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type CPUSetTestCase struct {
	description string
	pod         *v1.Pod
	// "cpuset", "share", "allcpus"
	cpusetType string
}

func generatePodSharePool() *v1.Pod {
	pod := generatePodCommon()
	return pod
}

func generatePodCPUSet() *v1.Pod {
	pod := generatePodCommon()
	container := &pod.Spec.Containers[0]
	container.Resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
			v1.ResourceMemory: resource.MustParse("100Mi"),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
			v1.ResourceMemory: resource.MustParse("100Mi"),
		},
	}
	pod.Annotations = map[string]string{}
	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = "{\"containers\":[{\"name\":\"pod-base\",\"resource\":{\"cpu\":{\"cpuSet\":{\"spreadStrategy\":\"sameCoreFirst\",\"cpuIDs\":[]}}}}]}"
	return pod
}

func generatePodAllCPUs() *v1.Pod {
	pod := generatePodCommon()
	container := &pod.Spec.Containers[0]
	container.Resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    *resource.NewMilliQuantity(0, resource.DecimalSI),
			v1.ResourceMemory: *resource.NewQuantity(0, resource.BinarySI),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU:    *resource.NewMilliQuantity(2000, resource.DecimalSI),
			v1.ResourceMemory: resource.MustParse("100Mi"),
		},
	}
	pod.Annotations = map[string]string{}
	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = "{\"containers\":[{\"name\":\"pod-base\",\"resource\":{\"cpu\":{\"BindingStrategy\":\"BindAllCPUs\",\"cpuSet\":{\"spreadStrategy\":\"sameCoreFirst\",\"cpuIDs\":[]}}}}]}"
	return pod
}

func doCPUSetTestCase(f *framework.Framework, testCase *CPUSetTestCase) {
	framework.Logf("Start to test case %q", testCase.description)
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name

	// Step1: Create pod
	testPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Step3: Get the running pod.
	By("get created pod")
	getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	framework.Logf("getPod: %v", getPod.Annotations)
	Expect(err).NotTo(HaveOccurred(), "get pod err")

	// Step4: Get expect cpuset
	nodeName := getPod.Spec.NodeName
	node, err := f.ClientSet.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get node err")

	var expectCPUSet cpuset.CPUSet
	switch testCase.cpusetType {
	case "share":
		cpus, exists := util.GetDefaultCPUSetFromNodeAnnotation(node)
		if !exists {
			framework.Failf("Failed to get default cpuset from node annotation")
		}
		expectCPUSet = cpus
	case "cpuset":
		cpus, exists := util.GetCPUsFromPodAnnotation(getPod, containerName)
		if !exists {
			framework.Failf("Failed to get cpuset from pod annotation")
		}
		expectCPUSet = cpus
	case "allcpus":
		cpus, exists := util.GetNodeAllCPUs(node)
		if !exists {
			framework.Failf("Failed to get node all cpusfrom node annotation")
		}
		expectCPUSet = cpus
	}

	// Step5: Get actual cpuset from container.
	// Kubelet will reset all container cpus every 10s
	checkSuccess := false
	actualCPUSet := cpuset.NewCPUSet()
	for i := 0; i < 3; i++ {
		actualCPUSet = util.GetContainerCPUSet(f, getPod, containerName)
		if actualCPUSet.Equals(expectCPUSet) {
			checkSuccess = true
			break
		}
		time.Sleep(10 * time.Second)
	}
	if !checkSuccess {
		framework.Failf("expectCPUSet: %v, but get actualCPUSet: %v", expectCPUSet, actualCPUSet)
	}

}

var _ = Describe("[sigma-kubelet][cpuset] cpuset", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	It("[smoke][Serial] cpuset: share pool test", func() {
		testCase := &CPUSetTestCase{
			description: "share pool test",
			pod:         generatePodSharePool(),
			cpusetType:  "share",
		}
		doCPUSetTestCase(f, testCase)
	})
	It("[smoke] cpuset: cpuset test", func() {
		testCase := &CPUSetTestCase{
			description: "cpuset test",
			pod:         generatePodCPUSet(),
			cpusetType:  "cpuset",
		}
		doCPUSetTestCase(f, testCase)
	})
	// allcpus bindingStrategy will bind cotnainer to all cpus.
	It("[smoke] cpuset: allcpus bindingstrategy test", func() {
		testCase := &CPUSetTestCase{
			description: "cpubindingstrategy allcpus",
			pod:         generatePodAllCPUs(),
			cpusetType:  "allcpus",
		}
		doCPUSetTestCase(f, testCase)
	})
})
