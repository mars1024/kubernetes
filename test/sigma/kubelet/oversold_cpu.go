package kubelet

import (
	"encoding/json"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubelet/sysctl"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-kubelet][Serial] oversold cpu", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	var testPod *v1.Pod
	var nodeIP string // testPod 所在的宿主机的IP
	BeforeEach(func() {
		By("Load a pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "fail to load pod from file")

		By("update pod cpuID")
		if len(pod.Annotations) == 0 {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = getAllocSpec()
		addResources(pod)

		By("Create a pod")
		testPod = f.PodClient().Create(pod)

		By("Waiting for pods to come up.")
		err = framework.WaitForPodRunningInNamespace(f.ClientSet, testPod)

		By("Get node IP")
		getPod, err := f.PodClient().Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		nodeIP = getPod.Status.HostIP
	})

	It("[smoke] create pod with cpu over sold ,should not be admitted", func() {
		By("add cpu over quota")
		podFromAPIServer, err := f.PodClient().Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		nodeName := podFromAPIServer.Spec.NodeName
		framework.AddOrUpdateLabelOnNode(f.ClientSet, nodeName, sigmak8sapi.LabelCPUOverQuota, "1.0")
		defer framework.RemoveLabelOffNode(f.ClientSet, nodeName, sigmak8sapi.LabelCPUOverQuota)

		By("Load another pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		podAnother, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "fail to load pod from file")
		podAnother.Name = "another-pod"

		By("update pod cpuID")
		if len(podAnother.Annotations) == 0 {
			podAnother.Annotations = make(map[string]string)
		}
		// Use pod-base's allocSpec and specify another-pod's NodeName to avoid schedule.
		allocSpecStr, _ := podFromAPIServer.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
		podAnother.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = allocSpecStr
		// Admission will deny the pod if resource doesn't match cpuid.
		addResources(podAnother)
		// If we don't specify NodeName, scheduler will modify cpuids.
		podAnother.Spec.NodeName = nodeName

		By("update pod NodeAffinity")
		addNodeAffinityFromNodeIP(podAnother, nodeIP)

		By("Create another pod")
		testPod2 := f.PodClient().Create(podAnother)
		defer util.DeletePod(f.ClientSet, testPod2)

		By("Waiting for pods not to come up.")
		err = framework.WaitForPodNoLongerRunningInNamespace(f.ClientSet, testPod2.Name, testPod2.Namespace)
		Expect(err).NotTo(HaveOccurred())

		By("check failed reason")
		podFromAPIServer, err = f.PodClient().Get(testPod2.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(podFromAPIServer.Status.Reason).To(Equal(sysctl.PodForbiddenReason))
		Expect(podFromAPIServer.Status.Phase).To(Equal(v1.PodFailed))
	})

	It("cpu over quota bigger than 1.0, should be admitted", func() {
		By("add cpu over quota")
		podFromAPIServer, err := f.PodClient().Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		nodeName := podFromAPIServer.Spec.NodeName
		framework.Logf("node name is %s", nodeName)
		framework.AddOrUpdateLabelOnNode(f.ClientSet, nodeName, sigmak8sapi.LabelCPUOverQuota, "1.5")
		defer framework.RemoveLabelOffNode(f.ClientSet, nodeName, sigmak8sapi.LabelCPUOverQuota)

		By("Load another pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		podAnother, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "fail to load pod from file")
		podAnother.Name = "another-pod"

		By("update pod cpuID")
		if len(podAnother.Annotations) == 0 {
			podAnother.Annotations = make(map[string]string)
		}
		// Use pod-base's allocSpec and specify another-pod's NodeName to avoid schedule.
		allocSpecStr, _ := podFromAPIServer.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
		podAnother.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = allocSpecStr
		addResources(podAnother)
		podAnother.Spec.NodeName = nodeName

		By("update pod NodeAffinity")
		addNodeAffinityFromNodeIP(podAnother, nodeIP)

		By("Create a pod")
		testPod2 := f.PodClient().Create(podAnother)
		defer util.DeletePod(f.ClientSet, testPod2)

		By("Waiting for pods to come up.")
		err = framework.WaitForPodRunningInNamespace(f.ClientSet, testPod2)
		Expect(err).NotTo(HaveOccurred())
	})
})

func getAllocSpec() string {
	allocSpec := sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: "pod-base",
				Resource: sigmak8sapi.ResourceRequirements{
					CPU: sigmak8sapi.CPUSpec{
						CPUSet: &sigmak8sapi.CPUSetSpec{
							SpreadStrategy: sigmak8sapi.SpreadStrategySameCoreFirst,
						},
					},
				},
			},
		},
	}
	allocSpecByte, _ := json.Marshal(allocSpec)
	return string(allocSpecByte)
}

func addNodeAffinityFromNodeIP(pod *v1.Pod, nodeIP string) {
	pod.Spec.Affinity = &v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: []v1.NodeSelectorTerm{
					{
						MatchExpressions: []v1.NodeSelectorRequirement{
							{
								Key:      sigmak8sapi.LabelNodeIP,
								Operator: v1.NodeSelectorOpIn,
								Values:   []string{nodeIP},
							},
						},
					},
				},
			},
		},
	}
}

func addResources(pod *v1.Pod) {
	resources := v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:              *resource.NewMilliQuantity(3000, resource.DecimalSI),
			v1.ResourceMemory:           resource.MustParse("512Mi"),
			v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU:              *resource.NewMilliQuantity(3000, resource.DecimalSI),
			v1.ResourceMemory:           resource.MustParse("512Mi"),
			v1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
		},
	}
	pod.Spec.Containers[0].Resources = resources
}
