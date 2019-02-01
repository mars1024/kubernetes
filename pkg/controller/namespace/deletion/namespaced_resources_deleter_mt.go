package deletion

import (
	multitenancy "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	v1clientset "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/dynamic"
)

// manually generated
func (c *namespacedResourcesDeleter) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.nsClient = c.nsClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1clientset.NamespaceInterface)
	copied.dynamicClient = c.dynamicClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(dynamic.Interface)
	copied.podsGetter = c.podsGetter.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1clientset.PodsGetter)
	return &copied
}
