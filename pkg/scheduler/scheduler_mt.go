package scheduler

import (
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/scheduler/volumebinder"
)

func (sched *Scheduler) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("bind scheduleOne with tenant info %#v", tenant)
	copyConfig := *sched.config
	copyConfig.NodeLister = copyConfig.NodeLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(algorithm.NodeLister)
	copyConfig.GetBinder = func(pod *v1.Pod) Binder {
		return sched.config.GetBinder(pod).(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(Binder)
	}
	copyConfig.PodConditionUpdater = copyConfig.PodConditionUpdater.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(PodConditionUpdater)
	copyConfig.PodPreemptor = copyConfig.PodPreemptor.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(PodPreemptor)
	copyConfig.VolumeBinder = copyConfig.VolumeBinder.ShallowCopyWithTenant(tenant).(*volumebinder.VolumeBinder)
	return &Scheduler{
		config: &copyConfig,
	}
}