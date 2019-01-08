package cronjob

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/client-go/kubernetes"
)

func (c *CronJobController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.kubeClient = c.kubeClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.jobControl = c.jobControl.(meta.TenantWise).ShallowCopyWithTenant(tenant).(jobControlInterface)
	copied.sjControl = c.sjControl.(meta.TenantWise).ShallowCopyWithTenant(tenant).(sjControlInterface)
	copied.podControl = c.podControl.(meta.TenantWise).ShallowCopyWithTenant(tenant).(podControlInterface)
	return &copied
}
