package cleaner

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	csrclient "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	certificateslisters "k8s.io/client-go/listers/certificates/v1beta1"
)

func (c *CSRCleanerController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.csrClient = c.csrClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(csrclient.CertificateSigningRequestInterface)
	copied.csrLister = c.csrLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(certificateslisters.CertificateSigningRequestLister)
	return &copied
}
