package replicaset

import (
	multitenancy "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	meta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	kubernetes "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/apps/v1"
	corev1 "k8s.io/client-go/listers/core/v1"
	controller "k8s.io/kubernetes/pkg/controller"
)

func (c *ReplicaSetController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.kubeClient = c.kubeClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.podControl = c.podControl.(meta.TenantWise).ShallowCopyWithTenant(tenant).(controller.PodControlInterface)
	copied.rsLister = c.rsLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.ReplicaSetLister)
	copied.podLister = c.podLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(corev1.PodLister)
	return &copied
}
