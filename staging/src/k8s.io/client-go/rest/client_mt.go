// +build multitenancy

package rest

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/transport"
)

func (c *RESTClient) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copiedHTTPClient := *copied.Client
	copiedHTTPClient.Transport = transport.NewTenantHeaderTwistedRoundTripper(tenant, copiedHTTPClient.Transport)
	copied.Client = &copiedHTTPClient
	return &copied
}
