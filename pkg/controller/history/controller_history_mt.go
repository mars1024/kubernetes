package history

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/listers/apps/v1"
)

func (c *realHistory) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copy := *c
	copy.client = c.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copy.lister = c.lister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.ControllerRevisionLister)
	return &copy
}
