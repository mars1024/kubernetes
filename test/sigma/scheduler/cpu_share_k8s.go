package scheduler

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/swarm"
)

var _ = Describe("[sigma-3.1][sigma-scheduler][cpushare][cpu]", func() {
	var cs clientset.Interface
	var nodeList *v1.NodeList

	nodeToAllocatableMapCPU := make(map[string]int64)
	nodeToAllocatableMapMem := make(map[string]int64)
	nodeToAllocatableMapEphemeralStorage := make(map[string]int64)
	nodesInfo := make(map[string]*v1.Node)
	f := framework.NewDefaultFramework(CPUSetNameSpace)
	f.AllNodesReadyTimeout = 3 * time.Second
	BeforeEach(func() {
		cs = f.ClientSet
		nodeList = &v1.NodeList{}

		masterNodes, nodeList = getMasterAndWorkerNodesOrDie(cs)

		for i, node := range nodeList.Items {
			waitNodeResourceReleaseComplete(node.Name)
			nodesInfo[node.Name] = &nodeList.Items[i]
			etcdNodeinfo := swarm.GetNode(node.Name)
			nodeToAllocatableMapCPU[node.Name] = int64(etcdNodeinfo.LocalInfo.CpuNum * 1000)
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
		// delete sigma2.0 allocplan if exists
		DeleteSigmaContainer(f)
	})

	// 非超卖场景下的SameCoreFirst，验证分配的物理核优先，sigma和k8s互相感知
	// 步骤 要求每个容器分配的cpu个数不能低于2个，否则这个case会验证失败
	// 1.  1/4 整机核 cpushare（预期成功）
	// 2.  1/4 整机核 cpuset（预期成功）
	// 3.  1/4 整机核 cpushare（预期成功）
	// 4.  1/4 整机核 cpuset（预期成功）

	// 验证结果
	//1. node节点的sharePool的值 = 整机cpu - cpuset容器的cpu
	//2. cpuset的cpu，容器内不重叠
	//3. cpuset的cpu，整机不重叠
	It("cpushare001", func() {
		framework.WaitForStableCluster(cs, masterNodes)

		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)
		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		requestedCPU := AllocatableCPU / 4
		requestedMemory := AllocatableMemory / 8 //保证一定能扩容出来
		requestedDisk := AllocatableDisk / 8     //保证一定能扩容出来

		// get nodeIP by node name
		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[sigmak8sapi.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &sigmak8sapi.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
		}
		testContext := &testContext{
			caseName:  "cpushare001",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
			testCases: tests,
			nodeName:  nodeName,
		}
		testContext.execTests(
			checkContainerCPUIDNotDuplicated,
			checkHostCPUIdNotDuplicated,
			checkSharePool,
			checkContainerShareCPUShouldBeNil,
		)
	})

	// 非超卖 k8s 单链路 cpushare 测试，检查 sharepool 的状态
	It("[p1] cpushareK8s.", func() {
		framework.WaitForStableCluster(cs, masterNodes)

		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		By(fmt.Sprintf("apply a label on the found node %s", nodeName))
		allocatableCPU := nodeToAllocatableMapCPU[nodeName]
		allocatableMemory := nodeToAllocatableMapMem[nodeName]
		allocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]
		framework.Logf("allocatableCpu: %d", allocatableCPU)
		framework.Logf("allocatableMemory: %d", allocatableMemory)
		framework.Logf("allocatableDisk: %d", allocatableDisk)

		pod1CPU := allocatableCPU / 2
		pod1Memory := allocatableMemory / 2
		pod1Disk := allocatableDisk / 2
		pod2CPU := allocatableCPU - pod1CPU
		pod2Memory := allocatableMemory - pod1Memory
		pod2Disk := allocatableDisk - pod1Disk

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[sigmak8sapi.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &sigmak8sapi.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}
		tests := []resourceCase{
			// test[0] pod1 cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod1CPU,
				mem:             pod1Memory,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[1] pod2 cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod2CPU,
				mem:             pod2Memory,
				ethstorage:      pod2Disk,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[3] pod3 cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod1CPU,
				mem:             pod1Memory,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
		}
		testContext := &testContext{
			caseName:  "cpushare-k8s-without-overquota",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
			testCases: tests,
			nodeName:  nodeName,
		}
		testContext.execTests(
			checkSharePool,
		)
	})
})
