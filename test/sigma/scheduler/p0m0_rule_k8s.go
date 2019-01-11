package scheduler

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/swarm"
	"k8s.io/kubernetes/test/sigma/util"
)

func addP0M0Rul() error {
	keyP0 := "/applications/metainfos/et15sqa/sigma-test-p0"
	valueP0 := `{"PriorityClass":"p0","PriorityConstaints":{"A7":{"m0":1,"p0":2,"p0+m0":2}},"UpdateTime":"2017-08-28 12:07:52"}`
	keyM0 := "/applications/metainfos/et15sqa/sigma-test-m0"
	valueM0 := `{"PriorityClass":"m0","PriorityConstaints":{"A7":{"m0":1,"p0":2,"p0+m0":2}},"UpdateTime":"2017-08-28 12:05:02"}`

	if err := swarm.AddP0M0Rules(keyP0, valueP0); err != nil {
		return err
	}

	if err := swarm.AddP0M0Rules(keyM0, valueM0); err != nil {
		return err
	}
	return nil
}

var _ = Describe("[sigma-3.1][sigma-scheduler][p0m0][p2]", func() {
	var cs clientset.Interface

	f := framework.NewDefaultFramework(CPUSetNameSpace)
	f.AllNodesReadyTimeout = 3 * time.Second

	BeforeEach(func() {
		// TODO: uncomment it when p0m0 rule in k8s is ready
		Skip("the p0m0 is not ready, skip the test")

		// add p0m0 rul
		err := addP0M0Rul()
		if err != nil {
			Skip("add p0m0 rule failed, skip the test")
		}

		cs = f.ClientSet
	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			DumpSchedulerState(f, 0)
		}
		DeleteSigmaContainer(f)
	})

	It("[p2] Pod should success or fail to be scheduled according to p0 constraint. ", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		// add /node/logicinfo/SmName
		swarm.CreateOrUpdateNodeLogicInfoSmName(nodeName, "A7")

		// Apply node affinity label to each node
		nodeAffinityKey := "node-for-p0m0-e2e-test"
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		By("create one p0 deployunit pod should be scheduled the specified node successfully")
		pod1 := createPausePod(f, pausePodConfig{
			Name:     "scheduler-p0m0-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
			Labels: map[string]string{
				"ali.DeployUnit": "sigma-test-p0",
			},
		})
		defer util.DeletePod(f.ClientSet, pod1)

		By("Wait the pod becomes running")
		framework.ExpectNoError(f.WaitForPodRunning(pod1.Name))

		By("create the second p0 deployunit pod should be scheduled the specified node successfully")
		pod2 := createPausePod(f, pausePodConfig{
			Name:     "scheduler-p0m0-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
			Labels: map[string]string{
				"ali.DeployUnit": "sigma-test-p0",
			},
		})
		defer util.DeletePod(f.ClientSet, pod2)

		By("Wait the pod becomes running")
		framework.ExpectNoError(f.WaitForPodRunning(pod2.Name))

		By("create the third p0 deployunit pod should failed due to p0=2 constrains")
		pod3 := createPausePod(f, pausePodConfig{
			Name:     "scheduler-p0m0-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
			Labels: map[string]string{
				"ali.DeployUnit": "sigma-test-p0",
			},
		})
		defer util.DeletePod(f.ClientSet, pod3)
		framework.Logf("expect pod failed to be scheduled.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod3.Name, pod3.Namespace, waitForPodRunningTimeout)
		Expect(err).ToNot(BeNil(), "expect err not be nil, got %s", err)
	})

	It("[p2] Pod should success or fail to be scheduled according to m0 constraint. ", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		// add /node/logicinfo/SmName
		swarm.CreateOrUpdateNodeLogicInfoSmName(nodeName, "A7")

		// Apply node affinity label to each node
		nodeAffinityKey := "node-for-p0m0-e2e-test"
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		By("create one m0 deployunit pod should be scheduled the specified node successfully")
		pod1 := createPausePod(f, pausePodConfig{
			Name:     "scheduler-p0m0-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
			Labels: map[string]string{
				"ali.DeployUnit": "sigma-test-m0",
			},
		})
		defer util.DeletePod(f.ClientSet, pod1)

		By("Wait the pod becomes running")
		framework.ExpectNoError(f.WaitForPodRunning(pod1.Name))

		By("create the seconde m0 deployunit pod should failed due to m0=1 constrains")
		pod2 := createPausePod(f, pausePodConfig{
			Name:     "scheduler-p0m0-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
			Labels: map[string]string{
				"ali.DeployUnit": "sigma-test-m0",
			},
		})
		defer util.DeletePod(f.ClientSet, pod2)
		framework.Logf("expect pod failed to be scheduled.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod2.Name, pod2.Namespace, waitForPodRunningTimeout)
		Expect(err).ToNot(BeNil(), "expect err not be nil, got %s", err)
	})

	It("[p2][ali] Pod should success or fail to be scheduled according to p0+m0 constraint. ", func() {
		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		// add /node/logicinfo/SmName
		swarm.CreateOrUpdateNodeLogicInfoSmName(nodeName, "A7")

		// Apply node affinity label to each node
		nodeAffinityKey := "node-for-p0m0-e2e-test"
		framework.AddOrUpdateLabelOnNode(cs, nodeName, nodeAffinityKey, nodeName)
		framework.ExpectNodeHasLabel(cs, nodeName, nodeAffinityKey, nodeName)
		defer framework.RemoveLabelOffNode(cs, nodeName, nodeAffinityKey)

		By("create one m0 deployunit pod should be scheduled the specified node successfully")
		pod1 := createPausePod(f, pausePodConfig{
			Name:     "scheduler-p0m0-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
			Labels: map[string]string{
				"ali.DeployUnit": "sigma-test-m0",
			},
		})
		defer util.DeletePod(f.ClientSet, pod1)

		By("Wait the pod becomes running")
		framework.ExpectNoError(f.WaitForPodRunning(pod1.Name))

		By("create the second p0 deployunit pod should be scheduled the specified node successfully")
		pod2 := createPausePod(f, pausePodConfig{
			Name:     "scheduler-p0m0-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
			Labels: map[string]string{
				"ali.DeployUnit": "sigma-test-p0",
			},
		})
		defer util.DeletePod(f.ClientSet, pod2)

		By("Wait the pod becomes running")
		framework.ExpectNoError(f.WaitForPodRunning(pod2.Name))

		By("create the second p0 deployunit pod should fail due to p0+m0=2 constraint.")
		pod3 := createPausePod(f, pausePodConfig{
			Name:     "scheduler-p0m0-" + string(uuid.NewUUID()),
			Affinity: util.GetAffinityNodeSelectorRequirement(nodeAffinityKey, []string{nodeName}),
			Labels: map[string]string{
				"ali.DeployUnit": "sigma-test-p0",
			},
		})
		defer util.DeletePod(f.ClientSet, pod3)
		framework.Logf("expect pod failed to be scheduled.")
		err := framework.WaitTimeoutForPodRunningInNamespace(cs, pod3.Name, pod3.Namespace, waitForPodRunningTimeout)
		Expect(err).ToNot(BeNil(), "expect err not be nil, got %s", err)
	})

})
