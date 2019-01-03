package scale

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/client-go/rest"
)

func (c *scaleClient) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.clientBase = c.clientBase.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(rest.Interface)
	return &copied
}
