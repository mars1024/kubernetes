package multitenancy

// Tenant is an interface for accessing antcloud-aks multitenancy meta information
type TenantInfo interface {
	// GetTenantID returns tenant id
	GetTenantID() string
	// GetWorkspaceID returns workspace id
	GetWorkspaceID() string
	// GetClusterID returns cluster id
	GetClusterID() string
}


type defaultTenant struct {
	tenantID    string
	workspaceID string
	clusterID   string
}

var _ TenantInfo = &defaultTenant{}

func (t *defaultTenant) GetTenantID() string {
	return t.tenantID
}

func (t *defaultTenant) GetWorkspaceID() string {
	return t.workspaceID
}

func (t *defaultTenant) GetClusterID() string {
	return t.clusterID
}

func NewTenantInfo(tenantID, workspaceID, clusterID string) TenantInfo {
	return &defaultTenant{tenantID, workspaceID, clusterID}
}
