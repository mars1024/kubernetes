package clusterroleaggregation

import (
	multitenancy "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	meta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	rbacclient "k8s.io/client-go/kubernetes/typed/rbac/v1"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
)

func (c *ClusterRoleAggregationController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.clusterRoleClient = c.clusterRoleClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(rbacclient.ClusterRolesGetter)
	copied.clusterRoleLister = c.clusterRoleLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(rbaclisters.ClusterRoleLister)
	return &copied
}
