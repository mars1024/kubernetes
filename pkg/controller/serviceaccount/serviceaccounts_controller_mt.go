package serviceaccount

import (
	multitenancy "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	meta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	kubernetes "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
)

func (c *ServiceAccountsController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.client = c.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.saLister = c.saLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.ServiceAccountLister)
	copied.nsLister = c.nsLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.NamespaceLister)
	return &copied
}

func (c *TokensController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.client = c.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.serviceAccounts = c.serviceAccounts.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.ServiceAccountLister)
	return &copied
}
