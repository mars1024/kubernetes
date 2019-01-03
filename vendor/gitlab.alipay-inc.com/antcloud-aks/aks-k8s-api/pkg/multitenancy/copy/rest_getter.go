package copy

import (
	"context"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"

	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/apiserver/pkg/endpoints/request"
)

func InjectRESTGetter(getter rest.Getter) rest.Getter {
	return &restGetter{getter}
}

var _ rest.Lister = &tenantRESTLister{}

type restGetter struct {
	rest.Getter
}

func (g *restGetter) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return tenantRESTGetter{
		g,
		tenant,
	}

}

type tenantRESTGetter struct {
	rest.Getter
	tenant multitenancy.TenantInfo
}

func (g *tenantRESTGetter) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	user, _ := request.UserFrom(ctx)
	if err := util.TransformTenantInfoToUser(g.tenant, user); err != nil {
		return nil, err
	}
	ctx = request.WithUser(ctx, user)
	return g.Getter.Get(ctx, name, options)
}
