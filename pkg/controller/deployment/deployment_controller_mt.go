package deployment

import (
	multitenancy "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	meta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	kubernetes "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/apps/v1"
	corev1 "k8s.io/client-go/listers/core/v1"
	controller "k8s.io/kubernetes/pkg/controller"
)

func (c *DeploymentController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.rsControl = c.rsControl.(meta.TenantWise).ShallowCopyWithTenant(tenant).(controller.RSControlInterface)
	copied.client = c.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.dLister = c.dLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.DeploymentLister)
	copied.rsLister = c.rsLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.ReplicaSetLister)
	copied.podLister = c.podLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(corev1.PodLister)
	return &copied
}
