package util

import "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"

type TenantHash struct {
	TenantID    string
	WorkspaceID string
	ClusterID   string
}

func GetHashFromTenant(tenant multitenancy.TenantInfo) TenantHash {
	return TenantHash{
		TenantID:    tenant.GetTenantID(),
		WorkspaceID: tenant.GetWorkspaceID(),
		ClusterID:   tenant.GetClusterID(),
	}
}

func (th TenantHash) TenantInfo() multitenancy.TenantInfo {
	return multitenancy.NewTenantInfo(th.TenantID, th.WorkspaceID, th.ClusterID)
}
