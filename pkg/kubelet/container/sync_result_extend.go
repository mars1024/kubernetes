package container

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// ContainerStatusClean  clean container state status which not exist
func (p *PodSyncResult) ContainerStateClean(pod *v1.Pod) {
	podNames := sets.String{}
	for _, value := range pod.Spec.Containers {
		podNames.Insert(value.Name)
	}
	for containerInfo := range p.StateStatus.Statuses {
		found := podNames.Has(containerInfo.Name)
		if !found {
			delete(p.StateStatus.Statuses, containerInfo)
		}
	}
}

// UpdateStateToPodAnnotation update state to pod annotation.
func (p *PodSyncResult) UpdateStateToPodAnnotation(pod *v1.Pod) {
	if len(p.StateStatus.Statuses) == 0 {
		return
	}
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string, 1)
	}
	stateStatusJSON, err := json.Marshal(p.StateStatus)
	if err != nil {
		msg := fmt.Sprintf("json marshal: %v err, error is %s", p.StateStatus, err.Error())
		glog.Error(msg)
		pod.Annotations[sigmak8sapi.AnnotationPodUpdateStatus] = msg
	} else {
		pod.Annotations[sigmak8sapi.AnnotationPodUpdateStatus] = string(stateStatusJSON)
	}
}
