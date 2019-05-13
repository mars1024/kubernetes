package scheduler

import (
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	apis "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-3.1][sigma-scheduler][time-sharing][Serial]", func() {
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
			{
				allocatable, found := node.Status.Allocatable[v1.ResourceCPU]
				Expect(found).To(Equal(true))
				nodeToAllocatableMapCPU[node.Name] = allocatable.Value() * 1000
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
		if CurrentGinkgoTestDescription().Failed {
			DumpSchedulerState(f, 0)
		}
		DeleteSigmaContainer(f)
	})

	//验证能正常创建分时调度的Pod,资源计数能正常扣减
	//1.获取节点A的CPU资源t
	//2.使用t/2的资源创建完2个淘宝Pod
	//3.再创建一个使用t/2 CPU的淘宝Pod
	//4.销毁所有的淘宝Pod
	//5.再创建2个使用t/2CPU的淘宝Pod
	//6.再创建一个使用t/2的淘宝Pod
	It("[smoke][p0][bvt][ant] timesharing_001 If only taobao existed, scheduler should allocate taobao normally.", func() {
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

		requestedCPU := allocatableCPU / 2
		requestedMemory := allocatableMemory / 8
		requestedDisk := allocatableDisk / 8

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
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cleanIndexes: []int{2, 0, 1},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
		}
		testContext := &testContext{
			caseName:  "timesharing_001",
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

	// 验证淘宝和普通容器共存,资源的释放和回收
	It("[smoke][p0][bvt][ant] timesharing_002 If taobao and normal both existed, scheduler should allocate them with normal resource.", func() {
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

		requestedCPU := allocatableCPU / 2
		verifyCPU := requestedCPU / 2
		requestedMemory := allocatableMemory / 8
		requestedDisk := allocatableDisk / 8

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
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cleanIndexes: []int{3, 2, 1, 0},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
		}
		testContext := &testContext{
			caseName:  "timesharing_002",
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

	//验证能正常创建分时调度的Pod,资源计数能正常扣减
	//1.获取节点A的CPU资源t
	//2.使用t/2的资源创建完2个会员Pod
	//3.再创建一个使用t/2 CPU的会员Pod
	//4.销毁所有的会员Pod
	//5.再创建2个使用t/2CPU的会员Pod
	//6.再创建一个使用t/2的会员Pod
	It("[smoke][p0][bvt][ant] timesharing_003 If only ant member existed, scheduler should allocate ant member normally.", func() {
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

		requestedCPU := allocatableCPU / 2
		requestedMemory := allocatableMemory / 8
		requestedDisk := allocatableDisk / 8

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
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cleanIndexes: []int{2, 0, 1},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
		}
		testContext := &testContext{
			caseName:  "timesharing_003",
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

	// 验证会员和普通容器共存,资源的释放和回收
	It("[smoke][p0][bvt][ant] timesharing_004 If ant member and normal both existed, scheduler should allocate them with normal resource.", func() {
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

		requestedCPU := allocatableCPU / 2
		verifyCPU := requestedCPU / 2
		requestedMemory := allocatableMemory / 8
		requestedDisk := allocatableDisk / 8

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
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cleanIndexes: []int{3, 2, 1, 0},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
		}
		testContext := &testContext{
			caseName:  "timesharing_004",
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

	//验证淘宝和会员混合共享CPUShare
	It("[smoke][p0][bvt][ant] timesharing_005 If taobao and member both existed, scheduler should allocate them with sharing resource.", func() {
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

		requestedCPU := allocatableCPU
		verifyCPU := requestedCPU / 4
		requestedMemory := allocatableMemory / 8
		requestedDisk := allocatableDisk / 8

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
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cleanIndexes: []int{3, 2, 1, 0},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: false,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
		}
		testContext := &testContext{
			caseName:  "timesharing_005",
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

	//验证淘宝和会员混合共享CPUSet
	It("[smoke][p0][bvt][ant] timesharing_006 If taobao and member both existed, scheduler should allocate them with sharing resource.", func() {
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

		requestedCPU := allocatableCPU
		verifyCPU := requestedCPU / 4
		requestedMemory := allocatableMemory / 8
		requestedDisk := allocatableDisk / 8

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
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				spreadStrategy:  "spread",
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				spreadStrategy:  "spread",
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				spreadStrategy:  "spread",
				shouldScheduled: false,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				spreadStrategy:  "spread",
				shouldScheduled: false,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cleanIndexes: []int{3, 2, 1, 0},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				spreadStrategy:  "spread",
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				spreadStrategy:  "spread",
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				spreadStrategy:  "spread",
				shouldScheduled: false,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             verifyCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				spreadStrategy:  "spread",
				shouldScheduled: false,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
		}
		testContext := &testContext{
			caseName:  "timesharing_006",
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

	//验证淘宝和会员混合共享Memory
	It("[smoke][p0][bvt][ant] timesharing_007 If taobao and member both existed, scheduler should allocate them with sharing memory resource.", func() {
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

		requestedCPU := allocatableCPU / 8
		requestedMemory := allocatableMemory / 2
		verifyMemory := requestedMemory / 2
		requestedDisk := allocatableDisk / 8

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
				keepAliveMemory: verifyMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				keepAliveMemory: verifyMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             verifyMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             verifyMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cleanIndexes: []int{3, 2, 1, 0},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				keepAliveMemory: verifyMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				keepAliveMemory: verifyMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             verifyMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             verifyMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
		}
		testContext := &testContext{
			caseName:  "timesharing_007",
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
	//验证普通Pod inplace update到淘宝,和会员混合共享resource
	It("[smoke][p0][bvt][ant] timesharing_008 update normal pod to taobao, and share resource with ant member.", func() {
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

		podCPU1 := allocatableCPU
		podMemory1 := allocatableMemory / 4
		podDisk1 := allocatableDisk / 4

		By("Request a pod with CPU/Memory/EphemeralStorage.")
		podsToDelete := []*v1.Pod{}

		defer func() {
			for _, pod := range podsToDelete {
				if pod == nil {
					continue
				}
				err := util.DeletePod(f.ClientSet, pod)
				if err != nil {
					framework.Logf("delete pod err :%+v", err)
				}
			}
		}()
		resourceList1 := v1.ResourceList{
			v1.ResourceCPU:              *resource.NewMilliQuantity(podCPU1, resource.DecimalSI),
			v1.ResourceMemory:           *resource.NewQuantity(podMemory1, resource.BinarySI),
			v1.ResourceEphemeralStorage: *resource.NewQuantity(podDisk1, resource.BinarySI),
		}

		resourceRequirements1 := v1.ResourceRequirements{
			Limits:   resourceList1,
			Requests: resourceList1,
		}

		//1. 创建普通Pod,使用全部CPU
		name := "inplace-update-" + string(uuid.NewUUID()) + "-1"
		pod := createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements1,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-1 to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		podsToDelete = append(podsToDelete, pod)
		Expect(err).NotTo(HaveOccurred())

		//check resource is used up
		resourceList1 = v1.ResourceList{
			v1.ResourceCPU:              *resource.NewMilliQuantity(podCPU1/4, resource.DecimalSI),
			v1.ResourceMemory:           *resource.NewQuantity(podMemory1, resource.BinarySI),
			v1.ResourceEphemeralStorage: *resource.NewQuantity(podDisk1, resource.BinarySI),
		}

		resourceRequirements2 := v1.ResourceRequirements{
			Limits:   resourceList1,
			Requests: resourceList1,
		}

		// 2. 创建普通Pod,预期失败
		name = "inplace-update-" + string(uuid.NewUUID()) + "-2"
		tmpPod := createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-2 to be scheduled failed.")
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, tmpPod.Name, tmpPod.Namespace)
		Expect(err).To(BeNil())

		err = util.DeletePod(f.ClientSet, tmpPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		// 3. 创建taobao Pod,预期失败
		name = "inplace-update-" + string(uuid.NewUUID()) + "-3"
		tmpPod = createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-3 to be scheduled failed.")
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, tmpPod.Name, tmpPod.Namespace)
		Expect(err).To(BeNil())

		err = util.DeletePod(f.ClientSet, tmpPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		// 4. 创建ant member Pod,预期失败
		name = "inplace-update-" + string(uuid.NewUUID()) + "-4"
		tmpPod = createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-4 to be scheduled failed.")
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, tmpPod.Name, tmpPod.Namespace)
		Expect(err).To(BeNil())

		err = util.DeletePod(f.ClientSet, tmpPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		// 5. update 普通Pod到taobao,预期成功
		doUpdateWithTimeSharingSuccess(cs, pod, apis.PodPromotionTypeTaobao.String())

		// 6. 创建taobaoPod(1/4CPU)，预期失败
		name = "inplace-update-" + string(uuid.NewUUID()) + "-5"
		tmpPod = createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-5 to be scheduled failed.")
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, tmpPod.Name, tmpPod.Namespace)
		Expect(err).To(BeNil())

		err = util.DeletePod(f.ClientSet, tmpPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		// 7. 创建ant member Pod(全部CPU)，预期成功
		name = "inplace-update-" + string(uuid.NewUUID()) + "-6"
		tmpPod = createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements1,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-6 to be scheduled successfully.")
		err = framework.WaitTimeoutForPodRunningInNamespace(cs, tmpPod.Name, tmpPod.Namespace, waitForPodRunningTimeout)
		podsToDelete = append(podsToDelete, tmpPod)
		Expect(err).NotTo(HaveOccurred())

		// 8. 创建ant member Pod, (1/4 CPU)，预期失败
		name = "inplace-update-" + string(uuid.NewUUID()) + "-7"
		tmpPod = createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-7 to be scheduled failed.")
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, tmpPod.Name, tmpPod.Namespace)
		Expect(err).To(BeNil())

		err = util.DeletePod(f.ClientSet, tmpPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

	})

	//验证普通Pod inplace update到会员,和淘宝混合共享resource
	It("[smoke][p0][bvt][ant] timesharing_009 update normal pod to ant member, and share resource with taobao.", func() {
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

		podCPU1 := allocatableCPU
		podMemory1 := allocatableMemory / 4
		podDisk1 := allocatableDisk / 4

		By("Request a pod with CPU/Memory/EphemeralStorage.")
		podsToDelete := []*v1.Pod{}

		defer func() {
			for _, pod := range podsToDelete {
				if pod == nil {
					continue
				}
				err := util.DeletePod(f.ClientSet, pod)
				if err != nil {
					framework.Logf("delete pod err :%+v", err)
				}
			}
		}()
		resourceList1 := v1.ResourceList{
			v1.ResourceCPU:              *resource.NewMilliQuantity(podCPU1, resource.DecimalSI),
			v1.ResourceMemory:           *resource.NewQuantity(podMemory1, resource.BinarySI),
			v1.ResourceEphemeralStorage: *resource.NewQuantity(podDisk1, resource.BinarySI),
		}

		resourceRequirements1 := v1.ResourceRequirements{
			Limits:   resourceList1,
			Requests: resourceList1,
		}

		//1. 创建普通Pod,使用全部CPU
		name := "inplace-update-" + string(uuid.NewUUID()) + "-1"
		pod := createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements1,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-1 to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		podsToDelete = append(podsToDelete, pod)
		Expect(err).NotTo(HaveOccurred())

		//check resource is used up
		resourceList1 = v1.ResourceList{
			v1.ResourceCPU:              *resource.NewMilliQuantity(podCPU1/4, resource.DecimalSI),
			v1.ResourceMemory:           *resource.NewQuantity(podMemory1, resource.BinarySI),
			v1.ResourceEphemeralStorage: *resource.NewQuantity(podDisk1, resource.BinarySI),
		}

		resourceRequirements2 := v1.ResourceRequirements{
			Limits:   resourceList1,
			Requests: resourceList1,
		}

		// 2. 创建普通Pod,预期失败
		name = "inplace-update-" + string(uuid.NewUUID()) + "-2"
		tmpPod := createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-2 to be scheduled failed.")
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, tmpPod.Name, tmpPod.Namespace)
		Expect(err).To(BeNil())

		err = util.DeletePod(f.ClientSet, tmpPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		// 3. 创建ant member Pod,预期失败
		name = "inplace-update-" + string(uuid.NewUUID()) + "-3"
		tmpPod = createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-3 to be scheduled failed.")
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, tmpPod.Name, tmpPod.Namespace)
		Expect(err).To(BeNil())

		err = util.DeletePod(f.ClientSet, tmpPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		// 4. 创建taobao Pod,预期失败
		name = "inplace-update-" + string(uuid.NewUUID()) + "-4"
		tmpPod = createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-4 to be scheduled failed.")
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, tmpPod.Name, tmpPod.Namespace)
		Expect(err).To(BeNil())

		err = util.DeletePod(f.ClientSet, tmpPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		// 5. update 普通Pod到ant member,预期成功
		doUpdateWithTimeSharingSuccess(cs, pod, apis.PodPromotionTypeAntMember.String())

		// 6. 创建ant member Pod(1/4CPU)，预期失败
		name = "inplace-update-" + string(uuid.NewUUID()) + "-5"
		tmpPod = createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-5 to be scheduled failed.")
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, tmpPod.Name, tmpPod.Namespace)
		Expect(err).To(BeNil())

		err = util.DeletePod(f.ClientSet, tmpPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		// 7. 创建taobao member Pod(全部CPU)，预期成功
		name = "inplace-update-" + string(uuid.NewUUID()) + "-6"
		tmpPod = createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements1,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-6 to be scheduled successfully.")
		err = framework.WaitTimeoutForPodRunningInNamespace(cs, tmpPod.Name, tmpPod.Namespace, waitForPodRunningTimeout)
		podsToDelete = append(podsToDelete, tmpPod)
		Expect(err).NotTo(HaveOccurred())

		// 8. 创建taobao Pod, (1/4 CPU)，预期失败
		name = "inplace-update-" + string(uuid.NewUUID()) + "-7"
		tmpPod = createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-7 to be scheduled failed.")
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, tmpPod.Name, tmpPod.Namespace)
		Expect(err).To(BeNil())

		err = util.DeletePod(f.ClientSet, tmpPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

	})
	It("[smoke][p0][bvt][ant] timesharing_010 update taobao pod to normal", func() {
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

		podCPU1 := allocatableCPU
		podMemory1 := allocatableMemory / 4
		podDisk1 := allocatableDisk / 4

		By("Request a pod with CPU/Memory/EphemeralStorage.")
		podsToDelete := []*v1.Pod{}

		defer func() {
			for _, pod := range podsToDelete {
				if pod == nil {
					continue
				}
				err := util.DeletePod(f.ClientSet, pod)
				if err != nil {
					framework.Logf("delete pod err :%+v", err)
				}
			}
		}()
		resourceList1 := v1.ResourceList{
			v1.ResourceCPU:              *resource.NewMilliQuantity(podCPU1, resource.DecimalSI),
			v1.ResourceMemory:           *resource.NewQuantity(podMemory1, resource.BinarySI),
			v1.ResourceEphemeralStorage: *resource.NewQuantity(podDisk1, resource.BinarySI),
		}

		resourceRequirements1 := v1.ResourceRequirements{
			Limits:   resourceList1,
			Requests: resourceList1,
		}

		//1. 创建taobaoPod,使用全部CPU
		name := "inplace-update-" + string(uuid.NewUUID()) + "-1"
		pod := createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements1,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-1 to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		podsToDelete = append(podsToDelete, pod)
		Expect(err).NotTo(HaveOccurred())

		resourceList1 = v1.ResourceList{
			v1.ResourceCPU:              *resource.NewMilliQuantity(podCPU1/4, resource.DecimalSI),
			v1.ResourceMemory:           *resource.NewQuantity(podMemory1, resource.BinarySI),
			v1.ResourceEphemeralStorage: *resource.NewQuantity(podDisk1, resource.BinarySI),
		}

		resourceRequirements2 := v1.ResourceRequirements{
			Limits:   resourceList1,
			Requests: resourceList1,
		}

		// 2. 创建普通 Pod(1/4CPU)，预期失败
		name = "inplace-update-" + string(uuid.NewUUID()) + "-2"
		tmpPod := createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-2 to be scheduled failed.")
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, tmpPod.Name, tmpPod.Namespace)
		Expect(err).To(BeNil())

		err = util.DeletePod(f.ClientSet, tmpPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		//3.update 到普通,预期成功
		doUpdateWithTimeSharingSuccess(cs, pod, apis.PodPromotionTypeNone.String())

		//4. 创建普通 Pod(1/4CPU)，预期失败
		name = "inplace-update-" + string(uuid.NewUUID()) + "-3"
		tmpPod = createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-3 to be scheduled failed.")
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, tmpPod.Name, tmpPod.Namespace)
		Expect(err).To(BeNil())

		err = util.DeletePod(f.ClientSet, tmpPod)
		Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")

		//5.删除所有pod
		for _, pod := range podsToDelete {
			if pod == nil {
				continue
			}
			err := util.DeletePod(f.ClientSet, pod)
			if err != nil {
				framework.Logf("delete pod err :%+v", err)
			}
		}
		podsToDelete = []*v1.Pod{}

		//6. 创建taobaoPod,使用全部CPU
		name = "inplace-update-" + string(uuid.NewUUID()) + "-4"
		pod = createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements1,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeTaobao.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-4 to be scheduled successfully.")
		err = framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		podsToDelete = append(podsToDelete, pod)
		Expect(err).NotTo(HaveOccurred())

		//7. 创建ant member Pod,使用(1/4)CPU
		name = "inplace-update-" + string(uuid.NewUUID()) + "-5"
		pod2 := createPausePod(f, pausePodConfig{
			Name:      name,
			Resources: &resourceRequirements2,
			Annotations: map[string]string{
				sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(
					name, sigmak8sapi.SpreadStrategySameCoreFirst),
			},
			Labels:   map[string]string{apis.LabelPodPromotionType: apis.PodPromotionTypeAntMember.String()},
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
		})
		framework.Logf("expect pod-5 to be scheduled successfully.")
		err = framework.WaitTimeoutForPodRunningInNamespace(cs, pod2.Name, pod2.Namespace, waitForPodRunningTimeout)
		podsToDelete = append(podsToDelete, pod2)
		Expect(err).NotTo(HaveOccurred())

		//8.pod update 到普通,预期失败
		doUpdateWithTimeSharingFailed(cs, pod, apis.PodPromotionTypeNone.String())
	})
})

func doUpdateWithTimeSharingFailed(client clientset.Interface, pod *v1.Pod, promotionType string) {
	pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	if err != nil {
		Expect(err).NotTo(HaveOccurred(), "get scheduled pod should succeed")
	}
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState] =
		sigmak8sapi.InplaceUpdateStateCreated

	pod.Labels[apis.LabelPodPromotionType] = promotionType

	framework.Logf("doUpdateWithNewResource, type: %+v", promotionType)
	doUpdate(client, pod)
	err = wait.PollImmediate(5*time.Second, 5*time.Minute, checkInplaceUpdateIsNotAccepted(client, pod))
	Expect(err).NotTo(HaveOccurred(), "inplace update should failed")

}
func doUpdateWithTimeSharingSuccess(client clientset.Interface, pod *v1.Pod, promotionType string) {
	pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	if err != nil {
		Expect(err).NotTo(HaveOccurred(), "get scheduled pod should succeed")
	}

	// increase resources of pod
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	pod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState] =
		sigmak8sapi.InplaceUpdateStateCreated

	pod.Labels[apis.LabelPodPromotionType] = promotionType

	framework.Logf("doUpdateWithNewResource, type: %+v", promotionType)
	doUpdate(client, pod)

	err = wait.PollImmediate(5*time.Second, 5*time.Minute, checkInplaceUpdateIsAccepted(client, pod))
	Expect(err).NotTo(HaveOccurred(), "inplace update should succeed")
}

func doUpdate(client clientset.Interface, pod *v1.Pod) {
	_, err := client.CoreV1().Pods(pod.Namespace).Update(pod)
	if err != nil {
		Expect(err).NotTo(HaveOccurred(), "update pod should succeed")
	}
}
func checkInplaceUpdateIsNotAccepted(client clientset.Interface, pod *v1.Pod) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		state, ok := pod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState]
		if !ok {
			return false, nil
		}

		if state == sigmak8sapi.InplaceUpdateStateAccepted ||
			state == sigmak8sapi.InplaceUpdateStateSucceeded {
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
