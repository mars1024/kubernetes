package serviceaccount

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/listers/core/v1"
)

func (s *serviceAccount) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *s
	copied.client = s.client.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.serviceAccountLister = s.serviceAccountLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(v1.ServiceAccountLister)
	copied.secretLister = s.secretLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(v1.SecretLister)
	return &copied
}
