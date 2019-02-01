package statefulset

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	clientset "k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// manually generated
func (c *realStatefulPodControl) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.client = c.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(clientset.Interface)
	copied.setLister = c.setLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(appslisters.StatefulSetLister)
	copied.podLister = c.podLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.PodLister)
	copied.pvcLister = c.pvcLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.PersistentVolumeClaimLister)
	return &copied
}
