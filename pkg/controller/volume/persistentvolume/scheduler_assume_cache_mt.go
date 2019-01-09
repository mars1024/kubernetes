package persistentvolume

import "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"

func (c *pvcAssumeCache) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return c
}

func (c *pvAssumeCache) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return c
}
