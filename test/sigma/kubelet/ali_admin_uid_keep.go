package kubelet

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"path/filepath"
	"strings"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-kubelet] Keep ali admin UID", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	var testPod *v1.Pod

	It("Generate ali admin uid automatic [pouch-only]", func() {
		By("Load a pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "fail to load pod from file")

		By("Update container env, assign it to docker vm")
		if len(pod.Spec.Containers[0].Env) == 0 {
			pod.Spec.Containers[0].Env = make([]v1.EnvVar, 0, 1)
		}
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, v1.EnvVar{
			Name:  "ali_admin_uid",
			Value: "0",
		})

		By("Create a pod")
		testPod = f.PodClient().Create(pod)
		defer util.DeletePod(f.ClientSet, testPod)

		By("Waiting for pods to come up.")
		err = framework.WaitForPodRunningInNamespace(f.ClientSet, testPod)
		framework.ExpectNoError(err, "waiting for server pod to start")

		By("Get ali admin uid env by exec cmd")
		out, err := util.PodExec(testPod, "/usr/bin/env | grep ali_admin_uid")
		framework.Logf("ali admin uid get by container exec is : %s", out)
		Expect(err).NotTo(HaveOccurred(), "Get ali admin uid env by  exec err")
		Expect(strings.Trim(out, "\n")).Should(MatchRegexp("^ali_admin_uid=[0-9]*$"))
		Expect(strings.Trim(out, "\n")).Should(Not(Equal("ali_admin_uid=0")))
	})

	It("Generate ali admin uid  by annotation", func() {
		By("Load a pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "fail to load pod from file")

		By("Update container env, assign it to docker vm")
		if len(pod.Spec.Containers[0].Env) == 0 {
			pod.Spec.Containers[0].Env = make([]v1.EnvVar, 0, 1)
		}
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, v1.EnvVar{
			Name:  "ali_admin_uid",
			Value: "123123123123123",
		})

		By("Create a pod")
		testPod = f.PodClient().Create(pod)
		defer util.DeletePod(f.ClientSet, testPod)

		By("Waiting for pods to come up.")
		err = framework.WaitForPodRunningInNamespace(f.ClientSet, testPod)
		framework.ExpectNoError(err, "waiting for server pod to start")

		By("Get ali admin uid env by exec cmd")
		out, err := util.PodExec(testPod, "/usr/bin/env | grep ali_admin_uid")
		framework.Logf("ali admin uid get by container exec is : %s", out)
		Expect(err).NotTo(HaveOccurred(), "Get ali admin uid env by  exec err")
		Expect("ali_admin_uid=123123123123123").Should(Equal(strings.Trim(out, "\n")))
	})

	It("[smoke] Generate ali admin uid  by annotation, when upgrade keep it", func() {
		patchDataAdd := `{"spec":{"containers":[{"name":"pod-base","command":["/bin/sh"],"args":["-c", "sleep 1000"]}]}}`
		containerName := "pod-base"
		successStr := "upgrade container success"

		By("Load a pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		Expect(err).NotTo(HaveOccurred(), "fail to load pod from file")

		By("Update container env, assign it to docker vm")
		if len(pod.Spec.Containers[0].Env) == 0 {
			pod.Spec.Containers[0].Env = make([]v1.EnvVar, 0, 1)
		}
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, v1.EnvVar{
			Name:  "ali_admin_uid",
			Value: "123123123123123",
		})

		By("Create a pod")
		testPod = f.PodClient().Create(pod)
		defer util.DeletePod(f.ClientSet, testPod)

		By("Waiting for pods to come up.")
		err = framework.WaitForPodRunningInNamespace(f.ClientSet, testPod)
		framework.ExpectNoError(err, "waiting for server pod to start")

		By("Get ali admin uid env by exec cmd")
		out, err := util.PodExec(testPod, "/usr/bin/env | grep ali_admin_uid")
		framework.Logf("ali admin uid get by container exec is : %s", out)
		Expect(err).NotTo(HaveOccurred(), "Get ali admin uid env by  exec err")
		Expect("ali_admin_uid=123123123123123").Should(Equal(strings.Trim(out, "\n")))

		By("Update pod, and ali admin uid keep")
		testPod, err = f.PodClient().Patch(testPod.Name, types.StrategicMergePatchType, []byte(patchDataAdd))
		Expect(err).NotTo(HaveOccurred(), "Container upgrade err")

		// Step4: Wait for upgrade action finished.
		By("Waiting until pod upgrade")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, testPod, containerName, 3*time.Minute, successStr, true)
		Expect(err).NotTo(HaveOccurred(), "upgrade pod err")

		By("Get ali admin uid env by exec cmd")
		out, err = util.PodExec(testPod, "/usr/bin/env | grep ali_admin_uid")
		framework.Logf("ali admin uid get by container exec is : %s", out)
		Expect(err).NotTo(HaveOccurred(), "Get ali admin uid env by  exec err")
		Expect("ali_admin_uid=123123123123123").Should(Equal(strings.Trim(out, "\n")))
	})
})
