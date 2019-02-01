package statefulset

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/listers/apps/v1"
)

// manually generated
func (c *realStatefulSetStatusUpdater) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.client = c.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.setLister = c.setLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.StatefulSetLister)
	return &copied
}
