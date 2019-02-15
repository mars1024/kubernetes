package scheduler

import (
	"encoding/json"
	"sort"
	"strconv"
	"sync"
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
 	"k8s.io/kubernetes/test/sigma/util"
)

type PodsToDelete struct {
	mu   sync.Mutex
	pods map[string]*v1.Pod
}

var _ = Describe("[sigma-3.1][sigma-scheduler][performance][Serial]", func() {
	var cs clientset.Interface
	var ns string
	var nodeList *v1.NodeList
	var systemPodsNo int

	nodeToAllocatableMapCPU := make(map[string]int64)
	nodesInfo := make(map[string]*v1.Node)

	ignoreLabels := framework.ImagePullerLabels

	f := framework.NewDefaultFramework("sigma-scheduler-performance")

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

		}
		pods, err := cs.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{})
		framework.ExpectNoError(err)
		for _, pod := range pods.Items {
			_, found := nodeToAllocatableMapCPU[pod.Spec.NodeName]
			if found && pod.Status.Phase != v1.PodSucceeded && pod.Status.Phase != v1.PodFailed {
				nodeToAllocatableMapCPU[pod.Spec.NodeName] -= getRequestedCPU(pod)
			}
		}
	})

	// 并发创建多个 Pod 的验证
	It("[performance][p2] performance_k8s_001 Create a lot of pods simultaneously.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		// Apply node label to each node
		nodeAffinityKey := "performance-e2e-test-" + string(uuid.NewUUID())
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		nodeInfo := nodesInfo[nodeName]
		// 设置成非超卖的机器
		if value, ok := nodeInfo.Labels[sigmak8sapi.LabelEnableOverQuota]; ok {
			if value == "true" {
				framework.AddOrUpdateLabelOnNode(cs, nodeName, sigmak8sapi.LabelEnableOverQuota, "false")
				defer framework.AddOrUpdateLabelOnNode(cs, nodeName, sigmak8sapi.LabelEnableOverQuota, "true")
			}
		}

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]

		// 使用 1 个 CPU 核做创建
		podCPU := int64(1000)

		numberOfPods := int(AllocatableCPU / podCPU)
		spreadStrategy := sigmak8sapi.SpreadStrategySameCoreFirst
		framework.Logf("AllocatableCPU: %d, numberOfPods: %d, spreadStrategy: %s", AllocatableCPU, numberOfPods, spreadStrategy)

		podsToDelete := &PodsToDelete{
			pods: make(map[string]*v1.Pod, numberOfPods),
		}

		// 起多个 goroutine 创建多个 Pod
		wg1 := &sync.WaitGroup{}
		for i := 1; i <= numberOfPods; i++ {
			wg1.Add(1)
			podName := "scheduler-e2e-performance-" + strconv.Itoa(i)
			go createPausePodsAndAddItToMap(f, podName, nodeName, nodeAffinityKey, podCPU, spreadStrategy, podsToDelete, wg1)
		}

		wg1.Wait()

		// 用一个 map 来记录每个 CPUID 被分配了几次
		allocatedCPUIDCountMap := make(map[int]int)

		// 等待 Pod 被调度成功，获取 Pod 的 CPUID 信息，并检查是否有核重叠
		time.Sleep(10 * time.Second)
		for name, pod := range podsToDelete.pods {
			err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, 10*time.Minute)
			Expect(err).NotTo(HaveOccurred())

			podRunning, err := f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			allocSpecStr := podRunning.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
			allocSpec := &sigmak8sapi.AllocSpec{}
			err = json.Unmarshal([]byte(allocSpecStr), allocSpec)
			Expect(err).NotTo(HaveOccurred())

			CPUIDs := allocSpec.Containers[0].Resource.CPU.CPUSet.CPUIDs
			sort.Ints(CPUIDs)
			framework.Logf("Pod[%s]Strategy[%s], CPUIDs: %v", name, spreadStrategy, CPUIDs)

			Expect(len(CPUIDs)).Should(Equal(int(podCPU/1000)), "length of CPUIDs should be equal to pod request CPUs")

			checkResult := checkCPUSetSpreadStrategy(CPUIDs, int(podCPU/1000), spreadStrategy, false)
			Expect(checkResult).Should(Equal(true), "checkCPUSetSpreadStrategy should pass")

			// 统计每个 CPUID 被分配的次数
			for _, cpuid := range CPUIDs {
				allocatedCPUIDCountMap[cpuid]++
			}

			for cpuid, count := range allocatedCPUIDCountMap {
				if count > 1 {
					framework.Logf("allocatedCPUIDCountMap[%d], count: %d", cpuid, count)
				}

				Expect(count).Should(Equal(1), "one cpuid should be allocated once and only once without over quota")
			}
		}

		// 删除 Pod
		wg2 := &sync.WaitGroup{}
		for _, pod := range podsToDelete.pods {
			if pod == nil {
				continue
			}
			wg2.Add(1)
			go func(pod *v1.Pod) {
				defer wg2.Done()
				err := util.DeletePod(f.ClientSet, pod)
				Expect(err).NotTo(HaveOccurred(), "delete pod should succeed")
			}(pod)
		}
		wg2.Wait()
	})
})

func createPausePodsAndAddItToMap(f *framework.Framework, podName, nodeName, nodeAffinityKey string, podCPU int64,
	strategy sigmak8sapi.SpreadStrategy, podsToDelete *PodsToDelete, wg *sync.WaitGroup) {
	framework.Logf("createPausePodsAndAddItToMap, podName: %s", podName)
	pod := createPausePod(f, pausePodConfig{
		Name: podName,
		Resources: &v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceCPU: *resource.NewMilliQuantity(podCPU, "DecimalSI"),
			},
			Requests: v1.ResourceList{
				v1.ResourceCPU: *resource.NewMilliQuantity(podCPU, "DecimalSI"),
			},
		},
		Annotations: map[string]string{
			sigmak8sapi.AnnotationPodAllocSpec: formatAllocSpecStringWithSpreadStrategy(podName, strategy),
		},
		Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
	})

	podsToDelete.mu.Lock()
	podsToDelete.pods[podName] = pod
	podsToDelete.mu.Unlock()
	wg.Done()
}
