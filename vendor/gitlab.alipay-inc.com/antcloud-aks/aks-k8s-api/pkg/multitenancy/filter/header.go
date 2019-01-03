package filter

import (
	"fmt"
	"net/http"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/authentication/user"
)

var (
	ErrNoTenantInfoRequest error = fmt.Errorf("no tenant info in the user request")
)

func OverrideTenantInfoForLegacyAdmin(headers http.Header, userInfo user.Info) (overrided bool, err error) {
	var overridingTenant multitenancy.TenantInfo

	isLegacyGlobalAdmin := util.IsMultiTenancyWiseAdmin(userInfo.GetName())

	// for a backward compatibilities
	if len(userInfo.GetExtra()) == 0 {
		if isLegacyGlobalAdmin {
			overridingTenant = multitenancy.GlobalAdminTenant
			overrided = true
		} else {
			return false, errors.NewBadRequest(fmt.Sprintf("invalid or malformed client certificate: %v", userInfo))
		}
	}

	if overridingTenant != nil {
		if err := util.TransformTenantInfoToUser(overridingTenant, userInfo); err != nil {
			return false, errors.NewBadRequest(err.Error())
		}
		overrided = true
	}
	return
}
