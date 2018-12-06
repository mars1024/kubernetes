package scheduler

import (
	"fmt"
	"time"

	sigmak8s "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/swarm"
	"k8s.io/kubernetes/test/sigma/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	SwarmMonoAPP = "mono-app"
	SwarmMonoDU  = "mono-du"
)

var _ = Describe("[ali][sigma-2.0+3.1][sigma-scheduler][node-mono][Serial]", func() {
	var cs clientset.Interface
	var nodeList *v1.NodeList
	var containersToDelete []string

	f := framework.NewDefaultFramework(CPUSetNameSpace)

	f.AllNodesReadyTimeout = 3 * time.Second

	BeforeEach(func() {
		cs = f.ClientSet
		nodeList = &v1.NodeList{}
		nodeList = &v1.NodeList{}

		masterNodes, nodeList = getMasterAndWorkerNodesOrDie(cs)
		// reset containers to-deleted
		containersToDelete = []string{}
	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			DumpSchedulerState(f, 0)
		}

		By("delete created containers")
		for _, containerID := range containersToDelete {
			if containerID == "" {
				continue
			}
			swarm.MustDeleteContainer(containerID)
		}
		DeleteSigmaContainer(f)
	})

	/*
		case: 测试 sigma2.0+3.1 混合链路，sigma2.0 独占主机应用可以感知 k8s 独占应用
		1. 创建一个 k8s 独占应用 A，期望调度成功到主机 nodeA
		2. 创建一个 sigma2.0 独占应用，指定调度到 nodeA，期望失败
	*/
	It("[ali] node_mono_mix_001 Sigma container should not be scheduled to the node contains mono pod.", func() {
		By("create a mono pod")
		appName := "app-" + string(uuid.NewUUID())
		duName := "du-" + appName
		podNamePrefix := "pod-mono-with-app-du-"
		nodeIP := nodeList.Items[0].Status.Addresses[0].Address

		pod := runPausePod(f, pausePodConfig{
			Name: podNamePrefix + string(uuid.NewUUID()),
			Labels: map[string]string{
				sigmak8s.LabelAppName:    appName,
				sigmak8s.LabelDeployUnit: duName,
			},
			Annotations: map[string]string{
				sigmak8s.AnnotationPodAllocSpec: allocSpecStrWithConstraints([]constraint{
					{
						key:   sigmak8s.LabelAppName,
						op:    metav1.LabelSelectorOpNotIn,
						value: appName,
					},
					{
						key:   sigmak8s.LabelDeployUnit,
						op:    metav1.LabelSelectorOpNotIn,
						value: duName,
					},
				}),
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(sigmak8s.LabelNodeIP, []string{nodeIP}),
		})
		defer util.DeletePod(f.ClientSet, pod)
		By("Wait the pod becomes running")
		framework.ExpectNoError(f.WaitForPodRunning(pod.Name))

		By("add global mono app/du rules")
		err := swarm.PutMonoAPPDURule(SwarmMonoAPP, SwarmMonoDU)
		defer swarm.RemoveSigmaGlobal()
		Expect(err).ShouldNot(HaveOccurred())

		By("Trying to launch a sigma2.0 containers with MaxInstancePerHost=1 on should fail.")
		containerLabels := map[string]string{
			"ali.AppName":            SwarmMonoAPP,
			"ali.AppDeployUnit":      SwarmMonoDU,
			"ali.SpecifiedNcIps":     nodeList.Items[0].Status.Addresses[0].Address,
			"ali.MaxInstancePerHost": "1",
		}

		name := "container-with-specified-ip"
		container, _ := swarm.CreateContainerSyncWithLabels(name, containerLabels)
		containersToDelete = append(containersToDelete, container.ID)
		Expect(container.IsScheduled()).Should(Equal(false), fmt.Sprintf("expect container failed to be scheduled, result: %+v", container))
	})

	/*
		case: 测试 sigma2.0+3.1 混合链路，k8s 独占主机应用可以感知 sigma2.0 独占应用
		1. 创建一个 sigma2.0 独占应用，指定调度到 nodeA，期望成功
		2. 创建一个 k8s 独占应用 A，指定调度到主机 nodeA，期望失败
	*/
	It("[ali] Pod should not be scheduled to node that have sigma 2.0 mono container.", func() {

		appName := "app-" + string(uuid.NewUUID())
		duName := "du-" + appName

		nodeName := GetNodeThatCanRunPod(f)
		// sleep 5s to wait for host.AllocPlanSize updated
		time.Sleep(5 * time.Second)

		node, _ := f.ClientSet.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})

		By("add global mono app/du rules")
		err := swarm.PutMonoAPPDURule(SwarmMonoAPP, SwarmMonoDU)
		defer swarm.RemoveSigmaGlobal()
		Expect(err).ShouldNot(HaveOccurred())

		By("Trying to launch one container with MaxInstancePerHost=1.")
		containerLabels := map[string]string{
			"ali.AppDeployUnit":      SwarmMonoDU,
			"ali.AppName":            SwarmMonoAPP,
			"ali.SpecifiedNcIps":     node.Labels[sigmak8s.LabelNodeIP],
			"ali.MaxInstancePerHost": "1",
		}

		name := "container-with-specified-ip"
		container, _ := swarm.CreateContainerSyncWithLabels(name, containerLabels)
		containersToDelete = append(containersToDelete, container.ID)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(container.IsScheduled()).Should(Equal(true),
			fmt.Sprintf("expect container to be scheduled successfully, result: %+v", container))

		By("Trying to launch a mono pod should fail")
		nodeIP := nodeList.Items[0].Status.Addresses[0].Address

		pod := createPausePod(f, pausePodConfig{
			Name: "pod-mono-" + string(uuid.NewUUID()),
			Labels: map[string]string{
				sigmak8s.LabelAppName:    appName,
				sigmak8s.LabelDeployUnit: duName,
			},
			Annotations: map[string]string{
				sigmak8s.AnnotationPodAllocSpec: allocSpecStrWithConstraints([]constraint{
					{
						key:   sigmak8s.LabelAppName,
						op:    metav1.LabelSelectorOpNotIn,
						value: appName,
					},
					{
						key:   sigmak8s.LabelDeployUnit,
						op:    metav1.LabelSelectorOpNotIn,
						value: duName,
					},
				}),
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(sigmak8s.LabelNodeIP, []string{nodeIP}),
		})
		defer util.DeletePod(f.ClientSet, pod)

		framework.Logf("expect pod failed to be scheduled.")
		err = framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)
	})
})
