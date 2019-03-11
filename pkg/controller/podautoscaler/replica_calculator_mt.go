package podautoscaler

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/controller/podautoscaler/metrics"
)

func (c *ReplicaCalculator) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.podLister = c.podLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.PodLister)
	copied.metricsClient = c.metricsClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(metrics.MetricsClient)
	return &copied
}
