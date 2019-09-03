package validation

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
)

func (r *DefaultRuleResolver) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *r
	copied.roleBindingLister = r.roleBindingLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(RoleBindingLister)
	copied.roleGetter = r.roleGetter.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(RoleGetter)
	copied.clusterRoleGetter = r.clusterRoleGetter.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(ClusterRoleGetter)
	copied.clusterRoleBindingLister = r.clusterRoleBindingLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(ClusterRoleBindingLister)
	return &copied
}
