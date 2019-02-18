package scheduler

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-3.1][sigma-scheduler][cpu-mode-update][Serial]", func() {
	var cs clientset.Interface
	var nodeList *v1.NodeList
	var systemPodsNo int
	var ns string

	nodeToAllocatableMapCPU := make(map[string]int64)
	nodeToAllocatableMapMem := make(map[string]int64)
	nodeToAllocatableMapEphemeralStorage := make(map[string]int64)

	ignoreLabels := framework.ImagePullerLabels
	f := framework.NewDefaultFramework(CPUSetNameSpace)
	f.AllNodesReadyTimeout = 3 * time.Second

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace.Name
		nodeList = &v1.NodeList{}
		masterNodes, nodeList = getMasterAndWorkerNodesOrDie(cs)

		systemPods, err := framework.GetPodsInNamespace(cs, ns, ignoreLabels)
		Expect(err).NotTo(HaveOccurred())
		systemPodsNo = 0
		for _, pod := range systemPods {
			if !masterNodes.Has(pod.Spec.NodeName) && pod.DeletionTimestamp == nil {
				systemPodsNo++
			}
		}

		err = framework.WaitForPodsRunningReady(cs, metav1.NamespaceSystem, int32(systemPodsNo), 0, framework.PodReadyBeforeTimeout, ignoreLabels)
		Expect(err).NotTo(HaveOccurred())

		err = framework.WaitForPodsSuccess(cs, metav1.NamespaceSystem, framework.ImagePullerLabels, framework.ImagePrePullingTimeout)
		Expect(err).NotTo(HaveOccurred())

		for _, node := range nodeList.Items {
			framework.Logf("logging pods the kubelet thinks is on node %s before test", node.Name)
			framework.PrintAllKubeletPods(cs, node.Name)
			waitNodeResourceReleaseComplete(node.Name)

			framework.Logf("calculate the available resource of node: %s", node.Name)
			nodeReady := false
			for _, condition := range node.Status.Conditions {
				if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
					nodeReady = true
					break
				}
			}
			if !nodeReady {
				continue
			}

			{
				allocatable, found := node.Status.Allocatable[v1.ResourceCPU]
				Expect(found).To(Equal(true))
				nodeToAllocatableMapCPU[node.Name] = allocatable.MilliValue()
			}
			{
				allocatable, found := node.Status.Allocatable[v1.ResourceMemory]
				Expect(found).To(Equal(true))
				nodeToAllocatableMapMem[node.Name] = allocatable.Value()
			}
			{
				allocatable, found := node.Status.Allocatable[v1.ResourceEphemeralStorage]
				Expect(found).To(Equal(true))
				nodeToAllocatableMapEphemeralStorage[node.Name] = allocatable.Value()
			}
		}
		pods, err := cs.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{})
		framework.ExpectNoError(err)
		for _, pod := range pods.Items {
			_, found := nodeToAllocatableMapCPU[pod.Spec.NodeName]
			if found && pod.Status.Phase != v1.PodSucceeded && pod.Status.Phase != v1.PodFailed {
				nodeToAllocatableMapCPU[pod.Spec.NodeName] -= getRequestedCPU(pod)
				nodeToAllocatableMapMem[pod.Spec.NodeName] -= getRequestedMem(pod)
				nodeToAllocatableMapEphemeralStorage[pod.Spec.NodeName] -= getRequestedStorageEphemeralStorage(pod)
			}
		}
	})

	JustAfterEach(func() {
	})

	// verify sigma3 cpu mode switch feature.
	// steps:
	// 0. select a node that will be used to provison pod
	// 1. first create a cpuset pod A
	// 2. change pod A to cpushare, check pod is successfully updated
	// 3. change pod A back to cpuset, check pod is successfully updated
	It("[smoke][p0][bvt][ant] scheduler_cpu_mode_update_001 A pod with cpu/mem/ephemeral-storage request should be scheduled on node with enough resource successfully. "+
		"And should be changed cpu mode successfully.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		// Apply node label to node
		nodeAffinityKey := "node-for-resource-e2e-test-" + string(uuid.NewUUID())
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		// allocatableCPU := nodeToAllocatableMapCPU[nodeName]
		allocatableMemory := nodeToAllocatableMapMem[nodeName]
		allocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		var podCPU1 int64 = 1
		podMemory1 := allocatableMemory * 5 / 10
		podDisk1 := allocatableDisk * 5 / 10

		By("Request a pod with CPU/Memory/EphemeralStorage.")

		podsToDelete := []*v1.Pod{}
		resourceList1 := v1.ResourceList{
			v1.ResourceCPU:              *resource.NewQuantity(podCPU1, "DecimalSI"),
			v1.ResourceMemory:           *resource.NewQuantity(podMemory1, "DecimalSI"),
			v1.ResourceEphemeralStorage: *resource.NewQuantity(podDisk1, "DecimalSI"),
		}

		// 50% node resource
		resourceRequirements1 := &v1.ResourceRequirements{
			Limits:   resourceList1,
			Requests: resourceList1,
		}

		name := "cpu-mode-update-1-" + string(uuid.NewUUID()) + "-1"
		pod := createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: resourceRequirements1,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})

		framework.Logf("Case, expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		podsToDelete = append(podsToDelete, pod)
		Expect(err).NotTo(HaveOccurred())

		// change pod to cpushare mode, this function will block until action succeed or fail.
		updatePodCPUMode(cs, pod, "cpushare")

		// change pod to cpuset mode, this function will block until action succeed or fail
		updatePodCPUMode(cs, pod, "cpuset")

		for _, pod := range podsToDelete {
			if pod == nil {
				continue
			}
			err := util.DeletePod(f.ClientSet, pod)
			Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
		}
	})
})

// updatePodCPUMode changes pod cpu mode from cpushare to cpuset, or otherwise
func updatePodCPUMode(client clientset.Interface, pod *v1.Pod, expectedMode string) {
	pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	if err != nil {
		Expect(err).NotTo(HaveOccurred(), "get scheduled pod should succeed")
	}

	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	allocSpec := &sigmak8sapi.AllocSpec{}
	err = json.Unmarshal([]byte(pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]), allocSpec)
	Expect(err).NotTo(HaveOccurred(), "pod alloc spec is invalid")

	if expectedMode == "cpushare" {
		// switch to cpushare pod
		for index := range allocSpec.Containers {
			allocSpec.Containers[index].Resource.CPU.CPUSet = nil
		}
	} else if expectedMode == "cpuset" {
		// switch to cpushare pod
		for index := range allocSpec.Containers {
			allocSpec.Containers[index].Resource.CPU.CPUSet = &sigmak8sapi.CPUSetSpec{
				SpreadStrategy: sigmak8sapi.SpreadStrategySameCoreFirst,
				CPUIDs:         []int{},
			}
		}
	}

	data, _ := json.Marshal(allocSpec)

	pod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState] = sigmak8sapi.InplaceUpdateStateCreated
	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)

	_, err = client.CoreV1().Pods(pod.Namespace).Update(pod)
	if err != nil {
		Expect(err).NotTo(HaveOccurred(), "update pod should succeed")
	}

	err = wait.PollImmediate(3*time.Second, 1*time.Minute, checkCPUModeUpdateIsAccepted(client, pod, expectedMode))
	Expect(err).NotTo(HaveOccurred(), "pod cpu mode update should succeed")
}

func checkCPUModeUpdateIsAccepted(client clientset.Interface, pod *v1.Pod, expectedMode string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		state, ok := pod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState]
		if !ok {
			return false, nil
		}

		if state == sigmak8sapi.InplaceUpdateStateFailed {
			return false, fmt.Errorf("change pod cpu mode failed")
		}

		if state != sigmak8sapi.InplaceUpdateStateAccepted &&
			state != sigmak8sapi.InplaceUpdateStateSucceeded {
			framework.Logf("checkInplaceUpdateIsAccepted, state: %s", state)
			return false, nil
		}

		// Get pod and check cpu mode
		allocSpecStr := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
		allocSpec := &sigmak8sapi.AllocSpec{}
		err = json.Unmarshal([]byte(allocSpecStr), allocSpec)
		if err != nil {
			return false, err
		}

		for index := range allocSpec.Containers {
			if expectedMode == "cpushare" && allocSpec.Containers[index].Resource.CPU.CPUSet != nil {
				framework.Logf("container %s cpumode is not cpushare", allocSpec.Containers[index].Name)
				return false, nil
			}

			if expectedMode == "cpuset" && allocSpec.Containers[index].Resource.CPU.CPUSet == nil {
				framework.Logf("container %s cpumode is not cpuset", allocSpec.Containers[index].Name)
				return false, nil
			}
		}

		framework.Logf("checkCPUModeUpdateIsAccepted, state: %s", state)
		return state == sigmak8sapi.InplaceUpdateStateSucceeded, nil
	}
}
