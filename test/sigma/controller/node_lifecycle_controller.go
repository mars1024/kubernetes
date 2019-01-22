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
	v1pod "k8s.io/kubernetes/pkg/api/v1/pod"
	nodeutil "k8s.io/kubernetes/pkg/controller/util/node"
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

	// UnreachableTaintTemplate is the taint for when a node becomes unreachable.
	UnreachableTaintTemplate = &v1.Taint{
		Key:    algorithm.TaintNodeUnreachable,
		Effect: v1.TaintEffectNoExecute,
	}

	// NotReadyTaintTemplate is the taint for when a node is not ready for
	// executing pods
	NotReadyTaintTemplate = &v1.Taint{
		Key:    algorithm.TaintNodeNotReady,
		Effect: v1.TaintEffectNoExecute,
	}
)

type sigmaletOperator interface {
	Stop(node *v1.Node) error
	Start(node *v1.Node) error
}

var (
	o sigmaletOperator = &alipaySAService{}
)

var (
	MalfunctionNoExecuteTaint = v1.Taint{
		Key:    "node.sigma.ali/lifecycle",
		Value:  "malfunction",
		Effect: v1.TaintEffectNoExecute,
	}
)

var _ = Describe("[sigma-controller][node-lifecycle]", func() {
	f := framework.NewDefaultFramework("sigma-ant-controller")
	It("[sigma-controller][node-lifecycle] taint Node with :NoSchedule, without :NoExecute", func() {
		By("Get a Node from cluster")
		nodes, err := f.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred(), "[ListNode] List node failed.")
		Expect(nodes.Items).NotTo(BeZero(), "[ListNode] No nodes in cluster.")
		node, err := f.ClientSet.CoreV1().Nodes().Get(nodes.Items[0].Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "[ListNode] get node failed.")

		pod, err := createEvictTestPod(f.ClientSet, f.Namespace.Name, node.Name)
		Expect(err).NotTo(HaveOccurred(), "[CreatePod] create test pod failed.")
		defer func() {
			err = f.ClientSet.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred(), "[ClearPod] clear test pod failed.")
		}()
		pod, err = waitPodReadyStatus(f.ClientSet, pod.Namespace, pod.Name, true)
		Expect(err).NotTo(HaveOccurred(), "[CreatePod] wait for pod ready failed.")

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

			if taintutils.TaintExists(node.Spec.Taints, NotReadyTaintTemplate) ||
				taintutils.TaintExists(node.Spec.Taints, UnreachableTaintTemplate) {
				return true, fmt.Errorf("node has :NoExecute taint")
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

		// TODO: 这里 err 也有可能是 GetPod 发生的。必须判断这个 err 是 timeout
		pod, err = waitPodReadyStatus(f.ClientSet, pod.Namespace, pod.Name, false)
		Expect(err).To(HaveOccurred(), "[PodStatus] pod condition ready had been false when Node is NotReady.")

		By("restart sigmalet")
		err = o.Start(node)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("restart sigmalet failed, err: %v", err))

		By("wait for clear Node Taint :NoSchedule")
		err = nodeWithoutNotReadyTaint(f.ClientSet, node.Name)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("wait node clear taint timeout, err: %v", err))
	})

	It("[sigma-controller][node-lifecycle] evict Pod using Label", func() {
		By("Get a Node from cluster")
		nodes, err := f.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred(), "[ListNode] List node failed.")
		Expect(nodes.Items).NotTo(BeZero(), "[ListNode] No nodes in cluster.")
		node, err := f.ClientSet.CoreV1().Nodes().Get(nodes.Items[0].Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "[ListNode] get node failed.")

		By("create a test Pod")
		pod, err := createEvictTestPod(f.ClientSet, f.Namespace.Name, node.Name)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("create a test Pod failed, err: %v", err))
		defer func() {
			var err error
			for i := 0; i < 10; i++ {
				if err = f.ClientSet.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{}); nil == err {
					return
				}
			}
			glog.Errorf("delete test Pod failed: %v", err)
		}()

		By("taint Node with node.sigma.ali/lifecycle=offline:NoExecute")
		taintSuccess := false
		for i := 0; i < 10; i++ {
			if taintSuccess = nodeutil.SwapNodeControllerTaint(f.ClientSet, []*v1.Taint{&MalfunctionNoExecuteTaint}, nil, node); taintSuccess {
				break
			}
		}
		Expect(taintSuccess).To(Equal(true), "should taint success")
		defer func() {
			for i := 0; i < 10; i++ {
				if nodeutil.SwapNodeControllerTaint(f.ClientSet, nil, []*v1.Taint{&MalfunctionNoExecuteTaint}, node) {
					return
				}
			}
			glog.Errorf("un-taint Node failed")
		}()

		By("wait Pod has evict Label")
		err = wait.Poll(time.Second*10, time.Minute*3, func() (done bool, err error) {
			if pod, err = f.ClientSet.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{}); nil != err {
				return false, err
			}

			if v, ok := pod.Labels["pod.sigma.ali/eviction"]; ok && v == "true" {
				return true, nil
			}

			return false, nil
		})
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("wait Pod has evict Label err: %v", err))
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

func createEvictTestPod(client clientset.Interface, nsName string, nodeName string) (*v1.Pod, error) {
	By("create a test pod")

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nodelifecyclectrl" + time.Now().Format("20160607123450"),
			Namespace: nsName,
			Labels: map[string]string{
				"sigma.ali/site":                  "test",
				"sigma.ali/app-name":              "common-app",
				"sigma.ali/instance-group":        "pouch-test_testhost",
				"sigma.alibaba-inc.com/app-unit":  "CENTER_UNIT.center",
				"sigma.alibaba-inc.com/app-stage": "DAILY",
				"sigma.ali/deploy-unit":           "dev",
				"meta.k8s.alipay.com/zone":        "AZ00A",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "nginx",
					Image: "reg.docker.alibaba-inc.com/antk8s/nginx",
				},
			},
			NodeName: nodeName,
		},
	}

	testPod, err := client.CoreV1().Pods(nsName).Create(pod)
	glog.Info("create pod config:%v", pod)

	return testPod, err
}

func waitPodReadyStatus(client clientset.Interface, nsName string, podName string, ready bool) (pod *v1.Pod, err error) {
	err = wait.Poll(time.Second*5, time.Minute*1, func() (done bool, err error) {
		pod, err = client.CoreV1().Pods(nsName).Get(podName, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		currentReady := v1pod.IsPodReady(pod)

		if currentReady == ready {
			return true, nil
		} else {
			return false, nil
		}
	})

	if nil != err {
		return pod, fmt.Errorf("wait for Pod condition ready to %v error: %v", ready, err)
	}

	return pod, err
}
