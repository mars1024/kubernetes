package violatecheck

import (
	"k8s.io/api/core/v1"
	statsapi "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/autopilot/slo"
)

const (
	//ViolateKey kepkg/kubelet/autonomy/autopilot/violatecheck/violate_framework.go:23y define in pod annotation.
	ViolateKey = "/violate"
	//AnnotationRT key define in pod annotation.
	AnnotationRT = "security.rt"
	//AnnotationQPS key define in pod annotation.
	AnnotationQPS = "security.qps"
	//AnnotationCPI key define in pod annotation.
	AnnotationCPI = "security.cpi"
)

// ViolateConfig for control the violate do or not.
type ViolateConfig struct {
	SwitchOn bool `json:"switchOn"`
}

// ViolateNeedData define the method to get violate target data.
type ViolateNeedData interface {
	GetPodByName(podnamespace string, podname string) (*v1.Pod, bool)

	// ListPodStats returns the stats of all the containers managed by pods.
	ListPodStats() ([]statsapi.PodStats, error)

	//return predicate, history ,five or fifteen values.
	slo.RuntimeSLOPredictValue
}

// CheckManager define the violate check logical.pkg/kubelet/autonomy/autopilot/violatecheck/violate_framework.go:23
type CheckManager interface {

	//ViolateCheck the core check part.
	ViolateCheck(annotations map[string]string) ([]string, error)

	//ContainerViolateStats return container violate state true or false,and which item violate.
	ContainerViolateState(cpi float32, rt float32, qps float32, pod *v1.Pod) ([]slo.ContainerSLOType, error)
}
