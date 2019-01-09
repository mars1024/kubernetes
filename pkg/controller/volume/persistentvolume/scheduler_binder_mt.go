package persistentvolume

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
)

func (b *volumeBinder) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	tenantVolumeBinder := *b
	tenantPvcCache := b.pvcCache.(multitenancymeta.TenantWise)
	tenantVolumeBinder.pvcCache = tenantPvcCache.ShallowCopyWithTenant(tenant).(PVCAssumeCache)
	tenantPvCache := b.pvCache.(multitenancymeta.TenantWise)
	tenantVolumeBinder.pvCache = tenantPvCache.ShallowCopyWithTenant(tenant).(PVAssumeCache)
	return &tenantVolumeBinder
}