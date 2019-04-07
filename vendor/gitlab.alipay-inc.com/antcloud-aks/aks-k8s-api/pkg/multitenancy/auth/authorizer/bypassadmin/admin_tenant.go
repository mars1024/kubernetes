package bypassadmin

import (
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"github.com/golang/glog"
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
	tenant, err := util.TransformTenantInfoFromUser(requestAttributes.GetUser())
	if err != nil {
		glog.Warning("fail to extract tenant info from user: %v", requestAttributes.GetUser())
		return authorizer.DecisionNoOpinion, "", nil
	}
	if util.IsMultiTenancyWiseTenant(tenant) {
		return authorizer.DecisionAllow, "", nil
	}
	return authorizer.DecisionNoOpinion, "", nil
}
