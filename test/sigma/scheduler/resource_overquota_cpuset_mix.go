package scheduler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/swarm"
)

var _ = Describe("[sigma-2.0+3.1][sigma-scheduler][resource][Serial]", func() {
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

	// 超卖场景下，验证混合链路的 cpuset cpu-over-quota = 2
	// 记录超卖后可分配的 cpu 额度 X
	// 每次新创建 pod 后要检查每个核的使用次数是否超过quota
	// 整个过程循环 2 次
	// test1：cpu-over-quota=2
	// 第一轮：测试 spread
	// 1. 创建 pod1：k8s, cpu=X/2, spread   成功，     剩余cpu=X/2
	// 2. 创建 pod2：sigma, cpu=X/2, spread   成功，     剩余cpu=0
	// 3. 创建 pod3：k8s, cpu=X/2, spread   失败，    剩余cpu=0
	// 4. 创建 pod4：sigma, cpu=X/2, spread   失败，    剩余cpu=0
	// 5. 删除 pod1～pod4
	// 第二轮：测试 samecorefirst
	// 6. 创建 pod5：k8s, cpu=X/2, samecorefirst   成功，     剩余cpu=X/2
	// 7. 创建 pod6：sigma, cpu=X/2, samecorefirst   成功，     剩余cpu=0
	// 8. 创建 pod7：k8s, cpu=X/2, samecorefirst   失败，    剩余cpu=0
	// 9. 创建 pod8：sigma, cpu=X/2, samecorefirst   失败，    剩余cpu=0
	// 10. 删除 pod5～pod8
	// 第三轮：测试混合策略
	// 11. 创建 pod9：k8s, cpu=X/4, spread              成功，    剩余cpu=3/4*X
	// 12. 创建 pod10：sigma, cpu=X/4, samecorefirst   成功，    剩余cpu=2/4*X
	// 13. 创建 pod11：k8s, cpu=X/4, spread              成功，    剩余cpu=1/4*X
	// 14. 创建 pod12：sigma, cpu=X/4, samecorefirst   成功，    剩余cpu=0
	// 15. 创建 pod13：k8s, cpu=X/4，随机策略           失败，    剩余cpu=0
	// 16. 创建 pod14：sigma, cpu=X/4，随机策略           失败，    剩余cpu=0
	// 17. 删除 pod9～pod14
	// 第四轮：测试随机混合策略
	// 18. 创建 pod15～pod22：链路随机，cpu=X/8，随机策略           成功，    剩余cpu=0
	// 19. 创建 pod23：链路随机，cpu=X/8，随机策略    失败，    剩余cpu=0
	It("[p1] cpusetOverquotaMix001 cpu-over-quota=2.", func() {
		framework.WaitForStableCluster(cs, masterNodes)

		nodeName := GetNodeThatCanRunPod(f)
		By(fmt.Sprintf("apply a label on the found node %s", nodeName))

		allocatableCPU := nodeToAllocatableMapCPU[nodeName]
		allocatableMemory := nodeToAllocatableMapMem[nodeName]
		allocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		framework.Logf("allocatableCpu: %d", allocatableCPU)
		framework.Logf("allocatableMemory: %d", allocatableMemory)
		framework.Logf("allocatableDisk: %d", allocatableDisk)

		cpuOverQuotaRatio := 2.0
		allocatableCPUAfterQuota := int64(float64(allocatableCPU) * cpuOverQuotaRatio)

		requestedMemory := int64(1000000000) // 1G memory 确保能分配
		requestedDisk := int64(1000000000)   // 1G disk 确保能分配

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[sigmak8sapi.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &sigmak8sapi.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		strategies := []string{"sameCoreFirst", "spread"}
		requestTypes := []string{requestTypeKubernetes, requestTypeSigma}

		// 整个过程执行 times 次
		times := 2
		for t := 1; t <= times; t++ {
			// 第一轮：测试 spread
			requestedCPU := allocatableCPUAfterQuota / 2
			tests1 := []resourceCase{
				{
					cpu:             requestedCPU,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					spreadStrategy:  "spread",
				},
				{
					cpu:             requestedCPU,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeSigma,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  "spread",
				},
				{
					cpu:             requestedCPU,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					spreadStrategy:  "spread",
				},
				{
					cpu:             requestedCPU,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeSigma,
					shouldScheduled: false,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  "spread",
				},
				{
					cleanIndexes: []int{0, 1, 2, 3},
					requestType:  cleanResource,
				},
			}

			// 第二轮：测试 samecorefirst
			requestedCPU = allocatableCPUAfterQuota / 2
			tests2 := []resourceCase{
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
					requestType:     requestTypeSigma,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             requestedCPU,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             requestedCPU,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeSigma,
					shouldScheduled: false,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cleanIndexes: []int{5, 6, 7, 8},
					requestType:  cleanResource,
				},
			}

			// 第三轮：测试混合策略
			requestedCPU = allocatableCPUAfterQuota / 4
			tests3 := []resourceCase{
				{
					cpu:             requestedCPU,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					spreadStrategy:  "spread",
				},
				{
					cpu:             requestedCPU,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeSigma,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
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
					requestType:     requestTypeSigma,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  "spread",
				},
				{
					cpu:             requestedCPU,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					spreadStrategy:  strategies[rand.Int()%2],
				},
				{
					cpu:             requestedCPU,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeSigma,
					shouldScheduled: false,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  strategies[rand.Int()%2],
				},
				{
					cleanIndexes: []int{10, 11, 12, 13, 14, 15},
					requestType:  cleanResource,
				},
			}

			// 第四轮：测试随机混合策略
			requestedCPU = allocatableCPUAfterQuota / 8
			tests4 := []resourceCase{}
			for i := 1; i <= 9; i++ {
				test := resourceCase{
					cpu:             requestedCPU,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypes[rand.Int()%2],
					shouldScheduled: true,
					spreadStrategy:  strategies[rand.Int()%2],
				}
				if test.requestType == requestTypeKubernetes {
					test.affinityConfig = map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}}
				}
				if test.requestType == requestTypeSigma {
					test.affinityConfig = map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}}
				}
				if i == 9 {
					test.shouldScheduled = false
				}
				tests4 = append(tests4, test)
			}
			tests4 = append(tests4, resourceCase{
				cleanIndexes: []int{17, 18, 19, 20, 21, 22, 23, 24, 25},
				requestType:  cleanResource,
			})

			tests := []resourceCase{}
			tests = append(tests, tests1...)
			tests = append(tests, tests2...)
			tests = append(tests, tests3...)
			tests = append(tests, tests4...)

			testCtx := &testContext{
				caseName:          "resourceOverQuotaMix",
				cs:                cs,
				localInfo:         localInfo,
				f:                 f,
				testCases:         tests,
				CPUOverQuotaRatio: cpuOverQuotaRatio,
				nodeName:          nodeName,
			}

			testCtx.execTests(
				checkCPUSetOverquotaRate,
				checkSharePool,
			)
		}

	})

	// 超卖场景下，验证混合链路的 cpuset cpu-over-quota = 1.5
	// 步骤与 cpu-over-quota = 2 一样
	It("[p1] cpusetOverquotaMix002 cpu-over-quota=1.5.", func() {
		framework.WaitForStableCluster(cs, masterNodes)

		nodeName := GetNodeThatCanRunPod(f)
		By(fmt.Sprintf("apply a label on the found node %s", nodeName))

		allocatableCPU := nodeToAllocatableMapCPU[nodeName]
		allocatableMemory := nodeToAllocatableMapMem[nodeName]
		allocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		framework.Logf("allocatableCpu: %d", allocatableCPU)
		framework.Logf("allocatableMemory: %d", allocatableMemory)
		framework.Logf("allocatableDisk: %d", allocatableDisk)

		cpuOverQuotaRatio := 1.5
		allocatableCPUAfterQuota := int64(float64(allocatableCPU) * cpuOverQuotaRatio)

		requestedMemory := int64(1000000000) // 1G memory 确保能分配
		requestedDisk := int64(1000000000)   // 1G disk 确保能分配

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[sigmak8sapi.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &sigmak8sapi.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		strategies := []string{"sameCoreFirst", "spread"}
		requestTypes := []string{requestTypeKubernetes, requestTypeSigma}

		// 整个过程执行 times 次
		times := 2
		for t := 1; t <= times; t++ {
			// 第一轮：测试 spread
			// CPUSet 的测试，申请的核数必须是整数
			requestedCPU1 := allocatableCPUAfterQuota / 2
			requestedCPU1 = (requestedCPU1 / 1000) * 1000
			requestedCPU2 := allocatableCPUAfterQuota - requestedCPU1
			tests1 := []resourceCase{
				{
					cpu:             requestedCPU1,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					spreadStrategy:  "spread",
				},
				{
					cpu:             requestedCPU2,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeSigma,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  "spread",
				},
				{
					cpu:             requestedCPU2,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					spreadStrategy:  "spread",
				},
				{
					cpu:             requestedCPU1,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeSigma,
					shouldScheduled: false,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  "spread",
				},
				{
					cleanIndexes: []int{0, 1, 2, 3},
					requestType:  cleanResource,
				},
			}

			// 第二轮：测试 samecorefirst
			tests2 := []resourceCase{
				{
					cpu:             requestedCPU1,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             requestedCPU2,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeSigma,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             requestedCPU2,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             requestedCPU1,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeSigma,
					shouldScheduled: false,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cleanIndexes: []int{5, 6, 7, 8},
					requestType:  cleanResource,
				},
			}

			// 第三轮：测试混合策略
			// CPUSet 的测试，申请的核数必须是整数
			requestedCPU3 := allocatableCPUAfterQuota / 4
			requestedCPU3 = (requestedCPU3 / 1000) * 1000
			tests3 := []resourceCase{
				{
					cpu:             requestedCPU3,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					spreadStrategy:  "spread",
				},
				{
					cpu:             requestedCPU3,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeSigma,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             requestedCPU3,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             requestedCPU3,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeSigma,
					shouldScheduled: true,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  "spread",
				},
				{
					cpu:             2 * requestedCPU3,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					spreadStrategy:  strategies[rand.Int()%2],
				},
				{
					cpu:             2 * requestedCPU3,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypeSigma,
					shouldScheduled: false,
					affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
					spreadStrategy:  strategies[rand.Int()%2],
				},
				{
					cleanIndexes: []int{10, 11, 12, 13, 14, 15},
					requestType:  cleanResource,
				},
			}

			// 第四轮：测试随机混合策略
			// CPUSet 的测试，申请的核数必须是整数
			requestedCPU4 := allocatableCPUAfterQuota / 8
			requestedCPU4 = (requestedCPU4 / 1000) * 1000
			tests4 := []resourceCase{}
			for i := 1; i <= 9; i++ {
				test := resourceCase{
					cpu:             requestedCPU4,
					mem:             requestedMemory,
					ethstorage:      requestedDisk,
					requestType:     requestTypes[rand.Int()%2],
					shouldScheduled: true,
					spreadStrategy:  strategies[rand.Int()%2],
				}
				if test.requestType == requestTypeKubernetes {
					test.affinityConfig = map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}}
				}
				if test.requestType == requestTypeSigma {
					test.affinityConfig = map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}}
				}
				if i == 9 {
					test.cpu = 2 * requestedCPU4
					test.shouldScheduled = false
				}
				tests4 = append(tests4, test)
			}
			tests4 = append(tests4, resourceCase{
				cleanIndexes: []int{17, 18, 19, 20, 21, 22, 23, 24, 25},
				requestType:  cleanResource,
			})

			tests := []resourceCase{}
			tests = append(tests, tests1...)
			tests = append(tests, tests2...)
			tests = append(tests, tests3...)
			tests = append(tests, tests4...)

			testCtx := &testContext{
				caseName:          "resourceOverQuotaMix",
				cs:                cs,
				localInfo:         localInfo,
				f:                 f,
				testCases:         tests,
				CPUOverQuotaRatio: cpuOverQuotaRatio,
				nodeName:          nodeName,
			}

			testCtx.execTests(
				checkCPUSetOverquotaRate,
				checkSharePool,
			)
		}
	})
})
