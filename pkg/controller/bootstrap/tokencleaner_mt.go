package bootstrap

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"

	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/listers/core/v1"
)

func (tc *TokenCleaner) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *tc
	copied.client = tc.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.secretLister = tc.secretLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(corev1.SecretLister)
	return &copied
}
