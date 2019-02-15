package scheduler

import (
	"math"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8s "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
 	"k8s.io/kubernetes/test/sigma/util"
	testutils "k8s.io/kubernetes/test/utils"
)

var balancePodLabel = map[string]string{"name": "priority-balanced-memory"}

var podRequestedResource = &v1.ResourceRequirements{
	Limits: v1.ResourceList{
		v1.ResourceMemory: resource.MustParse("100Mi"),
		v1.ResourceCPU:    resource.MustParse("100m"),
	},
	Requests: v1.ResourceList{
		v1.ResourceMemory: resource.MustParse("100Mi"),
		v1.ResourceCPU:    resource.MustParse("100m"),
	},
}

// createBalancedPodForNodes creates a pod per node that asks for enough resources to make all nodes have the same mem/cpu usage ratio.
func createBalancedPodForNodes(f *framework.Framework, cs clientset.Interface, ns string, nodes []v1.Node, requestedResource *v1.ResourceRequirements, ratio float64) error {
	// find the max, if the node has the max,use the one, if not,use the ratio parameter
	var maxCPUFraction, maxMemFraction float64 = ratio, ratio
	var cpuFractionMap = make(map[string]float64)
	var memFractionMap = make(map[string]float64)
	for _, node := range nodes {
		cpuFraction, memFraction := computeCPUMemFraction(cs, node, requestedResource)
		cpuFractionMap[node.Name] = cpuFraction
		memFractionMap[node.Name] = memFraction
		if cpuFraction > maxCPUFraction {
			maxCPUFraction = cpuFraction
		}
		if memFraction > maxMemFraction {
			maxMemFraction = memFraction
		}
	}
	// we need the max one to keep the same cpu/mem use rate
	ratio = math.Max(maxCPUFraction, maxMemFraction)
	for _, node := range nodes {
		memAllocatable, found := node.Status.Allocatable[v1.ResourceMemory]
		Expect(found).To(Equal(true))
		memAllocatableVal := memAllocatable.Value()

		cpuAllocatable, found := node.Status.Allocatable[v1.ResourceCPU]
		Expect(found).To(Equal(true))
		cpuAllocatableMil := cpuAllocatable.MilliValue()

		needCreateResource := v1.ResourceList{}
		cpuFraction := cpuFractionMap[node.Name]
		memFraction := memFractionMap[node.Name]
		needCreateResource[v1.ResourceCPU] = *resource.NewMilliQuantity(int64((ratio-cpuFraction)*float64(cpuAllocatableMil)), resource.DecimalSI)

		needCreateResource[v1.ResourceMemory] = *resource.NewQuantity(int64((ratio-memFraction)*float64(memAllocatableVal)), resource.BinarySI)

		err := testutils.StartPods(cs, 1, ns, string(uuid.NewUUID()),
			*initPausePod(f, pausePodConfig{
				Name:   "",
				Labels: balancePodLabel,
				Resources: &v1.ResourceRequirements{
					Limits:   needCreateResource,
					Requests: needCreateResource,
				},
				NodeName: node.Name,
			}), true, framework.Logf)

		if err != nil {
			return err
		}
	}

	for _, node := range nodes {
		By("Compute Cpu, Mem Fraction after create balanced pods.")
		computeCPUMemFraction(cs, node, requestedResource)
	}

	return nil
}

func computeCPUMemFraction(cs clientset.Interface, node v1.Node, resource *v1.ResourceRequirements) (float64, float64) {
	framework.Logf("ComputeCPUMemFraction for node: %v", node.Name)
	totalRequestedCPUResource := resource.Requests.Cpu().MilliValue()
	totalRequestedMemResource := resource.Requests.Memory().Value()
	allpods, err := cs.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		framework.Failf("Expect error of invalid, got : %v", err)
	}
	for _, pod := range allpods.Items {
		if pod.Spec.NodeName == node.Name {
			framework.Logf("Pod for on the node: %v, Cpu: %v, Mem: %v", pod.Name, getRequestedCPU(pod), getRequestedMem(pod))
			totalRequestedCPUResource += getRequestedCPU(pod)
			totalRequestedMemResource += getRequestedMem(pod)
		}
	}
	cpuAllocatable, found := node.Status.Allocatable[v1.ResourceCPU]
	Expect(found).To(Equal(true))
	cpuAllocatableMil := cpuAllocatable.MilliValue()

	cpuFraction := float64(totalRequestedCPUResource) / float64(cpuAllocatableMil)
	memAllocatable, found := node.Status.Allocatable[v1.ResourceMemory]
	Expect(found).To(Equal(true))
	memAllocatableVal := memAllocatable.Value()
	memFraction := float64(totalRequestedMemResource) / float64(memAllocatableVal)

	framework.Logf("Node: %v, totalRequestedCpuResource: %v, cpuAllocatableMil: %v, cpuFraction: %v", node.Name, totalRequestedCPUResource, cpuAllocatableMil, cpuFraction)
	framework.Logf("Node: %v, totalRequestedMemResource: %v, memAllocatableVal: %v, memFraction: %v", node.Name, totalRequestedMemResource, memAllocatableVal, memFraction)

	return cpuFraction, memFraction
}

var _ = Describe("[sigma-3.1][sigma-scheduler][podaffinity][Serial]", func() {
	var cs clientset.Interface
	var ns string
	var nodeList *v1.NodeList

	nodeToAllocatableMapCPU := make(map[string]int64)
	nodeToAllocatableMapMem := make(map[string]int64)
	nodeToAllocatableMapEphemeralStorage := make(map[string]int64)
	nodesInfo := make(map[string]*v1.Node)
	f := framework.NewDefaultFramework(CPUSetNameSpace)

	f.AllNodesReadyTimeout = 3 * time.Second
	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace.Name
		nodeList = &v1.NodeList{}

		masterNodes, nodeList = getMasterAndWorkerNodesOrDie(cs)
		if len(nodeList.Items) == 0 {
			Fail("get no nodes to schedule")
		}
		for i, node := range nodeList.Items {
			waitNodeResourceReleaseComplete(node.Name)
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
	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			DumpSchedulerState(f, 0)
		}
		DeleteSigmaContainer(f)
	})

	// Pod anti-affinity 应用互斥验证
	// 前置：要求节点数至少为 3，否则 skip
	// 步骤：
	// 1. 分配一个 sigma.ali/app-name=APP_A 的 pod1 到 node1（预期成功）
	// 2. 分配一个 sigma.ali/app-name=APP_B 的 pod2 到 node2（预期成功）
	// 3. 均衡节点的 CPU 和 Memory 利用率 （预期成功）
	// 4. 分配一个与 sigma.ali/app-name=APP_A 或 sigma.ali/app-name=APP_B 实例都互斥的 pod3（预期成功，且没有分配到 node1 或者 node2 上）
	It("[p1] pod_affinity_k8s_001 Pod should be scheduled to the node that satisfies the alloc-spec.affinity.podAntiAffinity terms", func() {
		if len(nodeList.Items) < 3 {
			Skip("SKIP: this test needs at least 3 nodes!")
		}

		By("Trying to launch APP_A pod1 on node1")
		nodeIP1 := nodeList.Items[0].Status.Addresses[0].Address
		pod1 := runPausePod(f, pausePodConfig{
			Name:     "scheduler-e2e-" + string(uuid.NewUUID()),
			Labels:   map[string]string{sigmak8s.LabelAppName: "APP_A"},
			Affinity: util.GetAffinityNodeSelectorRequirement(sigmak8s.LabelNodeIP, []string{nodeIP1}),
		})

		By("Trying to launch APP_B pod2 on node2")
		nodeIP2 := nodeList.Items[1].Status.Addresses[0].Address
		pod2 := runPausePod(f, pausePodConfig{
			Name:     "scheduler-e2e-" + string(uuid.NewUUID()),
			Labels:   map[string]string{sigmak8s.LabelAppName: "APP_B"},
			Affinity: util.GetAffinityNodeSelectorRequirement(sigmak8s.LabelNodeIP, []string{nodeIP2}),
		})

		By("Trying to balance node cpu, mem usage.")
		// make the nodes have balanced cpu, mem usage
		err := createBalancedPodForNodes(f, cs, ns, nodeList.Items, podRequestedResource, 0.6)
		framework.ExpectNoError(err)

		By("Trying to launch pod3 and not with APP_A or APP_B pod")
		pod3 := createPausePod(f, pausePodConfig{
			Resources: podRequestedResource,
			Name:      "scheduler-e2e-" + string(uuid.NewUUID()),
			Annotations: map[string]string{
				sigmak8s.AnnotationPodAllocSpec: allocSpecStrWithConstraints([]constraint{
					{
						key:      sigmak8s.LabelAppName,
						op:       metav1.LabelSelectorOpIn,
						value:    "APP_A",
						maxCount: 1,
					},
					{
						key:      sigmak8s.LabelAppName,
						op:       metav1.LabelSelectorOpIn,
						value:    "APP_B",
						maxCount: 1,
					},
				}),
			},
		})
		By("Wait the pod3 becomes running")
		framework.ExpectNoError(f.WaitForPodRunning(pod3.Name))
		pod3, err = cs.CoreV1().Pods(ns).Get(pod3.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)
		By("Verify the pod3 was not scheduled with pod1 or pod2")
		Expect(pod3.Spec.NodeName).NotTo(Equal(pod1.Spec.NodeName))
		Expect(pod3.Spec.NodeName).NotTo(Equal(pod2.Spec.NodeName))
	})

	// Pod anti-affinity 应用互斥(LabelSelectorOperator=In&maxCount=1)和独占(LabelSelectorOperator=NotIn)验证 验证AppName
	// 步骤：
	// 1. 在每个node节点分配一个sigma.ali/app-name=APP_A的Pod1（预期成功）
	// 2. 分配一个sigma.ali/app-name=APP_A且要求sigma.ali/app-name=APP_A的maxCount=1的Pod2（预期失败）
	// 3. 分配一个sigma.ali/app-name=APP_A且要求sigma.ali/app-name=APP_A的maxCount=2的Pod3(非对称检查)（预期成功）
	// 4. 分配一个sigma.ali/app-name=APP_B且要求sigma.ali/app-name=APP_B独占的Pod4（预期失败）
	It("[smoke][p0][bvt] pod_affinity_k8s_002: Pod should not be schedule to the node that satisfies PodAntiAffinity In and NotIn operators", func() {
		nodeName := nodeList.Items[0].Name
		By("Trying to launch a APP_A pod1 on each node.")

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		requestedCPU := AllocatableCPU / 8       //保证一定能扩容出来
		requestedMemory := AllocatableMemory / 8 //保证一定能扩容出来
		requestedDisk := AllocatableDisk / 8     //保证一定能扩容出来

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				podConstraints: []constraint{
					{sigmak8s.LabelAppName, metav1.LabelSelectorOpIn, "APP_A", 1},
				},
				labels:         map[string]string{sigmak8s.LabelAppName: "APP_A"},
				affinityConfig: map[string][]string{sigmak8s.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
				podConstraints: []constraint{
					{sigmak8s.LabelAppName, metav1.LabelSelectorOpIn, "APP_A", 1},
				},
				labels:         map[string]string{sigmak8s.LabelAppName: "APP_A"},
				affinityConfig: map[string][]string{sigmak8s.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				podConstraints: []constraint{
					{sigmak8s.LabelAppName, metav1.LabelSelectorOpIn, "APP_A", 2},
				},
				labels:         map[string]string{sigmak8s.LabelAppName: "APP_A"},
				affinityConfig: map[string][]string{sigmak8s.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
				labels:          map[string]string{sigmak8s.LabelAppName: "APP_B"},
				podConstraints: []constraint{
					{key: sigmak8s.LabelAppName, op: metav1.LabelSelectorOpNotIn, value: "APP_B"},
				},
				affinityConfig: map[string][]string{sigmak8s.LabelNodeIP: {nodeIP}},
			},
		}
		testContext := &testContext{
			caseName:  "pod_affinity_k8s_002",
			cs:        cs,
			localInfo: nil,
			f:         f,
			testCases: tests,
			nodeName:  nodeName,
		}
		// 清理创建出来的POD数据
		testContext.execTests()
	})

	// Pod anti-affinity 打散（maxCount=2）验证 验证deployUnit
	// 步骤：
	// 1. 获取一个可分配容器的节点的Name（预期成功）
	// 2. 在步骤1获取的节点上连续分配两个sigma.ali/deploy-unit=DU_1且要求sigma.ali/deploy-unit=DU_1的maxCount=2的Pod（预期成功）
	// 3. 在步骤1获取的节点上再次分配第三个sigma.ali/deploy-unit=DU_1且要求sigma.ali/deploy-unit=DU_1的maxCount=2的Pod（预期失败）
	// 4. 删除请求的第三个pod （预期成功）
	// 5. 删除请求的第二个pod （预期成功）
	// 6. 在步骤1获取的节点上再次分配第四个sigma.ali/deploy-unit=DU_1且要求sigma.ali/deploy-unit=DU_1的maxCount=2的Pod （预期成功）
	It("pod_affinity_k8s_003: Pod should be scheduled successfully if it satisfy the maxcount constraints and pod should fail if it exceeds maxcount constraints", func() {
		nodeName := nodeList.Items[0].Name
		By("Trying to launch a APP_A pod1 on each node.")

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		requestedCPU := AllocatableCPU / 8       //保证一定能扩容出来
		requestedMemory := AllocatableMemory / 8 //保证一定能扩容出来
		requestedDisk := AllocatableDisk / 8     //保证一定能扩容出来

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				podConstraints: []constraint{
					{sigmak8s.LabelDeployUnit, metav1.LabelSelectorOpIn, "DU_1", 2},
				},
				labels:         map[string]string{sigmak8s.LabelDeployUnit: "DU_1"},
				affinityConfig: map[string][]string{sigmak8s.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				podConstraints: []constraint{
					{sigmak8s.LabelDeployUnit, metav1.LabelSelectorOpIn, "DU_1", 2},
				},
				labels:         map[string]string{sigmak8s.LabelDeployUnit: "DU_1"},
				affinityConfig: map[string][]string{sigmak8s.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
				podConstraints: []constraint{
					{sigmak8s.LabelDeployUnit, metav1.LabelSelectorOpIn, "DU_1", 2},
				},
				labels:         map[string]string{sigmak8s.LabelDeployUnit: "DU_1"},
				affinityConfig: map[string][]string{sigmak8s.LabelNodeIP: {nodeIP}},
			},
			{
				requestType:  cleanResource,
				cleanIndexes: []int{1, 2},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				podConstraints: []constraint{
					{sigmak8s.LabelDeployUnit, metav1.LabelSelectorOpIn, "DU_1", 2},
				},
				labels:         map[string]string{sigmak8s.LabelDeployUnit: "DU_1"},
				affinityConfig: map[string][]string{sigmak8s.LabelNodeIP: {nodeIP}},
			},
		}
		testContext := &testContext{
			caseName:  "pod_affinity_k8s_003",
			cs:        cs,
			localInfo: nil,
			f:         f,
			testCases: tests,
		}

		testContext.execTests()
	})
})
