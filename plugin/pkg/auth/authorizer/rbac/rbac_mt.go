// +build multitenancy

// Package rbac implements the authorizer.Authorizer interface using roles base access control.
package rbac

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	rbacv1listers "k8s.io/client-go/listers/rbac/v1"
)

func (g *RoleGetter) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return &RoleGetter{
		Lister: g.Lister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(rbacv1listers.RoleLister),
	}
}

func (l *RoleBindingLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return &RoleBindingLister{
		Lister: l.Lister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(rbacv1listers.RoleBindingLister),
	}
}

func (g *ClusterRoleGetter) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return &ClusterRoleGetter{
		Lister: g.Lister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(rbacv1listers.ClusterRoleLister),
	}
}

func (l *ClusterRoleBindingLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return &ClusterRoleBindingLister{
		Lister: l.Lister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(rbacv1listers.ClusterRoleBindingLister),
	}
}
