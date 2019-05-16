package kubelet

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
	"path/filepath"
	"strconv"
)

type ignoreContainerReadyTest struct {
	ignoreContainerReady bool
}

var _ = Describe("[sigma-kubelet][ant] ignore sidecar container ready", func() {
	f := framework.NewDefaultFramework("e2e-ak8s-kubelet")
	It("When generate pod ready condition, ignore some containers which have special env ", func() {
		doIgnoreContainerReayTest(f, ignoreContainerReadyTest{true})
	})
	It("[smoke] When generate pod ready condition, don't ignore containers which not  have special env", func() {
		doIgnoreContainerReayTest(f, ignoreContainerReadyTest{false})
	})
})

func doIgnoreContainerReayTest(f *framework.Framework, testcase ignoreContainerReadyTest) {
	var testPod *v1.Pod
	By("Load a pod from file")
	podFile := filepath.Join(util.TestDataDir, "pod-base.json")
	pod, err := util.LoadPodFromFile(podFile)
	Expect(err).NotTo(HaveOccurred(), "fail to load pod from file")
	By("Add a new container conf")
	container := v1.Container{
		Image:           "reg.docker.alibaba-inc.com/ali/os:7u2",
		ImagePullPolicy: "IfNotPresent",
		Name:            "pod-base-add",
		Env: []v1.EnvVar{
			{
				Name:  sigmak8sapi.EnvIgnoreReady,
				Value: strconv.FormatBool(testcase.ignoreContainerReady),
			},
		},
		ReadinessProbe: &v1.Probe{
			Handler: v1.Handler{
				Exec: &v1.ExecAction{
					Command: []string{
						"cat /tmp/ignore_container_ready", // it will fail forever,and container will not ready forever
					},
				},
			},
		},
	}
	pod.Spec.Containers = append(pod.Spec.Containers, container)
	By("Create a pod")
	testPod = f.PodClient().Create(pod)
	defer util.DeletePod(f.ClientSet, testPod)
	By("Waiting for pods to come up.")
	err = framework.WaitForPodRunningInNamespace(f.ClientSet, testPod)
	framework.ExpectNoError(err, "waiting for server pod to start")
	By("check pod is ready ")
	if testcase.ignoreContainerReady {
		Expect(f.PodClient().PodIsReady(testPod.Name)).To(BeTrue(), "Expect pod's Ready condition to be true")
	} else {
		Expect(f.PodClient().PodIsReady(testPod.Name)).To(BeFalse(), "Expect pod's Ready condition to be false")
	}
}
