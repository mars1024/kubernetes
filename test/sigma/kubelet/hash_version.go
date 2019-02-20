package kubelet

import (
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	sigmautil "k8s.io/kubernetes/pkg/kubelet/sigma"
	"k8s.io/kubernetes/pkg/util/slice"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

type hashVersionTestCase struct {
	pod *v1.Pod
}

// updateContainerHashVersion can change container's hashVersion:
// 1. stop sigma-slave
// 2. update container hashVersion label
// 3. start sigma-slave
// TODO: Support stop and start sigmalet in minisigma.
func updateContainerHashVersion(hostSn string, hostIP string, runtimeType util.ContainerdType, containerID string) {
	//Stop sigmalet
	command := "cmd://systemctl(stop sigma-slave)"
	_, err := util.ResponseFromStarAgentTask(command, hostIP, hostSn)
	Expect(err).NotTo(HaveOccurred(), "failed to stop sigmalet")

	//Update container's label
	updateCommand := ""
	if runtimeType == util.ContainerdTypeDocker {
		updateCommand = fmt.Sprintf(`cmd://docker(update -l "annotation.sigma.alibaba-inc.com.hashVersion"="100.14.5" %s)`, containerID)
	} else {
		updateCommand = fmt.Sprintf(`cmd://pouch(update -l "annotation.sigma.alibaba-inc.com.hashVersion"="100.14.5" %s)`, containerID)
	}
	_, err = util.ResponseFromStarAgentTask(updateCommand, hostIP, hostSn)
	Expect(err).NotTo(HaveOccurred(), "update cotnainer failed")

	defer func() {
		//Start sigmalet
		command = "cmd://systemctl(start sigma-slave)"
		_, err = util.ResponseFromStarAgentTask(command, hostIP, hostSn)
		Expect(err).NotTo(HaveOccurred(), "failed to stop sigmalet")
	}()
}

// restartContainer can stop a container then start it.
func restartContainer(client clientset.Interface, pod *v1.Pod) {
	containerName := pod.Spec.Containers[0].Name
	namespace := pod.Namespace

	err := util.StopContainer(client, pod, namespace, containerName)
	Expect(err).NotTo(HaveOccurred(), "stop pod err")
	err = util.StartContainer(client, pod, namespace, containerName)
	Expect(err).NotTo(HaveOccurred(), "start pod err")
}

// getContainerID can get container's id from container status.
func getContainerID(pod *v1.Pod) (string, error) {
	if len(pod.Status.ContainerStatuses) == 0 {
		return "", fmt.Errorf("Invalid pod status: %+v", pod.Status)
	}
	segs := strings.Split(pod.Status.ContainerStatuses[0].ContainerID, "//")
	if len(segs) != 2 {
		return "", fmt.Errorf("Failed to get container id from pod: %v", segs)
	}
	return segs[1], nil
}

// doHashVersionTestCase tests four conditions:
// 1. create a new container and check hashVersion;
// 2. modify env of pod with higher hashVersion(trigger upgrade);
// 3. modify image of pod with higher hashVersion(trigger upgrade);
// 4. modify other field of pod with higher hashVersion(not triiger upgrade).
func doHashVersionTestCase(f *framework.Framework, testCase *hashVersionTestCase) {
	pod := testCase.pod
	containerName := pod.Spec.Containers[0].Name
	upgradeSuccessStr := "upgrade container success"

	// Step1: Create pod
	By("create pod")
	testPod, err := util.CreatePod(f.ClientSet, pod, f.Namespace.Name)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Step3: Get created container
	By("Get created pod")
	getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get pod err")
	hostIP := getPod.Status.HostIP
	hostSn := util.GetHostSnFromHostIp(hostIP)

	// Step4: Get and check hashVersion.
	By("Get and check hashVersion")
	containerID, err := getContainerID(getPod)
	Expect(err).NotTo(HaveOccurred(), "get container id err")

	// Step5: Check hashVersion
	By("Get and check hashVersion")
	format := `'{{index .Config.Labels "annotation.sigma.alibaba-inc.com.hashVersion"}}'`
	hashVersionStr, err := util.GetContainerInspectField(hostIP, containerID, format)
	framework.Logf("hash version result: %s", hashVersionStr)
	Expect(err).NotTo(HaveOccurred(), "failed get hashVersion from container inspect")
	hashVersionStr = strings.Replace(hashVersionStr, "\n", "", -1)

	Expect(hashVersionStr).Should(Equal(sigmautil.VERSION_CURRENT))

	// Step6: Get runtime type
	By("Get runtimeType")
	runtimeType, err := util.GetContainerDType(hostIP)
	Expect(err).NotTo(HaveOccurred(), "get runtime type error")

	// Only test in ant environment.
	// Because it is not easy to stop and start sigmalet in minisigma.
	env := os.Getenv("TESTER")
	notDefaultValueEnvs := []string{"ant"}
	if slice.ContainsString(notDefaultValueEnvs, env, nil) {
		// Step7: Update container's hashVersion
		By("Update container's hashVersion directly")
		updateContainerHashVersion(hostSn, hostIP, runtimeType, containerID)

		// Step8: Upgrade by change Env
		By("Upgrade higher hashVersion container by changing env")
		patchData := `{"spec":{"containers":[{"name":"pod-base","env":[{"name":"Upgrade","value":"2"}]}]}}`
		upgradedPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(patchData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		// Step9: Wait for upgrade action finished.
		By("Wait until pod is upgraded (change Env)")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 3*time.Minute, upgradeSuccessStr, true)
		Expect(err).NotTo(HaveOccurred(), "upgrade pod err")

		// Step8: Clear Upgrade's status so we can do next upgrade.
		By("Clear Upgrade's status")
		restartContainer(f.ClientSet, upgradedPod)

		// Step9: Get new pod
		By("Get new pod and refresh container id")
		getPod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "get pod err")
		containerID, err = getContainerID(getPod)
		Expect(err).NotTo(HaveOccurred(), "get container id err")

		// Step10: Update container's hashVersion
		By("Update container's hashVersion directly")
		updateContainerHashVersion(hostSn, hostIP, runtimeType, containerID)

		// Step11: Upgrade by change Image
		By("Upgrade higher hashVersion container by changing Image")
		patchData = `{"spec":{"containers":[{"name":"pod-base","image":"reg.docker.alibaba-inc.com/sigma-x/mysql:test-v2"}]}}`
		upgradedPod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(patchData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		// Step12: Wait for upgrade action finished.
		By("Wait until pod is upgraded (change Image)")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 3*time.Minute, upgradeSuccessStr, true)
		Expect(err).NotTo(HaveOccurred(), "upgrade pod err")

		// Step13: Clear Upgrade's status so we can do next upgrade.
		By("Clear Upgrade's status")
		restartContainer(f.ClientSet, upgradedPod)

		/// Step14: Get new pod
		By("Get new pod and refresh container id")
		getPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(testPod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "get pod err")
		containerID, err := getContainerID(getPod)
		Expect(err).NotTo(HaveOccurred(), "get container id err")

		// Step15: Update container's hashVersion
		By("Update container's hashVersion directly")
		updateContainerHashVersion(hostSn, hostIP, runtimeType, containerID)

		// Step16: Upgrade by change privileged
		By("Upgrade higher hashVersion container by changing privileged")
		patchData = `{"spec":{"containers":[{"name":"pod-base","securityContext":{"privileged":false}}]}}`
		upgradedPod, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, types.StrategicMergePatchType, []byte(patchData))
		Expect(err).NotTo(HaveOccurred(), "patch pod err")

		// Step17: Wait for upgrade action finished.
		// Change privileged should not triggered Upgrade.
		// So timeout error is expected.
		By("Wait until pod is upgraded (change privileged)")
		err = util.WaitTimeoutForContainerUpdateStatus(f.ClientSet, upgradedPod, containerName, 1*time.Minute, upgradeSuccessStr, true)
		Expect(err).To(HaveOccurred(), "expected upgrade pod err")
	}

}

var _ = Describe("[sigma-kubelet][hash-version] test for hashVersion: check current hashVersion and deal with higher hashVersion pod", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	It("[smoke][Serial] hash version", func() {
		pod := generateRunningPod()
		testCase := hashVersionTestCase{
			pod: pod,
		}
		if len(pod.Spec.Containers[0].Env) == 0 {
			pod.Spec.Containers[0].Env = []v1.EnvVar{}
		}
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, v1.EnvVar{Name: "Upgrade", Value: "1"})

		isPrivileged := true
		securityContext := &v1.SecurityContext{
			Privileged: &isPrivileged,
		}
		pod.Spec.Containers[0].SecurityContext = securityContext

		doHashVersionTestCase(f, &testCase)
	})
})
