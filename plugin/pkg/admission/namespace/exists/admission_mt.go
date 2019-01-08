package exists

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

// SetInternalKubeClientSet implements the WantsInternalKubeClientSet interface.
func (e *Exists) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *e
	copied.client = e.client.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(internalclientset.Interface)
	copied.namespaceLister = e.namespaceLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.NamespaceLister)
	return &copied
}
