package statefulset

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"

	"k8s.io/kubernetes/pkg/controller/history"
)

// manually generated
func (c *defaultStatefulSetControl) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.podControl = c.podControl.(meta.TenantWise).ShallowCopyWithTenant(tenant).(StatefulPodControlInterface)
	copied.controllerHistory = c.controllerHistory.(meta.TenantWise).ShallowCopyWithTenant(tenant).(history.Interface)
	copied.statusUpdater = c.statusUpdater.(meta.TenantWise).ShallowCopyWithTenant(tenant).(StatefulSetStatusUpdaterInterface)
	return &copied
}
