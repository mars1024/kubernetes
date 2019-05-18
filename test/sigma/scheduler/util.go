package scheduler

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	"gitlab.alibaba-inc.com/sigma/sigma-k8s-extensions/pkg/apis/apps/v1beta1"
	extclientset "gitlab.alibaba-inc.com/sigma/sigma-k8s-extensions/pkg/client/clientset"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/util/system"
	"k8s.io/kubernetes/test/sigma/env"
	"k8s.io/kubernetes/test/sigma/swarm"
)

// variable set in BeforeEach, never modified afterwards
var masterNodes sets.String

// the timeout of waiting for pod running.
var waitForPodRunningTimeout = 5 * time.Minute

const (
	dafaultPausePod = "default-pause-pod"
	containerPrefix = "container-"
)

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
	ResourcesForMultiContainers       []v1.ResourceRequirements
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

	if conf.Annotations == nil {
		conf.Annotations = make(map[string]string)
	}

	if _, ok := conf.Annotations[alipaysigmak8sapi.AnnotationZappinfo]; !ok {
		conf.Annotations[alipaysigmak8sapi.AnnotationZappinfo] =
			`{"spec":{"appName":"scheduler-e2e-app-name","zone":"GZ00B","serverType":"DOCKER","fqdn":""},"status":{"registered":true}}`
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

	if len(conf.ResourcesForMultiContainers) > 0 {
		pod.Spec.Containers = []v1.Container{}
	}

	for i, r := range conf.ResourcesForMultiContainers {
		c := v1.Container{
			Name:      containerPrefix + strconv.Itoa(i),
			Image:     pauseImage,
			Ports:     conf.Ports,
			Resources: r,
		}
		pod.Spec.Containers = append(pod.Spec.Containers, c)
	}

	resourceRequests := v1.ResourceList{
		v1.ResourceCPU:              *resource.NewMilliQuantity(10, resource.DecimalSI),
		v1.ResourceMemory:           *resource.NewQuantity(1024*1024*512, resource.BinarySI),
		v1.ResourceEphemeralStorage: *resource.NewQuantity(1024*1024*1024, resource.BinarySI),
	}

	resourceLimits := v1.ResourceList{
		v1.ResourceCPU:              *resource.NewMilliQuantity(10, resource.DecimalSI),
		v1.ResourceMemory:           *resource.NewQuantity(1024*1024*512, resource.BinarySI),
		v1.ResourceEphemeralStorage: *resource.NewQuantity(1024*1024*1024, resource.BinarySI),
	}

	dafaultResourceRequirements := v1.ResourceRequirements{
		Limits:   resourceLimits,
		Requests: resourceRequests,
	}

	if pod.Name == dafaultPausePod {
		pod.Spec.Containers[0].Resources = dafaultResourceRequirements
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

	// wait until pod is deleted
	timeout := 2 * time.Minute
	t := time.Now()
	for {
		_, err := f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil && strings.Contains(err.Error(), "not found") {
			framework.Logf("pod %s has been removed", pod.Name)
			return pod.Spec.NodeName
		}
		if time.Since(t) >= timeout {
			framework.Logf("Gave up waiting for pod %s is removed after %v seconds",
				pod.Name, time.Since(t).Seconds())
			return ""
		}
		framework.Logf("Retrying to check whether pod %s is removed", pod.Name)
		time.Sleep(1 * time.Second)
	}

	return pod.Spec.NodeName
}

func getRequestedCPU(pod v1.Pod) int64 {
	var result int64
	for _, container := range pod.Spec.Containers {
		result += container.Resources.Requests.Cpu().MilliValue()
	}
	return result
}

func getRequestedColocationCPU(pod v1.Pod) int64 {
	var result int64
	for _, container := range pod.Spec.Containers {
		value, found := container.Resources.Requests[alipaysigmak8sapi.SigmaBEResourceName]
		if found {
			result += value.Value()
		}
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

// GetNodeThatCanRunPod return a node name that can run pod.
func GetNodeThatCanRunPod(f *framework.Framework) string {
	By("Trying to launch a pod without a label to get a node which can launch it.")
	return runPodAndGetNodeName(f, pausePodConfig{Name: dafaultPausePod})
}

// GetNodeThatCanRunConlocationPod return a node name that can run colocation pod.
func GetNodeThatCanRunColocationPod(f *framework.Framework) string {
	By("Trying to launch a pod without a label to get a colocation node which can launch it.")
	return runPodAndGetNodeName(f, pausePodConfig{
		Name: dafaultPausePod,
		Labels: map[string]string{
			sigmak8sapi.LabelPodQOSClass: string(sigmak8sapi.SigmaQOSBestEffort),
		},
	})
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

func formatAllocSpecStringWithSpreadStrategyForMultiContainers(name string, strategy sigmak8sapi.SpreadStrategy, count int) string {
	allocSpecRequest := &sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{},
	}

	for i := 0; i < count; i++ {
		c := sigmak8sapi.Container{
			Name: containerPrefix + strconv.Itoa(i),
			Resource: sigmak8sapi.ResourceRequirements{
				CPU: sigmak8sapi.CPUSpec{
					CPUSet: &sigmak8sapi.CPUSetSpec{
						SpreadStrategy: strategy,
					},
				},
			},
		}
		allocSpecRequest.Containers = append(allocSpecRequest.Containers, c)
	}

	allocSpecBytes, err := json.Marshal(&allocSpecRequest)
	if err != nil {
		return ""
	}

	return string(allocSpecBytes)
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
							CPUIDs:         []int{},
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
		swarm.DumpNodeState(node.Name)
		if env.Tester == env.TesterAnt {
			continue
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

// the same function as in framework, except that skip the taint node check.
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

// the same function as in framework, except that skip the taint node check.
func getMasterAndColocationWorkerNodesOrDie(c clientset.Interface) (sets.String, *v1.NodeList) {
	nodes := &v1.NodeList{}
	masters := sets.NewString()
	labelsMap := map[string]string{
		alipaysigmak8sapi.LabelIsColocation: "true",
	}
	selector := labels.SelectorFromSet(labels.Set(labelsMap))
	all, err := c.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
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

func NewPreviewClient(kubeconfig string) (*extclientset.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	cs, err := extclientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

func createPreview(tc *testContext, previewReq *v1beta1.CapacityPreview) *v1beta1.CapacityPreview {
	pod, err := tc.PreviewClient.AppsV1beta1().CapacityPreviews(tc.f.Namespace.Name).Create(previewReq)
	framework.ExpectNoError(err)
	return pod
}

func WaitTimeoutForPreviewFinishInNamespace(previewClient *extclientset.Clientset, previewName, namespace string, timeout time.Duration) error {
	return wait.PollImmediate(framework.Poll, timeout, previewFinish(previewClient, previewName, namespace))
}

func previewFinish(previewClient *extclientset.Clientset, previewName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		preview, err := previewClient.AppsV1beta1().CapacityPreviews(namespace).Get(previewName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		switch preview.Status.Phase {
		case v1beta1.PreviewPhasePending:
			return false, fmt.Errorf("preview in pending")
		case v1beta1.PreviewPhaseCompleted:
			return true, nil
		}
		return false, nil
	}
}
