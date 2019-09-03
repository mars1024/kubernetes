package garbagecollector

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/client-go/dynamic"
)

func (gc *GarbageCollector) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *gc
	copied.dynamicClient = gc.dynamicClient.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(dynamic.Interface)
	return &copied
}
