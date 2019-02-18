package scheduler

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"

	"fmt"

	"k8s.io/kubernetes/test/e2e/framework"
 	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-3.1][sigma-scheduler][resource][Serial]", func() {
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
				allocatable, found := node.Status.Allocatable[v1.ResourceCPU]
				Expect(found).To(Equal(true))
				nodeToAllocatableMapCPU[node.Name] = allocatable.Value()*1000
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

	// 基本资源 CPU/Memory/Disk quota 验证
	// 前置：集群中单节点上可分配的 CPU/Memory/Disk 资源均大于 0

	// 步骤：
	// 1. 获取一个可调度的节点，记录可分配的 cpu 额度 X，memory 的额度 Y，disk 额度 Z
	// 2. 在 node 上打上一个随机的标签 key=NodeName
	// 3. k8s 创建一个单容器 PodA，NodeAffinity 设置 key=NodeName，
	//    并且 Requests.CPU 为 1/2 * X，Requests.Memory 为 1/2 * Y，Requests.EphemeralStorage 为 1/2 * Z
	// 4. 观察调度结果，并获取当前节点可分配的 cpu、memory、disk 的额度，分别记录为 X1、Y1、Z1
	// 5. k8s 创建第二个单容器 PodB， NodeAffinity 设置 key=NodeName，
	//    并且 Requests.CPU 为 X - (1/2 * X) ，Requests.Memory 为 Y - (1/2 * Y)，Requests.EphemeralStorage 为 Z - (1/2 * Z)
	// 6. 观察调度结果，并获取当前节点可分配的 cpu、memory、disk 的额度，分别记录为 X2、Y2、Z2
	// 7. k8s 创建第三个单容器 PodC，参数和 PodA 保持一致
	// 8. 观察调度结果，并获取当前节点可分配的 cpu、memory、disk 的额度，分别记录为 X3、Y3、Z3
	// 9. 删掉 PodA、PodB 和 PodC，获取当前节点可分配的 cpu、memory、disk 的额度，分别记录为 X4、Y4、Z4
	// 10. 重复 3-8 步骤
	// 11. 删掉 PodB 和 PodC，再次创建 PodB 和 PodC，观察调度结果

	// 验证结果：
	// 1. 步骤 4 中，PodA 调度成功，且 Pod.Spec.NodeName = 此 Node，
	//    剩余 cpu 额度 X1 = X - (1/2 * X)，memory 额度 Y1 = Y - (1/2 * Y)，disk 额度 Z1 = Z - (1/2 * Z)
	// 2. 步骤 6 中，PodB 调度成功，且 Pod.Spec.NodeName = 此 Node，
	//    剩余的 cpu 额度 X2 = 0，memory 额度 Y2 = 0，disk 额度 Z2 = 0
	// 3. 步骤 8 中，PodC 调度失败，剩余的 cpu 额度 X3 = 0，memory 额度 Y3 = 0，disk 额度 Z3 = 0
	// 4. 步骤 9 中，剩余的 cpu 额度 X4 = X，memory 额度 Y4 = Y，disk 额度 Z4 = Z
	// 5. 步骤 10 中的调度结果，符合上述 1-4 的描述
	// 6. 步骤 11 中，PodB 可以调度成功，PodC 调度失败
	It("[smoke][p0][bvt] resource_k8s_001 A pod with cpu/mem/ephemeral-storage request should be scheduled on node with enough resource successfully. "+
		"While fail to be scheduled on node with insufficient resource", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		// Apply node label to each node
		nodeAffinityKey := "node-for-resource-e2e-test-" + string(uuid.NewUUID())
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		pod1CPU := AllocatableCPU * 5 / 10
		pod1Memory := AllocatableMemory * 5 / 10
		pod1Disk := AllocatableDisk * 5 / 10

		pod2CPU := AllocatableCPU - pod1CPU
		pod2Memory := AllocatableMemory - pod1Memory
		pod2Disk := AllocatableDisk - pod1Disk

		By("Request a pod with CPU/Memory/EphemeralStorage.")
		tests := []struct {
			cpu                       resource.Quantity
			mem                       resource.Quantity
			ethstorage                resource.Quantity
			expectedAvailableResource []int64 // CPU/Memory/Disk
			expectedScheduleResult    bool
		}{
			{
				cpu:                       *resource.NewMilliQuantity(pod1CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod1Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod1Disk, "DecimalSI"),
				expectedAvailableResource: []int64{pod2CPU, pod2Memory, pod2Disk},
				expectedScheduleResult:    true,
			},
			{
				cpu:                       *resource.NewMilliQuantity(pod2CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod2Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod2Disk, "DecimalSI"),
				expectedAvailableResource: []int64{0, 0, 0},
				expectedScheduleResult:    true,
			},
			{
				cpu:                       *resource.NewMilliQuantity(pod2CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod2Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod2Disk, "DecimalSI"),
				expectedAvailableResource: []int64{0, 0, 0},
				expectedScheduleResult:    false,
			},
		}

		podsToDelete := []*v1.Pod{}
		// 循环 3 次
		loopTime := 3
		for j := 1; j <= loopTime; j++ {
			for i, test := range tests {
				pod := createPausePod(f, pausePodConfig{
					Name: "scheduler-e2e-resource-" + strconv.Itoa(i) + "-" + string(uuid.NewUUID()),
					Resources: &v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:              test.cpu,
							v1.ResourceMemory:           test.mem,
							v1.ResourceEphemeralStorage: test.ethstorage,
						},
						Requests: v1.ResourceList{
							v1.ResourceCPU:              test.cpu,
							v1.ResourceMemory:           test.mem,
							v1.ResourceEphemeralStorage: test.ethstorage,
						},
					},
					Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
				})

				if test.expectedScheduleResult == true {
					framework.Logf("Case[%d], expect pod to be scheduled successfully.", i)
					err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
					podsToDelete = append(podsToDelete, pod)
					Expect(err).NotTo(HaveOccurred())
				} else {
					framework.Logf("Case[%d], expect pod failed to be scheduled.", i)
					podsToDelete = append(podsToDelete, pod)
					err := framework.WaitForPodNameUnschedulableInNamespace(cs, pod.Name, pod.Namespace)
					Expect(err).To(BeNil(), "expect err be nil, got %s", err)
				}

				ar := getAvailableResourceOnNode(f, nodeName)
				for j := 0; j < len(ar); j++ {
					framework.Logf("Case[%d], AvailableResource[%d]: %d.", i, j, ar[j])
					Expect(ar[j]).Should(Equal(test.expectedAvailableResource[j]), "available resource should match to expected")
				}
			}

			// 只有前两次才全部删除，第三次单独处理
			if j != 3 {
				for _, pod := range podsToDelete {
					if pod == nil {
						continue
					}
					err := util.DeletePod(f.ClientSet, pod)
					Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
				}

				By("Get available resource after all pods are deleted.")
				expectedAvailableResourceAfterAllPodsAreDeleted := []int64{AllocatableCPU, AllocatableMemory, AllocatableDisk}

				ar := getAvailableResourceOnNode(f, nodeName)
				for j := 0; j < len(ar); j++ {
					framework.Logf("AvailableResource[%d]: %d.", j, ar[j])
					Expect(ar[j]).Should(Equal(expectedAvailableResourceAfterAllPodsAreDeleted[j]), "available resource should match to expected")
				}
				podsToDelete = []*v1.Pod{}
				continue
			}

			// 第三次，只删掉 PodB 和 PodC
			for index, pod := range podsToDelete {
				if pod == nil {
					continue
				}
				if index > 0 {
					err := util.DeletePod(f.ClientSet, pod)
					Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
				}
			}

			// 最后一次，只创建 PodB 和 PodC
			tests2 := []struct {
				cpu                       resource.Quantity
				mem                       resource.Quantity
				ethstorage                resource.Quantity
				expectedAvailableResource []int64 // CPU/Memory/Disk
				expectedScheduleResult    bool
			}{tests[1], tests[2]}

			for i, test := range tests2 {
				pod := createPausePod(f, pausePodConfig{
					Name: "scheduler-e2e-resource-" + strconv.Itoa(i) + "-" + string(uuid.NewUUID()),
					Resources: &v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:              test.cpu,
							v1.ResourceMemory:           test.mem,
							v1.ResourceEphemeralStorage: test.ethstorage,
						},
						Requests: v1.ResourceList{
							v1.ResourceCPU:              test.cpu,
							v1.ResourceMemory:           test.mem,
							v1.ResourceEphemeralStorage: test.ethstorage,
						},
					},
					Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
				})

				if test.expectedScheduleResult == true {
					framework.Logf("Case[%d], expect pod to be scheduled successfully.", i)
					err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
					podsToDelete = append(podsToDelete, pod)
					Expect(err).NotTo(HaveOccurred())
				} else {
					framework.Logf("Case[%d], expect pod failed to be scheduled.", i)
					podsToDelete = append(podsToDelete, pod)
					err := framework.WaitForPodNameUnschedulableInNamespace(cs, pod.Name, pod.Namespace)
					Expect(err).To(BeNil(), "expect err be nil, got %s", err)
				}
			}
		}
	})

	It("resource_k8s_002 A pod with over quota cpu request should be scheduled on node with enough resource successfully."+
		"While fail to be scheduled on node with insufficient resource", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		// Apply node affinity label to each node
		nodeAffinityKey := "node-for-resource-e2e-test"
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		// Apply over quota labels and taints to each node
		framework.AddOrUpdateLabelOnNode(cs, nodeName, sigmak8sapi.LabelEnableOverQuota, "true")
		framework.ExpectNodeHasLabel(cs, nodeName, sigmak8sapi.LabelEnableOverQuota, "true")
		defer framework.RemoveLabelOffNode(cs, nodeName, sigmak8sapi.LabelEnableOverQuota)

		framework.AddOrUpdateLabelOnNode(cs, nodeName, sigmak8sapi.LabelCPUOverQuota, "2")
		framework.ExpectNodeHasLabel(cs, nodeName, sigmak8sapi.LabelCPUOverQuota, "2")
		defer framework.RemoveLabelOffNode(cs, nodeName, sigmak8sapi.LabelCPUOverQuota)

		overQuotaTaint := v1.Taint{
			Key:    sigmak8sapi.LabelEnableOverQuota,
			Value:  "true",
			Effect: v1.TaintEffectNoSchedule,
		}
		framework.AddOrUpdateTaintOnNode(cs, nodeName, overQuotaTaint)
		framework.ExpectNodeHasTaint(cs, nodeName, &overQuotaTaint)
		defer framework.RemoveTaintOffNode(cs, nodeName, overQuotaTaint)

		podRequestCPU := nodeToAllocatableMapCPU[nodeName]
		allocatableMemory := nodeToAllocatableMapMem[nodeName]
		allocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		framework.Logf("podRequestCPU: %d", podRequestCPU)
		framework.Logf("allocatableMemory: %d", allocatableMemory)
		framework.Logf("allocatableDisk: %d", allocatableDisk)

		By("Request a pod with over quota CPU.")
		tests := []struct {
			cpu                    resource.Quantity
			expectedScheduleResult bool
		}{
			{
				cpu: *resource.NewMilliQuantity(podRequestCPU, "DecimalSI"),
				expectedScheduleResult: true,
			},
			{
				cpu: *resource.NewMilliQuantity(podRequestCPU, "DecimalSI"),
				expectedScheduleResult: true,
			},
			{
				cpu: *resource.NewMilliQuantity(podRequestCPU, "DecimalSI"),
				expectedScheduleResult: false,
			},
		}

		podsToDelete := make([]*v1.Pod, len(tests))
		for i, test := range tests {
			pod := createPausePod(f, pausePodConfig{
				Name: "scheduler-" + string(uuid.NewUUID()),
				Resources: &v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU: test.cpu,
					},
				},
				Affinity: &v1.Affinity{
					NodeAffinity: &v1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
							NodeSelectorTerms: []v1.NodeSelectorTerm{
								{
									MatchExpressions: []v1.NodeSelectorRequirement{
										{
											Key:      nodeAffinityKey,
											Operator: v1.NodeSelectorOpIn,
											Values:   []string{nodeName},
										},
									},
								},
								{
									MatchExpressions: []v1.NodeSelectorRequirement{
										{
											Key:      sigmak8sapi.LabelEnableOverQuota,
											Operator: v1.NodeSelectorOpIn,
											Values:   []string{"true"},
										},
									},
								},
							},
						},
					},
				},
				Tolerations: []v1.Toleration{{Key: sigmak8sapi.LabelEnableOverQuota, Value: "true", Effect: v1.TaintEffectNoSchedule}},
			})

			if test.expectedScheduleResult == true {
				framework.Logf("Case[%d], expect pod to be scheduled successfully.", i)
				err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
				podsToDelete = append(podsToDelete, pod)
				Expect(err).NotTo(HaveOccurred())
			} else {
				framework.Logf("Case[%d], expect pod failed to be scheduled.", i)
				podsToDelete = append(podsToDelete, pod)
				err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
				Expect(err).To(BeNil(), "expect err to be nil, got %s", err)
			}
		}

		for _, pod := range podsToDelete {
			if pod == nil {
				continue
			}
			err := util.DeletePod(f.ClientSet, pod)
			Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
		}
	})

	It("resource_k8s_003 A pod with extended resource request should be scheduled on node with enough resource successfully."+
		"While fail to be scheduled on node with insufficient resource.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		// Apply node label to each node
		nodeAffinityKey := "node-for-resource-e2e-test"
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		// defile a extended resource
		exdResource := "sigma3.e2e.test/extended"
		exdResourceValue := "2"
		By("Submit a PATCH HTTP request to the API server to specify the available quantity of new resource")

		var nodes []v1.Node
		for _, v := range nodeList.Items {
			if v.Name == nodeName {
				nodes = append(nodes, v)
			}
		}
		err := util.PatchNodeExtendedResource(f, nodes, exdResource, exdResourceValue)
		Expect(err).ToNot(HaveOccurred())

		// remove the extended resource from nodes
		defer func() {
			data1 := []byte(`[{"op": "remove", "path": "/status/capacity/sigma3.e2e.test~1extended"}]`)
			util.PatchNodeStatusJsonPathType(f, nodes, data1)
			data2 := []byte(`[{"op": "remove", "path": "/status/allocatable/sigma3.e2e.test~1extended"}]`)
			util.PatchNodeStatusJsonPathType(f, nodes, data2)
		}()

		By("Request a pod with new resource.")
		tests := []struct {
			key      v1.ResourceName
			quantity string
			ok       bool
		}{
			{
				key:      v1.ResourceName(exdResource),
				quantity: "1",
				ok:       true,
			},
			{
				key:      v1.ResourceName(exdResource),
				quantity: "3",
				ok:       false,
			},
		}

		podsToDelete := make([]*v1.Pod, len(tests))
		for _, test := range tests {

			pod := createPausePod(f, pausePodConfig{
				Name: "scheduler-" + string(uuid.NewUUID()),
				Resources: &v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceName(test.key): resource.MustParse(test.quantity),
					},
					Requests: v1.ResourceList{
						v1.ResourceName(test.key): resource.MustParse(test.quantity),
					},
				},
				Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
			})

			if test.ok == true {
				By("expect pod to be scheduled successfully.")
				err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
				podsToDelete = append(podsToDelete, pod)
				Expect(err).NotTo(HaveOccurred())
			} else {

				By("expect pod failed to be scheduled .")
				podsToDelete = append(podsToDelete, pod)
				err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
				Expect(err).To(BeNil(), "expect err to be nil, got %s", err)
			}
		}
		for _, pod := range podsToDelete {
			if pod == nil {
				continue
			}
			util.DeletePod(f.ClientSet, pod)
		}
	})

	// OverQuota 下对不同 cpuset 绑核策略的验证 cpu-over-quota = 2
	// 步骤：
	// 获取一个可调度的节点，记录超卖后可分配的 cpu 额度 X
	// 给机器打上超卖标签
	// sigma.ali/cpu-over-quota=2
	// sigma.ali/is-over-quota=true
	// 下面整个过程重复 2 次
	// 第一轮：测试 spread
	// 1. 创建 pod1：cpu=X/2, spread 成功，剩余 cpu=X/2
	// 2. 创建 pod2：cpu=X/2, spread 成功，剩余 cpu=0
	// 3. 创建 pod3：cpu=X/2, spread 失败，剩余 cpu=0
	// 4. 删除 pod1~pod3
	// 5. 第二轮：测试 samecorefirst
	// 6. 创建 pod4：cpu=X/2, samecorefirst 成功，剩余 cpu=X/2
	// 7. 创建 pod5：cpu=X/2, samecorefirst 成功，剩余 cpu=0
	// 8. 创建 pod6：cpu=X/2, samecorefirst 失败，剩余 cpu=0
	// 9. 删除 pod4~pod6
	// 第三轮：测试混合策略
	// 10. 创建 pod7：cpu=X/4, spread 成功，剩余 cpu=3/4*X
	// 11. 创建 pod8：cpu=X/4, samecorefirst 成功，剩余 cpu=2/4*X
	// 12. 创建 pod9：cpu=X/4, spread 成功，剩余 cpu=1/4*X
	// 13. 创建 pod10：cpu=X/4, samecorefirst 成功，剩余 cpu=0
	// 14. 创建 pod11：cpu=X/4，随机策略失败，剩余 cpu=0
	// 15. 删除 pod7～pod11
	// 第四轮：测试随机混合策略
	// 16. 创建 pod12～pod19：cpu=X/8，随机策略成功，剩余 cpu=0
	// 17. 创建 pod20：cpu=X/8，随机策略失败，剩余 cpu=0
	It("[p1] cpusetK8sOverquota001 cpu-over-quota=2.", func() {
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

		cpuOverQuotaRatio := 2.0

		allocatableCPUAfterQuota := int64(float64(allocatableCPU) * cpuOverQuotaRatio)

		request1CPU := (allocatableCPUAfterQuota / 2000) * 1000
		request2CPU := (allocatableCPUAfterQuota / 4000) * 1000
		request3CPU := (allocatableCPUAfterQuota / 8000) * 1000

		podMemory := int64(1024 * 1024 * 1024)
		podDisk := int64(1024 * 1024 * 1024)

		rest1CPU := allocatableCPUAfterQuota - request1CPU
		rest2CPU := allocatableCPUAfterQuota - request2CPU*3
		rest3CPU := allocatableCPUAfterQuota - request3CPU*7

		framework.Logf("request1CPU: %d, rest1CPU: %d", request1CPU, rest1CPU)
		framework.Logf("request2CPU: %d, rest2CPU: %d", request2CPU, rest2CPU)
		framework.Logf("request3CPU: %d, rest3CPU: %d", request3CPU, rest3CPU)

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[sigmak8sapi.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &sigmak8sapi.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		strategy := []string{"spread", "sameCoreFirst"}
		for t := 1; t <= 2; t++ {
			tests := []resourceCase{
				// 第一轮：测试 spread
				{
					cpu:             request1CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "spread",
				},
				{
					cpu:             rest1CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "spread",
				},
				{
					cpu:             request1CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					cpushare:        false,
					spreadStrategy:  "spread",
				},
				{
					cleanIndexes: []int{0, 1, 2},
					requestType:  cleanResource,
				},
				// 第二轮：测试 samecorefirst
				{
					cpu:             request1CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             rest1CPU, // 剩余的所有 CPU
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             request1CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					cpushare:        false,
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cleanIndexes: []int{4, 5, 6},
					requestType:  cleanResource,
				},
				// 第三轮：测试混合策略
				{
					cpu:             request2CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "spread",
				},
				{
					cpu:             request2CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             request2CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "spread",
				},
				{
					cpu:             rest2CPU, // 剩余的所有 CPU
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             request2CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cleanIndexes: []int{8, 9, 10, 11, 12},
					requestType:  cleanResource,
				},
				// 第四轮：测试随机混合策略
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             rest3CPU, // 剩余的所有 CPU
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cleanIndexes: []int{14, 15, 16, 17, 18, 19, 20, 21, 22},
					requestType:  cleanResource,
				},
			}

			testContext := &testContext{
				caseName:          "cpusetK8sOverquota001",
				cs:                cs,
				localInfo:         localInfo,
				f:                 f,
				testCases:         tests,
				CPUOverQuotaRatio: cpuOverQuotaRatio,
				nodeName:          nodeName,
			}

			testContext.execTests(
				checkCPUSetOverquotaRate,
			)
		}

	})

	// OverQuota 下 cpuset 不同策略的验证  cpu-over-quota = 1.5
	// 测试步骤和 cpu-over-quota = 2 完全相同
	It("[p1] cpusetK8sOverquota002 cpu-over-quota=1.5.", func() {
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

		cpuOverQuotaRatio := 1.5

		allocatableCPUAfterQuota := int64(float64(allocatableCPU) * cpuOverQuotaRatio)

		request1CPU := (allocatableCPUAfterQuota / 2000) * 1000
		request2CPU := (allocatableCPUAfterQuota / 4000) * 1000
		request3CPU := (allocatableCPUAfterQuota / 8000) * 1000

		podMemory := int64(1024 * 1024 * 1024)
		podDisk := int64(1024 * 1024 * 1024)

		rest1CPU := allocatableCPUAfterQuota - request1CPU
		rest2CPU := allocatableCPUAfterQuota - request2CPU*3
		rest3CPU := allocatableCPUAfterQuota - request3CPU*7

		framework.Logf("request1CPU: %d, rest1CPU: %d", request1CPU, rest1CPU)
		framework.Logf("request2CPU: %d, rest2CPU: %d", request2CPU, rest2CPU)
		framework.Logf("request3CPU: %d, rest3CPU: %d", request3CPU, rest3CPU)

		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address
		localInfoString := nodesInfo[nodeName].Annotations[sigmak8sapi.AnnotationLocalInfo]
		Expect(localInfoString == "").ShouldNot(BeTrue(), fmt.Sprintf("nodeName:%s, localInfoString is empty", nodeName))
		localInfo := &sigmak8sapi.LocalInfo{}
		if err := json.Unmarshal([]byte(localInfoString), localInfo); err != nil {
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("nodeName:%s, localInfoString:%v parse error", nodeName, localInfoString))
		}

		strategy := []string{"spread", "sameCoreFirst"}
		for t := 1; t <= 2; t++ {
			tests := []resourceCase{
				// 第一轮：测试 spread
				{
					cpu:             request1CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "spread",
				},
				{
					cpu:             rest1CPU, // 剩余的所有 CPU
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "spread",
				},
				{
					cpu:             request1CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					cpushare:        false,
					spreadStrategy:  "spread",
				},
				{
					cleanIndexes: []int{0, 1, 2},
					requestType:  cleanResource,
				},
				// 第二轮：测试 samecorefirst
				{
					cpu:             request1CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             rest1CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             request1CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					cpushare:        false,
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cleanIndexes: []int{4, 5, 6},
					requestType:  cleanResource,
				},
				// 第三轮：测试混合策略
				{
					cpu:             request2CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "spread",
				},
				{
					cpu:             request2CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             request2CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "spread",
				},
				{
					cpu:             rest2CPU, // 剩余的所有 CPU
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  "sameCoreFirst",
				},
				{
					cpu:             request2CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cleanIndexes: []int{8, 9, 10, 11, 12},
					requestType:  cleanResource,
				},
				// 第四轮：测试随机混合策略
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             rest3CPU, // 剩余的所有 CPU
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: true,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cpu:             request3CPU,
					mem:             podMemory,
					ethstorage:      podDisk,
					affinityConfig:  map[string][]string{sigmak8sapi.LabelNodeIP: {nodeIP}},
					requestType:     requestTypeKubernetes,
					shouldScheduled: false,
					cpushare:        false,
					spreadStrategy:  strategy[rand.Int()%2],
				},
				{
					cleanIndexes: []int{14, 15, 16, 17, 18, 19, 20, 21, 22},
					requestType:  cleanResource,
				},
			}

			testContext := &testContext{
				caseName:          "cpusetK8sOverquota001",
				cs:                cs,
				localInfo:         localInfo,
				f:                 f,
				testCases:         tests,
				CPUOverQuotaRatio: cpuOverQuotaRatio,
				nodeName:          nodeName,
			}

			testContext.execTests(
				checkCPUSetOverquotaRate,
			)
		}
	})
})
