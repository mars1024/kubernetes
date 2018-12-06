package scheduler

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	sigmak8s "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/env"
	"k8s.io/kubernetes/test/sigma/swarm"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-3.1][sigma-scheduler][node-mono][Serial]", func() {
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
	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			DumpSchedulerState(f, 0)
		}
		DeleteSigmaContainer(f)
	})
	/*
		Case 描述：验证应用独占主机
		测试步骤：
		1. 创建一个独占应用 A，期望调度成功
		2. 检查独占应用所在的主机 nodeA，有且只有独占应用 A
		3. 创建第二个相同的独占应用 A，检查 如果集群只有一个主机，则一定调度到 nodeA；
		   否则，如果测试环境是集团环境，则调度规则默认为 balance，一定调度到其他主机
		4. 创建一个普通 pod ，如果集群只有一个主机，则检查 pod 调度失败
	*/
	It("[p2] node_mono_001 Pod of mono app and du should be scheduled to the node that only has this mono app.", func() {
		// TODO: uncomment it when bug is fixed
		Skip("skip due to bug:https://aone.alibaba-inc.com/project/770309/issue/16894405")
		By("create a mono pod")
		appName := "app-" + string(uuid.NewUUID())
		duName := "du-" + appName
		podNamePrefix := "pod-mono-with-app-du-"
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
		})
		defer util.DeletePod(f.ClientSet, pod)
		By("Wait the pod becomes running")
		framework.ExpectNoError(f.WaitForPodRunning(pod.Name))

		By("check there is only mono app in the node")
		nodeName := pod.Spec.NodeName
		AllocatableMapCPU := nodeToAllocatableMapCPU[nodeName]
		allNs, _ := f.ClientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
		for _, ns := range allNs.Items {
			allPods, _ := f.ClientSet.CoreV1().Pods(ns.Namespace).List(metav1.ListOptions{})
			for _, v := range allPods.Items {
				if v.Spec.NodeName == nodeName {
					Expect(v.Labels[sigmak8s.LabelAppName]).Should(Equal(appName))
					Expect(v.Labels[sigmak8s.LabelDeployUnit]).Should(Equal(duName))
					// calculate the rest allocable cpu
					AllocatableMapCPU -= getRequestedCPU(v)
				}
			}
		}

		By("alloc the second pod with the rest cpu should be scheduled successfully")
		pod2 := runPausePod(f, pausePodConfig{
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
			Resources: &v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: *resource.NewMilliQuantity(AllocatableMapCPU, "DecimalSI"),
				},
				Requests: v1.ResourceList{
					v1.ResourceCPU: *resource.NewMilliQuantity(AllocatableMapCPU, "DecimalSI"),
				},
			},
		})
		defer util.DeletePod(f.ClientSet, pod2)

		By("Wait the pod becomes running")
		framework.ExpectNoError(f.WaitForPodRunning(pod2.Name))
		// 如果策略是 堆叠：判断调度到同一台机器，如果策略是 balance：如果有多台机器，判断调度到其他机器
		// 集团默认策略是 balance
		if len(nodeList.Items) > 1 {
			if env.Tester == env.TesterJituan {
				By("In jituan environment, schedule strategy is balance, expect the second pod to be scheduled to another node")
				Expect(pod2.Spec.NodeName).ShouldNot(Equal(nodeName))
			} else if env.Tester == env.TesterAnt {
				By("In ant envrironment, schedule strategy is packed, expect the second pod to be scheduled to the same node")
				Expect(pod2.Spec.NodeName).Should(Equal(nodeName))
			}
		}

		if len(nodeList.Items) == 1 {
			Expect(pod2.Spec.NodeName).Should(Equal(nodeName))
		}

		By("alloc a non mono pod, should not be scheduled to the node")
		pod3 := createPausePod(f, pausePodConfig{
			Name: "pod-non-momo-" + string(uuid.NewUUID()),
		})
		defer util.DeletePod(f.ClientSet, pod3)

		if len(nodeList.Items) == 1 {
			framework.Logf("expect pod failed to be scheduled.")
			err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod3.Name, f.Namespace.Name)
			Expect(err).To(BeNil(), "expect err be nil, got %s", err)
		}
	})

	/*
		Case 描述：验证两个不同应用独占主机
		测试步骤：
		1. 创建一个独占应用 A，期望调度成功到主机 nodeA
		2. 创建第二个的独占应用 B，检查 如果集群只有一个主机，则一定调度失败；
		   否则，如果有多个主机，则调度成功，且没有调度到 nodeA
	*/
	It("[smoke][p0][bvt] node_mono_002 Two different node mono pods should not be scheduled in the same node", func() {
		By("create a mono pod")
		appName := "app-" + string(uuid.NewUUID())
		duName := "du-" + appName
		podNamePrefix := "pod-mono-with-app-du-"
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
		})
		defer util.DeletePod(f.ClientSet, pod)

		By("Wait the pod becomes running")
		framework.ExpectNoError(f.WaitForPodRunning(pod.Name))
		nodeName := pod.Spec.NodeName

		By("alloc the second different mono pod")
		appName2 := "app-" + string(uuid.NewUUID())
		duName2 := "du-" + appName
		pod2 := createPausePod(f, pausePodConfig{
			Name: podNamePrefix + string(uuid.NewUUID()),
			Labels: map[string]string{
				sigmak8s.LabelAppName:    appName2,
				sigmak8s.LabelDeployUnit: duName2,
			},
			Annotations: map[string]string{
				sigmak8s.AnnotationPodAllocSpec: allocSpecStrWithConstraints([]constraint{
					{
						key:   sigmak8s.LabelAppName,
						op:    metav1.LabelSelectorOpNotIn,
						value: appName2,
					},
					{
						key:   sigmak8s.LabelDeployUnit,
						op:    metav1.LabelSelectorOpNotIn,
						value: duName2,
					},
				}),
			},
		})
		defer util.DeletePod(f.ClientSet, pod2)

		if len(nodeList.Items) == 1 {
			By("In one node cluster, pod2 should fail to be scheduled")
			err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod2.Name, f.Namespace.Name)
			Expect(err).To(BeNil(), "expect err be nil, got %s", err)
		}

		if len(nodeList.Items) > 1 {
			nodeIP := ""
			for _, v := range nodeList.Items {
				if nodeName == v.Name {
					for _, addr := range v.Status.Addresses {
						if addr.Type == v1.NodeInternalIP {
							nodeIP = addr.Address
						}
					}
					break
				}
			}
			By("In multi nodes cluster, pod2 should be scheduled to a different node.")
			framework.ExpectNoError(f.WaitForPodRunning(pod2.Name))
			logrus.Info("pod2 is scheduled to node:", pod2.Status.HostIP)
			Expect(nodeIP).ToNot(Equal(pod2.Status.HostIP))
		}
	})

	/*
		Case 描述：验证独占主机应用无法调度到已存在普通 pod 的主机
		测试步骤：
		1. 创建一个普通pod，期望调度成功到主机 nodeA
		2. 创建第二个的独占应用 A，检查 如果集群只有一个主机，则一定调度失败；
		   否则，如果有多个主机，则调度成功，且没有调度到 nodeA
	*/
	It("node_mono_003 A node mono pod should fail to be scheduled to the node with ordinary pod", func() {
		By("create a ordinary pod")
		pod := runPausePod(f, pausePodConfig{
			Name: "non-mono-pod-" + string(uuid.NewUUID()),
			Labels: map[string]string{
				sigmak8s.LabelAppName:    "testApp",
				sigmak8s.LabelDeployUnit: "testDu",
			},
		})
		defer util.DeletePod(f.ClientSet, pod)

		By("Wait the pod becomes running")
		framework.ExpectNoError(f.WaitForPodRunning(pod.Name))
		nodeName := pod.Spec.NodeName

		By("create a mono pod")
		appName := "app-" + string(uuid.NewUUID())
		duName := "du-" + appName
		podNamePrefix := "pod-mono-with-app-du-"
		pod2 := createPausePod(f, pausePodConfig{
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
		})
		defer util.DeletePod(f.ClientSet, pod2)

		if len(nodeList.Items) == 1 {
			By("In one node cluster, pod2 should fail to be scheduled")
			err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod2.Name, f.Namespace.Name)
			Expect(err).To(BeNil(), "expect err be nil, got %s", err)
		}

		if len(nodeList.Items) > 1 {
			By("In multi nodes cluster, pod2 should be scheduled to a different node.")
			nodeIP := ""
			for _, v := range nodeList.Items {
				if nodeName == v.Name {
					for _, addr := range v.Status.Addresses {
						if addr.Type == v1.NodeInternalIP {
							nodeIP = addr.Address
						}
					}
					break
				}
			}
			By("In multi nodes cluster, pod2 should be scheduled to a different node.")
			framework.ExpectNoError(f.WaitForPodRunning(pod2.Name))
			logrus.Info("pod2 is scheduled to node:", pod2.Status.HostIP)
			Expect(nodeIP).ToNot(Equal(pod2.Status.HostIP))
		}
	})

})
