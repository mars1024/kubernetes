/*
Copyright 2018 Alipay.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller_test

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	v1node "k8s.io/kubernetes/pkg/api/v1/node"
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	taintutils "k8s.io/kubernetes/pkg/util/taints"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	UnreachableNoScheduleTaintTemplate = &v1.Taint{
		Key:    algorithm.TaintNodeUnreachable,
		Effect: v1.TaintEffectNoSchedule,
	}

	NotReadyNoScheduleTaintTemplate = &v1.Taint{
		Key:    algorithm.TaintNodeNotReady,
		Effect: v1.TaintEffectNoSchedule,
	}
)

type sigmaletOperator interface {
	Stop(node *v1.Node) error
	Start(node *v1.Node) error
}

var (
	o sigmaletOperator = &alipaySAService{}
)

var _ = Describe("[ant][sigma-alipay-controller][node-lifecycle]", func() {
	f := framework.NewDefaultFramework("sigma-ant-controller")
	It("[sigma-alipay-controller][NodeLifeCycle] taint Node with :NoSchedule", func() {
		By("Get a Node from cluster")
		nodes, err := f.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred(), "[ListNode] List node failed.")
		Expect(nodes.Items).NotTo(BeZero(), "[ListNode] No nodes in cluster.")
		node, err := f.ClientSet.CoreV1().Nodes().Get(nodes.Items[0].Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "[ListNode] get node failed.")

		By("kill sigmalet")
		defer func() {
			o.Start(node)

			err = nodeWithoutNotReadyTaint(f.ClientSet, node.Name)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("wait node recover without taint timeout, err: %v", err))
		}()
		err = o.Stop(node)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("stop sigmalet failed, err: %v", err))

		By("wait for Node Taint Updated with :NoSchedule")
		err = wait.Poll(time.Second*30, time.Minute*3, func() (done bool, err error) {
			node, err := f.ClientSet.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
			if err != nil {
				return true, err
			}

			_, condition := v1node.GetNodeCondition(&node.Status, v1.NodeReady)
			if nil == condition {
				return true, fmt.Errorf("node condition 'Ready' is not found")
			}

			if condition.Status == v1.ConditionTrue {
				return false, nil
			} else if condition.Status == v1.ConditionFalse {
				if taintutils.TaintExists(node.Spec.Taints, NotReadyNoScheduleTaintTemplate) {
					return true, nil
				} else {
					return false, nil
				}
			} else if condition.Status == v1.ConditionUnknown {
				if taintutils.TaintExists(node.Spec.Taints, UnreachableNoScheduleTaintTemplate) {
					return true, nil
				} else {
					return false, nil
				}
			}

			return true, fmt.Errorf("never reach")
		})
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("wait node taint timeout, err: %v", err))

		By("restart sigmalet")
		err = o.Start(node)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("restart sigmalet failed, err: %v", err))

		By("wait for clear Node Taint :NoSchedule")
		err = nodeWithoutNotReadyTaint(f.ClientSet, node.Name)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("wait node clear taint timeout, err: %v", err))
	})
})

func nodeWithoutNotReadyTaint(client clientset.Interface, nodeName string) error {
	return wait.Poll(time.Second*10, time.Minute*3, func() (done bool, err error) {
		node, err := client.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		_, condition := v1node.GetNodeCondition(&node.Status, v1.NodeReady)
		if nil == condition {
			return true, fmt.Errorf("node condition 'Ready' is not found")
		}

		if condition.Status == v1.ConditionTrue {
			if !taintutils.TaintExists(node.Spec.Taints, NotReadyNoScheduleTaintTemplate) &&
				!taintutils.TaintExists(node.Spec.Taints, UnreachableNoScheduleTaintTemplate) {
				return true, nil
			} else {
				return false, nil
			}
		} else if condition.Status == v1.ConditionFalse {
			return false, nil
		} else if condition.Status == v1.ConditionUnknown {
			return false, nil
		}

		return true, fmt.Errorf("never reach")
	})
}

type alipaySAService struct {
}

func (s *alipaySAService) Stop(node *v1.Node) error {
	sn := node.Labels["sigma.ali/node-sn"]
	ip := node.Labels["sigma.ali/node-ip"]

	// TODO: ip 和 sn 会不会获取失败？

	resp, err := util.ResponseFromStarAgentTask("cmd://systemctl stop sigma-slave", ip, sn)

	glog.Infof("stop sigmalet resp: %s", resp)

	return err
}

func (s *alipaySAService) Start(node *v1.Node) error {
	sn := node.Labels["sigma.ali/node-sn"]
	ip := node.Labels["sigma.ali/node-ip"]

	// TODO: ip 和 sn 会不会获取失败？

	resp, err := util.ResponseFromStarAgentTask("cmd://systemctl restart sigma-slave", ip, sn)

	glog.Infof("restart sigmalet resp: %s", resp)

	return err
}
