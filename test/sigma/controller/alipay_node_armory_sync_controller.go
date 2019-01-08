package controller_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	k8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/kubernetes/test/e2e/framework"
	alipayapis "gitlab.alipay-inc.com/sigma/apis/pkg/apis"

)

/*
1. 随机取一个node，获取有无annotation
2. 删除Annotation，然后查看是否更新
3. 再更新Annotation，查看是否更新annotation
 */

const (
	AnnotationNodeArmorySync = "meta.k8s.alipay.com/last-armory-sync"
)

var _ = Describe("[ant][sigma-alipay-controller][node-armory-sync]", func() {
	f := framework.NewDefaultFramework("sigma-ant-controller")
	//node-armory-sync在测试时设置的是1min同步一次，生产中30min同步一次，所以每次patch node之后等待61s等任务完成；
	It("[sigma-alipay-controller][NodeArmorySync] choose a node without annotation last-armory-sync, should patch labels and annotations.", func() {
		By("Get a Node from cluster and init node, no annotation last-armory-sync.")
		nodeName := GetNodeAndPatchInfo(f, "")

		By("Update node, wait controller resync and patch labels.")
		time.Sleep(61 * time.Second)
		//expect isUpdate=true.
		CheckNodeLabelsAndAnnotation(f, nodeName, "", true)

	})
	It("[sigma-alipay-controller][NodeArmorySync2] choose a node with annotation last-armory-syn and value more than 60s, labels and annotations should be updated.", func() {
		By("Get a Node from cluster and init node, annotation last-armory-sync delta-T more than 1min.")
		//当前时间-100s，一定是大于更新阈值，会触发更新操作；
		currentTime := fmt.Sprintf("%v", time.Now().UnixNano()-100000000000)
		framework.Logf("currentTime:%s", currentTime)
		nodeName := GetNodeAndPatchInfo(f, currentTime)

		By("Update node, wait controller patch labels.")
		time.Sleep(61 * time.Second)
		//expect isUpdate=true.
		CheckNodeLabelsAndAnnotation(f, nodeName, currentTime, true)
	})

	It("[sigma-alipay-controller][NodeArmorySync3] choose a node with annotation last-armory-sync and value less than 60s, labels and annotations should not be updated.", func() {
		By("Get a Node from cluster and init node, annotation last-armory-sync delta-T less than 1min.")
		//当前时间+10s，一定是小于更新阈值60s，不会触发更新操作；
		currentTime := fmt.Sprintf("%v", time.Now().UnixNano()+10000000000)
		nodeName := GetNodeAndPatchInfo(f, currentTime)
		By("Update node, wait controller resync, expect un-updated.")
		time.Sleep(61 * time.Second)
		//expect isUpdate=false.
		CheckNodeLabelsAndAnnotation(f, nodeName, currentTime, false)
	})
})

//GetNodeAndPatchInfo() choose one node patch info, return node name.
func GetNodeAndPatchInfo(f *framework.Framework, currentTime string) string {
	By("choose node.")
	nodes, err := f.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred(), "[ListNode] List node failed.")
	Expect(nodes.Items).NotTo(BeZero(), "[ListNode] No nodes in alipay dev.")
	node, err := f.ClientSet.CoreV1().Nodes().Get(nodes.Items[0].Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "[ListNode] get node failed.")
	//resync every 60s
	framework.Logf("currentTime:%s", currentTime)
	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		node, err := f.ClientSet.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		InitNode(node)
		//update last-armory-sync time.
		if currentTime != "" {
			node.Annotations[AnnotationNodeArmorySync] = currentTime
		}
		_, err = f.ClientSet.CoreV1().Nodes().Update(node)
		return err
	})
	Expect(err).To(BeNil(), "[PatchNode] Init node failed.")
	return node.Name
}

//InitNode() delete exist label before test.
func InitNode(node *corev1.Node) {
	delete(node.Annotations, AnnotationNodeArmorySync)
	delete(node.Labels, k8sapi.LabelAppName)
	delete(node.Labels, k8sapi.LabelRack)
	delete(node.Labels, k8sapi.LabelASW)
	delete(node.Labels, k8sapi.LabelNodeArmoryHostname)
	delete(node.Labels, k8sapi.LabelMachineModel)
	delete(node.Labels, k8sapi.LabelRoom)
	delete(node.Labels, k8sapi.LabelPhyIPRange)
	delete(node.Labels, k8sapi.LabelNetLogicSite)
	delete(node.Labels, k8sapi.LabelPOD)
	delete(node.Labels, k8sapi.LabelLogicPOD)
	delete(node.Labels, k8sapi.LabelParentServiceTag)
	delete(node.Labels, k8sapi.LabelDSWCluster)
	delete(node.Labels, k8sapi.LabelSecurityDomain)
	delete(node.Labels, k8sapi.LabelSite)
	delete(node.Labels, k8sapi.LabelNetArchVersion)
	delete(node.Labels, alipayapis.LabelModel)
	delete(node.Labels, alipayapis.LabelIDCManagerState)
}

//CheckLabelExist() check lable is exist.
func CheckLabelExist(label map[string]string, key string) bool {
	_, ok := label[key]
	return ok
}

//CheckNodeLabelsAndAnnotation() check node info. labels and annotation
func CheckNodeLabelsAndAnnotation(f *framework.Framework, name, currentTime string, isUpdate bool) {
	getNode, err := f.ClientSet.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "[GetNode] get node failed.")
	framework.Logf("Node Labels:%v, annotations:%v", getNode.Labels, getNode.Annotations)
	var matcher types.GomegaMatcher
	if isUpdate {
		matcher = BeTrue()
		Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelNodeIP)).To(BeTrue(), "[GetNode] Label node-ip does not exist.")
		Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelNodeSN)).To(BeTrue(), "[GetNode] Label node-sn does not exist.")
	} else {
		matcher = BeFalse()
	}
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelNodeArmoryHostname)).To(matcher, "[GetNode] Unexpected Label hostname.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelParentServiceTag)).To(matcher, "[GetNode] Unexpected Label parentservicetag.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelRack)).To(matcher, "[GetNode] Unexpected Label rack.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelRoom)).To(matcher, "[GetNode] Unexpected Label room.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelMachineModel)).To(matcher, "[GetNode] Unexpected Label machine model.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelASW)).To(matcher, "[GetNode] Unexpected Label.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelPhyIPRange)).To(matcher, "[GetNode] Unexpected Label phyip range.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelNetLogicSite)).To(matcher, "[GetNode] Unexpected Label net logic site.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelDSWCluster)).To(matcher, "[GetNode] Unexpected Label net dsw cluster.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelPOD)).To(matcher, "[GetNode] Unexpected Label  pod.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelLogicPOD)).To(matcher, "[GetNode] Unexpected Label  logicpod.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelSecurityDomain)).To(matcher, "[GetNode] Unexpected Label  security domain.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelSite)).To(matcher, "[GetNode] Unexpected Label site.")
	Expect(CheckLabelExist(getNode.Labels, k8sapi.LabelNetArchVersion)).To(matcher, "[GetNode] Unexpected label netArchVersion")
	Expect(CheckLabelExist(getNode.Labels, alipayapis.LabelIDCManagerState)).To(matcher, "[GetNode] Unexpected label idcManagerState")
	Expect(CheckLabelExist(getNode.Labels, alipayapis.LabelModel)).To(matcher, "[GetNode] Unexpected label Model")
	lastArmorySync, ok := getNode.Annotations[AnnotationNodeArmorySync]
	Expect(ok).To(BeTrue(), "[GetNode] Annotation last-armory-sync doesn't patch.")
	if isUpdate {
		if currentTime != "" {
			Expect(lastArmorySync).NotTo(Equal(currentTime), "[GetNode] Annotation last-armory-sync should not be updated.")
		} else {
			Expect(lastArmorySync).NotTo(BeEmpty(), "[GetNode] Annotation last-armory-sync should not be updated.")
		}
	} else {
		Expect(lastArmorySync).To(Equal(currentTime), "[GetNode] Annotation last-armory-sync should not be updated.")
	}
}
