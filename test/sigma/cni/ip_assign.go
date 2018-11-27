package cni

import (
	"path/filepath"
	"time"

	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/env"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-cni]", func() {
	f := framework.NewDefaultFramework("sigma-cni")
	var testPod *v1.Pod

	It("[smoke] create one pod whose ip is assigned by IPAM", func() {
		By("create a pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred())

		testPod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		By("check pod is running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(getPod.Status.HostIP).NotTo(BeEmpty(), "status.HostIP should not be empty")
		Expect(getPod.Status.PodIP).NotTo(BeEmpty(), "status.PodIP should not be empty")
	})

	It("[overlay] create one pod whose ip is assigned by overlay network", func() {
		By("create a pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred())
		// set node affinity to make sure using overlay node
		pod.Spec.Affinity = &v1.Affinity{
			NodeAffinity: &v1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
					NodeSelectorTerms: []v1.NodeSelectorTerm{
						{
							MatchExpressions: []v1.NodeSelectorRequirement{
								{
									Key:      "sigma.ali/is-overlay-network",
									Operator: v1.NodeSelectorOpIn,
									Values:   []string{"true"},
								},
							},
						},
					},
				},
			},
		}
		if env.GetTester() == env.TesterJituan {
			framework.Logf("set overlay network toleration")
			pod.Spec.Tolerations = append(pod.Spec.Tolerations, v1.Toleration{
				Key:      sigmak8sapi.LabelIsOverlayNetwork,
				Operator: v1.TolerationOpEqual,
				Value:    "true",
				Effect:   v1.TaintEffectNoSchedule,
			})
			ret, err := json.Marshal(pod)
			Expect(err).NotTo(HaveOccurred(), "json.Marshal pod err")
			framework.Logf(string(ret))
		}
		testPod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		By("check pod is running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(getPod.Status.HostIP).NotTo(BeEmpty(), "status.HostIP should not be empty")
		Expect(getPod.Status.PodIP).NotTo(BeEmpty(), "status.PodIP should not be empty")
		framework.Logf("overlay pod[%s] is created on node[%s]", getPod.Status.PodIP, getPod.Status.HostIP)
	})
})
