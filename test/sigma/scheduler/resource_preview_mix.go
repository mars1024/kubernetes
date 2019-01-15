package scheduler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/kubernetes/test/sigma/swarm"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = Describe("[sigma-2.0+3.1][sigma-scheduler][resource][Serial]", func() {
	var cs clientset.Interface
	var nodeList *v1.NodeList

	nodeToAllocatableMapCPU := make(map[string]int64)
	nodeToAllocatableMapMem := make(map[string]int64)
	nodeToAllocatableMapEphemeralStorage := make(map[string]int64)

	nodesInfo := make(map[string]*v1.Node)

	f := framework.NewDefaultFramework("sigma-scheduler")

	BeforeEach(func() {
		cs = f.ClientSet
		nodeList = &v1.NodeList{}
		masterNodes, nodeList = getMasterAndWorkerNodesOrDie(cs)
		for i, node := range nodeList.Items {
			waitNodeResourceReleaseComplete(node.Name)
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

	It("preview_001 A pod with cpu/mem/ephemeral-storage request should be scheduled on node with enough resource with both k8s and sigma API."+
		"While fail to be scheduled on node with insufficient resource", func() {

		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		framework.Logf("get one node to schedule, nodeName: %s", nodeName)

		AllocatableCPU := nodeToAllocatableMapCPU[nodeName]
		AllocatableMemory := nodeToAllocatableMapMem[nodeName]
		AllocatableDisk := nodeToAllocatableMapEphemeralStorage[nodeName]

		requestedCPU := AllocatableCPU / 2
		requestedMemory := AllocatableMemory / 2
		requestedDisk := AllocatableDisk / 2

		By("Request a pod with CPU/Memory/EphemeralStorage.")
		nodeIP := nodesInfo[nodeName].Status.Addresses[0].Address

		tests := []resourceCase{
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
				shouldScheduled: true,
				spreadStrategy:  "sameCoreFirst",
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypePreview,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: true,
				cpushare:        true,
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypeKubernetes,
				shouldScheduled: true,
				cpushare:        true,
				affinityConfig:  map[string][]string{api.LabelNodeIP: {nodeIP}},
			},
			{
				cpu:             requestedCPU,
				mem:             requestedMemory,
				ethstorage:      requestedDisk,
				requestType:     requestTypePreview,
				affinityConfig:  map[string][]string{"ali.SpecifiedNcIps": {nodeIP}},
				shouldScheduled: false,
				cpushare:        true,
			},
		}

		testContext := &testContext{
			caseName:  "preview_001",
			cs:        cs,
			localInfo: nil,
			f:         f,
			testCases: tests,
		}

		testContext.execTests()
	})
})
