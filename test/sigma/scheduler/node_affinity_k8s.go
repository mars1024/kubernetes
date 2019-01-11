package scheduler

import (
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

	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/kubernetes/test/sigma/swarm"
)

var _ = Describe("[sigma-3.1][sigma-scheduler][node-affinity]", func() {
	var cs clientset.Interface
	var nodeList *v1.NodeList
	nodesInfo := make(map[string]*v1.Node)
	nodeToAllocatableMapCPU := make(map[string]int64)
	nodeToAllocatableMapMem := make(map[string]int64)
	nodeToAllocatableMapEphemeralStorage := make(map[string]int64)

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

	It("[smoke][p0][bvt] node_affinity_k8s_001 A pod with node IP label should be scheduled to the specified node.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		nodeIP := ""

		for _, v := range nodeList.Items {
			if v.Name == nodeName {
				for _, addr := range v.Status.Addresses {
					if addr.Type == v1.NodeInternalIP {
						nodeIP = addr.Address
					}
				}
				break
			}
		}
		Expect(nodeIP).ToNot(Equal(""))

		By("Request a pod with specified node IP")

		pod := createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorRequirement(api.LabelNodeIP, []string{nodeIP}),
		})

		By("expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeName).To(Equal(newPod.Spec.NodeName))
		util.DeletePod(f.ClientSet, newPod)
	})

	It("[smoke][p0][bvt] node_affinity_k8s_002 A pod with node IP label that doesn't exist on any node should fail to be scheduled.", func() {
		nodeIP := []string{}
		nodeName := []string{}

		for _, v := range nodeList.Items {
			nodeName = append(nodeName, v.Name)
			for _, addr := range v.Status.Addresses {
				if addr.Type == v1.NodeInternalIP {
					nodeIP = append(nodeIP, addr.Address)
				}
			}
		}
		badIP := "127.0.0.1"
		Expect(nodeIP).ToNot(ContainElement(badIP))

		By("Request a pod with specified non existed node IP")

		pod := createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorRequirement(api.LabelNodeIP, []string{badIP}),
		})

		By("Expect pod to fail to be scheduled.")
		err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)
		defer util.DeletePod(f.ClientSet, pod)
	})

	// Pod with a set of IP labels expect to fail to be scheduled, if all the nodes with the IP labels are full of pods and even there are other free nodes.
	It("node_affinity_k8s_003 A pod with node affinity to a set of nodes can only be scheduled to any available nodes within the set of nodes.", func() {
		nodeIPs := []string{}
		nodeName := []string{}

		for _, v := range nodeList.Items {
			nodeName = append(nodeName, v.Name)
			for _, addr := range v.Status.Addresses {
				if addr.Type == v1.NodeInternalIP {
					nodeIPs = append(nodeIPs, addr.Address)
				}
			}
		}
		Expect(len(nodeIPs)).To(Equal(len(nodeName)))

		if len(nodeIPs) < 2 {
			Skip("Nodes number is less than 2, skip.")
		}

		framework.Logf("Have node IPs: %+v", nodeIPs)
		// With PackedResourceAllocation strategy, tests[0] and tests[1] will fill up the nodeIPs[0] and nodeIPs[1], and tests[2] will fail.
		tests := []struct {
			cpu        resource.Quantity
			mem        resource.Quantity
			ethstorage resource.Quantity
			ip         []string
			ok         bool
		}{{

			cpu:        *resource.NewMilliQuantity(nodeToAllocatableMapCPU[nodeName[0]], "DecimalSI"),
			mem:        *resource.NewQuantity(nodeToAllocatableMapMem[nodeName[0]], "DecimalSI"),
			ethstorage: *resource.NewQuantity(nodeToAllocatableMapEphemeralStorage[nodeName[0]], "DecimalSI"),
			ip:         []string{nodeIPs[0], nodeIPs[1]},
			ok:         true,
		}, {
			cpu:        *resource.NewMilliQuantity(nodeToAllocatableMapCPU[nodeName[1]], "DecimalSI"),
			mem:        *resource.NewQuantity(nodeToAllocatableMapMem[nodeName[1]], "DecimalSI"),
			ethstorage: *resource.NewQuantity(nodeToAllocatableMapEphemeralStorage[nodeName[1]], "DecimalSI"),
			ip:         []string{nodeIPs[0], nodeIPs[1]},
			ok:         true,
		}, {
			cpu:        *resource.NewMilliQuantity(nodeToAllocatableMapCPU[nodeName[1]], "DecimalSI"),
			mem:        *resource.NewQuantity(nodeToAllocatableMapMem[nodeName[1]], "DecimalSI"),
			ethstorage: *resource.NewQuantity(nodeToAllocatableMapEphemeralStorage[nodeName[1]], "DecimalSI"),
			ip:         []string{nodeIPs[0], nodeIPs[1]},
			ok:         false,
		}}

		podsToDelete := make([]*v1.Pod, len(tests))
		for i, test := range tests {
			framework.Logf("Case[%d], request a pod with node IPs: %+v", i, test.ip)
			pod := createPausePod(f, pausePodConfig{
				Name: "scheduler-" + string(uuid.NewUUID()),
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
				Affinity: util.GetAffinityNodeSelectorRequirement(api.LabelNodeIP, test.ip),
			})

			if test.ok == true {
				By("expect pod to be scheduled successfully.")
				err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
				podsToDelete = append(podsToDelete, pod)
				Expect(err).NotTo(HaveOccurred())
				newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
				Expect(test.ip).To(ContainElement(newPod.Status.HostIP))

			} else {
				By("Expect pod to fail to be scheduled.")
				podsToDelete = append(podsToDelete, pod)
				err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
				Expect(err).To(BeNil(), "expect err be nil, got %s", err)
			}
		}
		for _, pod := range podsToDelete {
			if pod == nil {
				continue
			}
			util.DeletePod(f.ClientSet, pod)
		}
	})

	It("node_affinity_k8s_004 A pod specified with multi IPs, will be scheduled to good IPs only.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

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
		badIP := "127.0.0.1"

		tests := []struct {
			cpu        resource.Quantity
			mem        resource.Quantity
			ethstorage resource.Quantity
			ip         []string
			ok         bool
		}{{

			cpu:        *resource.NewMilliQuantity(nodeToAllocatableMapCPU[nodeName], "DecimalSI"),
			mem:        *resource.NewQuantity(nodeToAllocatableMapMem[nodeName], "DecimalSI"),
			ethstorage: *resource.NewQuantity(nodeToAllocatableMapEphemeralStorage[nodeName], "DecimalSI"),
			ip:         []string{nodeIP, badIP},
			ok:         true,
		}, {
			cpu:        *resource.NewMilliQuantity(nodeToAllocatableMapCPU[nodeName], "DecimalSI"),
			mem:        *resource.NewQuantity(nodeToAllocatableMapMem[nodeName], "DecimalSI"),
			ethstorage: *resource.NewQuantity(nodeToAllocatableMapEphemeralStorage[nodeName], "DecimalSI"),
			ip:         []string{nodeIP, badIP},
			ok:         false,
		}}

		podsToDelete := make([]*v1.Pod, len(tests))
		for _, test := range tests {
			By("Request a pod with specified node IPs")
			pod := createPausePod(f, pausePodConfig{
				Name: "scheduler-" + string(uuid.NewUUID()),
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
				Affinity: util.GetAffinityNodeSelectorRequirement(api.LabelNodeIP, test.ip),
			})

			if test.ok == true {
				By("expect pod to be scheduled successfully.")
				err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
				podsToDelete = append(podsToDelete, pod)
				Expect(err).NotTo(HaveOccurred())
				newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
				Expect(nodeIP).To(Equal(newPod.Status.HostIP))
			} else {
				By("Expect pod to fail to be scheduled.")
				podsToDelete = append(podsToDelete, pod)
				err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
				Expect(err).To(BeNil(), "expect err be nil, got %s", err)
			}
		}
		for _, pod := range podsToDelete {
			if pod == nil {
				continue
			}
			util.DeletePod(f.ClientSet, pod)
		}
	})

	//Node affinity 验证：指定 IP 忽略强制标签
	//步骤：
	//1. 选取一个可调度节点
	//2. 给节点打上强制标签（Taints+Labels）
	//3. 指定IP，忽略强制标（Toleration+Labels）创建 Pod，预期创建成功
	It("node_affinity_k8s_005 A pod specified with a node IP and tolerations, will be scheduled to the node, "+
		"even if the node has taints.", func() {
		By("Get one node to schedule.")
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		nodeIP := ""

		for _, v := range nodeList.Items {
			if v.Name == nodeName {
				for _, addr := range v.Status.Addresses {
					if addr.Type == v1.NodeInternalIP {
						nodeIP = addr.Address
					}
				}
				break
			}
		}
		Expect(nodeIP).ToNot(Equal(""))

		mandatoryLabelKey := "mandatory.k8s.alipay.com/appEnv"
		mandatoryLabelValue := "prepub"

		By("Add mandatory label.")
		framework.AddOrUpdateLabelOnNode(cs, nodeName, mandatoryLabelKey, mandatoryLabelValue)
		framework.ExpectNodeHasLabel(cs, nodeName, mandatoryLabelKey, mandatoryLabelValue)
		defer framework.RemoveLabelOffNode(cs, nodeName, mandatoryLabelKey)

		taint := &v1.Taint{
			Key:    mandatoryLabelKey,
			Value:  mandatoryLabelValue,
			Effect: v1.TaintEffectNoSchedule,
		}

		framework.AddOrUpdateTaintOnNode(cs, nodeName, *taint)
		framework.ExpectNodeHasTaint(cs, nodeName, taint)
		defer framework.RemoveTaintOffNode(cs, nodeName, *taint)

		By("Request a pod with toleration.")
		tol := v1.Toleration{
			Key:      mandatoryLabelKey,
			Operator: v1.TolerationOpEqual,
			Value:    mandatoryLabelValue,
			Effect:   v1.TaintEffectNoSchedule,
		}

		pod := createPausePod(f, pausePodConfig{
			Name:        "scheduler-" + string(uuid.NewUUID()),
			Affinity:    util.GetAffinityNodeSelectorRequirement(api.LabelNodeIP, []string{nodeIP}),
			Tolerations: []v1.Toleration{tol},
		})

		By("Expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeName).To(Equal(newPod.Spec.NodeName))
		util.DeletePod(f.ClientSet, newPod)
	})

	//Node affinity 验证：指定IP忽略强制标签
	//步骤：
	//1. 选取一个可调度节点
	//2. 给节点打上强制标签（Taints+Labels）
	//3. 指定IP，创建 Pod，预期创建失败
	It("node_affinity_k8s_006 A pod specified with a node IP, will not be scheduled to the node, if the node has mandatory labels.", func() {
		By("Get one node to schedule.")
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		nodeIP := ""

		for _, v := range nodeList.Items {
			if v.Name == nodeName {
				for _, addr := range v.Status.Addresses {
					if addr.Type == v1.NodeInternalIP {
						nodeIP = addr.Address
					}
				}
				break
			}
		}
		Expect(nodeIP).ToNot(Equal(""))

		mandatoryLabelKey := "mandatory.k8s.alipay.com/appEnv"
		mandatoryLabelValue := "prepub"

		By("Add mandatory label.")
		framework.AddOrUpdateLabelOnNode(cs, nodeName, mandatoryLabelKey, mandatoryLabelValue)
		framework.ExpectNodeHasLabel(cs, nodeName, mandatoryLabelKey, mandatoryLabelValue)
		defer framework.RemoveLabelOffNode(cs, nodeName, mandatoryLabelKey)

		taint := &v1.Taint{
			Key:    mandatoryLabelKey,
			Value:  mandatoryLabelValue,
			Effect: v1.TaintEffectNoSchedule,
		}

		framework.AddOrUpdateTaintOnNode(cs, nodeName, *taint)
		framework.ExpectNodeHasTaint(cs, nodeName, taint)
		defer framework.RemoveTaintOffNode(cs, nodeName, *taint)

		By("Request a pod.")
		pod := createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorRequirement(api.LabelNodeIP, []string{nodeIP}),
		})

		By("Expect pod to be scheduled failed.")
		defer util.DeletePod(cs, pod)
		err := framework.WaitForPodNameUnschedulableInNamespace(cs, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)
	})

	It("node_affinity_k8s_007 A pod specified with predefined labels, will be scheduled to the nodes that have the labels.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		nodeNames := []string{}

		nodes := map[string]v1.Node{}
		for _, v := range nodeList.Items {
			nodes[v.Name] = v
			nodeNames = append(nodeNames, v.Name)
		}
		rack, rok := nodes[nodeName].Labels[api.LabelRack]
		serviceTag, sok := nodes[nodeName].Labels[api.LabelParentServiceTag]
		if !rok || !sok {
			Skip("Nodes has no Rack or ParentServiceTag label, skip.")
		}
		aff := util.GetAffinityNodeSelectorRequirement(api.LabelRack, []string{rack})
		aff, _ = util.AddMatchExpressionsToFirstNodeSelectorTermsOfAffinity(api.LabelParentServiceTag, []string{serviceTag}, aff)

		By("Request a pod with predefined node label")
		pod := createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: aff,
		})

		By("expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeNames).To(ContainElement(newPod.Spec.NodeName))
		Expect(nodes[newPod.Spec.NodeName].Labels[api.LabelRack]).To(Equal(rack))
		Expect(nodes[newPod.Spec.NodeName].Labels[api.LabelParentServiceTag]).To(Equal(serviceTag))
		util.DeletePod(f.ClientSet, newPod)
	})

	It("node_affinity_k8s_008 A pod specified with user defined labels, will be scheduled to the nodes that have the labels.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		customLabelKey := "custom.k8s.alipay.com/a"
		customLabelValue := "b"

		framework.AddOrUpdateLabelOnNode(cs, nodeName, customLabelKey, customLabelValue)
		framework.ExpectNodeHasLabel(cs, nodeName, customLabelKey, customLabelValue)
		defer framework.RemoveLabelOffNode(cs, nodeName, customLabelKey)

		By("Request a pod with custom node label")

		pod := createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorRequirement(customLabelKey, []string{customLabelValue}),
		})
		defer util.DeletePod(f.ClientSet, pod)

		By("expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeName).To(Equal(newPod.Spec.NodeName))
		util.DeletePod(f.ClientSet, newPod)
	})

	//Node affinity 验证 NotIn
	//步骤：
	//1. 给所有节点打上 a=b 普通标
	//2. 使用 label NotIn a=b 创建 pod，预期创建失败
	It("node_affinity_k8s_009 A pod specified with NotIn user defined labels, will not be scheduled to the nodes that have the labels.", func() {
		customLabelKey := "custom.k8s.alipay.com/a"
		customLabelValue := "b"

		for _, v := range nodeList.Items {
			By("Add custom label to " + v.Name)
			framework.AddOrUpdateLabelOnNode(cs, v.Name, customLabelKey, customLabelValue)
			framework.ExpectNodeHasLabel(cs, v.Name, customLabelKey, customLabelValue)
			defer framework.RemoveLabelOffNode(cs, v.Name, customLabelKey)
		}
		By("Request a pod")
		pod := createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorNotInRequirement(customLabelKey, []string{customLabelValue}),
		})

		By("expect pod to be scheduled failed.")
		defer util.DeletePod(cs, pod)
		err := framework.WaitForPodNameUnschedulableInNamespace(cs, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)
	})

	//Node affinity 验证 NotIn
	//步骤：
	//1. 给部分节点打上 a=b 普通标
	//2. 使用 label NotIn a=b 创建 pod，预期创建成功，且调度到没有打标的 node 上
	It("node_affinity_k8s_010 A pod specified with NotIn user defined labels, will be scheduled to the nodes that do not have the labels.", func() {
		if len(nodeList.Items) < 2 {
			Skip("Nodes number is less than 2, skip.")
		}
		nodeName := nodeList.Items[0].Name
		customLabelKey := "custom.k8s.alipay.com/a"
		customLabelValue := "b"

		for _, v := range nodeList.Items[1:] {
			By("Add custom label to " + v.Name)
			framework.AddOrUpdateLabelOnNode(cs, v.Name, customLabelKey, customLabelValue)
			framework.ExpectNodeHasLabel(cs, v.Name, customLabelKey, customLabelValue)
			defer framework.RemoveLabelOffNode(cs, v.Name, customLabelKey)
		}
		By("Request a pod")
		pod := createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorNotInRequirement(customLabelKey, []string{customLabelValue}),
		})

		By("expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		defer util.DeletePod(cs, newPod)
		Expect(err).NotTo(HaveOccurred())
		Expect(nodeName).To(Equal(newPod.Spec.NodeName))
	})

	It("node_affinity_k8s_011 A pod specified with exclusive all node IPs, will be scheduled failed.", func() {
		nodeIPs := []string{}
		for _, v := range nodeList.Items {
			for _, addr := range v.Status.Addresses {
				if addr.Type == v1.NodeInternalIP {
					nodeIPs = append(nodeIPs, addr.Address)
				}
			}
		}

		framework.Logf("Request a pod with NotIn node IPs: %+v", nodeIPs)

		pod := createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorNotInRequirement(api.LabelNodeIP, nodeIPs),
		})

		framework.Logf("Before wait, Pod.Spec.Affinity.NodeAffinity: %+v", pod.Spec.Affinity.NodeAffinity)

		By("Expect pod to fail to be scheduled.")
		defer util.DeletePod(f.ClientSet, pod)
		err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)
	})

	It("node_affinity_k8s_012 A pod specified to exclude some node IPs, will be scheduled to other nodes.", func() {
		nodeIPs := []string{}
		for _, v := range nodeList.Items {
			for _, addr := range v.Status.Addresses {
				if addr.Type == v1.NodeInternalIP {
					nodeIPs = append(nodeIPs, addr.Address)
				}
			}
		}
		if len(nodeIPs) < 2 {
			Skip("Nodes number is less than 2, skip.")
		}

		By("Request a pod with NotIn node IPs")

		pod := createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorNotInRequirement(api.LabelNodeIP, nodeIPs[1:]),
		})

		By("expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(nodeIPs[0]).To(Equal(newPod.Status.HostIP))
		util.DeletePod(f.ClientSet, newPod)
	})

	//Node affinity 验证 NotIn && In
	//步骤：
	//1. 获取可以调度的空闲节点 nodeIP
	//2. 使用label NotIn && In nodeIP 创建 pod，预期创建失败
	//3. 使用label NotIn || In nodeIP 创建 pod，预期创建成功
	It("node_affinity_k8s_013 A pod specified with NotIn and In labels, will be failed to schedule; with NotIn or In labels, will be successfully scheduled.", func() {
		By("Get one node to schedule.")
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		nodeIP := ""

		for _, v := range nodeList.Items {
			if v.Name == nodeName {
				for _, addr := range v.Status.Addresses {
					if addr.Type == v1.NodeInternalIP {
						nodeIP = addr.Address
					}
				}
				break
			}
		}
		Expect(nodeIP).ToNot(Equal(""))

		By("Request a pod.")
		aff := util.GetAffinityNodeSelectorNotInRequirement(api.LabelNodeIP, []string{nodeIP})
		util.AddMatchExpressionsToFirstNodeSelectorTermsOfAffinity(api.LabelNodeIP, []string{nodeIP}, aff)
		pod := createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: aff,
		})
		By("Expect pod to be scheduled failed.")
		defer util.DeletePod(cs, pod)
		err := framework.WaitForPodNameUnschedulableInNamespace(cs, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)

		By("Request a pod.")
		aff = util.GetAffinityNodeSelectorNotInRequirement(api.LabelNodeIP, []string{nodeIP})
		util.AddMatchExpressionsToNewNodeSelectorTermsOfAffinity(api.LabelNodeIP, []string{nodeIP}, aff)
		pod = createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: aff,
		})

		By("Expect pod to be scheduled successfully.")
		err = framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		defer util.DeletePod(f.ClientSet, newPod)

		Expect(err).NotTo(HaveOccurred())
		Expect(newPod.Spec.NodeName).ToNot(Equal(""))
	})

	It("node_affinity_k8s_014 A pod specified with machine types, will be scheduled to the nodes belongs to the machine types, and in weight order.", func() {
		nodes := map[string]v1.Node{}
		machineTypes := []string{}
		nodeNames := []string{}
		for _, v := range nodeList.Items {
			nodes[v.Name] = v
			nodeNames = append(nodeNames, v.Name)
			sm, _ := v.Labels[api.LabelMachineModel]
			machineTypes = append(machineTypes, sm)
		}

		if len(nodeNames) < 2 {
			Skip("Nodes number is less than 2, skip.")
		}

		aff := util.GetAffinityNodeSelectorRequirement(api.LabelMachineModel, machineTypes[:2])
		weightFactor := 100 / len(machineTypes)
		for i, sm := range machineTypes {
			if i > 1 {
				break
			}
			aff = util.AddMatchExpressionsToPerferredScheduleOfAffinity(api.LabelMachineModel, []string{sm}, int32(100-i*weightFactor), aff)
		}

		By("Request a pod with machine types.")

		pod := createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: aff,
		})

		By("expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())

		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(nodeNames).To(ContainElement(newPod.Spec.NodeName))
		Expect(nodes[newPod.Spec.NodeName].Labels[api.LabelMachineModel]).To(Equal(machineTypes[0]))
		util.DeletePod(f.ClientSet, newPod)
	})

	It("node_affinity_k8s_015 A pod specified with machine types that no node belongs to, will be failed to schedule.", func() {
		machineTypes := []string{"X", "Y"}
		aff := util.GetAffinityNodeSelectorRequirement(api.LabelMachineModel, machineTypes)
		weightFactor := 100 / len(machineTypes)
		for i, sm := range machineTypes {
			aff = util.AddMatchExpressionsToPerferredScheduleOfAffinity(api.LabelMachineModel, []string{sm}, int32(100-i*weightFactor), aff)
		}

		By("Request a pod with machine types.")
		pod := createPausePod(f, pausePodConfig{
			Name:     "scheduler-" + string(uuid.NewUUID()),
			Affinity: aff,
		})

		By("Expect pod to be scheduled failed.")
		defer util.DeletePod(cs, pod)
		err := framework.WaitForPodNameUnschedulableInNamespace(cs, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)
	})

	It("node_affinity_k8s_016 A pod specified with mandatory labels, will be scheduled to the nodes that have the mandatory labels.", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		mandatoryLabelKey := "mandatory.k8s.alipay.com/appEnv"
		mandatoryLabelValue := "prepub"

		By("add mandatory label")
		framework.AddOrUpdateLabelOnNode(cs, nodeName, mandatoryLabelKey, mandatoryLabelValue)
		framework.ExpectNodeHasLabel(cs, nodeName, mandatoryLabelKey, mandatoryLabelValue)
		defer framework.RemoveLabelOffNode(cs, nodeName, mandatoryLabelKey)

		taint := &v1.Taint{
			Key:    mandatoryLabelKey,
			Value:  mandatoryLabelValue,
			Effect: v1.TaintEffectNoSchedule,
		}

		framework.AddOrUpdateTaintOnNode(cs, nodeName, *taint)
		framework.ExpectNodeHasTaint(cs, nodeName, taint)
		defer framework.RemoveTaintOffNode(cs, nodeName, *taint)

		By("Request a pod with mandatory label")
		tol := v1.Toleration{
			Key:      mandatoryLabelKey,
			Operator: v1.TolerationOpEqual,
			Value:    mandatoryLabelValue,
			Effect:   v1.TaintEffectNoSchedule,
		}

		pod := createPausePod(f, pausePodConfig{
			Name:        "scheduler-" + string(uuid.NewUUID()),
			Affinity:    util.GetAffinityNodeSelectorRequirement(mandatoryLabelKey, []string{mandatoryLabelValue}),
			Tolerations: []v1.Toleration{tol},
		})
		defer util.DeletePod(f.ClientSet, pod)

		By("expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeName).To(Equal(newPod.Spec.NodeName))
		util.DeletePod(f.ClientSet, newPod)
	})

	It("node_affinity_k8s_017 A pod specified with mandatory labels, will scheduled failed if all the nodes have no the mandatory labels.", func() {
		mandatoryLabelKey := "mandatory.k8s.alipay.com/appEnv"
		mandatoryLabelValue := "prepub"

		By("Request a pod with mandatory label")
		tol := v1.Toleration{
			Key:      mandatoryLabelKey,
			Operator: v1.TolerationOpEqual,
			Value:    mandatoryLabelValue,
			Effect:   v1.TaintEffectNoSchedule,
		}

		pod := createPausePod(f, pausePodConfig{
			Name:        "scheduler-" + string(uuid.NewUUID()),
			Affinity:    util.GetAffinityNodeSelectorRequirement(mandatoryLabelKey, []string{mandatoryLabelValue}),
			Tolerations: []v1.Toleration{tol},
		})
		defer util.DeletePod(f.ClientSet, pod)

		By("expect pod to be scheduled failed.")
		err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)
	})

	It("node_affinity_k8s_018 A pod can not be scheduled to the nodes that have the mandatory labels.", func() {
		mandatoryLabelKey := "mandatory.k8s.alipay.com/appEnv"
		mandatoryLabelValue := "prepub"
		taint := &v1.Taint{
			Key:    mandatoryLabelKey,
			Value:  mandatoryLabelValue,
			Effect: v1.TaintEffectNoSchedule,
		}

		for _, v := range nodeList.Items {
			By("add mandatory label to " + v.Name)
			framework.AddOrUpdateLabelOnNode(cs, v.Name, mandatoryLabelKey, mandatoryLabelValue)
			framework.ExpectNodeHasLabel(cs, v.Name, mandatoryLabelKey, mandatoryLabelValue)
			defer framework.RemoveLabelOffNode(cs, v.Name, mandatoryLabelKey)

			framework.AddOrUpdateTaintOnNode(cs, v.Name, *taint)
			framework.ExpectNodeHasTaint(cs, v.Name, taint)
			defer framework.RemoveTaintOffNode(cs, v.Name, *taint)
		}

		By("Request a pod")

		pod := createPausePod(f, pausePodConfig{
			Name: "scheduler-" + string(uuid.NewUUID()),
		})
		defer util.DeletePod(f.ClientSet, pod)

		By("expect pod to be scheduled failed.")
		err := framework.WaitForPodNameUnschedulableInNamespace(f.ClientSet, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)
	})

	//Nodeaffinity 验证：带 toleration 的 pod 可以调度到强制标节点
	//步骤：
	//1. 给所有机器打上强制标
	//2. 忽略指定的 taints 创建 Pod，预期创建成功
	It("node_affinity_k8s_019 A pod tolerates mandatory labels, can be scheduled to the nodes tainted with the mandatory labels.", func() {
		mandatoryLabelKey := "mandatory.k8s.alipay.com/appEnv"
		mandatoryLabelValue := "prepub"

		taint := &v1.Taint{
			Key:    mandatoryLabelKey,
			Value:  mandatoryLabelValue,
			Effect: v1.TaintEffectNoSchedule,
		}

		for _, v := range nodeList.Items {
			By("Add mandatory label to node.")
			framework.AddOrUpdateLabelOnNode(cs, v.Name, mandatoryLabelKey, mandatoryLabelValue)
			framework.ExpectNodeHasLabel(cs, v.Name, mandatoryLabelKey, mandatoryLabelValue)
			defer framework.RemoveLabelOffNode(cs, v.Name, mandatoryLabelKey)

			framework.AddOrUpdateTaintOnNode(cs, v.Name, *taint)
			framework.ExpectNodeHasTaint(cs, v.Name, taint)
			defer framework.RemoveTaintOffNode(cs, v.Name, *taint)
		}
		By("Request a pod with toleration.")
		tol := v1.Toleration{
			Key:      mandatoryLabelKey,
			Operator: v1.TolerationOpEqual,
			Value:    mandatoryLabelValue,
			Effect:   v1.TaintEffectNoSchedule,
		}

		pod := createPausePod(f, pausePodConfig{
			Name:        "scheduler-" + string(uuid.NewUUID()),
			Tolerations: []v1.Toleration{tol},
		})

		By("expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		defer util.DeletePod(f.ClientSet, newPod)
		Expect(err).NotTo(HaveOccurred())
	})

	//Nodeaffinity 验证：带 toleration 的 pod 可以调度到强制标节点
	//步骤：
	//1. 忽略指定的 taints 创建 Pod，预期创建成功
	It("node_affinity_k8s_020 A pod tolerates mandatory labels, can be scheduled to the normal nodes.", func() {
		By("Request a pod with toleration.")
		mandatoryLabelKey := "mandatory.k8s.alipay.com/appEnv"
		mandatoryLabelValue := "prepub"
		tol := v1.Toleration{
			Key:      mandatoryLabelKey,
			Operator: v1.TolerationOpEqual,
			Value:    mandatoryLabelValue,
			Effect:   v1.TaintEffectNoSchedule,
		}

		pod := createPausePod(f, pausePodConfig{
			Name:        "scheduler-" + string(uuid.NewUUID()),
			Tolerations: []v1.Toleration{tol},
		})

		By("expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		defer util.DeletePod(f.ClientSet, newPod)
		Expect(err).NotTo(HaveOccurred())
	})

	It("node_affinity_k8s_021 A pod ignore mandatory labels, can be scheduled to the nodes that do not have the mandatory labels.", func() {
		if len(nodeList.Items) < 2 {
			Skip("Nodes number is less than 2, skip.")
		}

		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		nodeIP := ""

		for _, v := range nodeList.Items {
			if nodeName != v.Name {
				for _, addr := range v.Status.Addresses {
					if addr.Type == v1.NodeInternalIP {
						nodeIP = addr.Address
					}
				}
				break
			}
		}
		Expect(nodeIP).ToNot(Equal(""))

		mandatoryLabelKey := "mandatory.k8s.alipay.com/appEnv"
		mandatoryLabelValue := "prepub"

		By("add mandatory label")
		framework.AddOrUpdateLabelOnNode(cs, nodeName, mandatoryLabelKey, mandatoryLabelValue)
		framework.ExpectNodeHasLabel(cs, nodeName, mandatoryLabelKey, mandatoryLabelValue)
		defer framework.RemoveLabelOffNode(cs, nodeName, mandatoryLabelKey)

		taint := &v1.Taint{
			Key:    mandatoryLabelKey,
			Value:  mandatoryLabelValue,
			Effect: v1.TaintEffectNoSchedule,
		}

		framework.AddOrUpdateTaintOnNode(cs, nodeName, *taint)
		framework.ExpectNodeHasTaint(cs, nodeName, taint)
		defer framework.RemoveTaintOffNode(cs, nodeName, *taint)

		By("Request a pod with mandatory label")
		tol := v1.Toleration{
			Key:      mandatoryLabelKey,
			Operator: v1.TolerationOpEqual,
			Value:    mandatoryLabelValue,
			Effect:   v1.TaintEffectNoSchedule,
		}

		aff := util.GetAffinityNodeSelectorRequirement(api.LabelNodeIP, []string{nodeIP})
		pod := createPausePod(f, pausePodConfig{
			Name:        "scheduler-" + string(uuid.NewUUID()),
			Affinity:    aff,
			Tolerations: []v1.Toleration{tol},
		})
		defer util.DeletePod(f.ClientSet, pod)

		By("expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeIP).To(Equal(newPod.Status.HostIP))
		util.DeletePod(f.ClientSet, newPod)
	})

	//Nodeaffinity 验证：operator 是 Exists,可以调度到强制标机器上
	//步骤：
	//1. 忽略所有 taints 创建 Pod，预期创建成功
	//2. 给所有机器打上强制标
	//3. 忽略所有 taints 创建 Pod，预期创建成功
	It("node_affinity_k8s_022 A pod tolerates all taints, can be scheduled to the nodes tainted with labels or normal nodes.", func() {
		tol := v1.Toleration{
			Operator: v1.TolerationOpExists,
			Effect:   v1.TaintEffectNoSchedule,
		}

		By("Request a pod with toleration.")
		pod := createPausePod(f, pausePodConfig{
			Name:        "scheduler-" + string(uuid.NewUUID()),
			Tolerations: []v1.Toleration{tol},
		})

		By("expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		defer util.DeletePod(f.ClientSet, newPod)
		Expect(newPod.Spec.NodeName).ToNot(Equal(""))
		Expect(err).NotTo(HaveOccurred())

		mandatoryLabelKey := "mandatory.k8s.alipay.com/appEnv"
		mandatoryLabelValue := "prepub"

		taint := &v1.Taint{
			Key:    mandatoryLabelKey,
			Value:  mandatoryLabelValue,
			Effect: v1.TaintEffectNoSchedule,
		}

		for _, v := range nodeList.Items {
			By("Add mandatory label to node.")
			framework.AddOrUpdateLabelOnNode(cs, v.Name, mandatoryLabelKey, mandatoryLabelValue)
			framework.ExpectNodeHasLabel(cs, v.Name, mandatoryLabelKey, mandatoryLabelValue)
			defer framework.RemoveLabelOffNode(cs, v.Name, mandatoryLabelKey)

			framework.AddOrUpdateTaintOnNode(cs, v.Name, *taint)
			framework.ExpectNodeHasTaint(cs, v.Name, taint)
			defer framework.RemoveTaintOffNode(cs, v.Name, *taint)
		}
		By("Request a pod with toleration.")
		pod = createPausePod(f, pausePodConfig{
			Name:        "scheduler-" + string(uuid.NewUUID()),
			Tolerations: []v1.Toleration{tol},
		})

		By("expect pod to be scheduled successfully.")
		err = framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err = cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		defer util.DeletePod(f.ClientSet, newPod)
		Expect(newPod.Spec.NodeName).ToNot(Equal(""))
		Expect(err).NotTo(HaveOccurred())
	})

	//Nodeaffinity 验证：多个强制标有效性
	//步骤：
	//1. 给一台机器打上多个强制标
	//2. 给其它机器打上单个强制标
	//3. 带上多个标签创建 Pod，预期创建成功
	//4. 带上单个标签创建 Pod，预期创建成功
	//5. 给其它机器打上多个强制标
	//6. 带上单个标签创建 Pod，预期创建失败
	It("node_affinity_k8s_023 A pod with multi mandatory labels, can be scheduled to the nodes with multi mandatory labels.", func() {
		if len(nodeList.Items) < 2 {
			Skip("Nodes number is less than 2, skip.")
		}
		nodeName := nodeList.Items[0].Name

		labelOneKey := "mandatory.k8s.alipay.com/appEnv"
		labelOneValue := "prepub"

		labelTwoKey := "mandatory.k8s.alibaba.com/server.owner"
		labelTwoValue := "prism"

		taintOne := &v1.Taint{
			Key:    labelOneKey,
			Value:  labelOneValue,
			Effect: v1.TaintEffectNoSchedule,
		}
		taintTwo := &v1.Taint{
			Key:    labelTwoKey,
			Value:  labelTwoValue,
			Effect: v1.TaintEffectNoSchedule,
		}
		framework.AddOrUpdateLabelOnNode(cs, nodeName, labelOneKey, labelOneValue)
		framework.ExpectNodeHasLabel(cs, nodeName, labelOneKey, labelOneValue)
		defer framework.RemoveLabelOffNode(cs, nodeName, labelOneKey)

		framework.AddOrUpdateLabelOnNode(cs, nodeName, labelTwoKey, labelTwoValue)
		framework.ExpectNodeHasLabel(cs, nodeName, labelTwoKey, labelTwoValue)
		defer framework.RemoveLabelOffNode(cs, nodeName, labelTwoKey)

		framework.AddOrUpdateTaintOnNode(cs, nodeName, *taintOne)
		framework.ExpectNodeHasTaint(cs, nodeName, taintOne)
		defer framework.RemoveTaintOffNode(cs, nodeName, *taintOne)
		framework.AddOrUpdateTaintOnNode(cs, nodeName, *taintTwo)
		framework.ExpectNodeHasTaint(cs, nodeName, taintTwo)
		defer framework.RemoveTaintOffNode(cs, nodeName, *taintTwo)

		for _, v := range nodeList.Items[1:] {
			By("Add mandatory label to node.")
			framework.AddOrUpdateLabelOnNode(cs, v.Name, labelOneKey, labelOneValue)
			framework.ExpectNodeHasLabel(cs, v.Name, labelOneKey, labelOneValue)
			defer framework.RemoveLabelOffNode(cs, v.Name, labelOneKey)
			framework.AddOrUpdateTaintOnNode(cs, v.Name, *taintOne)
			framework.ExpectNodeHasTaint(cs, v.Name, taintOne)
			defer framework.RemoveTaintOffNode(cs, v.Name, *taintOne)
		}

		tolOne := v1.Toleration{
			Key:      labelOneKey,
			Operator: v1.TolerationOpEqual,
			Value:    labelOneValue,
			Effect:   v1.TaintEffectNoSchedule,
		}
		tolTwo := v1.Toleration{
			Key:      labelTwoKey,
			Operator: v1.TolerationOpEqual,
			Value:    labelTwoValue,
			Effect:   v1.TaintEffectNoSchedule,
		}

		aff := util.GetAffinityNodeSelectorRequirement(labelOneKey, []string{labelOneValue})
		aff, _ = util.AddMatchExpressionsToFirstNodeSelectorTermsOfAffinity(labelTwoKey, []string{labelTwoValue}, aff)
		By("Request a pod with multi toleration.")
		pod := createPausePod(f, pausePodConfig{
			Name:        "scheduler-" + string(uuid.NewUUID()),
			Affinity:    aff,
			Tolerations: []v1.Toleration{tolOne, tolTwo},
		})

		By("Expect pod to be scheduled successfully.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err := cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		defer util.DeletePod(f.ClientSet, newPod)
		Expect(newPod.Spec.NodeName).To(Equal(nodeName))
		Expect(err).NotTo(HaveOccurred())

		By("Request a pod with one toleration.")
		aff = util.GetAffinityNodeSelectorRequirement(labelOneKey, []string{labelOneValue})
		pod = createPausePod(f, pausePodConfig{
			Name:        "scheduler-" + string(uuid.NewUUID()),
			Affinity:    aff,
			Tolerations: []v1.Toleration{tolOne},
		})

		By("expect pod to be scheduled successfully.")
		err = framework.WaitTimeoutForPodRunningInNamespace(cs, pod.Name, pod.Namespace, waitForPodRunningTimeout)
		Expect(err).NotTo(HaveOccurred())
		newPod, err = cs.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		defer util.DeletePod(f.ClientSet, newPod)
		Expect(newPod.Spec.NodeName).ToNot(Equal(nodeName))
		Expect(err).NotTo(HaveOccurred())

		for _, v := range nodeList.Items[1:] {
			By("Add mandatory label to node.")
			framework.AddOrUpdateLabelOnNode(cs, v.Name, labelTwoKey, labelTwoValue)
			framework.ExpectNodeHasLabel(cs, v.Name, labelTwoKey, labelTwoValue)
			defer framework.RemoveLabelOffNode(cs, v.Name, labelTwoKey)
			framework.AddOrUpdateTaintOnNode(cs, v.Name, *taintTwo)
			framework.ExpectNodeHasTaint(cs, v.Name, taintTwo)
			defer framework.RemoveTaintOffNode(cs, v.Name, *taintTwo)
		}

		By("Request a pod with one toleration.")
		aff = util.GetAffinityNodeSelectorRequirement(labelOneKey, []string{labelOneValue})
		pod = createPausePod(f, pausePodConfig{
			Name:        "scheduler-" + string(uuid.NewUUID()),
			Affinity:    aff,
			Tolerations: []v1.Toleration{tolOne},
		})

		By("expect pod to be scheduled failed.")
		defer util.DeletePod(cs, pod)
		err = framework.WaitForPodNameUnschedulableInNamespace(cs, pod.Name, f.Namespace.Name)
		Expect(err).To(BeNil(), "expect err be nil, got %s", err)
	})
})
