package multitenancy

// We identify a tenant by x509 pki infrastructure. the "Organization" segments with the following
// prefixes will be filtered into user's extra info, which is to say, all these prefixes shall not
// present in the handler logic or server runtime.
const (
	UserGroupMultiTenancyPrefix = "multitenancy:"

	UserGroupTenantIDPrefix    = UserGroupMultiTenancyPrefix + "tenant:"
	UserGroupWorkspaceIDPrefix = UserGroupMultiTenancyPrefix + "workspace:"
	UserGroupClusterIDPrefix   = UserGroupMultiTenancyPrefix + "cluster:"
)

const (
	UserGroupMultiTenancyImpersonated = "multitenancy:impersonated"
)

const (
	GlobalAdminTenantNamePrefix = "admin::"

	//Deprecated
	GlobalAdminTenantTenantID = "admin"
	//Deprecated
	GlobalAdminTenantWorkspaceID = "default"
	//Deprecated
	GlobalAdminTenantClusterID = "default"
)

var (
	// GlobalAdminTenant is a in-memory tenant who has the privilege to operate all tenants.
	// Deprecated
	GlobalAdminTenant TenantInfo = &defaultTenant{
		tenantID:    GlobalAdminTenantTenantID,
		workspaceID: GlobalAdminTenantWorkspaceID,
		clusterID:   GlobalAdminTenantClusterID,
	}

	AKSAdminTenant TenantInfo = &defaultTenant{
		tenantID:    "admin::aks",
		workspaceID: "default",
		clusterID:   "default",
	}
)

const (
	UserSystemAlipayAdmin = "system:admin"

	// Compatibility with aks-tooling
	UserKubeAPIServer = "kubeapiserver"
)
