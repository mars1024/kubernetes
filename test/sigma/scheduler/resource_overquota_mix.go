package scheduler

import (
	"encoding/json"
	"fmt"
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

	//--------------------------------------------------------------------------
	// 超卖场景下，验证混合链路的资源校验， 包括cpu/memory/disk
	// cpu-over-quota = 2       memory-over-quota=1.1
	// 步骤：
	// 获取一个可调度的节点，记录超卖后可分配的 cpu 额度 X，memory 的额度 Y，disk 额度 Z
	// 给机器打上超卖标签
	// sigma.ali/cpu-over-quota=2
	// sigma.ali/is-over-quota=true
	// sigma.ali/memory-over-quota=1.1
	// 1. 创建 pod1：k8s，cpu=X/2，mem=Y/2，disk=Z/2， 成功，   剩余cpu=X/2，mem=Y/2，disk=Z/2
	// 2. 创建 pod2：sigma，cpu=X/2-1，mem=Y/2-1k，disk=Z/2-1k， 成功，剩余cpu=1，mem=1k，disk=1k
	// 3. 创建 pod3：k8s，cpu=X/2，mem=1k，disk=1k，  失败， 剩余cpu=1，mem=1k，disk=1k
	// 4. 创建 pod4：k8s，cpu=1，mem=Y/2，disk=1k，  失败， 剩余cpu=1，mem=1k，disk=1k
	// 5. 创建 pod5：k8s，cpu=1，mem=1k，disk=Z/2，  失败， 剩余cpu=1，mem=1k，disk=1k
	// 6. 创建 pod6：sigma，cpu=X/2，mem=1k，disk=1k，  失败， 剩余cpu=1，mem=1k，disk=1k
	// 7. 创建 pod7：sigma，cpu=1，mem=Y/2，disk=1k，  失败， 剩余cpu=1，mem=1k，disk=1k
	// 8. 创建 pod8：sigma，cpu=1，mem=1k，disk=Z/2，  失败， 剩余cpu=1，mem=1k，disk=1k
	// 9. 创建 pod9：k8s，cpu=1，mem=1k，disk=1k，  成功， 剩余cpu=0，mem=0，disk=0
	// 10. 删除 pod3～9
	// 11. 创建 pod10：sigma，cpu=1，mem=1k，disk=1k，  成功， 剩余cpu=0，mem=0，disk=0
	// 12. 创建 pod11：k8s，cpu=X/2，mem=Y/2，disk=Z/2，  失败， 剩余cpu=0，mem=0，disk=0
	// 13. 创建 pod12：sigma，cpu=X/2，mem=Y/2，disk=Z/2，  失败， 剩余cpu=0，mem=0，disk=0
	// 14. 删除 pod10～12,pod2
	// 15. 创建 pod13：sigma，cpu=X/2+1，mem=Y/2+1k, disk=Z/2+1k， 失败，剩余cpu=X/2，mem=Y/2，disk=Z/2
	// 16. 创建 pod14：k8s，cpu=X/2+1，mem=Y/2+1k, disk=Z/2+1k， 失败，剩余cpu=X/2，mem=Y/2，disk=Z/2
	// 17. 创建 pod15：sigma，cpu=X/2，mem=Y/2, disk=Z/2，   陈工，  剩余cpu=0，mem=0，disk=0
	// 17. 删除pod1，13，14，15
	// 18. 创建pod16：sigma，cpu=X/2，mem=Y/2，disk=Z/2， 成功，   剩余cpu=X/2，mem=Y/2，disk=Z/2
	// 19. 创建pod17：k8s，cpu=X/2，mem=Y/2，disk=Z/2， 成功，   剩余cpu=0，mem=0，disk=0
	// 20. 创建pod18：sigma，cpu=X/2，mem=Y/2，disk=Z/2， 失败，   剩余cpu=0，mem=0，disk=0
	It("[p1] resourceOverquotaMix001 cpu-over-quota=2 and memory-over-quota=1.1.", func() {
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
		memOverQuotaRatio := 1.1

		allocatableCPUAfterQuota := int64(float64(allocatableCPU) * cpuOverQuotaRatio)
		allocatableMemoryAfterQuota := int64(float64(allocatableMemory) * memOverQuotaRatio)
		allocatableDiskAfterQuota := allocatableDisk

		pod1CPU := allocatableCPUAfterQuota / 2
		pod1Memory := allocatableMemoryAfterQuota / 2
		pod1Disk := allocatableDiskAfterQuota / 2

		pod2CPU := allocatableCPUAfterQuota - pod1CPU
		pod2Memory := allocatableMemoryAfterQuota - pod1Memory
		pod2Disk := allocatableDiskAfterQuota - pod1Disk

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[sigmak8sapi.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &sigmak8sapi.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			// test[0] pod1 k8s，cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod1CPU,
				mem:             pod1Memory,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[1] pod2 sigma，cpu=X/2-1，mem=Y/2-1k，disk=Z/2-1k
			{
				cpu:             pod2CPU - 1000,
				mem:             pod2Memory - 1024*1024*1024,
				ethstorage:      pod2Disk - 1024*1024*1024,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[2] pod3 k8s，cpu=X/2，mem=1k，disk=1k
			{
				cpu:             pod1CPU,
				mem:             1024 * 1024 * 1024,
				ethstorage:      1024 * 1024 * 1024,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[3] pod4 k8s，cpu=1，mem=Y/2，disk=1k
			{
				cpu:             1000,
				mem:             pod1Memory,
				ethstorage:      1024 * 1024 * 1024,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[4] pod5 k8s，cpu=1，mem=1k，disk=Z/2
			{
				cpu:             1000,
				mem:             1024 * 1024 * 1024,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[5] pod6 sigma，cpu=X/2，mem=1k，disk=1k
			{
				cpu:             pod1CPU,
				mem:             1024 * 1024 * 1024,
				ethstorage:      1024 * 1024 * 1024,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[6] pod7 sigma，cpu=1，mem=Y/2，disk=1k
			{
				cpu:             1000,
				mem:             pod1Memory,
				ethstorage:      1024 * 1024 * 1024,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[7] pod8 sigma，cpu=1，mem=1k，disk=Z/2
			{
				cpu:             1000,
				mem:             1024 * 1024 * 1024,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[8] pod9 k8s，cpu=1，mem=1k，disk=1k
			{
				cpu:             1000,
				mem:             100 * 1024 * 1024,
				ethstorage:      100 * 1024 * 1024,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[9] delete pod3~9
			{
				cleanIndexes: []int{2, 3, 4, 5, 6, 7, 8},
				requestType:  cleanResource,
			},
			// test[10] pod10 sigma，cpu=1，mem=1k，disk=1k
			{
				cpu:             1000,
				mem:             1024 * 1024 * 1024,
				ethstorage:      1024 * 1024 * 1024,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[11] pod11 k8s，cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod1CPU,
				mem:             pod1Memory,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[12] pod12 sigma，cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod1CPU,
				mem:             pod1Memory,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[13] delete pod10~12, pod2
			{
				cleanIndexes: []int{1, 10, 11, 12},
				requestType:  cleanResource,
			},
			// test[14] pod13 sigma，cpu=X/2+1，mem=Y/2+1k, disk=Z/2+1k
			{
				cpu:             pod2CPU + 1000,
				mem:             pod2Memory + 1024*1024*1024,
				ethstorage:      pod2Disk + 1024*1024*1024,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[15] pod14 k8s，cpu=X/2+1，mem=Y/2+1k, disk=Z/2+1k
			{
				cpu:             pod2CPU + 1000,
				mem:             pod2Memory + 1024*1024*1024,
				ethstorage:      pod2Disk + 1024*1024*1024,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[16] pod15 sigma，cpu=X/2，mem=Y/2, disk=Z/2
			{
				cpu:             pod2CPU,
				mem:             pod2Memory,
				ethstorage:      pod2Disk,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[17] delete pod1，13，14, 15
			{
				cleanIndexes: []int{0, 14, 15, 16},
				requestType:  cleanResource,
			},
			// test[18] pod16 sigma，cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod1CPU,
				mem:             pod1Memory,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[19] pod17 k8s，cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod2CPU,
				mem:             pod2Memory,
				ethstorage:      pod2Disk,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[20] pod18 sigma，cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod2CPU,
				mem:             pod2Memory,
				ethstorage:      pod2Disk,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: false,
				cpushare:        true,
			},
		}

		testContext := &testContext{
			caseName:             "resourceOverQuotaMix",
			cs:                   cs,
			localInfo:            localInfo,
			f:                    f,
			testCases:            tests,
			CPUOverQuotaRatio:    cpuOverQuotaRatio,
			MemoryOverQuotaRatio: memOverQuotaRatio,
			nodeName:             nodeName,
		}

		testContext.execTests(
			checkCPUSetOverquotaRate,
			checkSharePool,
		)
	})

	It("[p1] resourceOverquotaMix002 cpu-over-quota=1.5 and memory-over-quota=1.1.", func() {
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
		memOverQuotaRatio := 1.1

		allocatableCPUAfterQuota := int64(float64(allocatableCPU) * cpuOverQuotaRatio)
		allocatableMemoryAfterQuota := int64(float64(allocatableMemory) * memOverQuotaRatio)
		allocatableDiskAfterQuota := allocatableDisk

		pod1CPU := allocatableCPUAfterQuota / 2
		pod1Memory := allocatableMemoryAfterQuota / 2
		pod1Disk := allocatableDiskAfterQuota / 2

		pod2CPU := allocatableCPUAfterQuota - pod1CPU
		pod2Memory := allocatableMemoryAfterQuota - pod1Memory
		pod2Disk := allocatableDiskAfterQuota - pod1Disk

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[sigmak8sapi.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &sigmak8sapi.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			// test[0] pod1 k8s，cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod1CPU,
				mem:             pod1Memory,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[1] pod2 sigma，cpu=X/2-1，mem=Y/2-1k，disk=Z/2-1k
			{
				cpu:             pod2CPU - 1000,
				mem:             pod2Memory - 1024*1024*1024,
				ethstorage:      pod2Disk - 1024*1024*1024,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[2] pod3 k8s，cpu=X/2，mem=1k，disk=1k
			{
				cpu:             pod1CPU,
				mem:             1024 * 1024 * 1024,
				ethstorage:      1024 * 1024 * 1024,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[3] pod4 k8s，cpu=1，mem=Y/2，disk=1k
			{
				cpu:             1000,
				mem:             pod1Memory,
				ethstorage:      1024 * 1024 * 1024,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[4] pod5 k8s，cpu=1，mem=1k，disk=Z/2
			{
				cpu:             1000,
				mem:             1024 * 1024 * 1024,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[5] pod6 sigma，cpu=X/2，mem=1k，disk=1k
			{
				cpu:             pod1CPU,
				mem:             1024 * 1024 * 1024,
				ethstorage:      1024 * 1024 * 1024,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[6] pod7 sigma，cpu=1，mem=Y/2，disk=1k
			{
				cpu:             1000,
				mem:             pod1Memory,
				ethstorage:      1024 * 1024 * 1024,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[7] pod8 sigma，cpu=1，mem=1k，disk=Z/2
			{
				cpu:             1000,
				mem:             1024 * 1024 * 1024,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[8] pod9 k8s，cpu=1，mem=1k，disk=1k
			{
				cpu:             1000,
				mem:             100 * 1024 * 1024,
				ethstorage:      100 * 1024 * 1024,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[9] delete pod3~9
			{
				cleanIndexes: []int{2, 3, 4, 5, 6, 7, 8},
				requestType:  cleanResource,
			},
			// test[10] pod10 sigma，cpu=1，mem=1k，disk=1k
			{
				cpu:             1000,
				mem:             1024 * 1024 * 1024,
				ethstorage:      1024 * 1024 * 1024,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[11] pod11 k8s，cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod1CPU,
				mem:             pod1Memory,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[12] pod12 sigma，cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod1CPU,
				mem:             pod1Memory,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[13] delete pod10~12, pod2
			{
				cleanIndexes: []int{1, 10, 11, 12},
				requestType:  cleanResource,
			},
			// test[14] pod13 sigma，cpu=X/2+1，mem=Y/2+1k, disk=Z/2+1k
			{
				cpu:             pod2CPU + 1000,
				mem:             pod2Memory + 1024*1024*1024,
				ethstorage:      pod2Disk + 1024*1024*1024,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[15] pod14 k8s，cpu=X/2+1，mem=Y/2+1k, disk=Z/2+1k
			{
				cpu:             pod2CPU + 1000,
				mem:             pod2Memory + 1024*1024*1024,
				ethstorage:      pod2Disk + 1024*1024*1024,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			// test[16] pod15 sigma，cpu=X/2，mem=Y/2, disk=Z/2
			{
				cpu:             pod2CPU,
				mem:             pod2Memory,
				ethstorage:      pod2Disk,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[17] delete pod1，13，14, 15
			{
				cleanIndexes: []int{0, 14, 15, 16},
				requestType:  cleanResource,
			},
			// test[18] pod16 sigma，cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod1CPU,
				mem:             pod1Memory,
				ethstorage:      pod1Disk,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[19] pod17 k8s，cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod2CPU,
				mem:             pod2Memory,
				ethstorage:      pod2Disk,
				affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			// test[20] pod18 sigma，cpu=X/2，mem=Y/2，disk=Z/2
			{
				cpu:             pod2CPU,
				mem:             pod2Memory,
				ethstorage:      pod2Disk,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.EnableOverQuota": {"true"}},
				requestType:     requestTypeSigma,
				shouldScheduled: false,
				cpushare:        true,
			},
		}

		testContext := &testContext{
			caseName:             "resourceOverQuotaMix",
			cs:                   cs,
			localInfo:            localInfo,
			f:                    f,
			testCases:            tests,
			CPUOverQuotaRatio:    cpuOverQuotaRatio,
			MemoryOverQuotaRatio: memOverQuotaRatio,
			nodeName:             nodeName,
		}

		testContext.execTests(
			checkCPUSetOverquotaRate,
			checkSharePool,
		)
	})

	// 超卖 k8s 单链路 cpushare 测试，检查 sharepool 的状态
	It("[p1] cpushareK8sOverquota.", func() {
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
		memOverQuotaRatio := 1.1

		allocatableCPUAfterQuota := int64(float64(allocatableCPU) * cpuOverQuotaRatio)
		allocatableMemoryAfterQuota := int64(float64(allocatableMemory) * memOverQuotaRatio)
		allocatableDiskAfterQuota := allocatableDisk

		pod1CPU := allocatableCPUAfterQuota / 2
		pod1Memory := allocatableMemoryAfterQuota / 2
		pod1Disk := allocatableDiskAfterQuota / 2

		pod2CPU := allocatableCPUAfterQuota - pod1CPU
		pod2Memory := allocatableMemoryAfterQuota - pod1Memory
		pod2Disk := allocatableDiskAfterQuota - pod1Disk

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
			caseName:             "cpushare-k8s-with-overquota",
			cs:                   cs,
			localInfo:            localInfo,
			f:                    f,
			testCases:            tests,
			CPUOverQuotaRatio:    cpuOverQuotaRatio,
			MemoryOverQuotaRatio: memOverQuotaRatio,
			nodeName:             nodeName,
		}

		testContext.execTests(
			checkSharePool,
		)
	})
})
