// +build multitenancy

package v1

import (
	"k8s.io/client-go/rest"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
)


func (e *events) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *e
	copied.client = e.client.(*rest.RESTClient).ShallowCopyWithTenant(tenant).(*rest.RESTClient)
	return &copied
}
