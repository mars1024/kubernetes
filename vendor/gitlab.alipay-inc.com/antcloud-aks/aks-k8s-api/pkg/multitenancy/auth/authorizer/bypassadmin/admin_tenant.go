package bypassadmin

import (
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
)

type byPassAdminTenantAuthz struct{}

var _ authorizer.Authorizer = byPassAdminTenantAuthz{}

func NewByPassingAdminTenantAuthorizer() authorizer.Authorizer {
	return byPassAdminTenantAuthz{}
}

func (byPassAdminTenantAuthz) Authorize(requestAttributes authorizer.Attributes) (authorizer.Decision, string, error) {
	if util.IsMultiTenancyWiseAdmin(requestAttributes.GetUser().GetName()) {
		return authorizer.DecisionAllow, "", nil
	}
	return authorizer.DecisionNoOpinion, "", nil
}
