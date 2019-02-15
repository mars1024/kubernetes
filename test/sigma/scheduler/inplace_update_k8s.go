package scheduler

import (
	"encoding/json"
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
	"k8s.io/kubernetes/test/sigma/swarm"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-3.1][sigma-scheduler][inplace-update][Serial]", func() {
	var cs clientset.Interface
	var nodeList *v1.NodeList
	var systemPodsNo int
	var ns string

	nodeToAllocatableMapCPU := make(map[string]int64)
	nodeToAllocatableMapMem := make(map[string]int64)
	nodeToAllocatableMapEphemeralStorage := make(map[string]int64)

	nodesInfo := make(map[string]*v1.Node)
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

		for i, node := range nodeList.Items {
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

			nodesInfo[node.Name] = &nodeList.Items[i]
			//etcdNodeinfo := swarm.GetNode(node.Name)
			//nodeToAllocatableMapCPU[node.Name] = int64(etcdNodeinfo.LocalInfo.CpuNum * 1000)
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
		if CurrentGinkgoTestDescription().Failed {
			DumpSchedulerState(f, 0)
		}
		DeleteSigmaContainer(f)
	})

	// 验证 Inplace Update 功能在调度器中的支持
	// 前置：集群中单节点上可分配的 CPU/Memory/Disk 资源均大于 0

	// 步骤：
	// 1. 获取一个可调度的节点，记录可分配的 cpu 额度 X，memory 的额度 Y，disk 额度 Z
	// 2. 在 node 上打上一个随机的标签 key=NodeName
	// 3. k8s 创建一个单容器 PodA，NodeAffinity 设置 key=NodeName，
	//    并且 Requests.CPU 为 1/2 * X，Requests.Memory 为 1/2 * Y，Requests.EphemeralStorage 为 1/2 * Z
	// 4. 观察调度结果，并获取当前节点可分配的 cpu、memory、disk 的额度，分别记录为 X1、Y1、Z1
	// 5. k8s 对 PodA 创建 update 请求，资源均变大到 3/4 额度，观察调度结果
	// 6. k8s 对 PodA 创建 update 请求，资源均变大到 4/4 额度，观察调度结果
	// 7. 再缩小到 1/2 额度，观察调度结果
	// 8. 再创建一个 1/2 额度的 PodB，观察调度结果

	// 验证结果：
	// 每一步均调度成功，且 inplace update state 为 accepted
	// TODO(kubo.cph) 下面的剩余资源检查还没做
	// 1. 步骤 4 中，PodA 调度成功，且 Pod.Spec.NodeName = 此 Node，
	//    剩余 cpu 额度 X1 = X - (1/2 * X)，memory 额度 Y1 = Y - (1/2 * Y)，disk 额度 Z1 = Z - (1/2 * Z)
	// 2. 步骤 5 中，PodA 调度成功，且 Pod.Spec.NodeName = 此 Node，
	//    剩余的 cpu 额度 X2 = (1/4 * X)，memory 额度 Y2 = (1/4 * Y)，disk 额度 Z2 = (1/4 * Z)
	// 3. 步骤 6 中，PodA 调度成功，且 Pod.Spec.NodeName = 此 Node，
	//    剩余的 cpu 额度 X3 = (3/4 * X)，memory 额度 Y3 = (3/4 * Y)，disk 额度 Z3 = (3/4 * Z)
	It("[smoke][p0][bvt][ant] scheduler_inplace_update_001 A pod with cpu/mem/ephemeral-storage request should be scheduled on node with enough resource successfully. "+
		"And should be updated successfully.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		// Apply node label to each node
		nodeAffinityKey := "node-for-resource-e2e-test-" + string(uuid.NewUUID())
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		allocatableCPU := nodeToAllocatableMapCPU[nodeName]
		allocatableMemory := nodeToAllocatableMapMem[nodeName]
		allocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		podCPU1 := allocatableCPU * 5 / 10
		podMemory1 := allocatableMemory * 5 / 10
		podDisk1 := allocatableDisk * 5 / 10

		podCPU2 := allocatableCPU * 75 / 100
		podMemory2 := allocatableMemory * 75 / 100
		podDisk2 := allocatableDisk * 75 / 100

		By("Request a pod with CPU/Memory/EphemeralStorage.")

		podsToDelete := []*v1.Pod{}
		resourceList1 := v1.ResourceList{
			v1.ResourceCPU:              *resource.NewMilliQuantity(podCPU1, "DecimalSI"),
			v1.ResourceMemory:           *resource.NewQuantity(podMemory1, "DecimalSI"),
			v1.ResourceEphemeralStorage: *resource.NewQuantity(podDisk1, "DecimalSI"),
		}

		// 50% node resource
		resourceRequirements1 := &v1.ResourceRequirements{
			Limits:   resourceList1,
			Requests: resourceList1,
		}

		resourceList2 := v1.ResourceList{
			v1.ResourceCPU:              *resource.NewMilliQuantity(podCPU2, "DecimalSI"),
			v1.ResourceMemory:           *resource.NewQuantity(podMemory2, "DecimalSI"),
			v1.ResourceEphemeralStorage: *resource.NewQuantity(podDisk2, "DecimalSI"),
		}

		// 75% node resource
		resourceRequirements2 := v1.ResourceRequirements{
			Limits:   resourceList2,
			Requests: resourceList2,
		}

		resourceList3 := v1.ResourceList{
			v1.ResourceCPU:              *resource.NewMilliQuantity(allocatableCPU, "DecimalSI"),
			v1.ResourceMemory:           *resource.NewQuantity(allocatableMemory, "DecimalSI"),
			v1.ResourceEphemeralStorage: *resource.NewQuantity(allocatableDisk, "DecimalSI"),
		}

		// 100% node resource
		resourceRequirements3 := v1.ResourceRequirements{
			Limits:   resourceList3,
			Requests: resourceList3,
		}

		name := "inplace-update-1-" + string(uuid.NewUUID()) + "-1"
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

		doUpdateWithNewResource(cs, pod, resourceRequirements2)
		doUpdateWithNewResource(cs, pod, resourceRequirements3)
		doUpdateWithNewResource(cs, pod, *resourceRequirements1)

		name = "inplace-update-1-" + string(uuid.NewUUID()) + "-2"
		pod2 := createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: resourceRequirements1,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})

		framework.Logf("expect the second pod to be scheduled successfully.")
		err = framework.WaitTimeoutForPodRunningInNamespace(cs, pod2.Name, pod2.Namespace, waitForPodRunningTimeout)
		podsToDelete = append(podsToDelete, pod2)
		Expect(err).NotTo(HaveOccurred())

		for _, pod := range podsToDelete {
			if pod == nil {
				continue
			}
			err := util.DeletePod(f.ClientSet, pod)
			Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
		}
	})
})

func doUpdateWithNewResource(client clientset.Interface, pod *v1.Pod, resource v1.ResourceRequirements) {
	pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	if err != nil {
		Expect(err).NotTo(HaveOccurred(), "get scheduled pod should succeed")
	}

	// increase resources of pod
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState] =
		sigmak8sapi.InplaceUpdateStateCreated
	pod.Spec.Containers[0].Resources = resource

	_, err = client.CoreV1().Pods(pod.Namespace).Update(pod)
	if err != nil {
		Expect(err).NotTo(HaveOccurred(), "update pod should succeed")
	}

	err = wait.PollImmediate(3*time.Second, 5*time.Minute, checkInplaceUpdateIsAccepted(client, pod))
	Expect(err).NotTo(HaveOccurred(), "inplace update should succeed")
}

func checkInplaceUpdateIsAccepted(client clientset.Interface, pod *v1.Pod) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		state, ok := pod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState]
		if !ok {
			return false, nil
		}

		if state != sigmak8sapi.InplaceUpdateStateAccepted &&
			state != sigmak8sapi.InplaceUpdateStateSucceeded {
			framework.Logf("checkInplaceUpdateIsAccepted, state: %s", state)
			return false, nil
		}

		// Get pod and check CPUIDs.
		allocSpecStr := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
		allocSpec := &sigmak8sapi.AllocSpec{}
		err = json.Unmarshal([]byte(allocSpecStr), allocSpec)
		if err != nil {
			return false, err
		}

		CPUIDs := allocSpec.Containers[0].Resource.CPU.CPUSet.CPUIDs
		cpuRequest := pod.Spec.Containers[0].Resources.Requests[v1.ResourceCPU]
		cpuRequestCount := cpuRequest.Value()
		if cpuRequestCount != int64(len(CPUIDs)) {
			framework.Logf("cpuRequestCount[%d] is not equal to len(CPUIDs)[%d]",
				cpuRequestCount, len(CPUIDs))
			return false, nil
		}

		framework.Logf("checkInplaceUpdateIsAccepted, state: %s", state)
		return true, nil
	}
}
