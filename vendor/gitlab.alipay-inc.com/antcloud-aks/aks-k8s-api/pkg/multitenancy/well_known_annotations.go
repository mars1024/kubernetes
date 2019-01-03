package multitenancy

// Tenancy info will be injected into every resource object's annotations via the following
// keys, and *these annotations shall never be modified*, neither normal tenant nor global admin.
const (
	// Deprecated
	MultiTenancyAnnotationKeyTenantID = "alpha.cloud.alipay.com/tenant-id"
	// Deprecated
	MultiTenancyAnnotationKeyWorkspaceID = "alpha.cloud.alipay.com/workspace-id"
	// Deprecated
	MultiTenancyAnnotationKeyClusterID = "alpha.cloud.alipay.com/cluster-id"

	AnnotationServiceOwner = "service.beta.cloud.alipay.com/owner"
)

const (
	AnnotationCafeMinionClusterID  = "aks.cafe.sofastack.io/mc"
	AnnotationCafeIngoreServiceNet = "aks.cafe.sofastack.io/ignore-service-net"
	AnnotationCafeAKSPopulated     = "aks.cafe.sofastack.io/aks-populated"
)

// We inject tenant info into user's "extra" (which is a map). The following are the keys defining
// how we transform the tenant info into a user object and how extract tenancy from user.
const (
	UserExtraInfoTenantID    = "antcloud-aks-tenant-id"
	UserExtraInfoWorkspaceID = "antcloud-aks-workspace-id"
	UserExtraInfoClusterID   = "antcloud-aks-cluster-id"
)

const (
	X509CertificateTenantIDPrefix    = "multitenancy:tenant:"
	X509CertificateWorkspaceIDPrefix = "multitenancy:workspace:"
	X509CertificateClusterIDPrefix   = "multitenancy:cluster:"
)
