package copy

import (
	"context"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"

	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apimachinery/pkg/runtime"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/apiserver/pkg/endpoints/request"
)

func InjectRESTLister(lister rest.Lister) rest.Lister {
	return &restLister{lister}
}

var _ rest.Lister = &tenantRESTLister{}

type restLister struct {
	rest.Lister
}

func (l *restLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return tenantRESTLister{
		l,
		tenant,
	}

}

type tenantRESTLister struct {
	rest.Lister
	tenant multitenancy.TenantInfo
}

func (l *tenantRESTLister) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	user, _ := request.UserFrom(ctx)
	if err := util.TransformTenantInfoToUser(l.tenant, user); err != nil {
		return nil, err
	}
	ctx = request.WithUser(ctx, user)
	return l.Lister.List(ctx, options)
}
