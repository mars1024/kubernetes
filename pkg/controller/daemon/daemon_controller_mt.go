package daemon

import (
	multitenancy "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	meta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	kubernetes "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/apps/v1"
	corev1 "k8s.io/client-go/listers/core/v1"
	controller "k8s.io/kubernetes/pkg/controller"
)

func (c *DaemonSetsController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.kubeClient = c.kubeClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.podControl = c.podControl.(meta.TenantWise).ShallowCopyWithTenant(tenant).(controller.PodControlInterface)
	copied.crControl = c.crControl.(meta.TenantWise).ShallowCopyWithTenant(tenant).(controller.ControllerRevisionControlInterface)
	copied.dsLister = c.dsLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.DaemonSetLister)
	copied.historyLister = c.historyLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.ControllerRevisionLister)
	copied.podLister = c.podLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(corev1.PodLister)
	copied.nodeLister = c.nodeLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(corev1.NodeLister)
	return &copied
}
