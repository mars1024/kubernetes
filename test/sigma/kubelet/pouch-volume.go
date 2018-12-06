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
	pod            *v1.Pod
	checkCommand   string
	resultKeywords []string
	checkMethod    string
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
	result := f.ExecShellInContainer(testPod.Name, containerName, testCase.checkCommand)
	framework.Logf("command resut: %v", result)
	checkResult(testCase.checkMethod, result, testCase.resultKeywords)
}

var _ = Describe("[sigma-kubelet][pouch-volume] check pouch-volume", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")
	pouchVolumeDriver := "alipay/pouch-volume"
	// There are four items in /var/cache of mysql:test-v1 image.
	It("[smoke][ant] check pouch volume: image path exists", func() {
		pod := generateRunningPod()
		volumeMount := v1.VolumeMount{
			Name:      "disk",
			MountPath: "/var/cache",
		}
		image := pod.Spec.Containers[0].Image
		volume := v1.Volume{
			Name: "disk",
			VolumeSource: v1.VolumeSource{
				FlexVolume: &v1.FlexVolumeSource{
					Driver:  pouchVolumeDriver,
					Options: map[string]string{"image": image, "imagePath": "/var/cache/."},
				},
			},
		}
		pod.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{volumeMount}
		pod.Spec.Volumes = []v1.Volume{volume}
		testCase := pouchVolumeTestCase{
			pod:            pod,
			checkCommand:   "ls /var/cache/ | wc -l",
			resultKeywords: []string{"4"},
			checkMethod:    checkMethodEqual,
		}
		doPouchVolumeTestCase(f, &testCase)
	})
	It("[ant] check pouch volume: image path doesn't exists", func() {
		pod := generateRunningPod()
		volumeMount := v1.VolumeMount{
			Name:      "disk",
			MountPath: "/var/cache",
		}
		image := pod.Spec.Containers[0].Image
		volume := v1.Volume{
			Name: "disk",
			VolumeSource: v1.VolumeSource{
				FlexVolume: &v1.FlexVolumeSource{
					Driver:  pouchVolumeDriver,
					Options: map[string]string{"image": image, "imagePath": "/var/cachenoexists/."},
				},
			},
		}
		pod.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{volumeMount}
		pod.Spec.Volumes = []v1.Volume{volume}
		testCase := pouchVolumeTestCase{
			pod:            pod,
			checkCommand:   "ls /var/cache/ | wc -l",
			resultKeywords: []string{"0"},
			checkMethod:    checkMethodEqual,
		}
		doPouchVolumeTestCase(f, &testCase)
	})
})
