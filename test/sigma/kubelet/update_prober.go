package kubelet

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var typeLivenessProbe = "LivenessProbe"
var typeReadinessProbe = "ReadinessProbe"

type updateProberTestCase struct {
	pod              *v1.Pod
	containerName    string
	probeType        string
	patchData        string
	patchType        types.PatchType
	checkCommandPre  string
	resultKeywordPre []string
	checkMethodPre   string
	shouldCheckPre   bool
	checkCommandPro  string
	resultKeywordPro []string
	checkMethodPro   string
	shouldCheckPro   bool
	waitSec          int
}

func doUpdateProberTestCase(f *framework.Framework, testCase *updateProberTestCase) {
	pod := testCase.pod
	containerName := testCase.containerName

	// Step1: Create pod
	By("create pod")
	testPod, err := util.CreatePod(f.ClientSet, pod, f.Namespace.Name)
	Expect(err).NotTo(HaveOccurred(), "create pod err")

	defer util.DeletePod(f.ClientSet, testPod)

	// Step2: Wait for container's creation finished.
	By("wait until pod running and have pod/host IP")
	err = util.WaitTimeoutForPodStatus(f.ClientSet, testPod, v1.PodRunning, 3*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "pod status is not running")

	// Do init work such as create a file in anonymous volume
	if testCase.shouldCheckPre {
		time.Sleep(time.Duration(testCase.waitSec) * time.Second)
		resultPre := f.ExecShellInContainer(testPod.Name, containerName, testCase.checkCommandPre)
		framework.Logf("commandPre resut: %v", resultPre)
		checkResult(testCase.checkMethodPre, resultPre, testCase.resultKeywordPre)
	}

	// Step3: Update container to tigger upgrade action.
	By("change container's prober")
	_, err = f.ClientSet.CoreV1().Pods(f.Namespace.Name).Patch(testPod.Name, testCase.patchType, []byte(testCase.patchData))
	Expect(err).NotTo(HaveOccurred(), "patch pod err")

	// Check command's result.
	if testCase.shouldCheckPro {
		time.Sleep(time.Duration(2*testCase.waitSec) * time.Second)
		resultPro := f.ExecShellInContainer(testPod.Name, containerName, testCase.checkCommandPro)
		framework.Logf("commandPro resut: %v", resultPro)
		checkResult(testCase.checkMethodPro, resultPro, testCase.resultKeywordPro)
	}
}

var _ = Describe("[sigma-kubelet][update-prober] check livenessProbe or readinessProbe update", func() {
	f := framework.NewDefaultFramework("sigma-kubelet")

	It("[smoke] update livenessProbe", func() {
		pod := generateRunningPod()
		livenessProbe := v1.Probe{
			Handler: v1.Handler{
				Exec: &v1.ExecAction{
					Command: []string{"/bin/bash", "-c", "echo helloLiveness1 > /home/helloLiveness1"},
				},
			},
			TimeoutSeconds: 2,
			PeriodSeconds:  5,
		}
		pod.Spec.Containers[0].LivenessProbe = &livenessProbe
		testCase := updateProberTestCase{
			pod:              pod,
			containerName:    pod.Spec.Containers[0].Name,
			probeType:        typeLivenessProbe,
			patchData:        `{"spec":{"containers":[{"name":"pod-base","livenessProbe":{"exec":{"command":["/bin/bash", "-c", "echo helloLiveness2 > /home/helloLiveness2"]}}}]}}`,
			patchType:        types.StrategicMergePatchType,
			checkCommandPre:  "cat /home/helloLiveness1",
			resultKeywordPre: []string{"helloLiveness1"},
			checkMethodPre:   checkMethodEqual,
			shouldCheckPre:   true,
			checkCommandPro:  "cat /home/helloLiveness2",
			resultKeywordPro: []string{"helloLiveness2"},
			checkMethodPro:   checkMethodEqual,
			shouldCheckPro:   true,
			waitSec:          7,
		}
		doUpdateProberTestCase(f, &testCase)
	})
	It("add livenessProbe", func() {
		pod := generateRunningPod()
		testCase := updateProberTestCase{
			pod:              pod,
			containerName:    pod.Spec.Containers[0].Name,
			probeType:        typeLivenessProbe,
			patchData:        `{"spec":{"containers":[{"name":"pod-base","livenessProbe":{"exec":{"command":["/bin/bash", "-c", "echo helloLiveness1 > /home/helloLiveness1"]}}}]}}`,
			patchType:        types.StrategicMergePatchType,
			checkCommandPre:  "",
			resultKeywordPre: []string{},
			checkMethodPre:   "",
			shouldCheckPre:   false,
			checkCommandPro:  "cat /home/helloLiveness1",
			resultKeywordPro: []string{"helloLiveness1"},
			checkMethodPro:   checkMethodEqual,
			shouldCheckPro:   true,
			waitSec:          7,
		}
		doUpdateProberTestCase(f, &testCase)
	})
	It("delete livenessProbe", func() {
		pod := generateRunningPod()
		livenessProbe := v1.Probe{
			Handler: v1.Handler{
				Exec: &v1.ExecAction{
					Command: []string{"/bin/bash", "-c", "echo helloLiveness1 > /home/helloLiveness1"},
				},
			},
			TimeoutSeconds: 2,
			PeriodSeconds:  5,
		}
		pod.Spec.Containers[0].LivenessProbe = &livenessProbe
		testCase := updateProberTestCase{
			pod:              pod,
			containerName:    pod.Spec.Containers[0].Name,
			probeType:        typeLivenessProbe,
			patchData:        `[{"op":"remove","path":"/spec/containers/0/livenessProbe","value":""}]`,
			patchType:        types.JSONPatchType,
			checkCommandPre:  "cat /home/helloLiveness1",
			resultKeywordPre: []string{"helloLiveness1"},
			checkMethodPre:   checkMethodEqual,
			shouldCheckPre:   true,
			checkCommandPro:  "echo '' > /home/helloLiveness1 && cat /home/helloLiveness1",
			resultKeywordPro: []string{"helloLiveness1"},
			checkMethodPro:   checkMethodNotEqual,
			shouldCheckPro:   true,
			waitSec:          7,
		}
		doUpdateProberTestCase(f, &testCase)
	})
	It("[smoke] update livenessProbe in multiContainer pod", func() {
		pod := generateMultiConRunningPod()
		livenessProbe := v1.Probe{
			Handler: v1.Handler{
				Exec: &v1.ExecAction{
					Command: []string{"/bin/bash", "-c", "echo helloLiveness1 > /home/helloLiveness1"},
				},
			},
			TimeoutSeconds: 2,
			PeriodSeconds:  5,
		}
		pod.Spec.Containers[1].LivenessProbe = &livenessProbe
		testCase := updateProberTestCase{
			pod:              pod,
			containerName:    pod.Spec.Containers[1].Name,
			probeType:        typeLivenessProbe,
			patchData:        `{"spec":{"containers":[{"name":"pod-base1","livenessProbe":{"exec":{"command":["/bin/bash", "-c", "echo helloLiveness2 > /home/helloLiveness2"]}}}]}}`,
			patchType:        types.StrategicMergePatchType,
			checkCommandPre:  "cat /home/helloLiveness1",
			resultKeywordPre: []string{"helloLiveness1"},
			checkMethodPre:   checkMethodEqual,
			shouldCheckPre:   true,
			checkCommandPro:  "cat /home/helloLiveness2",
			resultKeywordPro: []string{"helloLiveness2"},
			checkMethodPro:   checkMethodEqual,
			shouldCheckPro:   true,
			waitSec:          7,
		}
		doUpdateProberTestCase(f, &testCase)
	})
	It("[smoke] update readinessProbe", func() {
		pod := generateRunningPod()
		readinessProbe := v1.Probe{
			Handler: v1.Handler{
				Exec: &v1.ExecAction{
					Command: []string{"/bin/bash", "-c", "echo helloReadiness1 > /home/helloReadiness1"},
				},
			},
			TimeoutSeconds: 2,
			PeriodSeconds:  5,
		}
		pod.Spec.Containers[0].ReadinessProbe = &readinessProbe
		testCase := updateProberTestCase{
			pod:              pod,
			containerName:    pod.Spec.Containers[0].Name,
			probeType:        typeReadinessProbe,
			patchData:        `{"spec":{"containers":[{"name":"pod-base","readinessProbe":{"exec":{"command":["/bin/bash", "-c", "echo helloReadiness2 > /home/helloReadiness2"]}}}]}}`,
			patchType:        types.StrategicMergePatchType,
			checkCommandPre:  "cat /home/helloReadiness1",
			resultKeywordPre: []string{"helloReadiness1"},
			checkMethodPre:   checkMethodEqual,
			shouldCheckPre:   true,
			checkCommandPro:  "cat /home/helloReadiness2",
			resultKeywordPro: []string{"helloReadiness2"},
			checkMethodPro:   checkMethodEqual,
			shouldCheckPro:   true,
			waitSec:          7,
		}
		doUpdateProberTestCase(f, &testCase)
	})
	It("add readinessProbe", func() {
		pod := generateRunningPod()
		testCase := updateProberTestCase{
			pod:              pod,
			containerName:    pod.Spec.Containers[0].Name,
			probeType:        typeReadinessProbe,
			patchData:        `{"spec":{"containers":[{"name":"pod-base","readinessProbe":{"exec":{"command":["/bin/bash", "-c", "echo helloReadiness1 > /home/helloReadiness1"]}}}]}}`,
			patchType:        types.StrategicMergePatchType,
			checkCommandPre:  "",
			resultKeywordPre: []string{},
			checkMethodPre:   "",
			shouldCheckPre:   false,
			checkCommandPro:  "cat /home/helloReadiness1",
			resultKeywordPro: []string{"helloReadiness1"},
			checkMethodPro:   checkMethodEqual,
			shouldCheckPro:   true,
			waitSec:          7,
		}
		doUpdateProberTestCase(f, &testCase)
	})
	It("delete readinessProbe", func() {
		pod := generateRunningPod()
		readinessProbe := v1.Probe{
			Handler: v1.Handler{
				Exec: &v1.ExecAction{
					Command: []string{"/bin/bash", "-c", "echo helloReadiness1 > /home/helloReadiness1"},
				},
			},
			TimeoutSeconds: 2,
			PeriodSeconds:  5,
		}
		pod.Spec.Containers[0].ReadinessProbe = &readinessProbe
		testCase := updateProberTestCase{
			pod:              pod,
			containerName:    pod.Spec.Containers[0].Name,
			probeType:        typeReadinessProbe,
			patchData:        `[{"op":"remove","path":"/spec/containers/0/readinessProbe","value":""}]`,
			patchType:        types.JSONPatchType,
			checkCommandPre:  "cat /home/helloReadiness1",
			resultKeywordPre: []string{"helloReadiness1"},
			checkMethodPre:   checkMethodEqual,
			shouldCheckPre:   true,
			checkCommandPro:  "echo '' > /home/helloReadiness1 && cat /home/helloReadiness1",
			resultKeywordPro: []string{"helloReadiness1"},
			checkMethodPro:   checkMethodNotEqual,
			shouldCheckPro:   true,
			waitSec:          7,
		}
		doUpdateProberTestCase(f, &testCase)
	})
	It("[smoke] update readinessProbe in multiContainer pod", func() {
		pod := generateMultiConRunningPod()
		readinessProbe := v1.Probe{
			Handler: v1.Handler{
				Exec: &v1.ExecAction{
					Command: []string{"/bin/bash", "-c", "echo helloReadiness1 > /home/helloReadiness1"},
				},
			},
			TimeoutSeconds: 2,
			PeriodSeconds:  5,
		}
		pod.Spec.Containers[1].ReadinessProbe = &readinessProbe
		testCase := updateProberTestCase{
			pod:              pod,
			containerName:    pod.Spec.Containers[1].Name,
			probeType:        typeReadinessProbe,
			patchData:        `{"spec":{"containers":[{"name":"pod-base1","readinessProbe":{"exec":{"command":["/bin/bash", "-c", "echo helloReadiness2 > /home/helloReadiness2"]}}}]}}`,
			patchType:        types.StrategicMergePatchType,
			checkCommandPre:  "cat /home/helloReadiness1",
			resultKeywordPre: []string{"helloReadiness1"},
			checkMethodPre:   checkMethodEqual,
			shouldCheckPre:   true,
			checkCommandPro:  "cat /home/helloReadiness2",
			resultKeywordPro: []string{"helloReadiness2"},
			checkMethodPro:   checkMethodEqual,
			shouldCheckPro:   true,
			waitSec:          7,
		}
		doUpdateProberTestCase(f, &testCase)
	})
})
