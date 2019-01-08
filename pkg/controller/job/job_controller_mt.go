package job

import (
	multitenancy "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	meta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	kubernetes "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/batch/v1"
	corev1 "k8s.io/client-go/listers/core/v1"
	controller "k8s.io/kubernetes/pkg/controller"
)

func (c *JobController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.kubeClient = c.kubeClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.podControl = c.podControl.(meta.TenantWise).ShallowCopyWithTenant(tenant).(controller.PodControlInterface)
	copied.jobLister = c.jobLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.JobLister)
	copied.podStore = c.podStore.(meta.TenantWise).ShallowCopyWithTenant(tenant).(corev1.PodLister)
	return &copied
}
