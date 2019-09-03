package autoprovision

import (
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
)

func (p *Provision) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *p
	copied.client = p.client.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(internalclientset.Interface)
	copied.namespaceLister = p.namespaceLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.NamespaceLister)
	return &copied
}
