package scheduler

import (
	"fmt"
	"strings"
	"time"

	_ "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/env"
	"k8s.io/kubernetes/test/sigma/swarm"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("[sigma-2.0][sigma-scheduler][smoke][p1][nodeaffinity][Serial]", func() {
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

	It("A pod with node IP label should be scheduled to the specified node.", func() {

		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		waitNodeResourceReleaseComplete(nodeName)
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

		By("Trying to launch the container with label ali.SpecifiedNcIps:" + nodeIP)
		name := "container-with-specified-ncips" + string(uuid.NewUUID())
		containerLabels := map[string]string{
			"ali.SpecifiedNcIps": nodeIP,
			"ali.RequestId":      name,
			"ali.RequirementId":  name,
		}

		container, err := swarm.CreateContainerSyncWithLabels(name, containerLabels)
		containersToDelete = append(containersToDelete, container.ID)
		Expect(err).NotTo(HaveOccurred())

		By("Verify the container was scheduled to the expected node.")
		if env.GetTester() == "ant" {
			Expect(container.Host().HostIP).To(Equal(nodeIP))
		} else {
			// jituan 需要根据 containerID 重新获取 container 的信息
			// 这里参数必须用 name，因为这个是 requestId
			containerResult := swarm.GetRequestState(name)
			Expect(containerResult).ShouldNot(BeNil())
			Expect(containerResult.HostIP).To(Equal(nodeIP))
		}
	})

	It("A pod specified with a node IP, will not be scheduled to the node, if the node has mandatory labels.", func() {

		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		waitNodeResourceReleaseComplete(nodeName)
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

		mandatoryLabels := map[string]string{
			"a": "b",
		}
		extLabels := map[string]string{
			"c": "d",
		}
		nodeName = strings.ToUpper(nodeName)
		swarm.CreateOrUpdateNodeMandatoryLabel(nodeName, mandatoryLabels)
		swarm.EnsureNodeHasMandatoryLabels(nodeName, mandatoryLabels)
		defer swarm.DeleteNodeMandatoryLabels(nodeName, "a")

		swarm.CreateOrUpdateNodeLabel(nodeName, extLabels)
		swarm.EnsureNodeHasLabels(nodeName, extLabels)
		defer swarm.DeleteNodeLabels(nodeName, "c")

		By("Trying to launch the container with label ali.SpecifiedNcIps:" + nodeIP)
		containerLabels := map[string]string{
			"ali.SpecifiedNcIps": nodeIP,
		}

		name := "container-with-specified-ncips"
		container, err := swarm.CreateContainerSyncWithLabels(name, containerLabels)
		containersToDelete = append(containersToDelete, container.ID)

		if env.GetTester() == "ant" {
			Expect(err).NotTo(HaveOccurred())
			Expect(container.Host().HostIP).To(Equal(""))
		} else {
			Expect(container.IsScheduled()).Should(Equal(false), fmt.Sprintf("expect container failed to be scheduled, result: %+v", container))
		}
	})

	It("A pod specified with a node IP and IgnoreLabelBySpecifiedIp, will be scheduled to the node with mandatory labels.", func() {

		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		waitNodeResourceReleaseComplete(nodeName)
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

		mandatoryLabels := map[string]string{
			"a": "b",
		}
		extLabels := map[string]string{
			"c": "d",
		}
		nodeName = strings.ToUpper(nodeName)
		swarm.CreateOrUpdateNodeMandatoryLabel(nodeName, mandatoryLabels)
		swarm.EnsureNodeHasMandatoryLabels(nodeName, mandatoryLabels)
		defer swarm.DeleteNodeMandatoryLabels(nodeName, "a")

		swarm.CreateOrUpdateNodeLabel(nodeName, extLabels)
		swarm.EnsureNodeHasLabels(nodeName, extLabels)
		defer swarm.DeleteNodeLabels(nodeName, "c")

		name := "container-with-specified-ncips" + string(uuid.NewUUID())
		By("Trying to launch the container with label ali.SpecifiedNcIps:" + nodeIP)
		containerLabels := map[string]string{
			"ali.SpecifiedNcIps":           nodeIP,
			"ali.IgnoreLabelBySpecifiedIp": "true",
			"ali.RequestId":                name,
			"ali.RequirementId":            name,
		}

		container, err := swarm.CreateContainerSyncWithLabels(name, containerLabels)
		containersToDelete = append(containersToDelete, container.ID)
		Expect(err).NotTo(HaveOccurred())

		By("Verify the container was scheduled to the expected node.")
		if env.GetTester() == "ant" {
			Expect(container.Host().HostIP).To(Equal(nodeIP))
		} else {
			// jituan 需要根据 containerID 重新获取 container 的信息
			containerResult := swarm.GetRequestState(name)
			Expect(containerResult).ShouldNot(BeNil())
			Expect(containerResult.HostIP).To(Equal(nodeIP))
		}
	})

	It("A pod specified with a node IP and mandatory, will be scheduled to the node with mandatory labels.", func() {

		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		waitNodeResourceReleaseComplete(nodeName)
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

		mandatoryLabels := map[string]string{
			"a": "b",
		}
		extLabels := map[string]string{
			"c": "d",
		}
		nodeName = strings.ToUpper(nodeName)
		swarm.CreateOrUpdateNodeMandatoryLabel(nodeName, mandatoryLabels)
		swarm.EnsureNodeHasMandatoryLabels(nodeName, mandatoryLabels)
		defer swarm.DeleteNodeMandatoryLabels(nodeName, "a")

		swarm.CreateOrUpdateNodeLabel(nodeName, extLabels)
		swarm.EnsureNodeHasLabels(nodeName, extLabels)
		defer swarm.DeleteNodeLabels(nodeName, "c")

		name := "container-with-specified-ncips" + string(uuid.NewUUID())
		By("Trying to launch the container with label ali.SpecifiedNcIps:" + nodeIP)
		containerLabels := map[string]string{
			"ali.SpecifiedNcIps": nodeIP,
			"constraint:Label:a": "b",
			"ali.RequestId":      name,
			"ali.RequirementId":  name,
		}

		container, err := swarm.CreateContainerSyncWithLabels(name, containerLabels)
		containersToDelete = append(containersToDelete, container.ID)
		Expect(err).NotTo(HaveOccurred())

		By("Verify the container was scheduled to the expected node.")
		if env.GetTester() == "ant" {
			Expect(container.Host().HostIP).To(Equal(nodeIP))
		} else {
			// jituan 需要根据 containerID 重新获取 container 的信息
			containerResult := swarm.GetRequestState(name)
			Expect(containerResult).ShouldNot(BeNil())
			Expect(containerResult.HostIP).To(Equal(nodeIP))
		}
	})

	It("A pod specified with a node IP and ignore mandatory label, will be scheduled to the node with mandatory labels.", func() {
		// 集团不支持这个标签 "ali.IgnoreMandatory": "a=b"
		if env.Tester != "ant" {
			Skip("this test is for ant sigam2.0 only")
		}

		nodeName := GetNodeThatCanRunPod(f)
		Expect(nodeName).ToNot(BeNil())

		waitNodeResourceReleaseComplete(nodeName)
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

		mandatoryLabels := map[string]string{
			"a": "b",
		}
		extLabels := map[string]string{
			"c": "d",
		}
		nodeName = strings.ToUpper(nodeName)
		swarm.CreateOrUpdateNodeMandatoryLabel(nodeName, mandatoryLabels)
		swarm.EnsureNodeHasMandatoryLabels(nodeName, mandatoryLabels)
		defer swarm.DeleteNodeMandatoryLabels(nodeName, "a")

		swarm.CreateOrUpdateNodeLabel(nodeName, extLabels)
		swarm.EnsureNodeHasLabels(nodeName, extLabels)
		defer swarm.DeleteNodeLabels(nodeName, "c")

		name := "container-with-specified-ncips" + string(uuid.NewUUID())
		By("Trying to launch the container with label ali.SpecifiedNcIps:" + nodeIP)
		containerLabels := map[string]string{
			"ali.SpecifiedNcIps":  nodeIP,
			"ali.IgnoreMandatory": "a=b",
			"ali.RequestId":       name,
			"ali.RequirementId":   name,
		}

		container, err := swarm.CreateContainerSyncWithLabels(name, containerLabels)
		containersToDelete = append(containersToDelete, container.ID)
		Expect(err).NotTo(HaveOccurred())

		By("Verify the container was scheduled to the expected node.")
		if env.GetTester() == "ant" {
			Expect(container.Host().HostIP).To(Equal(nodeIP))
		} else {
			// jituan 需要根据 containerID 重新获取 container 的信息
			containerResult := swarm.GetRequestState(name)
			Expect(containerResult).ShouldNot(BeNil())
			Expect(containerResult.HostIP).To(Equal(nodeIP))
		}
	})
})
