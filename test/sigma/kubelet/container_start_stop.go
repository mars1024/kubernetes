package kubelet

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/golang/glog"

	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var _ = Describe("[sigma-kubelet] Container start/stop check", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	var testPod *v1.Pod
	var containerStateAnnotation = "{\"states\":{\"%s\":\"%s\"}}"

	It("[smoke] create one pod, stop its container by annotation, and then start it also by annotation", func() {
		By("create a pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		pod.Spec.Containers[0].Name = "container-start-stop"
		Expect(err).NotTo(HaveOccurred())

		testPod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred(), "create pod err")

		defer util.DeletePod(f.ClientSet, testPod)

		By("check pod is running and have pod/host IP")
		err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not running")

		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(getPod.Status.HostIP).NotTo(BeEmpty(), "status.HostIP should not be empty")
		Expect(getPod.Status.PodIP).NotTo(BeEmpty(), "status.PodIP should not be empty")
		containerName := getPod.Status.ContainerStatuses[0].Name

		By("stop container in pod by patching pod annotation")
		getPod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = fmt.Sprintf(containerStateAnnotation, containerName, "exited")
		glog.Info("pod annotation is :%v", getPod.Annotations)
		patchData, _ := json.Marshal(getPod)
		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, patchData)
		Expect(err).NotTo(HaveOccurred(), "patch pod error")

		By("check container should be stopped")
		// check pod status
		err = util.WaitTimeoutForPodStatus(f.ClientSet, getPod, v1.PodPending, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not pending, but should be pending")
		// check container update status from annotation
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, getPod, containerName, 1*time.Minute, "kill container success")
		Expect(err).NotTo(HaveOccurred(), "\"kill container success\" does not appear in container update status")
		// log into slave node and check container status, container should be stopped
		runOutput := util.GetDockerPsOutput(getPod.Status.HostIP, containerName)
		if runOutput == "" {
			runOutput = util.GetPouchPsOutput(getPod.Status.HostIP, containerName)
		}
		if !strings.Contains(runOutput, "Exited") && !strings.Contains(runOutput, "Stopped") {
			framework.Logf(runOutput)
			Fail("container status is not Exited or Stopped, but we expect it should be that")
		}

		By("start container again in pod by patching pod annotation")
		getPod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		getPod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = fmt.Sprintf(containerStateAnnotation, containerName, "running")
		glog.Info("pod annotation is :%v", getPod.Annotations)
		patchData, _ = json.Marshal(getPod)
		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, patchData)
		Expect(err).NotTo(HaveOccurred(), "patch pod error")

		By("check container should be running")
		// check pod status
		err = util.WaitTimeoutForPodStatus(f.ClientSet, getPod, v1.PodRunning, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "pod status is not Running, but should be Running")
		// log into slave node and check container status, now container should be up
		runOutput = util.GetDockerPsOutput(getPod.Status.HostIP, containerName)
		if runOutput == "" {
			runOutput = util.GetPouchPsOutput(getPod.Status.HostIP, containerName)
		}
		Expect(runOutput).To(ContainSubstring("Up"))
	})
	It("can't update container to running which pod restart policy is never", func() {
		containerName := "container-start-stop"
		By("create a pod from file")
		podFile := filepath.Join(util.TestDataDir, "pod-base.json")
		pod, err := util.LoadPodFromFile(podFile)
		pod.Spec.Containers[0].Name = containerName
		pod.Spec.RestartPolicy = v1.RestartPolicyNever
		Expect(err).NotTo(HaveOccurred())

		By("Create a pod")
		testPod = f.PodClient().Create(pod)
		defer util.DeletePod(f.ClientSet, testPod)

		By("Waiting for pods to come up.")
		err = framework.WaitForPodRunningInNamespace(f.ClientSet, testPod)
		framework.ExpectNoError(err, "waiting for server pod to start")

		By("patching pod annotation")
		getPod, err := f.PodClient().Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		if len(getPod.Annotations) == 0 {
			getPod.Annotations = make(map[string]string, 1)
		}
		getPod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = fmt.Sprintf(containerStateAnnotation, containerName, "running")
		patchData, _ := json.Marshal(getPod)
		_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, patchData)
		Expect(err).To(HaveOccurred())
		framework.Logf("%s", err.Error())
		Expect(err.Error()).To(ContainSubstring("pod restart policy is never, so container can't be started"))
	})
})
