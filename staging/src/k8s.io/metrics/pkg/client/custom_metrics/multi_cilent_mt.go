package custom_metrics

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
)

func (c *multiClient) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.newClient = func(gv schema.GroupVersion) (CustomMetricsClient, error) {
		client, err := c.newClient(gv)
		if err != nil {
			return nil, err
		}
		return client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(CustomMetricsClient), nil
	}
	return &copied
}
