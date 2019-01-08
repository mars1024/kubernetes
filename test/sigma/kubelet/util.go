package kubelet

import (
	"path/filepath"
	"strconv"
	"strings"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

var checkMethodContain string = "contain"
var checkMethodNotContain string = "not contain"
var checkMethodEqual string = "equal"
var checkMethodNotEqual string = "not equal"

// checkResult check result with keywords according to checkMethod.
func checkResult(checkMethod string, result string, keywords []string) {
	switch checkMethod {
	case checkMethodEqual:
		pass := false
		for _, keyword := range keywords {
			if result == keyword {
				pass = true
				break
			}
		}
		if !pass {
			framework.Failf("result %s doesn't equal any keyword: %s", result, keywords)
		}
	case checkMethodNotEqual:
		for _, keyword := range keywords {
			if result == keyword {
				framework.Failf("result %s shouldn't equal keyword: %s", result, keyword)
			}
		}
	case checkMethodContain:
		for _, keyword := range keywords {
			if !strings.Contains(result, keyword) {
				framework.Failf("result %s doesn't contain keyword: %s", result, keyword)
			}
		}
	case checkMethodNotContain:
		for _, keyword := range keywords {
			if strings.Contains(result, keyword) {
				framework.Failf("result %s shouldn't contain keyword: %s", result, keyword)
			}
		}
	default:
		framework.Failf("Unknown check method type")
	}
}

func generatePodCommon() *v1.Pod {
	podFileName := "pod-base.json"
	podFile := filepath.Join(util.TestDataDir, podFileName)
	pod, err := util.LoadPodFromFile(podFile)
	if err != nil {
		framework.Failf("Failed to load pod from file")
	}
	// name should be unique
	pod.Name = "pod-test-" + string(uuid.NewUUID())
	return pod
}

func generateRunningPod() *v1.Pod {
	pod := generatePodCommon()
	pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"
	pod.Annotations = map[string]string{}
	pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = `{"states":{"pod-base":"running"}}`
	return pod
}

func generateMultiConRunningPod() *v1.Pod {
	pod := generatePodCommon()
	pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"
	container1 := pod.Spec.Containers[0]
	container1.Name = "pod-base1"
	pod.Spec.Containers = append(pod.Spec.Containers, container1)
	container2 := pod.Spec.Containers[0]
	container2.Name = "pod-base2"
	pod.Spec.Containers = append(pod.Spec.Containers, container2)
	pod.Annotations = map[string]string{}
	pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = `{"states":{"pod-base":"running", "pod-base1":"running","pod-base2":"running"}}`
	return pod
}

func generateExitedPod() *v1.Pod {
	pod := generatePodCommon()
	pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"
	pod.Annotations = map[string]string{}
	pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = `{"states":{"pod-base":"exited"}}`
	return pod
}

func generateRunningPodWithEnv(envs map[string]string) *v1.Pod {
	pod := generatePodCommon()
	pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"
	pod.Annotations = map[string]string{}
	pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = `{"states":{"pod-base":"running"}}`
	// set extra information to pod.
	targetENVs := []v1.EnvVar{}
	for k, v := range envs {
		env := v1.EnvVar{
			Name:  k,
			Value: v,
		}
		targetENVs = append(targetENVs, env)
	}
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, targetENVs...)

	return pod
}

func generateRunningPodWithSpecHash(specHash string) *v1.Pod {
	pod := generatePodCommon()
	pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"
	pod.Annotations = map[string]string{}
	pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = `{"states":{"pod-base":"running"}}`
	pod.Annotations[sigmak8sapi.AnnotationPodSpecHash] = specHash
	return pod
}

func generateRunningPodWithWorkingDir(workingDir string) *v1.Pod {
	pod := generatePodCommon()
	pod.Spec.Containers[0].WorkingDir = workingDir
	pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"
	pod.Annotations = map[string]string{}
	pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = `{"states":{"pod-base":"running"}}`
	return pod
}

func generateRunningPodWithCmdArgs(command []string, args []string) *v1.Pod {
	pod := generatePodCommon()
	pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"
	pod.Spec.Containers[0].Command = command
	pod.Spec.Containers[0].Args = args
	pod.Annotations = map[string]string{}
	pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = `{"states":{"pod-base":"running"}}`
	return pod
}

func generateRunningPodWithPostStartHook(command []string) *v1.Pod {
	pod := generatePodCommon()
	pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/sigma-x/mysql:test-v1"
	lifecycle := &v1.Lifecycle{
		PostStart: &v1.Handler{
			Exec: &v1.ExecAction{command},
		},
	}
	pod.Spec.Containers[0].Lifecycle = lifecycle
	pod.Annotations = map[string]string{}
	pod.Annotations[sigmak8sapi.AnnotationContainerStateSpec] = `{"states":{"pod-base":"running"}}`
	return pod
}

func getCPUPeriod(pod *v1.Pod) int64 {
	if len(pod.Status.ContainerStatuses) == 0 {
		framework.Logf("Failed to get ContainerStatuses from pod: %s", pod.Name)
		return 0
	}
	segs := strings.Split(pod.Status.ContainerStatuses[0].ContainerID, "//")
	if len(segs) != 2 {
		framework.Logf("Failed to get ContainerID from pod: %s", pod.Name)
		return 0
	}
	containerID := segs[1]

	// Get CPUPeriod.
	format := "{{.HostConfig.CPUPeriod}}"
	cpuPeriodStr, err := util.GetContainerInspectField(pod.Status.HostIP, containerID, format)
	if err != nil {
		framework.Logf("Failed to get cpu period from pod: %s", pod.Name)
		return 0
	}
	cpuPeriod, err := strconv.ParseInt(strings.TrimSuffix(cpuPeriodStr, "\n"), 10, 64)
	if err != nil {
		framework.Logf("Failed to parse cpu period %s from pod: %s", cpuPeriodStr, pod.Name)
		return 0
	}
	return cpuPeriod
}

func getResourceRequirements(requests, limits v1.ResourceList) v1.ResourceRequirements {
	res := v1.ResourceRequirements{}
	res.Requests = requests
	res.Limits = limits
	return res
}

func getResourceList(cpu, memory string) v1.ResourceList {
	res := v1.ResourceList{}
	if cpu != "" {
		res[v1.ResourceCPU] = resource.MustParse(cpu)
	}
	if memory != "" {
		res[v1.ResourceMemory] = resource.MustParse(memory)
	}
	return res
}

func generateRunningPodWithInitResource(resources v1.ResourceRequirements) *v1.Pod {
	pod := generatePodCommon()
	pod.Spec.Containers[0].Image = "reg.docker.alibaba-inc.com/ali/os:7u2"
	pod.Spec.Containers[0].Resources = resources
	return pod
}
