package filter

import (
	"crypto/x509"
	"strings"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	apiserverx509 "k8s.io/apiserver/pkg/authentication/request/x509"
	"k8s.io/apiserver/pkg/authentication/user"
)

var CommonNameUserConversionWithMultiTenancy apiserverx509.UserConversion = apiserverx509.UserConversionFunc(func(chain []*x509.Certificate) (user.Info, bool, error) {
	if len(chain[0].Subject.CommonName) == 0 {
		return nil, false, nil
	}

	o := chain[0].Subject.Organization

	// TODO(zuoxiu.jm): refactor the following code, set extra in a loop over prefixes
	// filtering multitenacy infos
	var tenantID, workspaceID, clusterID string
	var userGroups []string
	for idx := range o {
		if strings.HasPrefix(o[idx], multitenancy.X509CertificateTenantIDPrefix) {
			tenantID = strings.TrimPrefix(o[idx], multitenancy.X509CertificateTenantIDPrefix)
			continue
		}
		if strings.HasPrefix(o[idx], multitenancy.X509CertificateWorkspaceIDPrefix) {
			workspaceID = strings.TrimPrefix(o[idx], multitenancy.X509CertificateWorkspaceIDPrefix)
			continue
		}
		if strings.HasPrefix(o[idx], multitenancy.X509CertificateClusterIDPrefix) {
			clusterID = strings.TrimPrefix(o[idx], multitenancy.X509CertificateClusterIDPrefix)
			continue
		}
		userGroups = append(userGroups, o[idx])
	}
	// ou format e.g.: group1:group2|tenant:workspace:cluster
	user := &user.DefaultInfo{
		Name:   chain[0].Subject.CommonName,
		Groups: userGroups,
		Extra:  make(map[string][]string),
	}
	if len(tenantID) > 0 && len(workspaceID) > 0 && len(clusterID) > 0 {
		user.Extra[multitenancy.UserExtraInfoTenantID] = []string{tenantID}
		user.Extra[multitenancy.UserExtraInfoWorkspaceID] = []string{workspaceID}
		user.Extra[multitenancy.UserExtraInfoClusterID] = []string{clusterID}
	}
	return user, true, nil
})
