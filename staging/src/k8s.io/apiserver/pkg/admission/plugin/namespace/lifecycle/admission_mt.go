package lifecycle

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/kubernetes"
)

func (l *Lifecycle) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *l
	copied.client = l.client.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.namespaceLister = l.namespaceLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.NamespaceLister)
	return &copied
}
