package scheduler

import (
	"encoding/json"
	"sort"
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

	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/swarm"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-3.1][sigma-scheduler][resource][Serial]", func() {
	var cs clientset.Interface
	var nodeList *v1.NodeList

	nodeToAllocatableMapCPU := make(map[string]int64)
	nodeToAllocatableMapMem := make(map[string]int64)
	nodeToAllocatableMapEphemeralStorage := make(map[string]int64)

	f := framework.NewDefaultFramework(CPUSetNameSpace)

	f.AllNodesReadyTimeout = 3 * time.Second

	BeforeEach(func() {
		cs = f.ClientSet
		nodeList = &v1.NodeList{}
		masterNodes, nodeList = getMasterAndWorkerNodesOrDie(cs)

		for _, node := range nodeList.Items {
			framework.Logf("logging pods the kubelet thinks is on node %s before test", node.Name)
			framework.PrintAllKubeletPods(cs, node.Name)

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

	// OverQuota 下基本资源 CPU/Memory/Disk quota 验证 cpu-over-quota = 2
	// 步骤：
	// 获取一个可调度的节点，记录超卖后可分配的 cpu 额度 X，memory 的额度 Y，disk 额度 Z
	// 给机器打上超卖标签
	// sigma.ali/cpu-over-quota=2
	// sigma.ali/is-over-quota=true
	// sigma.ali/memory-over-quota=1.1
	// 1. 创建 podA：cpu=1/2*X，mem=1/2*Y，disk=1/2*Z 成功，剩余 cpu=1/2*X，mem=1/2*Y，disk=1/2*Z
	// 2. 创建 podB：cpu=X-1/2*X-1，mem=Y-1/2*Y-1k，disk=Z-1/2*Z-1k 成功，剩余 cpu=1，mem=1k，disk=1k
	// 3. 创建 podC1：cpu=1/2*X，mem=1k，disk=1k 失败，资源不足，剩余 cpu=1，mem=1k，disk=1k
	// 4. 创建 podC2：cpu=1，mem=1/2*Y，disk=1k 失败，资源不足，剩余 cpu=1，mem=1k，disk=1k
	// 5. 创建 podC3：cpu=1，mem=1k，disk=1/2*Z 失败，资源不足，剩余 cpu=1，mem=1k，disk=1k
	// 6. 创建 podC4：cpu=1，mem=1k，disk=1k 成功，剩余 cpu=0，mem=0，disk=0
	// 7. 删除 podC1，C2，C3，C4，podB，剩余 cpu=X-1/2*X，mem=Y-1/2*Y，disk=Z-1/2*Z
	// 8. 创建 podD：cpu=1/2*X+1，mem=1/2*Y+1k，disk=1/2*Z+1k 失败，资源不足，剩余 cpu=X-1/2*X,mem=Y-1/2*Y，disk=Z-1/2*Z
	// 9. 创建 podE：cpu=1/2*X，mem=1/2*Y，disk=1/2*Z 成功，剩余 cpu=0，mem=0，disk=0
	// 10. 删除 podE，podA，剩余 cpu=X, mem=Y，disk=Z
	// 11. 创建 podF1：cpu=1/2*X，mem=1/2*Y，disk=1/2*Z 成功，剩余 cpu=1/2*X，mem=1/2*Y，disk=1/2*Z
	// 12. 创建 podF2：cpu=1/2*X，mem=1/2*Y，disk=1/2*Z 成功，剩余 cpu=0，mem=0，disk=0
	// 13 .删除 podF1，F2
	It("[p1] ResourceOverquotaK8s001 cpu-over-quota=2 & mem-over-quota=1.1.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		// Apply node affinity label to each node
		nodeAffinityKey := "node-for-resource-e2e-test" + string(uuid.NewUUID())
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

		framework.AddOrUpdateLabelOnNode(cs, nodeName, sigmak8sapi.LabelMemOverQuota, "1.1")
		framework.ExpectNodeHasLabel(cs, nodeName, sigmak8sapi.LabelMemOverQuota, "1.1")
		defer framework.RemoveLabelOffNode(cs, nodeName, sigmak8sapi.LabelMemOverQuota)

		// due to bug: 18296033, MUST also add overquota lable in sigma2.0
		swarm.SetNodeOverQuota(nodeName, 2, 1.1)
		defer swarm.SetNodeToNotOverQuota(nodeName)

		overQuotaTaint := v1.Taint{
			Key:    sigmak8sapi.LabelEnableOverQuota,
			Value:  "true",
			Effect: v1.TaintEffectNoSchedule,
		}

		framework.AddOrUpdateTaintOnNode(cs, nodeName, overQuotaTaint)
		framework.ExpectNodeHasTaint(cs, nodeName, &overQuotaTaint)
		defer framework.RemoveTaintOffNode(cs, nodeName, overQuotaTaint)

		allocatableCPU := nodeToAllocatableMapCPU[nodeName]
		allocatableMemory := nodeToAllocatableMapMem[nodeName]
		allocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		framework.Logf("allocatableCPU: %d", allocatableCPU)
		framework.Logf("allocatableMemory: %d", allocatableMemory)
		framework.Logf("allocatableDisk: %d", allocatableDisk)

		allocatableCPUAfterQuota := allocatableCPU * 2
		allocatableMemoryAfterQuota := int64(float64(allocatableMemory) * 1.1)
		allocatableDiskAfterQuota := allocatableDisk

		pod1CPU := (allocatableCPUAfterQuota / 1000) * 1000 / 2
		pod1Memory := allocatableMemoryAfterQuota * 5 / 10
		pod1Disk := allocatableDiskAfterQuota * 5 / 10

		pod2CPU := allocatableCPUAfterQuota - pod1CPU
		pod2Memory := allocatableMemoryAfterQuota - pod1Memory
		pod2Disk := allocatableDiskAfterQuota - pod1Disk

		By("Request a pod with over quota CPU/Memory/EphemeralStorage.")
		tests := []struct {
			cpu                       resource.Quantity
			mem                       resource.Quantity
			ethstorage                resource.Quantity
			expectedAvailableResource []int64 // CPU/Memory/Disk
			spreadStrategy            sigmak8sapi.SpreadStrategy
			expectedScheduleResult    bool
		}{
			// test[0] podA cpu = 1/2 * X, mem = 1/2 * Y，disk = 1/2 * Z
			{
				cpu:                       *resource.NewMilliQuantity(pod1CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod1Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod1Disk, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU, allocatableMemory - pod1Memory, allocatableDisk - pod1Disk},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    true,
			},
			// test[1] podB cpu = X - 1/2 * X - 1，mem = Y - 1/2 * Y - 1M，disk = Z - 1/2 * Z - 1M
			{
				cpu:                       *resource.NewMilliQuantity(pod2CPU-1000, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod2Memory-100*1024*1024, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod2Disk-100*1024*1024, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU + 1000, allocatableMemory - pod1Memory - pod2Memory + 100*1024*1024, allocatableDisk - pod1Disk - pod2Disk + 100*1024*1024},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    true,
			},
			// test[2] podC1 cpu = 1/2 * X, mem = 1M，disk = 1M
			{
				cpu:                       *resource.NewMilliQuantity(pod1CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(100*1024*1024, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(100*1024*1024, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU + 1000, allocatableMemory - pod1Memory - pod2Memory + 100*1024*1024, allocatableDisk - pod1Disk - pod2Disk + 100*1024*1024},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    false,
			},
			// test[3] podC2 cpu = 1, mem = 1/2 * Y，disk = 1M
			{
				cpu:                       *resource.NewMilliQuantity(1000, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod1Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(100*1024*1024, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU + 1000, allocatableMemory - pod1Memory - pod2Memory + 100*1024*1024, allocatableDisk - pod1Disk - pod2Disk + 100*1024*1024},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    false,
			},
			// test[4] podC3 cpu = 1, mem = 1M，disk = 1/2 * Z
			{
				cpu:                       *resource.NewMilliQuantity(1000, "DecimalSI"),
				mem:                       *resource.NewQuantity(100*1024*1024, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod1Disk, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU + 1000, allocatableMemory - pod1Memory - pod2Memory + 100*1024*1024, allocatableDisk - pod1Disk - pod2Disk + 100*1024*1024},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    false,
			},
			// test[5] podC4 cpu = 1, mem = 1M，disk = 1M
			{
				cpu:                       *resource.NewMilliQuantity(1000, "DecimalSI"),
				mem:                       *resource.NewQuantity(100*1024*1024, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(100*1024*1024, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU, allocatableMemory - pod1Memory - pod2Memory, 0},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    true,
			},
			// test[6] podD cpu = 1/2 * X + 1, mem = 1/2 * Y + 1M，disk = 1/2 * Z + 1M
			{
				cpu:                       *resource.NewMilliQuantity(pod2CPU+1000, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod2Memory+100*1024*1024, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod2Disk+100*1024*1024, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU, allocatableMemory - pod1Memory, allocatableDisk - pod1Disk},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    false,
			},
			// test[7] podE cpu = 1/2 * X, mem = 1/2 * Y，disk = 1/2 * Z
			{
				cpu:                       *resource.NewMilliQuantity(pod2CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod2Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod2Disk, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU, allocatableMemory - pod1Memory - pod2Memory, 0},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    true,
			},
			// test[8] podF1 cpu = 1/2 * X, mem = 1/2 * Y，disk = 1/2 * Z
			{
				cpu:                       *resource.NewMilliQuantity(pod1CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod1Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod1Disk, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU, allocatableMemory - pod1Memory, allocatableDisk - pod1Disk},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    true,
			},
			// test[9] podF2 cpu = 1/2 * X, mem = 1/2 * Y，disk = 1/2 * Z
			{
				cpu:                       *resource.NewMilliQuantity(pod2CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod2Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod2Disk, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU, allocatableMemory - pod1Memory - pod2Memory, 0},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    true,
			},
		}

		getCPUID := func(pod *v1.Pod) []int {
			// Get pod and check CPUIDs.
			podRunning, err := f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			allocSpecStr := podRunning.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
			allocSpec := &sigmak8sapi.AllocSpec{}
			err = json.Unmarshal([]byte(allocSpecStr), allocSpec)
			Expect(err).NotTo(HaveOccurred())

			CPUIDs := allocSpec.Containers[0].Resource.CPU.CPUSet.CPUIDs
			sort.Ints(CPUIDs)

			return CPUIDs
		}

		// 整个过程循环 2 次
		loopTime := 2
		for t := 1; t <= loopTime; t++ {
			podsToDelete := []*v1.Pod{}
			processorIDToCntMap := make(map[int]int)
			for i, test := range tests {
				podName := "scheduler-e2e-resource-" + strconv.Itoa(i) + "-" + string(uuid.NewUUID())
				allocSpecRequest := &sigmak8sapi.AllocSpec{
					Containers: []sigmak8sapi.Container{
						{
							Name: podName,
							Resource: sigmak8sapi.ResourceRequirements{
								CPU: sigmak8sapi.CPUSpec{
									CPUSet: &sigmak8sapi.CPUSetSpec{
										SpreadStrategy: test.spreadStrategy,
									},
								},
							},
						},
					},
				}

				allocSpecBytes, err := json.Marshal(&allocSpecRequest)
				if err != nil {
					return
				}

				pod := createPausePod(f, pausePodConfig{
					Name: podName,
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(allocSpecBytes),
					},
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

					// Get pod and check CPUIDs.
					CPUIDs := getCPUID(pod)
					framework.Logf("AllocSpec.CPUIDs: %v", CPUIDs)

					for _, cpu := range CPUIDs {
						processorIDToCntMap[cpu]++
					}
					checkResult := checkCPUOverQuotaCoreBinding(processorIDToCntMap, int(allocatableCPU/1000), 2.0)
					Expect(checkResult).Should(Equal(true), "checkCPUOverQuotaCoreBinding should pass")

				} else {
					framework.Logf("Case[%d], expect pod failed to be scheduled.", i)
					err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
					podsToDelete = append(podsToDelete, pod)
					Expect(err).To(BeNil(), "expect err be nil, got %s", err)
				}

				ar := getAvailableResourceOnNode(f, nodeName)
				for j := 0; j < len(ar); j++ {
					framework.Logf("Case[%d], AvailableResource[%d]: %d.", i, j, ar[j])
					Expect(ar[j]).Should(Equal(test.expectedAvailableResource[j]), "available resource should match to expected")
				}

				// test[5] podC4 创建成功后删除 pod C1，C2, C3, C4 和 podB
				if i == 5 {
					for j, pod := range podsToDelete {
						if j == 0 {
							continue
						}
						if pod == nil {
							continue
						}
						CPUIDs := getCPUID(pod)
						for _, cpu := range CPUIDs {
							processorIDToCntMap[cpu]--
						}
						err := util.DeletePod(f.ClientSet, pod)
						Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
					}

					By("Get available resource after all pods are deleted.")
					expectedAvailableResourceAfterAllPodsAreDeleted := []int64{allocatableCPU - pod1CPU, allocatableMemory - pod1Memory, allocatableDisk - pod1Disk}

					ar := getAvailableResourceOnNode(f, nodeName)
					for j := 0; j < len(ar); j++ {
						framework.Logf("AvailableResource[%d]: %d.", j, ar[j])
						Expect(ar[j]).Should(Equal(expectedAvailableResourceAfterAllPodsAreDeleted[j]), "available resource should match to expected")
					}
					continue
				}

				// test[7] podE 创建成功后删除 podA，podE
				if i == 7 {
					for j, pod := range podsToDelete {
						if j == 0 || j == 7 {
							if pod == nil {
								continue
							}
							CPUIDs := getCPUID(pod)
							for _, cpu := range CPUIDs {
								processorIDToCntMap[cpu]--
							}
							err := util.DeletePod(f.ClientSet, pod)
							Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
						}
					}

					By("Get available resource after all pods are deleted.")
					expectedAvailableResourceAfterAllPodsAreDeleted := []int64{allocatableCPU, allocatableMemory, allocatableDisk}

					ar := getAvailableResourceOnNode(f, nodeName)
					for j := 0; j < len(ar); j++ {
						framework.Logf("AvailableResource[%d]: %d.", j, ar[j])
						Expect(ar[j]).Should(Equal(expectedAvailableResourceAfterAllPodsAreDeleted[j]), "available resource should match to expected")
					}
					continue
				}

				// 最后删除所有容器
				if i == 9 {
					for j, pod := range podsToDelete {
						if j == 8 || j == 9 {
							if pod == nil {
								continue
							}
							CPUIDs := getCPUID(pod)
							for _, cpu := range CPUIDs {
								processorIDToCntMap[cpu]--
							}
							err := util.DeletePod(f.ClientSet, pod)
							Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
						}
					}
				}
			}
		}
	})

	// OverQuota 下基本资源 CPU/Memory/Disk quota 验证  cpu-over-quota = 1.5
	// 测试步骤和 cpu-over-quota = 2 完全相同
	It("[p1] ResourceOverquotaK8s002 cpu-over-quota=1.5 & mem-over-quota=1.1.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		// Apply node affinity label to each node
		nodeAffinityKey := "node-for-resource-e2e-test" + string(uuid.NewUUID())
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		// Apply over quota labels and taints to each node
		framework.AddOrUpdateLabelOnNode(cs, nodeName, sigmak8sapi.LabelEnableOverQuota, "true")
		framework.ExpectNodeHasLabel(cs, nodeName, sigmak8sapi.LabelEnableOverQuota, "true")
		defer framework.RemoveLabelOffNode(cs, nodeName, sigmak8sapi.LabelEnableOverQuota)

		framework.AddOrUpdateLabelOnNode(cs, nodeName, sigmak8sapi.LabelCPUOverQuota, "1.5")
		framework.ExpectNodeHasLabel(cs, nodeName, sigmak8sapi.LabelCPUOverQuota, "1.5")
		defer framework.RemoveLabelOffNode(cs, nodeName, sigmak8sapi.LabelCPUOverQuota)

		framework.AddOrUpdateLabelOnNode(cs, nodeName, sigmak8sapi.LabelMemOverQuota, "1.1")
		framework.ExpectNodeHasLabel(cs, nodeName, sigmak8sapi.LabelMemOverQuota, "1.1")
		defer framework.RemoveLabelOffNode(cs, nodeName, sigmak8sapi.LabelMemOverQuota)

		// due to bug: 18296033, MUST also add overquota lable in sigma2.0
		swarm.SetNodeOverQuota(nodeName, 1.5, 1.1)
		defer swarm.SetNodeToNotOverQuota(nodeName)

		overQuotaTaint := v1.Taint{
			Key:    sigmak8sapi.LabelEnableOverQuota,
			Value:  "true",
			Effect: v1.TaintEffectNoSchedule,
		}

		framework.AddOrUpdateTaintOnNode(cs, nodeName, overQuotaTaint)
		framework.ExpectNodeHasTaint(cs, nodeName, &overQuotaTaint)
		defer framework.RemoveTaintOffNode(cs, nodeName, overQuotaTaint)

		allocatableCPU := nodeToAllocatableMapCPU[nodeName]
		allocatableMemory := nodeToAllocatableMapMem[nodeName]
		allocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		framework.Logf("allocatableCPU: %d", allocatableCPU)
		framework.Logf("allocatableMemory: %d", allocatableMemory)
		framework.Logf("allocatableDisk: %d", allocatableDisk)

		allocatableCPUAfterQuota := int64(float64(allocatableCPU) * 1.5)
		allocatableMemoryAfterQuota := int64(float64(allocatableMemory) * 1.1)
		allocatableDiskAfterQuota := allocatableDisk

		pod1CPU := (allocatableCPUAfterQuota / 1000) * 1000 / 2
		pod1Memory := allocatableMemoryAfterQuota * 5 / 10
		pod1Disk := allocatableDiskAfterQuota * 5 / 10

		pod2CPU := allocatableCPUAfterQuota - pod1CPU
		pod2Memory := allocatableMemoryAfterQuota - pod1Memory
		pod2Disk := allocatableDiskAfterQuota - pod1Disk

		By("Request a pod with over quota CPU/Memory/EphemeralStorage.")
		tests := []struct {
			cpu                       resource.Quantity
			mem                       resource.Quantity
			ethstorage                resource.Quantity
			expectedAvailableResource []int64 // CPU/Memory/Disk
			spreadStrategy            sigmak8sapi.SpreadStrategy
			expectedScheduleResult    bool
		}{
			// test[0] podA cpu = 1/2 * X, mem = 1/2 * Y，disk = 1/2 * Z
			{
				cpu:                       *resource.NewMilliQuantity(pod1CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod1Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod1Disk, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU, allocatableMemory - pod1Memory, allocatableDisk - pod1Disk},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    true,
			},
			// test[1] podB cpu = X - 1/2 * X - 1，mem = Y - 1/2 * Y - 1M，disk = Z - 1/2 * Z - 1M
			{
				cpu:                       *resource.NewMilliQuantity(pod2CPU-1000, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod2Memory-100*1024*1024, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod2Disk-100*1024*1024, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU + 1000, allocatableMemory - pod1Memory - pod2Memory + 100*1024*1024, allocatableDisk - pod1Disk - pod2Disk + 100*1024*1024},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    true,
			},
			// test[2] podC1 cpu = 1/2 * X, mem = 1M，disk = 1M
			{
				cpu:                       *resource.NewMilliQuantity(pod1CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(100*1024*1024, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(100*1024*1024, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU + 1000, allocatableMemory - pod1Memory - pod2Memory + 100*1024*1024, allocatableDisk - pod1Disk - pod2Disk + 100*1024*1024},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    false,
			},
			// test[3] podC2 cpu = 1, mem = 1/2 * Y，disk = 1M
			{
				cpu:                       *resource.NewMilliQuantity(1000, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod1Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(100*1024*1024, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU + 1000, allocatableMemory - pod1Memory - pod2Memory + 100*1024*1024, allocatableDisk - pod1Disk - pod2Disk + 100*1024*1024},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    false,
			},
			// test[4] podC3 cpu = 1, mem = 1M，disk = 1/2 * Z
			{
				cpu:                       *resource.NewMilliQuantity(1000, "DecimalSI"),
				mem:                       *resource.NewQuantity(100*1024*1024, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod1Disk, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU + 1000, allocatableMemory - pod1Memory - pod2Memory + 100*1024*1024, allocatableDisk - pod1Disk - pod2Disk + 100*1024*1024},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    false,
			},
			// test[5] podC4 cpu = 1, mem = 1M，disk = 1M
			{
				cpu:                       *resource.NewMilliQuantity(1000, "DecimalSI"),
				mem:                       *resource.NewQuantity(100*1024*1024, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(100*1024*1024, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU, allocatableMemory - pod1Memory - pod2Memory, 0},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    true,
			},
			// test[6] podD cpu = 1/2 * X + 1, mem = 1/2 * Y + 1M，disk = 1/2 * Z + 1M
			{
				cpu:                       *resource.NewMilliQuantity(pod2CPU+1000, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod2Memory+100*1024*1024, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod2Disk+100*1024*1024, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU, allocatableMemory - pod1Memory, allocatableDisk - pod1Disk},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    false,
			},
			// test[7] podE cpu = 1/2 * X, mem = 1/2 * Y，disk = 1/2 * Z
			{
				cpu:                       *resource.NewMilliQuantity(pod2CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod2Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod2Disk, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU, allocatableMemory - pod1Memory - pod2Memory, 0},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    true,
			},
			// test[8] podF cpu = 1/2 * X, mem = 1/2 * Y，disk = 1/2 * Z
			{
				cpu:                       *resource.NewMilliQuantity(pod1CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod1Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod1Disk, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU, allocatableMemory - pod1Memory, allocatableDisk - pod1Disk},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    true,
			},
			// test[9] podF cpu = 1/2 * X, mem = 1/2 * Y，disk = 1/2 * Z
			{
				cpu:                       *resource.NewMilliQuantity(pod2CPU, "DecimalSI"),
				mem:                       *resource.NewQuantity(pod2Memory, "DecimalSI"),
				ethstorage:                *resource.NewQuantity(pod2Disk, "DecimalSI"),
				expectedAvailableResource: []int64{allocatableCPU - pod1CPU - pod2CPU, allocatableMemory - pod1Memory - pod2Memory, 0},
				spreadStrategy:            sigmak8sapi.SpreadStrategySameCoreFirst,
				expectedScheduleResult:    true,
			},
		}

		getCPUID := func(pod *v1.Pod) []int {
			// Get pod and check CPUIDs.
			podRunning, err := f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			allocSpecStr := podRunning.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
			allocSpec := &sigmak8sapi.AllocSpec{}
			err = json.Unmarshal([]byte(allocSpecStr), allocSpec)
			Expect(err).NotTo(HaveOccurred())

			CPUIDs := allocSpec.Containers[0].Resource.CPU.CPUSet.CPUIDs
			sort.Ints(CPUIDs)

			return CPUIDs
		}

		// 整个过程循环 2 次
		loopTime := 2
		for t := 1; t <= loopTime; t++ {
			podsToDelete := []*v1.Pod{}
			processorIDToCntMap := make(map[int]int)
			for i, test := range tests {
				podName := "scheduler-e2e-resource-" + strconv.Itoa(i) + "-" + string(uuid.NewUUID())
				allocSpecRequest := &sigmak8sapi.AllocSpec{
					Containers: []sigmak8sapi.Container{
						{
							Name: podName,
							Resource: sigmak8sapi.ResourceRequirements{
								CPU: sigmak8sapi.CPUSpec{
									CPUSet: &sigmak8sapi.CPUSetSpec{
										SpreadStrategy: test.spreadStrategy,
									},
								},
							},
						},
					},
				}

				allocSpecBytes, err := json.Marshal(&allocSpecRequest)
				if err != nil {
					return
				}

				pod := createPausePod(f, pausePodConfig{
					Name: podName,
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(allocSpecBytes),
					},
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
					framework.Logf("Case[%d][%d], expect pod to be scheduled successfully.", t, i)
					err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
					podsToDelete = append(podsToDelete, pod)
					Expect(err).NotTo(HaveOccurred())

					// Get pod and check CPUIDs.
					CPUIDs := getCPUID(pod)
					framework.Logf("AllocSpec.CPUIDs: %v", CPUIDs)

					for _, cpu := range CPUIDs {
						processorIDToCntMap[cpu]++
					}
					checkResult := checkCPUOverQuotaCoreBinding(processorIDToCntMap, int(allocatableCPU/1000), 1.5)
					Expect(checkResult).Should(Equal(true), "checkCPUOverQuotaCoreBinding should pass")

				} else {
					framework.Logf("Case[%d][%d], expect pod failed to be scheduled.", t, i)
					err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
					podsToDelete = append(podsToDelete, pod)
					Expect(err).To(BeNil(), "Case[%d][%d], expect err not be nil, got %s", t, i, err)
				}

				ar := getAvailableResourceOnNode(f, nodeName)
				for j := 0; j < len(ar); j++ {
					framework.Logf("Case[%d][%d], AvailableResource[%d]: %d.", t, i, j, ar[j])
					Expect(ar[j]).Should(Equal(test.expectedAvailableResource[j]), "available resource should match to expected")
				}

				// test[5] podC4 创建成功后删除 pod C1，C2, C3, C4 和 podB
				if i == 5 {
					for j, pod := range podsToDelete {
						if j == 0 {
							continue
						}
						if pod == nil {
							continue
						}
						CPUIDs := getCPUID(pod)
						for _, cpu := range CPUIDs {
							processorIDToCntMap[cpu]--
						}
						err := util.DeletePod(f.ClientSet, pod)
						Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
					}

					By("Get available resource after all pods are deleted.")
					expectedAvailableResourceAfterAllPodsAreDeleted := []int64{allocatableCPU - pod1CPU, allocatableMemory - pod1Memory, allocatableDisk - pod1Disk}

					ar := getAvailableResourceOnNode(f, nodeName)
					for j := 0; j < len(ar); j++ {
						framework.Logf("AvailableResource[%d]: %d.", j, ar[j])
						Expect(ar[j]).Should(Equal(expectedAvailableResourceAfterAllPodsAreDeleted[j]), "available resource should match to expected")
					}
					continue
				}

				// test[7] podE 创建成功后删除 podA，podE
				if i == 7 {
					for j, pod := range podsToDelete {
						if j == 0 || j == 7 {
							if pod == nil {
								continue
							}
							CPUIDs := getCPUID(pod)
							for _, cpu := range CPUIDs {
								processorIDToCntMap[cpu]--
							}
							err := util.DeletePod(f.ClientSet, pod)
							Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
						}
					}

					By("Get available resource after all pods are deleted.")
					expectedAvailableResourceAfterAllPodsAreDeleted := []int64{allocatableCPU, allocatableMemory, allocatableDisk}

					ar := getAvailableResourceOnNode(f, nodeName)
					for j := 0; j < len(ar); j++ {
						framework.Logf("AvailableResource[%d]: %d.", j, ar[j])
						Expect(ar[j]).Should(Equal(expectedAvailableResourceAfterAllPodsAreDeleted[j]), "available resource should match to expected")
					}
					continue
				}

				if i == 9 {
					for j, pod := range podsToDelete {
						if j == 8 || j == 9 {
							if pod == nil {
								continue
							}
							CPUIDs := getCPUID(pod)
							for _, cpu := range CPUIDs {
								processorIDToCntMap[cpu]--
							}
							err := util.DeletePod(f.ClientSet, pod)
							Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
						}
					}
				}
			}
		}
	})
})
