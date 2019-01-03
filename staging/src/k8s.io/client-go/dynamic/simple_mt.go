package dynamic

import (
	"k8s.io/client-go/rest"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
)

func (c *dynamicClient) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	var copiedRESTClient interface{} = c.client
	copied.client = copiedRESTClient.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(*rest.RESTClient)
	return &copied
}
