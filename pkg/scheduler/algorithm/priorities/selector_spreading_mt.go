package priorities

import (
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"

	"github.com/golang/glog"
)

func (s *SelectorSpread) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("priority selector spread with tenant %#v", tenant)
	tenantSelectorSpread := *s
	tenantServiceLister := tenantSelectorSpread.serviceLister.(multitenancymeta.TenantWise)
	tenantSelectorSpread.serviceLister = tenantServiceLister.ShallowCopyWithTenant(tenant).(algorithm.ServiceLister)
	tenantStatefulSetLister := tenantSelectorSpread.statefulSetLister.(multitenancymeta.TenantWise)
	tenantSelectorSpread.statefulSetLister = tenantStatefulSetLister.ShallowCopyWithTenant(tenant).(algorithm.StatefulSetLister)
	tenantReplicaSetLister := tenantSelectorSpread.replicaSetLister.(multitenancymeta.TenantWise)
	tenantSelectorSpread.replicaSetLister = tenantReplicaSetLister.ShallowCopyWithTenant(tenant).(algorithm.ReplicaSetLister)
	tenantControllerLister := tenantSelectorSpread.controllerLister.(multitenancymeta.TenantWise)
	tenantSelectorSpread.controllerLister = tenantControllerLister.ShallowCopyWithTenant(tenant).(algorithm.ControllerLister)
	return &tenantSelectorSpread
}

func (s *ServiceAntiAffinity) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("priority service anti affinity with tenant %#v", tenant)
	tenantServiceAntiAffinity := *s
	tenantPodLister := tenantServiceAntiAffinity.podLister.(multitenancymeta.TenantWise)
	tenantServiceAntiAffinity.podLister = tenantPodLister.ShallowCopyWithTenant(tenant).(algorithm.PodLister)
	tenantServiceLister := tenantServiceAntiAffinity.serviceLister.(multitenancymeta.TenantWise)
	tenantServiceAntiAffinity.serviceLister = tenantServiceLister.ShallowCopyWithTenant(tenant).(algorithm.ServiceLister)
	return &tenantServiceAntiAffinity
}