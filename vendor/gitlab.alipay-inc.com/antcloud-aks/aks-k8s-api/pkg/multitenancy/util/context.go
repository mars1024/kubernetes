package util

import (
	"context"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"k8s.io/apiserver/pkg/authentication/user"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

func NewEmptyContextWithTenant(tenant multitenancy.TenantInfo) context.Context {
	ctx := genericapirequest.NewContext()
	u := &user.DefaultInfo{Extra: make(map[string][]string)}
	TransformTenantInfoToUser(tenant, u)
	ctx = genericapirequest.WithUser(ctx, u)
	return ctx
}
