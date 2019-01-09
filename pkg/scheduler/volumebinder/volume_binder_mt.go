package volumebinder

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"

	"k8s.io/kubernetes/pkg/controller/volume/persistentvolume"
)

func (b *VolumeBinder) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	tenantVolumeBinder := *b
	tenantBinder := b.Binder.(multitenancymeta.TenantWise)
	tenantVolumeBinder.Binder = tenantBinder.ShallowCopyWithTenant(tenant).(persistentvolume.SchedulerVolumeBinder)
	return &tenantVolumeBinder
}