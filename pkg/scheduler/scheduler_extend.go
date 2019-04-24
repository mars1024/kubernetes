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
