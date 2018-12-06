package scheduler

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/swarm"
)

var _ = Describe("[sigma-2.0+3.1][sigma-scheduler][cpuset][cpu]", func() {
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
		DeleteSigmaContainer(f)
	})

	// case描述：1.5超卖场景下，验证分配的物理核优先sigma和k8s互相感知
	// 步骤 要求每个容器分配的cpu个数不能低于2个，否则这个case会验证失败
	// 1.  1/2 整机核 sigma（预期成功）
	// 2.  1/2 整机核 k8s（预期成功）
	// 3.  1/2 整机核 sigma（预期成功）
	// 4.  1/2 整机核 k8s（预期失败）

	// 验证结果
	// 1. 每个容器的cpu核都不重叠 checkContainerCpuIdNotDuplicate
	// 2. 每个核的超卖比不大于2 checkCpusetOverquotaRate

	It("overQuotaCPUSetMix001: ", func() {
		framework.WaitForStableCluster(cs, masterNodes)

		nodeName := GetNodeThatCanRunPod(f)

		By(fmt.Sprintf("apply a label on the found node %s", nodeName))

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

		cpuOverQuotaRatio := 1.5
		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				shouldScheduled: true,
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        false,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				shouldScheduled: true,
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				spreadStrategy:  "sameCoreFirst",
			},
		}
		testContext := &testContext{
			caseName:          "overQuotaCPUSetMix001",
			cs:                cs,
			localInfo:         localInfo,
			f:                 f,
			testCases:         tests,
			CPUOverQuotaRatio: cpuOverQuotaRatio,
			nodeName:          nodeName,
		}
		testContext.execTests(
			checkContainerCPUIDNotDuplicated,
			checkCPUSetOverquotaRate,
		)

	})

	// case描述：1.25 超卖场景下的cpu互斥，app1和app2应用的 CPU 互斥，app3普通应用
	// 用于：验证分配的物理核优先，sigma和k8s互相感知
	// 步骤 要求每个容器分配的cpu个数不能低于2个，否则这个case会验证失败
	// 1.  1/4 整机核 app1 sigma（预期成功）
	// 2.  1/4 整机核 app2 k8s（预期成功）
	// 3.  1/4 整机核 app3 sigma（预期成功）
	// 4.  1/4 整机核 app3 k8s（预期成功）
	// 5.  1/4 整机核 app3 sigma（预期成功）
	// 6.  1/4 整机核 app3 k8s（预期成功）
	// 验证结果
	// 1. app1的物理核不重叠， checkContainerCpuMutexCPUID
	// 2. app2的物理核不重叠， checkContainerCpuMutexCPUID
	// 3. app1和app2之间的物理核不重叠 checkHostCPUMutexCPUID
	// 4. 每个容器的逻辑核不重叠 checkContainerCPUIDNotDuplicated
	// 5. 每个核的超卖比不大于2 checkCpusetOverquotaRate
	It("[p2] overQuotaCPUSetMix002: ", func() {
		Skip("not implemented")

		nodeName := GetNodeThatCanRunPod(f)
		framework.Logf("get one node to schedule, nodeName: %s", nodeName)
		framework.WaitForStableCluster(cs, masterNodes)

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

		cpuOverQuotaRatio := 1.25

		// 2.0 给机器打超卖标
		swarm.SetNodeOverQuota(nodeName, cpuOverQuotaRatio, 1.0)
		defer swarm.SetNodeToNotOverQuota(nodeName)

		// 必须更新 global rule
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
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.AppName": {"app1"}, "ali.EnableOverQuota": {"true"}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				labels:          map[string]string{api.LabelAppName: "app2"},
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.AppName": {"app3"}, "ali.EnableOverQuota": {"true"}},
				shouldScheduled: true,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				labels:          map[string]string{api.LabelAppName: "app3"},
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.AppName": {"app3"}, "ali.EnableOverQuota": {"true"}},
				shouldScheduled: true,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				labels:          map[string]string{api.LabelAppName: "app3"},
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
			},
		}
		testContext := &testContext{
			caseName:          "overQuotaCPUSetMix002",
			cs:                cs,
			localInfo:         localInfo,
			f:                 f,
			globalRule:        globalRule,
			testCases:         tests,
			CPUOverQuotaRatio: cpuOverQuotaRatio,
			nodeName:          nodeName,
		}

		testContext.execTests(
			checkContainerCPUIDNotDuplicated,
			checkHostCPUIdNotDuplicated,
			checkContainerCpuMutexCPUID,
			checkHostCPUMutexCPUID,
			checkCPUSetOverquotaRate,
		)
	})

})
