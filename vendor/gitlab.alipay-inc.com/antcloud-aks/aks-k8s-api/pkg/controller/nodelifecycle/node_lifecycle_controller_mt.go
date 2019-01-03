package nodelifecycle

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/client-go/kubernetes"
	coordinationv1beta1 "k8s.io/client-go/listers/coordination/v1beta1"
	"k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/listers/extensions/v1beta1"
)

func (c *Controller) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.kubeClient = c.kubeClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.daemonSetStore = c.daemonSetStore.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1beta1.DaemonSetLister)
	copied.leaseLister = c.leaseLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(coordinationv1beta1.LeaseLister)
	copied.nodeLister = c.nodeLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.NodeLister)
	return &copied
}
