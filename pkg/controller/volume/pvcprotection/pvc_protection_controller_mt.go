package pvcprotection

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/listers/core/v1"
)

func (c *Controller) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.client = c.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.podLister = c.podLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(corev1.PodLister)
	copied.pvcLister = c.pvcLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(corev1.PersistentVolumeClaimLister)
	return &copied
}
