package throttle

import (
	"time"

	cadvisorapi "github.com/google/cadvisor/info/v1"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	"k8s.io/api/core/v1"
	statsapi "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/autopilot/slo"
)

// InputData composition of two interface.
type InputData interface {
	//ContainerInfo returns container info fit with name and req.
	ContainerInfo(name string, req *cadvisorapi.ContainerInfoRequest) (*cadvisorapi.ContainerInfo, error)

	//ContainerInfoV2 returns containers fit with name and options.
	ContainerInfoV2(name string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerInfo, error)

	//ContainerSpec Returns container spec.
	ContainerSpec(containerName string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerSpec, error)

	//ListPodStats returns the stats of all the containers managed by pods.
	ListPodStats() ([]statsapi.PodStats, error)

	// GetPodByName returns the spec of the pod with the name in the specified namespace.
	GetPodByName(namespace, name string) (*v1.Pod, bool)

	slo.RuntimeSLOPredictValue
}

// ParamRef as param for priority.
type ParamRef struct {
	Name          string
	SpecValues    interface{}
	CurrentValues interface{}
	HistoryValues interface{}
	PredictValues interface{}
}

// ContainerThrottlePriority select which container to be throttled or recovered.
type ContainerThrottlePriority interface {
	SelectCouldThrottleContainer(inputData InputData, containerUIDs []*ParamRef) (*ParamRef, error)
}

// Manager define throttle methods.
type Manager interface {
	ThrottleStatus() bool

	// init and run the throttle pix-period check job flow,each period-check worker do or not, that is controlled by flag.
	Start(executionIntervalSeconds time.Duration)

	// flag controller : start check throttle work at next period time.
	Enable()

	// flag controller : stop check throttle work at next period time.
	Disable()

	SelectThrottledContainerToRecover(inputData InputData, priority ContainerThrottlePriority) (string, error)

	SelectTheContainerToThrottle(inputData InputData, priority ContainerThrottlePriority) (string, int64, error)
}

// TODO the full framework for other throttle use case.
