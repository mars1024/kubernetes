package scheduler

import (
	"encoding/json"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
 	"k8s.io/kubernetes/test/sigma/util"

	// "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

var _ = Describe("[sigma-3.1][sigma-scheduler][cpuset][cpu]", func() {
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
			//etcdNodeinfo := swarm.GetNode(node.Name)
			//nodeToAllocatableMapCPU[node.Name] = int64(etcdNodeinfo.LocalInfo.CpuNum * 1000)
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
		// delete sigma 2.0 allocplan if exists
		DeleteSigmaContainer(f)
	})

	// 多 container 的 CPUSet 调度（非超卖）
	// k8s 非超卖的单链路节点上，绑核的操作不会重复使用相同的核，
	// 同一个核不会被超卖掉，即只能分配一次，资源回收之后，可以重复使用
	// 需要测试从 0->1->2->1->0 的复杂场景，
	// 并且各种绑核的策略需要交叉组合测试，以及策略降级的测试
	// 前置：集群中单节点上可分配的 CPU 资源大于 8

	// 注意，这个 case 和 cpuset_k8s_case000 基本上一样的，
	// 只是每个 pod 会随机创建 1-2 sidecar container，request = 0，
	// 模拟多 container 的 CPUSet 模式

	// 步骤：
	// 1. 获取 NodeA 上可分配的 CPU 额度 X
	// 2. k8s 创建新的 PodA，设置 Requests.CPU = 1/2 * X、Limits.CPU = 1/2 * X，
	//    同时设置 Pod.Annotations.AllocSpec...CpuSet.SpreadStrategy="spread"，
	//    观察调度结果，获取 Pod 上分配的 CPUIDs 信息，以及 Node 上的 CPUSharePool 信息
	// 3. k8s 创建新的 PodB，设置 Requests.CPU = X - (1/2 * X)、Limits.CPU = X - (1/2 * X)，
	//    同时设置 Pod.Annotations.AllocSpec...CpuSet.SpreadStrategy="sameCoreFirst"，
	//    观察调度结果，获取 Pod 上分配的 CPUIDs 信息，以及 Node 上的 CPUSharePool 信息
	// 4. 删掉所有 Pod，使用 1/4 的 CPU 额度，创建 2 个 spread 的 Pod，
	//    再创建 2 个 sameCoreFirst 的 Pod，观察调度结果，
	//    获取 Pod 上分配的 CPUIDs 信息，以及 Node 上的 CPUSharePool 信息
	// 5. 删掉所有 Pod，使用 1/8 的 CPU 额度，如果不能整除，那么剩下的作为最后一个 Pod 额度，
	//    随机选择 SpreadStrategy，创建 9 个 Pod，
	//    观察调度结果，获取 Pod 上分配的 CPUIDs 信息，以及 Node 上的 CPUSharePool 信息

	// 验证结果：
	// 1. 第二步第三步 Pod 调度成功，SpreadStrategy 符合预期，
	//    并且每个核只能被分配一次，Node 的 CPUSharePool 得到更新，减去了以及做了绑核的 CPUIDs
	// 2. 第四步中 Pod 调度成功，SpreadStrategy 符合预期，并且每个核只能被分配一次，
	//    Node 的 CPUSharePool 得到更新，减去了以及做了绑核的 CPUIDs
	// 3. 第五步中前九个 Pod 调度成功，SpreadStrategy 不做检查，每个核只能被分配一次，
	//    Node 的 CPUSharePool 得到更新，减去了以及做了绑核的 CPUIDs，第 9 个 Pod 调度失败

	It("[smoke][p0][bvt] cpuset_multi_containers_k8s_case001 A pod (multi container) with cpuset request should match the strategy, otherwise should down grade.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		// Apply node affinity label to each node
		nodeAffinityKey := "node-for-resource-e2e-test-" + string(uuid.NewUUID())
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		allocatableCPU := nodeToAllocatableMapCPU[nodeName] / 1000
		By("Request a pod with CPUSet strategy = spread.")

		// 做四次循环
		// 第一次，请求 1/2 的整机核，创建 "spread" 的 pod 3 个，前两个成功，第三个失败
		// 第二次，请求 1/2 的整机核，创建 "sameCoreFirst" 的 pod 3 个，前两个成功，第三个失败
		// 第三次，请求 1/4 的整机核，"spread"/"sameCoreFirst" 的 pod 各 2 个，均成功，再创建一个，失败
		// 第三次，请求 1/8 的整机核，"spread"/"sameCoreFirst" 随机选择，共创建 8 个 pod，均成功，再创建一个，失败

		type CPUSetTestCase struct {
			CPURequest        int64
			SpreadStrategy    sigmak8sapi.SpreadStrategy
			SkipCheckStrategy bool
			ExpectedResult    bool
			ContainerCount    int
		}

		loopTime := 4
		for loop := 1; loop <= loopTime; loop++ {
			tests := []CPUSetTestCase{}

			cpuRequest := allocatableCPU / 2
			testPodCount := 2
			spreadStrategy := sigmak8sapi.SpreadStrategySpread
			skipCheckStrategy := false
			expectedResult := true

			// 第二次，请求 1/2 的整机核，创建 "sameCoreFirst" 的容器 3 个
			if loop == 2 {
				testPodCount = 2
				spreadStrategy = sigmak8sapi.SpreadStrategySameCoreFirst
				skipCheckStrategy = false
			}

			// 第三次，请求 1/4 的整机核，"spread"/"sameCoreFirst" 的容器各 2 个
			if loop == 3 {
				cpuRequest = allocatableCPU / 4
				testPodCount = 4
				skipCheckStrategy = false
			}

			// 第四次，请求 1/8 的整机核，"spread"/"sameCoreFirst" 随机选择
			if loop == 4 {
				cpuRequest = allocatableCPU / 8
				testPodCount = 8
				skipCheckStrategy = true
			}

			for j := 1; j <= testPodCount; j++ {
				// 第三次，平分 SpreadStrategy
				if loop == 3 {
					spreadStrategy = sigmak8sapi.SpreadStrategySpread
					if j%2 == 0 {
						spreadStrategy = sigmak8sapi.SpreadStrategySameCoreFirst
					}
				}

				// 第三次，随机 SpreadStrategy
				if loop == 4 {
					spreadStrategies := []sigmak8sapi.SpreadStrategy{
						sigmak8sapi.SpreadStrategySpread,
						sigmak8sapi.SpreadStrategySameCoreFirst,
					}

					spreadStrategy = spreadStrategies[rand.Intn(len(spreadStrategies))]
				}

				caseItem := CPUSetTestCase{
					CPURequest:        cpuRequest,
					SpreadStrategy:    spreadStrategy,
					SkipCheckStrategy: skipCheckStrategy,
					ExpectedResult:    expectedResult,
					ContainerCount:    rand.Intn(3) + 1, // rand from 1 - 3
				}
				tests = append(tests, caseItem)
			}

			lastCaseItem := CPUSetTestCase{
				CPURequest:        cpuRequest,
				SpreadStrategy:    spreadStrategy,
				SkipCheckStrategy: skipCheckStrategy,
				ExpectedResult:    false,
				ContainerCount:    rand.Intn(3) + 1, // rand from 1 - 3,
			}

			tests = append(tests, lastCaseItem)

			for index, test := range tests {
				framework.Logf("Run tests[%d][%d]: %+v", loop, index, test)
			}

			podsToDelete := make([]*v1.Pod, len(tests))
			// 用一个 map 来记录每个 CPUID 被分配了几次
			allocatedCPUIDCountMap := make(map[int]int)
			for i, test := range tests {
				name := "e2e-resource-k8s-cpuset-" + strconv.Itoa(i) + "-" + string(uuid.NewUUID())
				resourcesForMultiContainers := []v1.ResourceRequirements{}
				for j := 0; j < test.ContainerCount; j++ {
					// TODO(kubo.cph): 目前在多个 container 的 cpuset 场景中，只有一个业务容器是有 request 值的，
					// 其他 container 都是 sidecar，request 都是 0，
					// 所以下面先把第一个 container 的 request 设置为和 pod 整体的一样；
					// 并且如果有多个 container 的 request > 0 的情况时，sigmascheduling admission 的部分也需要做修改，
					// 否则会报错 the count of cpuIDs is not match pod spec and this pod is not in inplace update process；
					usedCPURequest := int64(0)
					if j == 0 {
						usedCPURequest = test.CPURequest
					}

					framework.Logf("Case[%d]Container[%d], usedCPURequest: %d", i, j, usedCPURequest)
					r := v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU: *resource.NewQuantity(usedCPURequest, "DecimalSI"),
						},
					}

					resourcesForMultiContainers = append(resourcesForMultiContainers, r)
				}

				pod := createPausePod(f, pausePodConfig{
					Name: name,
					Labels: map[string]string{
						sigmak8sapi.LabelAppName:    "pod-app-name-for-resource-e2e-test",
						sigmak8sapi.LabelDeployUnit: "pod-deploy-unit-for-resource-e2e-test",
					},
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategyForMultiContainers(
							name, test.SpreadStrategy, test.ContainerCount),
					},
					Affinity:                    util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
					ResourcesForMultiContainers: resourcesForMultiContainers,
				})

				if test.ExpectedResult == true {
					framework.Logf("Case[%d], expect pod to be scheduled successfully.", i)
					err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
					podsToDelete = append(podsToDelete, pod)
					Expect(err).NotTo(HaveOccurred())

					// Get pod and check CPUIDs.
					podRunning, err := f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					allocSpecStr := podRunning.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
					allocSpec := &sigmak8sapi.AllocSpec{}
					err = json.Unmarshal([]byte(allocSpecStr), allocSpec)
					Expect(err).NotTo(HaveOccurred())

					CPUIDs := allocSpec.Containers[0].Resource.CPU.CPUSet.CPUIDs
					sort.Ints(CPUIDs)
					framework.Logf("Case[%d]Strategy[%s], CPUIDs: %v", i, test.SpreadStrategy, CPUIDs)

					for k, c := range allocSpec.Containers {
						framework.Logf("Case[%d]Strategy[%s], Containers[%d].Resource.CPU.CPUSet: %+v",
							i, test.SpreadStrategy, k, c.Resource.CPU.CPUSet)
						Expect(len(c.Resource.CPU.CPUSet.CPUIDs)).Should(Equal(len(CPUIDs)),
							"length of cpuids for all containers should be equal")
					}

					checkResult := checkCPUSetSpreadStrategy(CPUIDs, int(test.CPURequest), test.SpreadStrategy, test.SkipCheckStrategy)
					Expect(checkResult).Should(Equal(true), "checkCPUSetSpreadStrategy should pass")

					// 统计每个 CPUID 被分配的次数
					for _, cpuid := range CPUIDs {
						allocatedCPUIDCountMap[cpuid]++
					}

					for cpuid, count := range allocatedCPUIDCountMap {
						framework.Logf("Case[%d] allocatedCPUIDCountMap[%d], count: %d", i, cpuid, count)
						Expect(count).Should(Equal(1), "one cpuid should be allocated once and only once without over quota")
					}

					// TODO(kubo.cph): 这里可以顺便检查一下 Node 上面的 CPUSharePool 是不是动态调整了
				} else {
					framework.Logf("Case[%d], expect pod failed to be scheduled.", i)
					podsToDelete = append(podsToDelete, pod)
					err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
					Expect(err).To(BeNil(), "expect err to be nil, got %s", err)
				}
			}

			// 已经分配的 CPUIDs 的核数，要正好等于可分配的核数，即每个 CPUID 都被分配了一次
			Expect(len(allocatedCPUIDCountMap)).Should(Equal(int(allocatableCPU)), "all cpuids should be allocated once.")

			wg := &sync.WaitGroup{}
			for _, pod := range podsToDelete {
				if pod == nil {
					continue
				}
				wg.Add(1)
				go func(pod *v1.Pod) {
					defer wg.Done()
					err := util.DeletePod(f.ClientSet, pod)
					Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
				}(pod)
			}
			wg.Wait()
		}
	})
})
