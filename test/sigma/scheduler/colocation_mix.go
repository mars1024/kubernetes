package scheduler

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/swarm"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("[sigma-2.0+3.1][sigma-scheduler][ant][smoke][colocation]", func() {
	var cs clientset.Interface
	var nodeList *v1.NodeList

	nodeToAllocatableMapCPU := make(map[string]int64)
	nodeToAllocatableMapColocationCPU := make(map[string]int64)
	nodeToAllocatableMapMem := make(map[string]int64)
	nodeToAllocatableMapEphemeralStorage := make(map[string]int64)

	nodesInfo := make(map[string]*v1.Node)

	f := framework.NewDefaultFramework("colocation")

	f.AllNodesReadyTimeout = 3 * time.Second

	BeforeEach(func() {
		cs = f.ClientSet
		nodeList = &v1.NodeList{}

		masterNodes, nodeList = getMasterAndColocationWorkerNodesOrDie(cs)

		for i, node := range nodeList.Items {
			waitNodeResourceReleaseComplete(node.Name)
			nodesInfo[node.Name] = &nodeList.Items[i]
			etcdNodeinfo := swarm.GetNode(node.Name)
			nodeToAllocatableMapCPU[node.Name] = int64(etcdNodeinfo.LocalInfo.CpuNum * 1000)
			{
				allocatable, found :=
					node.Status.Allocatable[alipaysigmak8sapi.SigmaBEResourceName]
				Expect(found).To(Equal(true))
				nodeToAllocatableMapColocationCPU[node.Name] = allocatable.Value()
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
				podMem := getRequestedMem(pod)
				framework.Logf("podMem: %d", podMem)
				nodeToAllocatableMapCPU[pod.Spec.NodeName] -=
					getRequestedCPU(pod)
				nodeToAllocatableMapColocationCPU[pod.Spec.NodeName] -=
					getRequestedColocationCPU(pod)
				nodeToAllocatableMapMem[pod.Spec.NodeName] -=
					getRequestedMem(pod)
				nodeToAllocatableMapEphemeralStorage[pod.Spec.NodeName] -=
					getRequestedStorageEphemeralStorage(pod)
			}
		}
	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			DumpSchedulerState(f, 0)
		}
		// delete sigma 2.0 allocplan if exists
		DeleteSigmaContainer(f)
	})

	// 前提：
	// node 即属于 sigma 2.0 链路，也属于 sigma 3.1 链路，并且都支持混部；

	// 测试步骤：
	// 1. 获取混部资源的可分配值，记录为 A（以 3.1 的资源为主）；
	// 2. 创建 2.0 的混部容器，申请资源值 A/2，观察容器状态，通过调度器内存接口检查资源可用值和可分配值；
	// 3. 创建 3.1 的混部容器，申请资源值为 A/2，观察容器状态，通过调度器内存接口检查资源可用值和可分配值；
	// 4. 继续创建 2.0 的混部容器，申请资源为 1 milli cpu，观察容器状态通过调度器内存接口检查资源可用值和可分配值；
	// 5. 继续创建 3.1 的混部容器，申请资源为 1 milli cpu，观察容器状态通过调度器内存接口检查资源可用值和可分配值；
	// 6. 删掉 2.0 的混部，创建 3.1 的混部，可以成功，再创建 1 milli cpu，失败；
	// 7. 删掉 3.1 的混部，创建 2.0 的也可以成功，再创建 1 milli cpu，失败；
	// 8. 删掉所有混部容器，重复 2-7 的步骤，只是把 2.0 和 3.1 的创建顺序做调换；

	// 期望结果：
	// 1. A > 0；
	// 2. 可以正常创建，并且容器的根组是 sigma-stream，cpu 等参数正常，调度器内存中的资源状态正常；
	// 3. 可以正常创建；
	// 4. 创建失败，报错混部资源不足；
	// 5. 创建失败，报错混部资源不足；
	It("[ant] colocation_mix_001: colocation pod in mix node.", func() {
		if len(nodesInfo) == 0 {
			Skip("no colocation node,skip")
		}
		framework.WaitForStableCluster(cs, masterNodes)
		nodeName := GetNodeThatCanRunColocationPod(f)
		Expect(nodeName).ToNot(BeNil())
		if col, ok := nodesInfo[nodeName].Labels[alipaysigmak8sapi.LabelIsColocation]; !ok || col != "true" {
			Skip("not colocation node,skip")
		}
		framework.Logf("get one colocation node to schedule, nodeName: %s", nodeName)

		nodeNameUpper := strings.ToUpper(nodeName)
		swarm.CreateOrUpdateNodeColocation(nodeNameUpper)
		swarm.EnsureNodeUpdateColocation(nodeNameUpper, true)
		defer swarm.DeleteNodeColocation(nodeNameUpper)

		allocatableCPU := nodeToAllocatableMapCPU[nodeName]
		allocatableColocationCPU := nodeToAllocatableMapColocationCPU[nodeName]
		allocatableMemory := nodeToAllocatableMapMem[nodeName]
		allocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		framework.Logf("allocatableCPU: %d", allocatableCPU)
		framework.Logf("allocatableColocationCPU: %d", allocatableColocationCPU)
		framework.Logf("allocatableMemory: %d", allocatableMemory)
		framework.Logf("allocatableDisk: %d", allocatableDisk)

		requestedCPU := allocatableCPU / 2
		requestedMemory := allocatableMemory / 8 // 保证一定能扩容出来
		requestedDisk := allocatableDisk / 8     // 保证一定能扩容出来

		// get nodeIP by node name
		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[api.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(),
			fmt.Sprintf("nodeName: %s, localInfoString is empty", nodeName))
		localInfo := &api.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(),
				fmt.Sprintf("nodeName: %s, localInfoString: %v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			{
				cpu:         requestedCPU, // 1/2 colocation CPU
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "default",
				colocation:      true,
			},
			{
				cpu:             requestedCPU, // 1/2 colocation CPU
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes, // sigma 3.1
				shouldScheduled: true,
				spreadStrategy:  "spread",
				colocation:      true,
			},
			{
				cpu:         1,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: false,
				spreadStrategy:  "default",
				colocation:      true,
			},
			{
				cpu:             1,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes, // sigma 3.1
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				shouldScheduled: false,
				spreadStrategy:  "spread",
				colocation:      true,
			},
			{
				cleanIndexes: []int{0}, // 删掉 2.0 的混部容器
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU, // 1/2 colocation CPU
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes, // sigma 3.1
				shouldScheduled: true,
				spreadStrategy:  "spread",
				colocation:      true,
			},
			{
				cpu:         1,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: false,
				spreadStrategy:  "default",
				colocation:      true,
			},
			{
				cleanIndexes: []int{1}, // 删掉 3.1 的混部容器
				requestType:  cleanResource,
			},
			{
				cpu:         1,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "default",
				colocation:      true,
			},
			{
				cpu:             requestedCPU, // 1/2 colocation CPU
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes, // sigma 3.1
				shouldScheduled: false,
				spreadStrategy:  "spread",
				colocation:      true,
			},
		}
		testContext := &testContext{
			testCases: tests,
			caseName:  "colocation_mix_001",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
			nodeName:  nodeName,
		}
		testContext.execTests()

		framework.Logf("do colocation test once again")
		tests2 := []resourceCase{
			tests[1],
			tests[0],
			tests[3],
			tests[2],
			tests[4],
			tests[5],
			tests[6],
			tests[7],
			tests[8],
			tests[9],
		}
		testContext.testCases = tests2
		testContext.execTests()
	})

	// 前提：
	// node 支持混部资源，并且可分配值大于 0；

	// 测试步骤：
	// 1. 获取可分配的混部 CPU 资源 A，非混部 CPU 资源 B，内存资源 C；
	// 2. 创建容器请求 1/2 A 的混部 CPU 资源，1/8 C 的内存资源，获取调度器内存状态；
	// 3. 创建容器请求 1/2 B 的非混部 CPU 资源，1/8 C 的内存资源；
	// 4. 创建容器请求 1/2 A 的混部 CPU 资源，1/8 C 的内存资源（内存资源还有富余，混部 CPU 资源分满）；
	// 5. 创建容器请求 1/2 B 的非混部 CPU 资源，1/8 C 的内存资源（内存资源还有富余，非混部 CPU 资源分满）；
	// 6. 继续创建新容器，请求 1 milli core 的混部 CPU 资源；
	// 7. 继续创建新容器，请求 1 milli core 的非混部 CPU 资源；
	// 8. 删掉一个混部容器和非混部容器，重复 4-7 步；

	// 期望结果：
	// 1. A、B、C 大于 0；
	// 2. 容器创建成功，调度器内存中资源状态正常；
	// 3. 容器创建成功，调度器内存中资源状态正常；
	// 4. 容器创建成功，调度器内存中资源状态正常；
	// 5. 容器创建成功，调度器内存中资源状态正常；
	// 6. 混部容器创建失败，调度器内存中资源状态正常（即混部 CPU 资源分配达到上限）；
	// 7. 非混部容器创建失败，调度器内存中资源状态正常（即非混部 CPU 资源分配达到上限）；
	// 8. 结果同 4-7；
	It("[ant] colocation_mix_002: colocation pod in mix node.", func() {
		if len(nodesInfo) == 0 {
			Skip("no colocation node,skip")
		}
		framework.WaitForStableCluster(cs, masterNodes)
		nodeName := GetNodeThatCanRunColocationPod(f)
		Expect(nodeName).ToNot(BeNil())
		if col, ok := nodesInfo[nodeName].Labels[alipaysigmak8sapi.LabelIsColocation]; !ok || col != "true" {
			Skip("not colocation node,skip")
		}
		framework.Logf("get one colocation node to schedule, nodeName: %s", nodeName)

		nodeNameUpper := strings.ToUpper(nodeName)
		swarm.CreateOrUpdateNodeColocation(nodeNameUpper)
		swarm.EnsureNodeUpdateColocation(nodeNameUpper, true)
		defer swarm.DeleteNodeColocation(nodeNameUpper)

		allocatableCPU := nodeToAllocatableMapCPU[nodeName]
		allocatableColocationCPU := nodeToAllocatableMapColocationCPU[nodeName]
		allocatableMemory := nodeToAllocatableMapMem[nodeName]
		allocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		framework.Logf("allocatableCPU: %d", allocatableCPU)
		framework.Logf("allocatableColocationCPU: %d", allocatableColocationCPU)
		framework.Logf("allocatableMemory: %d", allocatableMemory)
		framework.Logf("allocatableDisk: %d", allocatableDisk)

		requestedCPU := allocatableCPU / 2
		requestedMemory := allocatableMemory / 8 // 保证一定能扩容出来
		requestedDisk := allocatableDisk / 8     // 保证一定能扩容出来

		// get nodeIP by node name
		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[api.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(),
			fmt.Sprintf("nodeName: %s, localInfoString is empty", nodeName))
		localInfo := &api.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(),
				fmt.Sprintf("nodeName: %s, localInfoString: %v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			{
				cpu:         requestedCPU, // 1/2 colocation CPU
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "default",
				colocation:      true,
			},
			{
				cpu:         requestedCPU, // 1/2 CPU
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeKubernetes, // sigma 3.1
				affinityConfig: map[string][]string{
					api.LabelNodeIP: {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "spread",
				colocation:      false,
			},
			{
				cpu:         requestedCPU, // 1/2 colocation CPU
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeKubernetes, // sigma 3.1
				affinityConfig: map[string][]string{
					api.LabelNodeIP: {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "spread",
				colocation:      true,
			},
			{
				cpu:         requestedCPU, // 1/2 colocation CPU
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "default",
				colocation:      false,
			},
			{
				cpu:         1,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: false,
				spreadStrategy:  "default",
				colocation:      true,
			},
			{
				cpu:             1000,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes, // sigma 3.1
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				shouldScheduled: false,
				spreadStrategy:  "spread",
				colocation:      false,
			},
			{
				cpu:             1,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes, // sigma 3.1
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				shouldScheduled: false,
				spreadStrategy:  "spread",
				colocation:      true,
			},
			{
				cpu:         1,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: false,
				spreadStrategy:  "default",
				colocation:      false,
			},
			{
				cleanIndexes: []int{0}, // 删掉 2.0 的混部容器
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU, // 1/2 colocation CPU
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeKubernetes, // sigma 3.1
				affinityConfig: map[string][]string{
					api.LabelNodeIP: {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "spread",
				colocation:      true,
			},
			{
				cpu:         1,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: false,
				spreadStrategy:  "default",
				colocation:      true,
			},
			{
				cleanIndexes: []int{1}, // 删掉 3.1 的非混部容器
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU, // 1/2 CPU
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma, // sigma 3.1
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "spread",
				colocation:      false,
			},
			{
				cpu:         1,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: false,
				spreadStrategy:  "default",
				colocation:      false,
			},
		}
		testContext := &testContext{
			testCases: tests,
			caseName:  "colocation_mix_002",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
			nodeName:  nodeName,
		}
		testContext.execTests()
	})

	// 前提：
	// node 支持混部资源，并且可分配值大于 0；

	// 测试步骤：
	// 1. 获取可分配的混部 CPU 资源 A，非混部 CPU 资源 B，内存资源 C；
	// 2. 创建容器请求 1/4 A 的混部 CPU 资源，1/4 C 的内存资源，获取调度器内存状态；
	// 3. 创建容器请求 1/4 B 的非混部 CPU 资源，1/4 C 的内存资源；
	// 4. 创建容器请求 1/4 A 的混部 CPU 资源，1/4 C 的内存资源；
	// 5. 创建容器请求 1/4 B 的非混部 CPU 资源，1/4 C 的内存资源（CPU 资源还有富余，内存资源分满）；
	// 6. 继续创建新容器，请求 1 byte 的内存资源；
	// 7. 删掉一个混部容器和非混部容器，重复 4-6 步；

	// 期望结果：
	// 1. A、B、C 大于 0；
	// 2. 容器创建成功，调度器内存中资源状态正常；
	// 3. 容器创建成功，调度器内存中资源状态正常；
	// 4. 容器创建成功，调度器内存中资源状态正常；
	// 5. 容器创建成功，调度器内存中资源状态正常；
	// 6. 容器创建失败（内存资源不足），调度器内存中资源状态正常；
	// 7. 结果同 4-6；
	It("[ant] colocation_mix_003: colocation pod in mix node.", func() {
		if len(nodesInfo) == 0 {
			Skip("no colocation node,skip")
		}
		framework.WaitForStableCluster(cs, masterNodes)
		nodeName := GetNodeThatCanRunColocationPod(f)
		Expect(nodeName).ToNot(BeNil())
		if col, ok := nodesInfo[nodeName].Labels[alipaysigmak8sapi.LabelIsColocation]; !ok || col != "true" {
			Skip("not colocation node,skip")
		}
		framework.Logf("get one colocation node to schedule, nodeName: %s", nodeName)

		nodeNameUpper := strings.ToUpper(nodeName)
		swarm.CreateOrUpdateNodeColocation(nodeNameUpper)
		swarm.EnsureNodeUpdateColocation(nodeNameUpper, true)
		defer swarm.DeleteNodeColocation(nodeNameUpper)

		allocatableCPU := nodeToAllocatableMapCPU[nodeName]
		allocatableColocationCPU := nodeToAllocatableMapColocationCPU[nodeName]
		allocatableMemory := nodeToAllocatableMapMem[nodeName]
		allocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		framework.Logf("allocatableCPU: %d", allocatableCPU)
		framework.Logf("allocatableColocationCPU: %d", allocatableColocationCPU)
		framework.Logf("allocatableMemory: %d", allocatableMemory)
		framework.Logf("allocatableDisk: %d", allocatableDisk)

		requestedCPU := allocatableCPU / 4       // 保证一定能扩容出来
		requestedMemory := allocatableMemory / 4 // 重点测试内存额度满了
		requestedDisk := allocatableDisk / 8     // 保证一定能扩容出来

		// 最后一个内存，使用总量减去前 3 个
		lastMemory := allocatableMemory - requestedMemory*3
		framework.Logf("lastMemory: %d", lastMemory)

		// get nodeIP by node name
		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[api.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(),
			fmt.Sprintf("nodeName: %s, localInfoString is empty", nodeName))
		localInfo := &api.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(),
				fmt.Sprintf("nodeName: %s, localInfoString: %v parse error", nodeName, localInfoString))
		}

		tests := []resourceCase{
			{
				cpu:         requestedCPU,     // 1/4 colocation CPU
				mem:         requestedMemory,  // 1/4 memory
				ethstorage:  requestedDisk,    // 1/4 disk
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "default",
				colocation:      true,
			},
			{
				cpu:         requestedCPU,          // 1/4 CPU
				mem:         requestedMemory,       // 1/4 memory
				ethstorage:  requestedDisk,         // 1/4 disk
				requestType: requestTypeKubernetes, // sigma 3.1
				affinityConfig: map[string][]string{
					api.LabelNodeIP: {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "spread",
				colocation:      false,
			},
			{
				cpu:         requestedCPU,          // 1/4 colocation CPU
				mem:         requestedMemory,       // 1/4 memory
				ethstorage:  requestedDisk,         // 1/4 disk
				requestType: requestTypeKubernetes, // sigma 3.1
				affinityConfig: map[string][]string{
					api.LabelNodeIP: {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "spread",
				colocation:      true,
			},
			{
				cpu:         requestedCPU,     // 1/4 colocation CPU
				mem:         lastMemory,       // last memory
				ethstorage:  requestedDisk,    // 1/4 disk
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "default",
				colocation:      false,
			},
			{
				cpu:         requestedCPU,     // 1/4 CPU
				mem:         1024 * 1024,      // 1Mi (内存分满了)
				ethstorage:  requestedDisk,    // 1/4 disk
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: false,
				spreadStrategy:  "default",
				colocation:      true,
			},
			{
				cpu:         requestedCPU,          // 1/4 colocation CPU
				mem:         1024 * 1024,           // 1Mi (内存分满了)
				ethstorage:  requestedDisk,         // 1/4 disk
				requestType: requestTypeKubernetes, // sigma 3.1
				affinityConfig: map[string][]string{
					api.LabelNodeIP: {nodeIP},
				},
				shouldScheduled: false,
				spreadStrategy:  "spread",
				colocation:      false,
			},
			{
				cleanIndexes: []int{0}, // 删掉 2.0 的混部容器
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU,          // 1/4 colocation CPU
				mem:         requestedMemory,       // 1/4 memory (内存又可以分配了)
				ethstorage:  requestedDisk,         // 1/4 disk
				requestType: requestTypeKubernetes, // sigma 3.1
				affinityConfig: map[string][]string{
					api.LabelNodeIP: {nodeIP},
				},
				shouldScheduled: true,
				spreadStrategy:  "spread",
				colocation:      true,
			},
			{
				cpu:         1,
				mem:         requestedMemory,  // 1/4 memory (内存又分满了)
				ethstorage:  requestedDisk,    // 1/4 disk
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: false,
				spreadStrategy:  "default",
				colocation:      true,
			},
			{
				cleanIndexes: []int{1}, // 删掉 3.1 的非混部容器
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU,     // 1/4 CPU
				mem:         requestedMemory,  // 1/4 memory (内存又可以分配了)
				ethstorage:  requestedDisk,    // 1/4 disk
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},

				shouldScheduled: true,
				spreadStrategy:  "spread",
				colocation:      false,
			},
			{
				cpu:         1,
				mem:         requestedMemory,  // 1/4 memory (内存又分满了)
				ethstorage:  requestedDisk,    // 1/4
				requestType: requestTypeSigma, // sigma 2.0
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP},
				},
				shouldScheduled: false,
				spreadStrategy:  "default",
				colocation:      false,
			},
		}
		testContext := &testContext{
			testCases: tests,
			caseName:  "colocation_mix_003",
			cs:        cs,
			localInfo: localInfo,
			f:         f,
			nodeName:  nodeName,
		}
		testContext.execTests()
	})
})
