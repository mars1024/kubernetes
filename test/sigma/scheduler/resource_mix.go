package scheduler

import (
	"fmt"
	"math"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"

	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/env"
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
			// 从 etcd localinfo 获取 cpu/mem/disk
			// 因为存在 sigma 和 k8s 资源不一致问题，以 sigma localinfo 为准
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

			if env.Tester == env.TesterAnt {
				plans := swarm.GetAllocPlans(node.Name)
				for _, plan := range plans {
					By(fmt.Sprintf("Get swarm alloc plan: nodename %s, cpuquota %d, memory %d, disk %d",
						node.Name, plan.CpuQuota, plan.Memory, plan.DiskQuota))
					if plan.CpusetMode != "cpushare" {
						nodeToAllocatableMapCPU[node.Name] -= int64(10 * plan.CpuQuota)
					}
					nodeToAllocatableMapMem[node.Name] -= plan.Memory
					rootHostPath := plan.DiskQuota["/"].HostPath
					for _, disk := range plan.DiskQuota {
						if disk.HostPath == rootHostPath {
							nodeToAllocatableMapEphemeralStorage[node.Name] -= disk.DiskSize
						}
					}
				}
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

	// 非超卖场景下，验证混合链路的资源校验
	// 步骤 要求每个容器分配的 CPU 个数不能低于 2 个，否则这个 case 会验证失败
	// 1.  1/2 整机核 cpushare k8s（预期成功）
	// 2.  剩余 整机核 cpushare sigma（预期成功）
	It("[smoke][p0] resourceMix001 CPU、内存、磁盘都满足场景下，验证分配成功", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		cpuSetModeLabel := "CpuSetMode"
		cpuSetModeLabels := map[string]string{
			cpuSetModeLabel: "share",
		}

		sigmaHostSN := strings.ToUpper(nodeName)
		swarm.CreateOrUpdateNodeLabel(sigmaHostSN, cpuSetModeLabels)
		swarm.EnsureNodeHasLabels(sigmaHostSN, cpuSetModeLabels)
		defer swarm.DeleteNodeLabels(sigmaHostSN, cpuSetModeLabel)

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		requestedCPU := AllocatableCPU / 2
		requestedMemory := AllocatableMemory / 2
		requestedDisk := AllocatableDisk / 2

		leftCPU := AllocatableCPU - requestedCPU
		leftMemory := AllocatableMemory - requestedMemory
		leftDisk := AllocatableDisk - requestedDisk

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cpu:             leftCPU,
				mem:             leftMemory,
				ethstorage:      leftDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				cpushare:        true,
			},
		}

		testContext := &testContext{
			caseName:  "resourceMix001",
			cs:        cs,
			localInfo: nil,
			f:         f,
			testCases: tests,
			nodeName:  nodeName,
		}

		testContext.execTests()
	})

	// 非超卖场景下，验证混合链路的资源校验（内存不足）
	// 步骤 要求每个容器分配的 CPU 个数不能低于 2 个，否则这个 case 会验证失败
	// 1.  1/2 整机资源 cpushare k8s（预期成功）
	// 2.  剩余 CPU 资源 - 1000, 剩余内存资源 - 1024, 剩余磁盘资源 - 1024, cpushare sigma（预期成功）
	// 3.  CPU 资源 1000, 内存资源 1024 + 1, 磁盘资源 1024, sigma（预期失败）
	It("resourceMix002 校验内存不足", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		cpuSetModeLabel := "CpuSetMode"
		cpuSetModeLabels := map[string]string{
			cpuSetModeLabel: "share",
		}

		sigmaHostSN := strings.ToUpper(nodeName)
		swarm.CreateOrUpdateNodeLabel(sigmaHostSN, cpuSetModeLabels)
		swarm.EnsureNodeHasLabels(sigmaHostSN, cpuSetModeLabels)
		defer swarm.DeleteNodeLabels(sigmaHostSN, cpuSetModeLabel)

		nodeAffinityKey := "node-for-resource-e2e-test"
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]
		requestedCPU := AllocatableCPU / 2
		requestedMemory := AllocatableMemory / 2
		requestedDisk := AllocatableDisk / 2
		leftCPU := AllocatableCPU - requestedCPU
		leftMemory := AllocatableMemory - requestedMemory
		leftDisk := AllocatableDisk - requestedDisk

		// TODO: 这里是 2fea0ca302ee352300fc20d24329718f5846afcd 添加的
		// 但是 sigma localinfo 和 kubelet 上报的内存不一致产生了错误，所以改回去
		//localInfo := swarm.GetNode(nodeName).LocalInfo
		//totalMem := localInfo.Memory
		//totalDisk := int64(0)
		//for _, disk := range localInfo.DiskInfos {
		//	if disk.IsBootDisk {
		//		totalDisk = disk.Size
		//		break
		//	}
		//}
		//leftMemory := totalMem - requestedMemory
		//leftDisk := totalDisk - requestedDisk
		//leftCPU := AllocatableCPU - requestedCPU
		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cpu:             leftCPU - 1000,
				mem:             leftMemory - 1024,
				ethstorage:      leftDisk - 1024,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cpu:             1000,
				mem:             1024 + 1,
				ethstorage:      1024,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: false,
				cpushare:        true,
			},
		}

		testContext := &testContext{
			caseName:  "resourceMix002",
			cs:        cs,
			localInfo: nil,
			f:         f,
			testCases: tests,
			nodeName:  nodeName,
		}
		// 清理创建出来的 Pod 数据
		testContext.execTests()
	})

	// 非超卖场景下，验证混合链路的资源校验 磁盘不足
	// 步骤 要求每个容器分配的 cpu 个数不能低于 2 个，否则这个 case 会验证失败
	// 1.  1/2 整机资源 cpushare k8s（预期成功）
	// 2.  剩余 CPU 资源 - 1000, 剩余内存资源 - 1024, 剩余磁盘资源 - 1024, cpushare sigma（预期成功）
	// 3.  CPU 资源 1000, 内存资源 1024, 磁盘资源 1024 + 1 sigma（预期失败）
	It("resourceMix003 校验磁盘不足", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		cpuSetModeLabel := "CpuSetMode"
		cpuSetModeLabels := map[string]string{
			cpuSetModeLabel: "share",
		}

		sigmaHostSN := strings.ToUpper(nodeName)
		swarm.CreateOrUpdateNodeLabel(sigmaHostSN, cpuSetModeLabels)
		swarm.EnsureNodeHasLabels(sigmaHostSN, cpuSetModeLabels)
		defer swarm.DeleteNodeLabels(sigmaHostSN, cpuSetModeLabel)

		nodeAffinityKey := "node-for-resource-e2e-test"
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]
		logrus.Infof("allocabal cpu:%d,mem:%d,systemdisk:%d", AllocatableCPU, AllocatableMemory, AllocatableDisk)
		requestedCPU := AllocatableCPU / 2
		requestedMemory := AllocatableMemory / 2
		requestedDisk := AllocatableDisk / 2
		logrus.Infof("request cpu:%d,mem:%d,systemdisk:%d", requestedCPU, requestedMemory, requestedDisk)

		leftCPU := AllocatableCPU - requestedCPU
		leftMemory := AllocatableMemory - requestedMemory
		leftDisk := AllocatableDisk - requestedDisk
		logrus.Infof("left cpu:%d,mem:%d,systemdisk:%d", leftCPU, leftMemory, leftDisk)

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cpu:             leftCPU - 1000,
				mem:             leftMemory - 1024,
				ethstorage:      leftDisk - 1024,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cpu:             1000,
				mem:             1024,
				ethstorage:      1024 + 1,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: false,
				cpushare:        true,
			},
		}

		testContext := &testContext{
			caseName:  "resourceMix003",
			cs:        cs,
			localInfo: nil,
			f:         f,
			testCases: tests,
			nodeName:  nodeName,
		}
		// 清理创建出来的 Pod 数据
		testContext.execTests()
	})

	// 混合链路 Container/Pod AllocateMode=host 验证
	// 步骤：
	// 1. host 模式 sigma （预期成功）
	// 2. 1/4 资源 k8s （预期失败）
	// 3. 销毁 1, 2 (预期成功)
	// 4. 1/4 资源 k8s （预期成功）
	// 5. 销毁 4 (预期成功)
	// 6. host 模式 sigma（预期成功）
	It("resourceMix004 sigma 2.0 AllocateMode=host and k8s.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		cpuSetModeLabel := "CpuSetMode"
		cpuSetModeLabels := map[string]string{
			cpuSetModeLabel: "share",
		}

		sigmaHostSN := strings.ToUpper(nodeName)
		swarm.CreateOrUpdateNodeLabel(sigmaHostSN, cpuSetModeLabels)
		swarm.EnsureNodeHasLabels(sigmaHostSN, cpuSetModeLabels)
		defer swarm.DeleteNodeLabels(sigmaHostSN, cpuSetModeLabel)

		nodeAffinityKey := "node-for-resource-e2e-test"
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]
		requestedCPU := AllocatableCPU / 4
		requestedMemory := AllocatableMemory / 4
		requestedDisk := AllocatableDisk / 4

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.AllocateMode": {"host"}},
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			{
				cleanIndexes: []int{0, 1},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cleanIndexes: []int{2},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}, "ali.AllocateMode": {"host"}},
				shouldScheduled: false,
				cpushare:        true,
			},
		}

		testContext := &testContext{
			caseName:  "resourceMix003",
			cs:        cs,
			localInfo: nil,
			f:         f,
			testCases: tests,
			nodeName:  nodeName,
		}

		testContext.execTests()
	})

	// 混合链路sigma多Volume对k8s ephemeral storage影响验证
	// 步骤：
	//1. 创建 sigma 2.0 的容器，并创建一个额外的数据卷,预期创建成功
	//		ali.Disk.1.MountPoint=/data1
	//		ali.Disk.1.AllowUseHostBootDisk=true
	//		ali.Disk.1.Size=Xm
	//2. 创建 k8s PodA，设置好余下的磁盘空间，预期创建成功
	//3. 创建 k8s PodB，磁盘quota采用sigma2.0的/data1磁盘空间，预期创建失败
	//4. 删除 sigma 2.0 的容器，使用 sigma 2.0 的 quota 创建 k8s PodB,预期创建成功
	//5. 删除 PodA，使用剩余的磁盘 quota 创建 sigma 2.0 的容器，预期创建成功
	//6. 删除PodB和第5步创建的sigma2.0容器，创建2个volume /data1 /data2的sigma2.0容器,预期创建成功
	//7. 创建PodC,磁盘quota为AllocatableDisk - requestedDisk - requestedVolume,预期创建失败
	//8. 创建PodD，磁盘quota为AllocatableDisk - requestedDisk - requestedVolume*2,预期创建失败
	//9. 删除第6步创建的容器，使用磁盘quota为requestedDisk + requestedVolume*2创建容器E，预期成功
	It("[ant] resourceMix005 sigma 2.0 volume and k8s.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		if env.Tester == env.TesterJituan {
			Skip("skip this test")
		}

		nodeAffinityKey := "node-for-resource-e2e-test"
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]
		requestedCPU := AllocatableCPU / 4
		requestedMemory := AllocatableMemory / 4
		requestedDisk := AllocatableDisk / 4
		requestedVolume := AllocatableDisk / 4
		requestedVolumeArgs := requestedVolume

		// 仅支持在单盘机器上运行测试
		// 做多盘检测
		// 发现IsBootDisk=false，且Type!=tmpfs/devtmpfs/overlay&&
		// !FileSystem.hasprefix(/dev/nbd ) && !FileSystem.hasprefix(/dev/vrbd ) && DiskType!=remote 的盘，则为多盘
		node := swarm.GetNode(nodeName)
		isMultiDisk := false
		if node.LocalInfo != nil {
			for _, disk := range node.LocalInfo.DiskInfos {
				if !disk.IsBootDisk && disk.Type != "tmpfs" &&
					disk.Type != "devtmpfs" && disk.Type != "overlay" &&
					!strings.HasPrefix(disk.FileSystem, "/dev/nbd") &&
					!strings.HasPrefix(disk.FileSystem, "/dev/vrbd") &&
					disk.DiskType != "remote" &&
					requestedDisk <= int64(math.Floor(float64(disk.Size)*node.LogicInfo.DiskOverQuota)) {
					isMultiDisk = true
					break
				}
			}
		}
		if isMultiDisk {
			Skip("Case should run on single disk host,so skip.")
		}
		// 对于 volume size，集团 2.0 环境也需要在 disksize 后加一位
		if env.Tester == env.TesterJituan {
			requestedVolumeArgs *= 10
		}
		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address

		tests := []resourceCase{
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":              {nodeIP},
					"ali.Disk.1.MountPoint":           {"/data1"},
					"ali.Disk.1.AllowUseHostBootDisk": {"true"},
					"ali.Disk.1.Size":                 {fmt.Sprintf("%d", requestedVolumeArgs)}},
				shouldScheduled: true,
				cpushare:        false,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      AllocatableDisk - requestedDisk - requestedVolume,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			{
				cleanIndexes: []int{0, 2},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk + requestedVolume,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cleanIndexes: []int{1},
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  AllocatableDisk - requestedDisk - requestedVolume,
				requestType: requestTypeSigma,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				cpushare:        false,
			},
			{
				cleanIndexes: []int{4, 6},
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":              {nodeIP},
					"ali.Disk.1.MountPoint":           {"/data1"},
					"ali.Disk.1.AllowUseHostBootDisk": {"true"},
					"ali.Disk.1.Exclusive":            {"share"},
					"ali.Disk.1.Size":                 {fmt.Sprintf("%d", requestedVolumeArgs)},
					"ali.Disk.2.MountPoint":           {"/data2"},
					"ali.Disk.2.AllowUseHostBootDisk": {"true"},
					"ali.Disk.2.Exclusive":            {"share"},
					"ali.Disk.2.Size":                 {fmt.Sprintf("%d", requestedVolumeArgs)}},
				shouldScheduled: true,
				cpushare:        false,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      AllocatableDisk - requestedDisk - requestedVolume,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: false,
				cpushare:        true,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      AllocatableDisk - requestedDisk - requestedVolume*2,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cleanIndexes: []int{8},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk + requestedVolume*2,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
		}

		testContext := &testContext{
			caseName:  "resourceMix005",
			cs:        cs,
			localInfo: nil,
			f:         f,
			testCases: tests,
			nodeName:  nodeName,
		}

		testContext.execTests()
	})

	// 混合链路sigma Update对k8s memory影响验证
	// 步骤：
	//  使用 Memory 做参考资源，线上只有 Memory 的 Update
	//  1. 创建 2.0 的容器 A,预期创建成功
	//  2. 使用余下的 quota 创建 k8s PodB,预期创建成功
	//  3. 使用 2.0 API，将 quota 加大第一步剩下的资源 update 容器 A,预期update失败
	//  4. 使用 2.0 API，将 quota 减小 update 容器 A，预期update成功
	//  5. 使用剩余 quota 创建 PodC，预期创建成功
	//  6. 删除容器 A，使用update后的 quota 创建容器 D，预期创建成功
	It("[ant] resourceMix006 sigma 2.0 update and k8s.", func() {
		// 集团不支持 update 修改 memory，只支持 update env/label/cpu
		// memory 仅混布链路支持，用 rebind
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		nodeAffinityKey := "node-for-resource-e2e-test"
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]
		requestedCPU := AllocatableCPU / 4
		requestedMemory := AllocatableMemory / 2
		requestedDisk := AllocatableDisk / 4

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address

		tests := []resourceCase{
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				cpushare:        false,
			},
			{
				cpu:             requestedCPU,
				mem:             AllocatableMemory - requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				mem: AllocatableMemory,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP}},
				requestType:     requestTypeSigmaUpdate,
				updateIndex:     0,
				shouldScheduled: false,
				cpushare:        false,
			},
			{
				mem: requestedMemory / 2,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps": {nodeIP}},
				requestType:     requestTypeSigmaUpdate,
				updateIndex:     0,
				shouldScheduled: true,
				cpushare:        false,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory - requestedMemory/2,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cleanIndexes: []int{0},
				requestType:  cleanResource,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory / 2,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
		}
		testContext := &testContext{
			caseName:       "resourceMix006",
			cs:             cs,
			localInfo:      nil,
			f:              f,
			testCases:      tests,
			nodeName:       nodeName,
			isSigmaNotMock: true,
		}
		testContext.execTests()

	})

	//混合链路preview：
	//指定IP=A preview
	//1. 挑选一台空机器A，使用固定的规格调用preview接口，IncreaseReplica=10000，得到可以创建的容器数c；
	//2.使用固定的规格调用preview接口设置IncreaseReplica=c+1，得到可以创建的容器c1；
	//3.在机器A上创建一个容器，调用preview接口，IncreaseReplica=10000，得到可以创建的容器数c2
	//4.在机器A上创建一个Pod，调用preview接口，IncreaseReplica=10000，得到可以创建的容器数c3
	//5.销毁所有容器和Pod，调用preview接口，IncreaseReplica=10000，得到可以创建的容器数c4
	//6.在机器A上创建N个容器，调用preview接口，IncreaseReplica=10000，得到可以创建的容器数c5
	//7.在机器A上销毁一个容器，调用preview接口，IncreaseReplica=10000，得到可以创建的容器数c6
	//8.销毁所有容器，调用preview接口，IncreaseReplica=10000，得到可以创建的容器数c7
	//9.在机器A上创建N个Pod，调用preview接口，IncreaseReplica=10000，得到可以创建的容器数c8
	//10.在机器A上销毁一个Pod，调用preview接口，IncreaseReplica=10000，得到可以创建的容器数c9
	//11.销毁所有Pod，调用preview接口，IncreaseReplica=10000，得到可以创建的容器数c10
	//
	//预期：
	//c=c1=c2+1=c3+2=c4=c5+N=c6+N-1=c7=c8+N=c9+N-1=c10
	It("resourceMix007 sigma 2.0 preview and k8s with specified IP.", func() {
		if env.Tester != env.TesterAnt {
			Skip("ant sigma 2.0 preview")
		}
		dp := "max-instance-filter-resourcemix008"
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		nodeAffinityKey := "node-for-resource-e2e-test"
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]
		requestedCPU := AllocatableCPU / 4
		requestedMemory := AllocatableMemory / 4
		requestedDisk := AllocatableDisk / 4

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address

		tests := []resourceCase{
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    4, //c
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"5"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    4, //c1
			},
			{ //index 2
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"1"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    3, //c2
			},
			{ //index 4
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{api.LabelDeployUnit: dp},
				shouldScheduled: true,
				cpushare:        false,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    2, //c3
			},
			{
				cleanIndexes: []int{2, 4},
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    4, //c4
			},
			{ //index 8
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"1"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
			},
			{ //index 9
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"1"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    2, //c5
			},
			{
				cleanIndexes: []int{8},
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    3, //c6
			},
			{
				cleanIndexes: []int{9},
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    4, //c7
			},
			{ //index 15
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{api.LabelDeployUnit: dp},
				shouldScheduled: true,
				cpushare:        false,
			},
			{ //index 16
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				affinityConfig:  map[string][]string{nodeAffinityKey: {nodeName}},
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{api.LabelDeployUnit: dp},
				shouldScheduled: true,
				cpushare:        false,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    2, //c8
			},
			{
				cleanIndexes: []int{15},
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    3, //c9
			},
			{
				cleanIndexes: []int{16},
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.SpecifiedNcIps":     {nodeIP},
					"ali.MaxInstancePerHost": {"100"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    4, //c10
			},
		}
		testContext := &testContext{
			caseName:  "resourceMix007",
			cs:        cs,
			localInfo: nil,
			f:         f,
			testCases: tests,
			nodeName:  nodeName,
		}
		testContext.execTests()
	})

	//混合链路preview：
	//非指定IP preview
	//1.设置maxInstancePerHost=1,继续调用preview接口，IncreaseReplica=10000，得到可以创建的容器c11
	//2.在A创建一台容器，继续按照10调用preview接口，得到可以创建的容器c12
	//3.销毁A上的容器，调用preview，得到可以创建的容器c13
	//4.在A创建一个Pod，继续按照10调用preview，得到可以创建的容器c14
	//5.销毁pod，继续调用preview，得到可以创建容器数c15
	//
	//预期：
	//c11=c12+1=c13=c14+1=c15
	It("resourceMix008 sigma 2.0 preview and k8s.", func() {
		if env.Tester != env.TesterAnt {
			Skip("ant sigma 2.0 preview")
		}
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		requestedCPU := int64(1000)         //1core
		requestedMemory := int64(524288000) //512M
		requestedDisk := int64(524288000)   //512M

		dp := "max-instance-filter-resourcemix008"
		testResourceCase := resourceCase{
			cpu:         requestedCPU,
			mem:         requestedMemory,
			ethstorage:  requestedDisk,
			requestType: requestTypeAntSigmaPreview,
			affinityConfig: map[string][]string{
				"ali.MaxInstancePerHost": {"1"},
				"ali.IncreaseReplica":    {"10000"},
				"ali.AppDeployUnit":      {dp},
			},
			shouldScheduled: true,
			cpushare:        false,
		}

		nodeNum := testPreviewSigmaCase(testResourceCase)

		tests := []resourceCase{
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.MaxInstancePerHost": {"1"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    nodeNum, //c11
			},
			{ //index 1
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeSigma,
				affinityConfig: map[string][]string{
					"ali.MaxInstancePerHost": {"1"},
					"ali.IncreaseReplica":    {"1"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.MaxInstancePerHost": {"1"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    nodeNum - 1, //c12
			},
			{
				cleanIndexes: []int{1},
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.MaxInstancePerHost": {"1"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    nodeNum, //c13
			},
			{ //index 5
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				labels:          map[string]string{api.LabelDeployUnit: dp},
				shouldScheduled: true,
				cpushare:        false,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.MaxInstancePerHost": {"1"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    nodeNum - 1, //c14
			},
			{
				cleanIndexes: []int{5},
				requestType:  cleanResource,
			},
			{
				cpu:         requestedCPU,
				mem:         requestedMemory,
				ethstorage:  requestedDisk,
				requestType: requestTypeAntSigmaPreview,
				affinityConfig: map[string][]string{
					"ali.MaxInstancePerHost": {"1"},
					"ali.IncreaseReplica":    {"10000"},
					"ali.AppDeployUnit":      {dp},
				},
				shouldScheduled: true,
				cpushare:        false,
				previewCount:    nodeNum, //c15
			},
		}
		testContext := &testContext{
			caseName:  "resourceMix008",
			cs:        cs,
			localInfo: nil,
			f:         f,
			testCases: tests,
			nodeName:  nodeName,
		}
		testContext.execTests()
	})
	//混合链路磁盘分配case：源于ant sigma2的bug
	//1. 创建一个使用全量boot盘sigma3的pod:预期成功
	//2. 创建一个使用1/2boot盘的sigma2的container:预期失败
	It("[smoke][p0] resourceMix009 ，验证分配磁盘正常", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())
		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address

		framework.Logf("get one node to schedule, nodeName: %s IP: %s", nodeName, nodeIP)

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		requestedCPU := AllocatableCPU / 4
		requestedMemory := AllocatableMemory / 4
		requestedDisk := AllocatableDisk / 2

		leftCPU := AllocatableCPU - requestedCPU
		leftMemory := AllocatableMemory - requestedMemory

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      AllocatableDisk,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cpu:             leftCPU,
				mem:             leftMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeSigma,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: false,
				cpushare:        true,
			},
		}

		testContext := &testContext{
			caseName:  "resourceMix009",
			cs:        cs,
			localInfo: nil,
			f:         f,
			testCases: tests,
			nodeName:  nodeName,
		}

		testContext.execTests()
	})
})
