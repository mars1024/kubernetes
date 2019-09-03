package metrics

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	corelisters "k8s.io/client-go/listers/core/v1"
)

func (collector *attachDetachStateCollector) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	cloned := *collector
	cloned.pvcLister = collector.pvcLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.PersistentVolumeClaimLister)
	cloned.pvLister = collector.pvLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.PersistentVolumeLister)
	cloned.podLister = collector.podLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.PodLister)
	return &cloned
}
