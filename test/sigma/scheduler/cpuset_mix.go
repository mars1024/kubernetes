package scheduler

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/env"
	"k8s.io/kubernetes/test/sigma/swarm"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("[sigma-2.0+3.1][sigma-scheduler][smoke][cpuset]", func() {
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

	// 非超卖场景下的 SameCoreFirst，验证分配的物理核优先，sigma 和 k8s 互相感知
	// 步骤 要求每个容器分配的 cpu 个数不能低于 2 个，否则这个 case 会验证失败
	// 1.  1/4 整机核 sigma（预期成功）
	// 2.  1/4 整机核 k8s（预期成功）
	// 3.  1/4 整机核 sigma（预期成功）
	// 4.  1/4 整机核 k8s（预期成功）
	// 5.  1/4 整机核 sigma （预期失败）
	// 6.  1/4 整机核 k8s（预期失败）

	// 验证结果
	// 1. 所有容器的 cpu 都不重叠
	// 2. 每个容器的 cpu 和都不重叠
	// 3. 每个容器的 cpu 的物理核 * 2 = 逻辑和
	It("[smoke][p0] cpuset_mix_001: Pod with SameCoreFirst strategy, cpuset not overlap, physical_cores*2 = logical_cores", func() {
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
		localInfoString := nodesInfo[nodeName].Annotations[api.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName: %s, localInfoString is empty", nodeName))
		localInfo := &api.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName: %s, localInfoString: %v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: false,
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				spreadStrategy:  "sameCoreFirst",
			},
		}
		testContext := &testContext{
			testCases: tests,
			caseName:  "cpuset_mix_001",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
			nodeName:  nodeName,
		}

		testContext.execTests(
			checkContainerCPUIDNotDuplicated,
			checkHostCPUIdNotDuplicated,
			checkContainerSameCoreFirst,
		)
	})

	// case描述：非超卖场景下的Spread，
	// 用于：验证分配的物理核优先，sigma和k8s互相感知
	// 步骤 要求每个容器分配的cpu个数不能低于2个，否则这个case会验证失败
	// 1.  1/4 整机核 sigma（预期成功）
	// 2.  1/4 整机核 k8s（预期成功）
	// 3.  1/4 整机核 sigma（预期成功）
	// 4.  1/4 整机核 k8s（预期成功）

	// 验证结果
	// 1. 所有容器的cpu都不重叠
	// 2. 每个容器的cpu都不重叠
	// 3. 每个容器的cpu的物理核不重叠
	It("[smoke][p0] cpuset_mix_002: Pod with Spread strategy, cpuset not overlap, physical core spread", func() {
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
		localInfoString := nodesInfo[nodeName].Annotations[api.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &api.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				spreadStrategy:  "default",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "spread",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				spreadStrategy:  "spread",
			},
		}
		testContext := &testContext{
			testCases: tests,
			caseName:  "cpuset_mix_002",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
			nodeName:  nodeName,
		}

		testContext.execTests(
			checkContainerCPUIDNotDuplicated,
			checkHostCPUIdNotDuplicated,
			checkContainerSpread,
		)
	})

	// case描述：非超卖场景下的cpu互斥，app1和app2应用互斥，app3普通应用
	// 用于：验证sigma和k8s之间的cpu分配明细是互相感知的
	// 步骤 要求每个容器分配的cpu个数不能低于2个，否则这个case会验证失败
	// 1.  1/4 整机核 app1 sigma（预期成功）
	// 2.  1/4 整机核 app2 k8s（预期成功）
	// 3.  1/4 整机核 app3 sigma（预期成功）
	// 4.  1/4 整机核 app3 k8s（预期成功）

	// 验证结果
	// 1. app1 的物理核不重叠，
	// 2. app2 的物理核不重叠，
	// 3. app1 和 app2 之间的物理核不重叠
	// 4. 每个容器的逻辑和不重叠
	// 5. 所有容器的逻辑和不重叠
	It("cpuset_mix_003: Pod with Spread strategy and CPU anti-affinity, cpuset not overlap, physical core spread", func() {
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
		localInfoString := nodesInfo[nodeName].Annotations[api.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &api.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}
		globalRule := &swarm.GlobalRules{
			UpdateTime: time.Now().Format(time.RFC3339),
			CpuSetMutex: swarm.CpuSetMutexDecs{
				AppConstraints: []string{"app1", "app2"},
			},
		}
		swarm.UpdateSigmaGlobalConfig(globalRule)

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.AppName": {"app1"}},
				spreadStrategy:  "default",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				labels:          map[string]string{api.LabelAppName: "app2"},
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "spread",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.AppName": {"app3"}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				labels:          map[string]string{api.LabelAppName: "app3"},
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				spreadStrategy:  "spread",
			},
		}
		testContext := &testContext{
			testCases:  tests,
			caseName:   "cpuset_mix_003",
			cs:         cs,
			localInfo:  localInfo,
			f:          f,
			globalRule: globalRule,
			nodeName:   nodeName,
		}
		testContext.execTests(
			checkContainerCPUIDNotDuplicated,
			checkHostCPUIdNotDuplicated,
			checkContainerCpuMutexCPUID,
			checkHostCPUMutexCPUID,
		)
	})
	// case描述：非超卖场景下的cpu分配k8s，sigma2.0混合不同策略交叉组合
	// 用于：验证sigma和k8s之间的cpu分配的正确性
	// 步骤 要求机器初始状态为空，每个容器分配的cpu个数不能低于2个，否则这个case会验证失败
	// 1.  1/4 整机核 sigma sameCoreFirst（预期成功）
	// 2.  1/4 整机核 k8s sameCoreFirst（预期成功）此时2和1用满1个socket
	// 3.  1/4 整机核 sigma default（预期成功）3在另外一个socket上，且spread
	// 4.  1/4 整机核 k8s spread（预期成功）4和3在一个socket上，且spread

	// 验证结果
	// 1. 容器满足在一个socket上，且sameCore
	// 2. 2和1用满1个socket
	// 3. 在余下的一个socket上，且spread
	// 4. 同3在一个socket上，且为spread
	It("cpuset_mix_004: Pod with Spread and sameCoreFirst strategy.", func() {
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
		localInfoString := nodesInfo[nodeName].Annotations[api.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &api.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				spreadStrategy:  "spread",
			},
		}
		testContext := &testContext{
			testCases: tests,
			caseName:  "cpuset_mix_004",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
		}
		testContext.execTests(
			checkContainerCPUIDNotDuplicated,
			checkHostCPUIdNotDuplicated,
			checkContainerCPUStrategyRightWithSameSocket,
		)
	})

	// case描述：非超卖场景下的cpu分配k8s，sigma2.0混合不同策略交叉组合
	// 用于：验证sigma和k8s之间的cpu分配的正确性
	// 步骤 要求机器初始状态为空，每个容器分配的cpu个数不能低于2个，否则这个case会验证失败
	// 1.  1/4 整机核 k8s spread（预期成功）在一个socket上，且spread
	// 2.  1/4 整机核 sigma default（预期成功）和1在一个socket上，且spread
	// 3.  1/4 整机核 sigma sameCoreFirst（预期成功）
	// 4.  1/4 整机核 k8s sameCoreFirst（预期成功）此时3和4用满1个socket

	// 验证结果
	// 1. 在一个socket上，且为spread
	// 2. 和1在一个socket上，且spread
	// 3. 容器满足在一个socket上，且sameCore
	// 4. 3和4用满1个socket
	It("cpuset_mix_005: Pod with Spread and sameCoreFirst strategy.", func() {
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
		localInfoString := nodesInfo[nodeName].Annotations[api.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &api.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				spreadStrategy:  "spread",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
		}
		testContext := &testContext{
			testCases: tests,
			caseName:  "cpuset_mix_005",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
		}
		testContext.execTests(
			checkContainerCPUIDNotDuplicated,
			checkHostCPUIdNotDuplicated,
			checkContainerCPUStrategyRightWithSameSocket,
		)
	})
	// case描述：非超卖场景下的cpu分配k8s，sigma2.0混合不同策略交叉组合
	// 用于：验证sigma和k8s之间的cpu分配的正确性
	// 步骤 要求机器初始状态为空，每个容器分配的cpu个数不能低于2个，否则这个case会验证失败
	// 1.  1/4 整机核 k8s pod A spread（预期成功）
	// 2.  1/4 整机核 sigma contaienr B default（预期成功）
	// 3.  1/4 整机核 sigma container C sameCoreFirst（预期成功）
	// 4.  1/4 整机核 k8s pod D sameCoreFirst（预期成功）
	// 5.  删除ABCD（预期成功）
	// 6.  整机核 k8s pod E sameCoreFirst（预期成功）
	// 7.  删除E（预期成功）
	// 8.  整机核 sigma container F sameCoreFirst（预期成功）

	// 验证结果
	// 1. 在一个socket上，且为spread
	// 2. 和1在一个socket上，且spread
	// 3. 容器满足在一个socket上，且sameCore
	// 4. 3和4用满1个socket
	// 6. 创建成功
	// 8. 创建成功
	It("cpuset_mix_006: Pod with Spread and sameCoreFirst strategy.", func() {
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
		localInfoString := nodesInfo[nodeName].Annotations[api.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &api.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				spreadStrategy:  "spread",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cleanIndexes: []int{0, 1, 2, 3},
				requestType:  cleanResource,
			},
			{
				cpu:             AllocatableCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cleanIndexes: []int{5},
				requestType:  cleanResource,
			},
			{
				cpu:             AllocatableCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
		}
		testContext := &testContext{
			testCases: tests,
			caseName:  "cpuset_mix_006",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
		}
		testContext.execTests(
			checkContainerCPUIDNotDuplicated,
			checkHostCPUIdNotDuplicated,
		)
	})
	// case描述：非超卖场景下的cpu分配k8s，sigma2.0 mixrun混合组合
	// 用于：验证sigma和k8s之间的cpu分配的正确性
	// 步骤 要求机器初始状态为空，每个容器分配的cpu个数不能低于2个，否则这个case会验证失败
	// 1.  整机核 sigma mixrun（预期成功）
	// 2.  1/2整机核 sigma default（预期成功）
	// 3.  整机核 sigma default（预期失败）
	// 4.  整机核 k8s （预期失败）
	// 5.  1/2整机核 k8s sameCoreFirst（预期成功）

	// 验证结果
	// 1. mixrun容器创建成功
	// 2. 可继续分配普通的容器
	// 5. 可以继续分配k8s容器
	It("[ant] cpuset_mix_007: Pod with mixrun case.", func() {
		framework.WaitForStableCluster(cs, masterNodes)

		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		requestedCPU := AllocatableCPU / 2
		requestedMemory := AllocatableMemory / 8 //保证一定能扩容出来
		requestedDisk := AllocatableDisk / 8     //保证一定能扩容出来

		// get nodeIP by node name
		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[api.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &api.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			{
				cpu:             AllocatableCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.CpuSetMode": {"mixrun"}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
			{
				cpu:             AllocatableCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: false,
				spreadStrategy:  "default",
			},
			{
				cpu:             AllocatableCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				spreadStrategy:  "spread",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				spreadStrategy:  "spread",
			},
		}
		testContext := &testContext{
			testCases: tests,
			caseName:  "cpuset_mix_007",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
		}
		testContext.execTests()
	})
	// case描述：非超卖场景下的cpu分配k8s，sigma2.0混合不同策略交叉组合
	// FIXME:sigma2.0 cpuset对cpu sharepool的资源扣减
	// 用于：验证sigma和k8s之间的cpuset分配后cpu sharepool的正确性
	// 步骤 要求机器初始状态为空，每个容器分配的cpu个数不能低于2个，否则这个case会验证失败
	// 1.  1/4 整机核 sigma sameCoreFirst（预期成功）
	// 2.  1/4 整机核 k8s sameCoreFirst（预期成功）此时3和4用满1个socket
	// 3.  1/4 整机核 k8s spread（预期成功）在一个socket上，且spread
	// 4.  1/4 整机核 sigma default（预期成功）和3在一个socket上，且spread

	// 验证结果
	// 1. 分配成功，cpu share pool变化
	// 2. 分配成功，cpu share pool变化
	// 3. 分配成功，cpu share pool变化
	// 4. 分配成功，cpu share pool变化
	It("[p2] cpuset_mix_008: Verify the cpu share pool.", func() {
		Skip("Need to fix.")
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
		localInfoString := nodesInfo[nodeName].Annotations[api.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &api.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				spreadStrategy:  "spread",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				spreadStrategy:  "spread",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
		}
		testContext := &testContext{
			testCases: tests,
			caseName:  "cpuset_mix_008",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
			nodeName:  nodeName,
		}
		testContext.execTests(
			checkContainerCPUIDNotDuplicated,
			checkHostCPUIdNotDuplicated,
			checkContainerCPUStrategyRightWithSameSocket,
			checkNodeSharePool,
		)
	})

	// case描述：非超卖场景下的cpu分配k8s，sigma2.0 cpushare混合组合
	// 用于：验证sigma和k8s之间的cpu分配的正确性
	// 步骤 要求机器初始状态为空，每个容器分配的cpu个数不能低于2个，否则这个case会验证失败
	// 1.  整机核 sigma cpushare（预期成功）
	// 2.  1/2整机核 sigma default（预期成功）
	// 3.  整机核 sigma default（预期失败）
	// 4.  整机核 k8s （预期失败）
	// 5.  1/2整机核 k8s sameCoreFirst（预期成功）

	// 验证结果
	// 1. cpushare容器创建成功
	// 2. 可继续分配普通的容器
	// 5. 可以继续分配k8s容器
	It("[ant] cpuset_mix_009: Pod with cpushare case.", func() {
		framework.WaitForStableCluster(cs, masterNodes)

		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		requestedCPU := AllocatableCPU / 2
		requestedMemory := AllocatableMemory / 8 //保证一定能扩容出来
		requestedDisk := AllocatableDisk / 8     //保证一定能扩容出来

		// get nodeIP by node name
		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[api.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &api.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			{
				cpu:             AllocatableCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.CpuSetMode": {"cpushare"}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
			{
				cpu:             AllocatableCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: false,
				spreadStrategy:  "default",
			},
			{
				cpu:             AllocatableCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				spreadStrategy:  "spread",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				spreadStrategy:  "spread",
			},
		}
		testContext := &testContext{
			testCases: tests,
			caseName:  "cpuset_mix_007",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
		}
		testContext.execTests()
	})
	// case描述：binpack
	// 用于：验证资源分配方式是使用最多优先
	// 步骤 要求机器初始状态为空，每个容器分配的cpu个数不能低于2个，否则这个case会验证失败
	// 1.  1核 sigma A （预期成功）
	// 2.  1核 k8s B （预期成功）
	// 3.  1核sigma C（预期成功）
	// 4.  删除A （预期成功）
	// 5.  1核 sigma D（预期成功）

	// 验证结果
	// 1. 所有pod和容器都在一台机器上创建
	It("[p2] cpuset_mix_010: Verify binpack strategy.", func() {
		if env.GetTester() != "ant" {
			Skip("Not ant, will skip.")
		}
		framework.WaitForStableCluster(cs, masterNodes)

		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		requestedCPU := int64(1000)
		requestedMemory := AllocatableMemory / 16 //保证一定能扩容出来
		requestedDisk := AllocatableDisk / 16     //保证一定能扩容出来

		// get nodeIP by node name
		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.CpuSetMode": {"default"}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				spreadStrategy:  "spread",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.CpuSetMode": {"default"}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
			{
				cleanIndexes: []int{0},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.CpuSetMode": {"default"}},
				shouldScheduled: true,
				spreadStrategy:  "default",
			},
		}
		tc := &testContext{
			testCases: tests,
			caseName:  "cpuset_mix_010",
			cs:        cs,
			f:         f,
		}
		defer cleanJob(tc)

		nodeName = ""
		for i, test := range tc.testCases {
			By(fmt.Sprintf("exec case:%d", i))
			nodeN := ""
			switch test.requestType {
			case requestTypeKubernetes:
				createAndVerifyK8sPod(tc, i, test)
				po := tc.resourceToDelete[i].pod
				newPod, err := tc.cs.CoreV1().Pods(po.Namespace).Get(po.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				nodeN = newPod.Spec.NodeName

			case requestTypeSigma:
				createAndVerifySigmaCase(tc, i, test)
				nodeN = strings.ToLower(tc.resourceToDelete[i].containerResult.HostSN)

			case cleanResource:
				framework.Logf("caseName: %v, caseIndex: %v, cleanResource, indexes: %v", tc.caseName, i, test.cleanIndexes)
				cleanContainers(tc, test.cleanIndexes)
			}
			if len(nodeN) > 0 {
				if len(nodeName) > 0 {
					Expect(nodeName).To(Equal(nodeN))
				} else {
					nodeName = nodeN
				}
			}
		}
	})
})
