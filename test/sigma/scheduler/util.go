package scheduler

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/common"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/sigma/env"
	"k8s.io/kubernetes/test/sigma/swarm"
	"k8s.io/kubernetes/pkg/util/system"
)

// variable set in BeforeEach, never modified afterwards
var masterNodes sets.String

// the timeout of waiting for pod running.
var waitForPodRunningTimeout = 5 * time.Minute

type pausePodConfig struct {
	Name                              string
	Affinity                          *v1.Affinity
	Annotations, Labels, NodeSelector map[string]string
	Resources                         *v1.ResourceRequirements
	Tolerations                       []v1.Toleration
	NodeName                          string
	Ports                             []v1.ContainerPort
	OwnerReferences                   []metav1.OwnerReference
	PriorityClassName                 string
}

func initPausePod(f *framework.Framework, conf pausePodConfig) *v1.Pod {
	pauseImage := util.SigmaPauseImage
	if pauseImage == "" {
		pauseImage = "k8s.gcr.io/pause-amd64"
	}

	if conf.Labels == nil {
		conf.Labels = make(map[string]string)
	}

	if _, ok := conf.Labels[sigmak8sapi.LabelInstanceGroup]; !ok {
		conf.Labels[sigmak8sapi.LabelInstanceGroup] = "scheduler-e2e-instance-group"
	}

	if _, ok := conf.Labels[sigmak8sapi.LabelSite]; !ok {
		conf.Labels[sigmak8sapi.LabelSite] = "scheduler-e2e-site"
	}

	if _, ok := conf.Labels[sigmak8sapi.LabelAppName]; !ok {
		conf.Labels[sigmak8sapi.LabelAppName] = "scheduler-e2e-app-name"
	}

	if _, ok := conf.Labels[sigmak8sapi.LabelDeployUnit]; !ok {
		conf.Labels[sigmak8sapi.LabelDeployUnit] = "scheduler-e2e-depoly-unit"
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            conf.Name,
			Labels:          conf.Labels,
			Annotations:     conf.Annotations,
			OwnerReferences: conf.OwnerReferences,
		},
		Spec: v1.PodSpec{
			NodeSelector: conf.NodeSelector,
			Affinity:     conf.Affinity,
			Containers: []v1.Container{
				{
					Name:  conf.Name,
					Image: pauseImage,
					Ports: conf.Ports,
				},
			},
			Tolerations:       conf.Tolerations,
			NodeName:          conf.NodeName,
			PriorityClassName: conf.PriorityClassName,
		},
	}
	if conf.Resources != nil {
		pod.Spec.Containers[0].Resources = *conf.Resources
	}
	if env.GetTester() == env.TesterJituan {
		pod.Spec.Tolerations = append(pod.Spec.Tolerations, v1.Toleration{
			Key:      sigmak8sapi.LabelResourcePool,
			Operator: v1.TolerationOpExists,
			Effect:   v1.TaintEffectNoSchedule,
		})
	}
	return pod
}

func createPausePod(f *framework.Framework, conf pausePodConfig) *v1.Pod {
	pod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(initPausePod(f, conf))
	framework.ExpectNoError(err)
	return pod
}

func runPausePod(f *framework.Framework, conf pausePodConfig) *v1.Pod {
	pod := createPausePod(f, conf)
	framework.ExpectNoError(framework.WaitForPodRunningInNamespace(f.ClientSet, pod))
	pod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(conf.Name, metav1.GetOptions{})
	framework.ExpectNoError(err)
	return pod
}

func runPodAndGetNodeName(f *framework.Framework, conf pausePodConfig) string {
	// launch a pod to find a node which can launch a pod. We intentionally do
	// not just take the node list and choose the first of them. Depending on the
	// cluster and the scheduler it might be that a "normal" pod cannot be
	// scheduled onto it.
	pod := runPausePod(f, conf)

	By("Explicitly delete pod here to free the resource it takes.")
	err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Delete(pod.Name, metav1.NewDeleteOptions(0))
	framework.ExpectNoError(err)

	return pod.Spec.NodeName
}

func getRequestedCPU(pod v1.Pod) int64 {
	var result int64
	for _, container := range pod.Spec.Containers {
		result += container.Resources.Requests.Cpu().MilliValue()
	}
	return result
}

func getRequestedMem(pod v1.Pod) int64 {
	var result int64
	for _, container := range pod.Spec.Containers {
		result += container.Resources.Requests.Memory().Value()
	}
	return result
}

func getRequestedStorageEphemeralStorage(pod v1.Pod) int64 {
	var result int64
	for _, container := range pod.Spec.Containers {
		result += container.Resources.Requests.StorageEphemeral().Value()
	}
	return result
}

// removeTaintFromNodeAction returns a closure that removes the given taint
// from the given node upon invocation.
func removeTaintFromNodeAction(cs clientset.Interface, nodeName string, testTaint v1.Taint) common.Action {
	return func() error {
		framework.RemoveTaintOffNode(cs, nodeName, testTaint)
		return nil
	}
}

// createPausePodAction returns a closure that creates a pause pod upon invocation.
func createPausePodAction(f *framework.Framework, conf pausePodConfig) common.Action {
	return func() error {
		_, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(initPausePod(f, conf))
		return err
	}
}

// WaitForSchedulerAfterAction performs the provided action and then waits for
// scheduler to act on the given pod.
func WaitForSchedulerAfterAction(f *framework.Framework, action common.Action, podName string, expectSuccess bool) {
	predicate := scheduleFailureEvent(podName)
	if expectSuccess {
		predicate = scheduleSuccessEvent(podName, "" /* any node */)
	}
	success, err := common.ObserveEventAfterAction(f, predicate, action)
	Expect(err).NotTo(HaveOccurred())
	Expect(success).To(Equal(true))
}

// TODO: upgrade calls in PodAffinity tests when we're able to run them
func verifyResult(c clientset.Interface, expectedScheduled int, expectedNotScheduled int, ns string) {
	allPods, err := c.CoreV1().Pods(ns).List(metav1.ListOptions{})
	framework.ExpectNoError(err)
	scheduledPods, notScheduledPods := framework.GetPodsScheduled(masterNodes, allPods)

	printed := false
	printOnce := func(msg string) string {
		if !printed {
			printed = true
			return msg
		}
		return ""
	}

	Expect(len(notScheduledPods)).To(Equal(expectedNotScheduled), printOnce(fmt.Sprintf("Not scheduled Pods: %#v", notScheduledPods)))
	Expect(len(scheduledPods)).To(Equal(expectedScheduled), printOnce(fmt.Sprintf("Scheduled Pods: %#v", scheduledPods)))
}

// verifyReplicasResult is wrapper of verifyResult for a group pods with same "name: labelName" label, which means they belong to same RC
func verifyReplicasResult(c clientset.Interface, expectedScheduled int, expectedNotScheduled int, ns string, labelName string) {
	allPods := getPodsByLabels(c, ns, map[string]string{"name": labelName})
	scheduledPods, notScheduledPods := framework.GetPodsScheduled(masterNodes, allPods)

	printed := false
	printOnce := func(msg string) string {
		if !printed {
			printed = true
			return msg
		}
		return ""
	}

	Expect(len(notScheduledPods)).To(Equal(expectedNotScheduled), printOnce(fmt.Sprintf("Not scheduled Pods: %#v", notScheduledPods)))
	Expect(len(scheduledPods)).To(Equal(expectedScheduled), printOnce(fmt.Sprintf("Scheduled Pods: %#v", scheduledPods)))
}

func getPodsByLabels(c clientset.Interface, ns string, labelsMap map[string]string) *v1.PodList {
	selector := labels.SelectorFromSet(labels.Set(labelsMap))
	allPods, err := c.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: selector.String()})
	framework.ExpectNoError(err)
	return allPods
}

func runAndKeepPodWithLabelAndGetNodeName(f *framework.Framework) (string, string) {
	// launch a pod to find a node which can launch a pod. We intentionally do
	// not just take the node list and choose the first of them. Depending on the
	// cluster and the scheduler it might be that a "normal" pod cannot be
	// scheduled onto it.
	By("Trying to launch a pod with a label to get a node which can launch it.")
	pod := runPausePod(f, pausePodConfig{
		Name:   "with-label-" + string(uuid.NewUUID()),
		Labels: map[string]string{"security": "S1"},
	})
	return pod.Spec.NodeName, pod.Name
}

// GetNodeThatCanRunPod return a nodename that can run pod.
func GetNodeThatCanRunPod(f *framework.Framework) string {
	By("Trying to launch a pod without a label to get a node which can launch it.")
	return runPodAndGetNodeName(f, pausePodConfig{Name: "without-label"})
}

func getNodeThatCanRunPodWithoutToleration(f *framework.Framework) string {
	By("Trying to launch a pod without a toleration to get a node which can launch it.")
	return runPodAndGetNodeName(f, pausePodConfig{Name: "without-toleration"})
}

// create pod which using hostport on the specified node according to the nodeSelector
func creatHostPortPodOnNode(f *framework.Framework, podName, ns, hostIP string, port int32, protocol v1.Protocol, nodeSelector map[string]string, expectScheduled bool) {
	createPausePod(f, pausePodConfig{
		Name: podName,
		Ports: []v1.ContainerPort{
			{
				HostPort:      port,
				ContainerPort: 80,
				Protocol:      protocol,
				HostIP:        hostIP,
			},
		},
		NodeSelector: nodeSelector,
	})

	err := framework.WaitForPodNotPending(f.ClientSet, ns, podName)
	if expectScheduled {
		framework.ExpectNoError(err)
	}
}

func scheduleSuccessEvent(podName, nodeName string) func(*v1.Event) bool {
	return func(e *v1.Event) bool {
		return e.Type == v1.EventTypeNormal &&
			e.Reason == "Scheduled" &&
			strings.HasPrefix(e.Name, podName) &&
			strings.Contains(e.Message, fmt.Sprintf("Successfully assigned %v to %v", podName, nodeName))
	}
}

func scheduleFailureEvent(podName string) func(*v1.Event) bool {
	return func(e *v1.Event) bool {
		return strings.HasPrefix(e.Name, podName) &&
			e.Type == "Warning" &&
			e.Reason == "FailedScheduling"
	}
}

// getAvailableResourceOnNode return available cpu/memory/disk size of this node.
// TODO, we need to take sigma node into account.
func getAvailableResourceOnNode(f *framework.Framework, nodeName string) []int64 {
	node, err := f.ClientSet.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	framework.ExpectNoError(err)

	var availableCPU, availableMemory, availableDisk int64 = 0, 0, 0

	allocatableCPU, found := node.Status.Allocatable[v1.ResourceCPU]
	Expect(found).To(Equal(true))
	availableCPU = allocatableCPU.MilliValue()

	allocatableMomory, found := node.Status.Allocatable[v1.ResourceMemory]
	Expect(found).To(Equal(true))
	availableMemory = allocatableMomory.Value()

	allocatableDisk, found := node.Status.Allocatable[v1.ResourceEphemeralStorage]
	Expect(found).To(Equal(true))
	availableDisk = allocatableDisk.Value()

	pods, err := f.ClientSet.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fields.Set{
			"spec.nodeName": nodeName,
		}.AsSelector().String(),
	})
	framework.ExpectNoError(err)

	for _, pod := range pods.Items {
		if pod.Status.Phase != v1.PodSucceeded && pod.Status.Phase != v1.PodFailed {
			availableCPU -= getRequestedCPU(pod)
			availableMemory -= getRequestedMem(pod)
			availableDisk -= getRequestedStorageEphemeralStorage(pod)
		}
	}

	return []int64{availableCPU, availableMemory, availableDisk}
}

func formatAllocSpecStringWithSpreadStrategy(name string, strategy sigmak8sapi.SpreadStrategy) string {
	allocSpecRequest := &sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			{
				Name: name,
				Resource: sigmak8sapi.ResourceRequirements{
					CPU: sigmak8sapi.CPUSpec{
						CPUSet: &sigmak8sapi.CPUSetSpec{
							SpreadStrategy: strategy,
						},
					},
				},
			},
		},
	}

	allocSpecBytes, err := json.Marshal(&allocSpecRequest)
	if err != nil {
		return ""
	}

	return string(allocSpecBytes)
}

// checkCPUSetSpreadStrategy check the given CPUIDs whether match the given strategy.
func checkCPUSetSpreadStrategy(cpuIDs []int, physicalCoreCount int, strategy sigmak8sapi.SpreadStrategy, skip bool) bool {
	if skip {
		return true
	}

	sort.Ints(cpuIDs)

	// Use a map to save physicalCoreID and it's processor (logic core) count.
	physicalCoreIDToProcessorCountMap := make(map[int]int)

	for i := 0; i < len(cpuIDs); i++ {
		coreID := cpuIDs[i] % physicalCoreCount
		physicalCoreIDToProcessorCountMap[coreID]++
	}

	switch strategy {
	case sigmak8sapi.SpreadStrategySpread:
		for k, v := range physicalCoreIDToProcessorCountMap {
			if v > 1 {
				framework.Logf("PhysicalCoreID[%d] has %d processors, not match spread strategy.", k, v)
				return false
			}
		}

		return true
	case sigmak8sapi.SpreadStrategySameCoreFirst:
		for k, v := range physicalCoreIDToProcessorCountMap {
			if len(physicalCoreIDToProcessorCountMap) >= 2 && v < 2 {
				framework.Logf("PhysicalCoreID[%d] has %d processors, not match sameCoreFirst strategy.", k, v)
				return false
			}
		}
		return true
	default:
		return false
	}
}

// 检查cpu overquota 下总核数不超过上限，每个核的使用次数不超过 overquota 允许的个数，例如：
// overquota = 2 时， 每个核不超过 2 次
// overquota = 1.5 时， 有一半的核不超过 2 次，剩下的不超过 1 次
func checkCPUOverQuotaCoreBinding(processorIDToCntMap map[int]int, cpuTotalNum int, overQuota float64) bool {
	overQuotaCpuNum := 0
	for _, cnt := range processorIDToCntMap {
		if cnt > int(math.Ceil(overQuota)) {
			return false
		} else {
			overQuotaCpuNum += cnt
		}
	}

	if overQuotaCpuNum > cpuTotalNum*int(math.Ceil(overQuota)) {
		return false
	}
	return true
}

// if cnt==0, print unlimit nodes
func DumpSchedulerState(f *framework.Framework, cnt int) {
	nodes, _ := f.ClientSet.Core().Nodes().List(metav1.ListOptions{})
	if nodes == nil {
		return
	}

	if cnt < 0 {
		cnt = 0
	}
	if cnt > len(nodes.Items) {
		cnt = len(nodes.Items)
	}
	for index, node := range nodes.Items {
		if index > cnt && cnt != 0 {
			break
		}
		allocplan := swarm.GetHostPod(node.Name)
		logrus.Infof("******** Output of scheduler allocplan")
		tmp, _ := json.MarshalIndent(allocplan, "  ", "  ")
		if tmp != nil {
			fmt.Println(string(tmp))
		} else {
			fmt.Println("allocplan is nil")
		}
	}
}

// DeleteSigmaContainer delete sigma2.0 container with specified appname and duname
func DeleteSigmaContainer(f *framework.Framework) error {
	nodes, _ := f.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
	if nodes == nil {
		return nil
	}
	for _, node := range nodes.Items {
		alloPlans := swarm.GetHostPod(node.Name)
		for _, alloc := range alloPlans {
			if alloc.AppName == "phyhost-ecs-trade" && alloc.DeployUnit == "container-test1" {
				swarm.Delete(alloc.InstanceSn)
			}
		}
	}
	return WaitSchedulerAllocPlanClean(f)
}

func WaitSchedulerAllocPlanClean(f *framework.Framework) error {
	if env.Tester == env.TesterJituan {
		return wait.Poll(2*time.Second, 5*time.Minute, func() (done bool, err error) {
			nodes, _ := f.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
			if nodes == nil {
				return true, nil
			}
			for _, node := range nodes.Items {
				podMap := swarm.GetHostPod(node.Name)
				if len(podMap) == 0 {
					return true, nil
				} else {
					for _, pod := range podMap {
						if pod.AppName == "phyhost-ecs-trade" && pod.DeployUnit == "container-test1" {
							return false, nil
						}
					}
				}
			}
			return true, nil
		})
	}
	return nil
}

// the same fuction as in framwork, except that skip the taint node check.
func getMasterAndWorkerNodesOrDie(c clientset.Interface) (sets.String, *v1.NodeList) {
	nodes := &v1.NodeList{}
	masters := sets.NewString()
	all, err := c.CoreV1().Nodes().List(metav1.ListOptions{})
	Expect(err).To(BeNil())
	for _, n := range all.Items {
		if system.IsMasterNode(n.Name) {
			masters.Insert(n.Name)
		} else if isNodeSchedulable(&n) {
			nodes.Items = append(nodes.Items, n)
		}
	}
	return masters, nodes
}

func isNodeSchedulable(node *v1.Node) bool {
	nodeReady := framework.IsNodeConditionSetAsExpected(node, v1.NodeReady, true)
	networkReady := framework.IsNodeConditionUnset(node, v1.NodeNetworkUnavailable) ||
		framework.IsNodeConditionSetAsExpectedSilent(node, v1.NodeNetworkUnavailable, false)
	return !node.Spec.Unschedulable && nodeReady && networkReady
}
