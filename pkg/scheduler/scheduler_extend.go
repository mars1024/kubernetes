package scheduler

import (
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/core"
)

func (sched *Scheduler) PatchAllocators(pod *v1.Pod, suggestedHost string) error {
	var err error
	if algo, ok := sched.config.Algorithm.(*core.GenericSchedulerExtend); ok {
		err = algo.Allocate(pod, suggestedHost)
		if err != nil {
			glog.Error(err)
			return err
		}
	}
	return err
}

func (sched *Scheduler) setPodAllocSpec(id, allocSpec string) {
	sched.inplacelock.Lock()
	defer sched.inplacelock.Unlock()

	if sched.inplacePodAllocSpec == nil {
		sched.inplacePodAllocSpec = map[string]string{}
	}
	sched.inplacePodAllocSpec[id] = allocSpec
}

func (sched *Scheduler) clearPodAllocSpec(id string) {
	sched.inplacelock.Lock()
	defer sched.inplacelock.Unlock()

	if sched.inplacePodAllocSpec == nil {
		return
	}
	delete(sched.inplacePodAllocSpec, id)
}

func (sched *Scheduler) getPodAllocSpec(id string) string {
	sched.inplacelock.Lock()
	defer sched.inplacelock.Unlock()

	if sched.inplacePodAllocSpec == nil {
		return ""
	}
	return sched.inplacePodAllocSpec[id]
}
