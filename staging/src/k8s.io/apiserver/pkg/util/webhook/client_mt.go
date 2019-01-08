package webhook

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
)

func (cm *ClientManager) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *cm
	copied.serviceResolver = cm.serviceResolver.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(ServiceResolver)
	return &copied
}
