package mutating

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/generic"
)

func  (a *Plugin) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *a
	copied.Webhook = a.Webhook.ShallowCopyWithTenant(tenant).(*generic.Webhook)
	return &copied
}
