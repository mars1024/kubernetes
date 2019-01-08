package exec

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

func (d *DenyExec) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *d
	copied.client = d.client.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(internalclientset.Interface)
	return &copied
}
