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
	"k8s.io/kubernetes/test/sigma/env"
)

var _ = Describe("[sigma-2.0+3.1][sigma-scheduler][podaffinity][Serial]", func() {
	var cs clientset.Interface
	var nodeList *v1.NodeList
	var containersToDelete []string

	f := framework.NewDefaultFramework(CPUSetNameSpace)

	f.AllNodesReadyTimeout = 3 * time.Second

	BeforeEach(func() {
		cs = f.ClientSet
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

	// 混合链路 Pod/Container anti-affinity 应用互斥验证
	// 前置：要求节点数至少为 3 个，否则 skip
	// 步骤：
	// 1. k8s 链路在 node1 上分配一个 APP-A 的 Pod（预期成功）
	// 2. sigma 2.0 链路在 node2 上分配一个 APP-B 的 Container（预期成功）
	// 3. k8s 链路分配一个与 APP-A 和 APP-B 都互斥的 APP-C Pod（预期成功，且不分配到 node1 或n ode2 上）
	// 4. sigma 2.0 链路分配一个与 APP-A 和 APP-B 都互斥（prohibit）的 APP-C 的 Container （预期成功，且不分配到 node1 或 node2 上）
	It("[p1] pod_affinity_mix_001 Pod and container should be scheduled to the node that satisfies the alloc-spec.affinity.podAntiAffinity terms or prohibit labels", func() {
		if len(nodeList.Items) < 3 {
			Skip("SKIP: this test needs at least 3 nodes!")
		}

		By("Trying to launch a APP-A pod on node[0].")

		waitNodeResourceReleaseComplete(nodeList.Items[0].Name)
		waitNodeResourceReleaseComplete(nodeList.Items[1].Name)

		nodeIP1 := nodeList.Items[0].Status.Addresses[0].Address
		pod := runPausePod(f, pausePodConfig{
			Name: "scheduler-e2e-" + string(uuid.NewUUID()),
			Labels: map[string]string{
				sigmak8s.LabelAppName: "APP-A",
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(sigmak8s.LabelNodeIP, []string{nodeIP1}),
		})

		By("Wait the APP-A pod becomes running")
		framework.ExpectNoError(f.WaitForPodRunning(pod.Name))

		By("Trying to launch a APP-B container on node[1].")
		nodeIP2 := nodeList.Items[1].Status.Addresses[0].Address
		containerLabels := map[string]string{
			"ali.AppName":        "APP-B",
			"ali.SpecifiedNcIps": nodeIP2,
		}

		name := "scheduler-e2e-" + string(uuid.NewUUID())
		container, err := swarm.CreateContainerSyncWithLabels(name, containerLabels)
		containersToDelete = append(containersToDelete, container.ID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(container.IsScheduled()).Should(Equal(true),
			fmt.Sprintf("expect container to be scheduled successfully, result: %+v", container))

		By("Trying to launch a APP-C pod not with APP-A or APP-B pod.")
		pod = runPausePod(f, pausePodConfig{
			Resources: podRequestedResource,
			Name:      "scheduler-e2e-" + string(uuid.NewUUID()),
			Annotations: map[string]string{
				sigmak8s.AnnotationPodAllocSpec: allocSpecStrWithConstraints([]constraint{
					{
						key:      sigmak8s.LabelAppName,
						op:       metav1.LabelSelectorOpIn,
						value:    "APP-A",
						maxCount: 1,
					},
					{
						key:      sigmak8s.LabelAppName,
						op:       metav1.LabelSelectorOpIn,
						value:    "APP-B",
						maxCount: 1,
					},
				}),
			},
			Labels: map[string]string{
				sigmak8s.LabelAppName: "APP-C",
			},
		})

		By("Verify the APP-C pod was scheduled to the expected node.")
		Expect(pod.Spec.NodeName).NotTo(Equal(nodeList.Items[0].Name))
		Expect(pod.Spec.NodeName).NotTo(Equal(nodeList.Items[1].Name))

		By("Trying to launch a APP-C container with label prohibit:Application=APP_A|APP_B")
		containerLabels = map[string]string{
			"ali.AppName":          "APP-C",
			"prohibit:Application": "APP-A|APP-B",
		}

		name = "scheduler-e2e-" + string(uuid.NewUUID())
		container, err = swarm.CreateContainerSyncWithLabels(name, containerLabels)
		containersToDelete = append(containersToDelete, container.ID)
		Expect(err).ShouldNot(HaveOccurred())
		By("Verify the APP-C container was scheduled to the expected node.")
		Expect(container.Host().HostSN).NotTo(Equal(nodeList.Items[0].Name))
		Expect(container.Host().HostSN).NotTo(Equal(nodeList.Items[1].Name))
	})

	// 混合链路Pod/container anti-affinity 应用互斥和独占验证
	// 步骤：
	// 1. sigma2.0链路在每个node节点分配一个APP-A的container（预期成功）
	// 2. k8s链路分配一个sigma.ali/app-name=APP_A且要求与sigma.ali/app-name=APP_A的maxCount=1的互斥的pod（预期失败）
	// 3. k8s链路分配一个sigma.ali/app-name=APP_B且要求sigma.ali/app-name=APP_B独占的Pod（预期失败）
	It("pod_affinity_mix_002 Pod should not be schedule to the node that satisfies PodAntiAffinity In and NotIn operators after containers which have same label have been scheduled.", func() {
		By("Trying to launch a APP-A container with SpecifiedNcIps on each node.")

		waitNodeResourceReleaseComplete(nodeList.Items[0].Name)
		nodeIP := nodeList.Items[0].Status.Addresses[0].Address

		containerLabels := map[string]string{
			"ali.AppName":        "APP-A",
			"ali.SpecifiedNcIps": nodeIP,
		}

		name := "scheduler-e2e-" + string(uuid.NewUUID())
		container, err := swarm.CreateContainerSyncWithLabels(name, containerLabels)
		Expect(err).ShouldNot(HaveOccurred())

		containersToDelete = append(containersToDelete, container.ID)
		Expect(container.IsScheduled()).Should(Equal(true),
			fmt.Sprintf("expect container to be scheduled successfully, result: %+v", container))

		By("Trying to launch another APP-A pod1 with maxCount=1.")
		pod := createPausePod(f, pausePodConfig{
			Resources: podRequestedResource,
			Name:      "scheduler-e2e-" + string(uuid.NewUUID()),
			Annotations: map[string]string{
				sigmak8s.AnnotationPodAllocSpec: allocSpecStrWithConstraints([]constraint{
					{
						key:      sigmak8s.LabelAppName,
						op:       metav1.LabelSelectorOpIn,
						value:    "APP-A",
						maxCount: 1,
					},
				}),
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(sigmak8s.LabelNodeIP, []string{nodeIP}),
			Labels: map[string]string{
				sigmak8s.LabelAppName: "APP-A",
			},
		})
		framework.Logf("expect pod failed to be scheduled.")
		err = framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)

		// 之前已经每台机器上 sigma 链路都 allocate 了 APP-A container，这时候 kubernetes 链路来 NotIn Operator 的 pod 应该分配不出来。
		By("Trying to launch a APP-B pod that does not want share the node with APP-A pod.")
		pod = createPausePod(f, pausePodConfig{
			Resources: podRequestedResource,
			Name:      "scheduler-e2e-" + string(uuid.NewUUID()),
			Annotations: map[string]string{
				sigmak8s.AnnotationPodAllocSpec: allocSpecStrWithConstraints([]constraint{
					{
						key:      sigmak8s.LabelAppName,
						op:       metav1.LabelSelectorOpNotIn,
						value:    "APP-B",
						maxCount: 1,
					},
				}),
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(sigmak8s.LabelNodeIP, []string{nodeIP}),
			Labels: map[string]string{
				sigmak8s.LabelAppName: "APP-B",
			},
		})
		framework.Logf("expect pod failed to be scheduled.")
		err = framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)
	})

	// 混合链路Pod/container anti-affinity 打散验证
	// 步骤：
	// 1. sigma链路在node1上连续分配两个DU_1且要求MaxInstancePerHost=2的container（预期成功）
	// 2. k8s链路在node1上分配一个sigma.ali/deploy-unit=DU_1且要求sigma.ali/deploy-unit=DU_1的maxCount=2的Pod（预期失败）

	It("pod_affinity_mix_003: Pod should be scheduled successfully if it satisfy the maxcount constrains and pod should fail if it exceeds maxcount constraints after containers which have same label have been scheduled.", func() {
		By("Trying to launch two containers with MaxInstancePerHost=2 on node[0] should pass.")
		waitNodeResourceReleaseComplete(nodeList.Items[0].Name)
		nodeIP := nodeList.Items[0].Status.Addresses[0].Address
		for i := 0; i < 2; i++ {
			containerLabels := map[string]string{
				"ali.AppDeployUnit":      "DU-1",
				"ali.SpecifiedNcIps":     nodeIP,
				"ali.MaxInstancePerHost": "2",
			}
			name := "scheduler-e2e-" + string(uuid.NewUUID())
			container, err := swarm.CreateContainerSyncWithLabels(name, containerLabels)
			containersToDelete = append(containersToDelete, container.ID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(container.IsScheduled()).Should(Equal(true),
				fmt.Sprintf("expect container to be scheduled successfully, result: %+v", container))
		}

		By("Trying to launch a DU-1 pod should fail")
		pod := createPausePod(f, pausePodConfig{
			Resources: podRequestedResource,
			Name:      "scheduler-e2e-" + string(uuid.NewUUID()),
			Annotations: map[string]string{
				sigmak8s.AnnotationPodAllocSpec: allocSpecStrWithConstraints([]constraint{
					{
						key:      sigmak8s.LabelDeployUnit,
						op:       metav1.LabelSelectorOpIn,
						value:    "DU-1",
						maxCount: 2,
					},
				}),
			},
			Labels: map[string]string{
				sigmak8s.LabelDeployUnit: "DU-1",
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(sigmak8s.LabelNodeIP, []string{nodeIP}),
		})
		framework.Logf("expect pod failed to be scheduled.")
		err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)
	})

	// 混合链路Container/Pod anti-affinity 应用互斥和独占验证
	// 步骤：
	// 1. k8s链路在每个node节点分配一个DU-1的pod（预期成功）
	// 2. sigma2.0链路分配一个DU-1且要求与MaxInstancePerHost=1的互斥的container（预期失败）
	// 3. sigma2.0链路分配一个DU-1且要求prohibit:Application=APP-1的container（预期失败）
	It("pod_affinity_mix_004, Sigma container should not be schedule to the node that satisfies MaxInstancePerHost=1 or Mono after pods which have same label have been scheduled.", func() {
		By("Trying to launch a DU-1 pod with nodeName on each node.")

		waitNodeResourceReleaseComplete(nodeList.Items[0].Name)

		nodeIP := nodeList.Items[0].Status.Addresses[0].Address
		runPausePod(f, pausePodConfig{
			Resources: podRequestedResource,
			Name:      "scheduler-e2e" + string(uuid.NewUUID()),
			Labels: map[string]string{
				sigmak8s.LabelAppName:    "APP-1",
				sigmak8s.LabelDeployUnit: "DU-1",
			},
			Affinity: util.GetAffinityNodeSelectorRequirement(sigmak8s.LabelNodeIP, []string{nodeIP}),
		})

		By("Trying to launch a DU-1 container with MaxInstancePerHost=1.")

		containerLabels := map[string]string{
			"ali.AppName":            "APP-1",
			"ali.AppDeployUnit":      "DU-1",
			"ali.MaxInstancePerHost": "1",
			"ali.SpecifiedNcIps":     nodeIP,
		}
		name := "scheduler-e2e-" + string(uuid.NewUUID())
		container, err := swarm.CreateContainerSyncWithLabels(name, containerLabels)
		containersToDelete = append(containersToDelete, container.ID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(container.IsScheduled()).Should(Equal(false), fmt.Sprintf("expect container failed to be scheduled, result: %+v", container))

		proValue := "APP-1"
		if env.Tester == env.TesterAnt {
			By("Trying to launch a container with prohibit:Application=APP-1.")
		} else {
			By("Trying to launch a container with prohibit:Application=DU-1.")
			proValue = "DU-1"
		}
		containerLabels2 := map[string]string{
			"ali.AppName":          "APP-2",
			"ali.AppDeployUnit":    "DU-2",
			"prohibit:Application": proValue,
			"ali.SpecifiedNcIps":   nodeIP,
		}
		name = "scheduler-e2e-" + string(uuid.NewUUID())
		container, err = swarm.CreateContainerSyncWithLabels(name, containerLabels2)
		containersToDelete = append(containersToDelete, container.ID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(container.IsScheduled()).Should(Equal(false), fmt.Sprintf("expect container failed to be scheduled, result: %+v", container))

		// TODO: 之前已经每台机器上kubernetes链路都allocate了APP_1 pod, 这时候sigma链路来的APP_2独占应用应该分配不出来
	})

	// 混合链路Container/Pod anti-affinity 打散（maxCount=2）验证
	// 步骤：
	// 1. k8s链路在每个节点上连续分配两个DU-1且要求maxCount=2的pod（预期成功）
	// 2. sigma链路分配一个DU-1且要求MaxInstancePerHost=2的Container（预期失败）
	It("pod_affinity_mix_005 Sigma container should be scheduled successfully if it satisfy the maxcount constrains and container should fail if it exceeds maxcount constraints after pods which have same label have been scheduled.", func() {

		By("Trying to launch two pods with maxcount=2 on node[0] should pass.")
		waitNodeResourceReleaseComplete(nodeList.Items[0].Name)

		nodeIP := nodeList.Items[0].Status.Addresses[0].Address
		for i := 0; i < 2; i++ {
			runPausePod(f, pausePodConfig{
				Resources: podRequestedResource,
				Name:      "scheduler-e2e-" + string(uuid.NewUUID()),
				Labels: map[string]string{
					sigmak8s.LabelAppName:    "APP-1",
					sigmak8s.LabelDeployUnit: "DU-1",
				},
				Annotations: map[string]string{
					sigmak8s.AnnotationPodAllocSpec: allocSpecStrWithConstraints([]constraint{
						{
							key:      sigmak8s.LabelDeployUnit,
							op:       metav1.LabelSelectorOpIn,
							value:    "DU-1",
							maxCount: 2,
						},
					}),
				},
				Affinity: util.GetAffinityNodeSelectorRequirement(sigmak8s.LabelNodeIP, []string{nodeIP}),
			})
		}

		By("Trying to launch a DU-1 container should fail")
		containerLabels := map[string]string{
			"ali.AppName":            "APP-1",
			"ali.AppDeployUnit":      "DU-1",
			"ali.MaxInstancePerHost": "2",
			"ali.SpecifiedNcIps":     nodeIP,
		}

		name := "scheduler-e2e-" + string(uuid.NewUUID())
		container, err := swarm.CreateContainerSyncWithLabels(name, containerLabels)
		containersToDelete = append(containersToDelete, container.ID)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(container.IsScheduled()).Should(Equal(false), fmt.Sprintf("expect container failed to be scheduled, result: %+v", container))
	})
})
