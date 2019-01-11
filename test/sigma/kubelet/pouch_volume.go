package kubelet

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type pouchVolumeTestCase struct {
	pod             *v1.Pod
	checkCommand    string
	resultKeywords  []string
	checkMethod     string
	isCheckProperty bool
}

func doPouchVolumeTestCase(f *framework.Framework, testCase *pouchVolumeTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name
	// Step1: Create pod
	By("create pod")
	testPod, err := util.CreatePod(f.ClientSet, pod, f.Namespace.Name)
	Expect(err).NotTo(HaveOccurred(), "create pod err")
	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod's status is not running")

	// Step3: Check command's result.
	By("check command result")
	result := f.ExecShellInContainer(testPod.Name, containerName, testCase.checkCommand)
	framework.Logf("command resut: %v", result)
	checkResult(testCase.checkMethod, result, testCase.resultKeywords)

	// Step4: Check property.
	// Designed only for image "reg.docker.alibaba-inc.com/ali/os:7u2"
	By("check property if needed")
	if testCase.isCheckProperty {
		command := "stat -c %a /home/admin"
		result := f.ExecShellInContainer(testPod.Name, containerName, command)
		framework.Logf("command result: %v", result)
		Expect(result).To(Equal("755"), "invalid right of /home/admin")

		command = "stat -c %U /home/admin"
		result = f.ExecShellInContainer(testPod.Name, containerName, command)
		framework.Logf("command result: %v", result)
		Expect(result).To(Equal("admin"), "invalid user of /home/admin")

		command = "stat -c %G /home/admin"
		result = f.ExecShellInContainer(testPod.Name, containerName, command)
		framework.Logf("command result: %v", result)
		Expect(result).To(Equal("admin"), "invalid group of /home/admin")

		command = "stat -c %a /home/admin/.bashrc"
		result = f.ExecShellInContainer(testPod.Name, containerName, command)
		framework.Logf("command result: %v", result)
		Expect(result).To(Equal("755"), "invalid right of /home/admin/.bashrc")

		command = "stat -c %U /home/admin/.bashrc"
		result = f.ExecShellInContainer(testPod.Name, containerName, command)
		framework.Logf("command result: %v", result)
		Expect(result).To(Equal("admin"), "invalid user of /home/admin/.bashrc")

		command = "stat -c %G /home/admin/.bashrc"
		result = f.ExecShellInContainer(testPod.Name, containerName, command)
		framework.Logf("command result: %v", result)
		Expect(result).To(Equal("admin"), "invalid group of /home/admin/.bashrc")
	}
}

var _ = Describe("[sigma-kubelet][pouch-volume] check pouch-volume", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	pouchVolumeDriver := "alipay/pouch-volume"
	It("[smoke][ant] check pouch volume: image path exists", func() {
		pod := generateRunningPod()
		volumeMount := v1.VolumeMount{
			Name:      "disk",
			MountPath: "/home/admin",
		}
		image := "reg.docker.alibaba-inc.com/ali/os:7u2"
		pod.Spec.Containers[0].Image = image
		// Set pod's command to avoid init script
		pod.Spec.Containers[0].Command = []string{"sh", "-c"}
		pod.Spec.Containers[0].Args = []string{"sleep 1000"}
		volume := v1.Volume{
			Name: "disk",
			VolumeSource: v1.VolumeSource{
				FlexVolume: &v1.FlexVolumeSource{
					Driver:  pouchVolumeDriver,
					Options: map[string]string{"image": image, "imagePath": "/home/admin/."},
				},
			},
		}
		pod.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{volumeMount}
		pod.Spec.Volumes = []v1.Volume{volume}
		testCase := pouchVolumeTestCase{
			pod:             pod,
			checkCommand:    "ls -a /home/admin | wc -l",
			resultKeywords:  []string{"5"},
			checkMethod:     checkMethodEqual,
			isCheckProperty: true,
		}
		doPouchVolumeTestCase(f, &testCase)
	})

	It("[ant] check pouch volume: image path doesn't exists", func() {
		pod := generateRunningPod()
		volumeMount := v1.VolumeMount{
			Name:      "disk",
			MountPath: "/home/admin",
		}
		image := "reg.docker.alibaba-inc.com/ali/os:7u2"
		pod.Spec.Containers[0].Image = image
		// Set pod's command to avoid init script
		pod.Spec.Containers[0].Command = []string{"sh", "-c"}
		pod.Spec.Containers[0].Args = []string{"sleep 1000"}
		volume := v1.Volume{
			Name: "disk",
			VolumeSource: v1.VolumeSource{
				FlexVolume: &v1.FlexVolumeSource{
					Driver:  pouchVolumeDriver,
					Options: map[string]string{"image": image, "imagePath": "/home/notexists/."},
				},
			},
		}
		pod.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{volumeMount}
		pod.Spec.Volumes = []v1.Volume{volume}
		testCase := pouchVolumeTestCase{
			pod:            pod,
			checkCommand:   "ls -a /home/admin | wc -l",
			resultKeywords: []string{"2"},
			checkMethod:    checkMethodEqual,
		}
		doPouchVolumeTestCase(f, &testCase)
	})
})
