package noderestriction

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	corev1lister "k8s.io/client-go/listers/core/v1"
)

func (p *nodePlugin) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *p
	copied.podsGetter = p.podsGetter.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corev1lister.PodLister)
	return &copied
}
