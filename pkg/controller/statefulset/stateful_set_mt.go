package statefulset

import (
	multitenancy "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	meta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	kubernetes "k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/listers/apps/v1"
	v1 "k8s.io/client-go/listers/core/v1"
	controller "k8s.io/kubernetes/pkg/controller"
)

func (c *StatefulSetController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.kubeClient = c.kubeClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.podControl = c.podControl.(meta.TenantWise).ShallowCopyWithTenant(tenant).(controller.PodControlInterface)
	copied.podLister = c.podLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.PodLister)
	copied.setLister = c.setLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(appsv1.StatefulSetLister)

	// manually generated
	copied.control = c.control.(meta.TenantWise).ShallowCopyWithTenant(tenant).(StatefulSetControlInterface)
	return &copied
}
