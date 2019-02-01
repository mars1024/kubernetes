package namespace

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/controller/namespace/deletion"
)

func (c *NamespaceController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.lister = c.lister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.NamespaceLister)

	// manually generated
	copied.namespacedResourcesDeleter = c.namespacedResourcesDeleter.(meta.TenantWise).ShallowCopyWithTenant(tenant).(deletion.NamespacedResourcesDeleterInterface)
	return &copied
}
