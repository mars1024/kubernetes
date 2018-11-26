package throttle

import (
	cadvisorapi "github.com/google/cadvisor/info/v1"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	"k8s.io/api/core/v1"
	statsapi "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/autopilot/slo"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
	"k8s.io/kubernetes/pkg/kubelet/server/stats"
)

// DefaultThrottleInputData the default implement of ThrottleInputData. Dependency on some providers.
type DefaultThrottleInputData struct {
	CadvisorClient cadvisor.Interface
	SketchClient   sketch.Provider
	StatsClient    stats.StatsProvider
}

var _ InputData = new(DefaultThrottleInputData)

// ContainerInfo returns the containerInfo with the specified name and request.
func (input *DefaultThrottleInputData) ContainerInfo(name string, req *cadvisorapi.ContainerInfoRequest) (*cadvisorapi.ContainerInfo, error) {
	return input.ContainerInfo(name, req)
}

// ContainerInfoV2 returns the containerInfoV2 with the specified name and options.
func (input *DefaultThrottleInputData) ContainerInfoV2(name string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerInfo, error) {
	return input.CadvisorClient.ContainerInfoV2(name, options)
}

// ContainerSpec Returns container spec.
func (input *DefaultThrottleInputData) ContainerSpec(containerName string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerSpec, error) {
	return input.CadvisorClient.ContainerSpec(containerName, options)
}

// ListPodStats returns the stats of all the containers managed by pods.
func (input *DefaultThrottleInputData) ListPodStats() ([]statsapi.PodStats, error) {
	return input.StatsClient.ListPodStats()
}

// GetPodByName returns the spec of the pod with the name in the specified
// namespace.
func (input *DefaultThrottleInputData) GetPodByName(namespace, name string) (*v1.Pod, bool) {
	return input.StatsClient.GetPodByName(namespace, name)
}

// Start impl .
func (input *DefaultThrottleInputData) Start() error {
	return input.SketchClient.Start()
}

// Stop impl.
func (input *DefaultThrottleInputData) Stop() {
	// TODO ...
}

// GetSketch .
func (input *DefaultThrottleInputData) GetSketch() {
	// TODO ...
}

// GetContainerRuntimeSLOValue .
func (input *DefaultThrottleInputData) GetContainerRuntimeSLOValue(sketchType slo.ContainerSLOType, podnamespace string, podname string, containerName string) (float32, error) {
	// TODO ...
	return 0, nil
}
