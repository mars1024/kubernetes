package violatecheck

import (
	"k8s.io/api/core/v1"
	statsapi "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/autopilot/slo"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch"
	"k8s.io/kubernetes/pkg/kubelet/server/stats"
)

// DefaultViolateDataProvider the default violate data impl.
type DefaultViolateDataProvider struct {
	//support input data to construct runtime checking need data.
	SketchClient sketch.Provider
	StatsClient  stats.StatsProvider
}

//GetContainerRuntimeSLOValue return container latest cpi value.
func (d *DefaultViolateDataProvider) GetContainerRuntimeSLOValue(sketchType slo.ContainerSLOType, podnamespace string, podname string, containerName string) (float32, error) {
	//TODO dependency on sketch.Provider's interface and implement. or access xperf directly.
	return 0, nil
}

//GetPodByName return pod under podnamespace and podname.
func (d *DefaultViolateDataProvider) GetPodByName(podnamespace string, podname string) (*v1.Pod, bool) {
	return d.StatsClient.GetPodByName(podnamespace, podname)
}

// ListPodStats returns the stats of all the containers managed by pods.
func (d *DefaultViolateDataProvider) ListPodStats() ([]statsapi.PodStats, error) {
	return d.StatsClient.ListPodStats()
}
