package disruption

import (
	multitenancy "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	meta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	kubernetes "k8s.io/client-go/kubernetes"
	appsv1beta1 "k8s.io/client-go/listers/apps/v1beta1"
	v1 "k8s.io/client-go/listers/core/v1"
	extensionsv1beta1 "k8s.io/client-go/listers/extensions/v1beta1"
	v1beta1 "k8s.io/client-go/listers/policy/v1beta1"
)

func (c *DisruptionController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.kubeClient = c.kubeClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.pdbLister = c.pdbLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1beta1.PodDisruptionBudgetLister)
	copied.podLister = c.podLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.PodLister)
	copied.rcLister = c.rcLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.ReplicationControllerLister)
	copied.rsLister = c.rsLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(extensionsv1beta1.ReplicaSetLister)
	copied.dLister = c.dLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(extensionsv1beta1.DeploymentLister)
	copied.ssLister = c.ssLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(appsv1beta1.StatefulSetLister)
	return &copied
}
