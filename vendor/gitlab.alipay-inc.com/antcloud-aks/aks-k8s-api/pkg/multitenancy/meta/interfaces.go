package meta

import "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"

type TenantWise interface {
	ShallowCopyWithTenant(info multitenancy.TenantInfo) interface{}
}
