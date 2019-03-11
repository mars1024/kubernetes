package metrics

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/custom_metrics"
	"k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/metrics/pkg/client/external_metrics"
)

func (c *resourceMetricsClient) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.client = c.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1beta1.PodMetricsesGetter)
	return &c
}

func (c *customMetricsClient) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.client = c.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(custom_metrics.CustomMetricsClient)
	return &c
}

func (c *HeapsterMetricsClient) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.podsGetter = c.podsGetter.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.PodsGetter)
	copied.services = c.services.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.ServiceInterface)
	return &copied
}

func (c *externalMetricsClient) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.client = c.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(external_metrics.ExternalMetricsClient)
	return &copied
}

func (c *restMetricsClient) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.resourceMetricsClient = c.ShallowCopyWithTenant(tenant).(*resourceMetricsClient)
	copied.customMetricsClient = c.ShallowCopyWithTenant(tenant).(*customMetricsClient)
	copied.externalMetricsClient = c.externalMetricsClient.ShallowCopyWithTenant(tenant).(*externalMetricsClient)
	return &c
}
