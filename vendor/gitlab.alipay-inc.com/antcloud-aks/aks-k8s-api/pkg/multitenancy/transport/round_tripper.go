package transport

import (
	"reflect"
	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"net/http"
	"k8s.io/client-go/transport"
)

type HTTPHeaderTransformer func(header http.Header)

type requestCanceler interface {
	CancelRequest(*http.Request)
}

func NewTenantHeaderTwistedRoundTripper(tenant multitenancy.TenantInfo, delegate http.RoundTripper) http.RoundTripper {
	if reflect.DeepEqual(tenant, multitenancy.GlobalAdminTenant) {
		return delegate
	}
	return NewHeaderTwistedRoundTripper(
		func(header http.Header) {
			header.Set(transport.ImpersonateUserHeader, "system:admin")
			header.Set(transport.ImpersonateUserExtraHeaderPrefix+multitenancy.UserExtraInfoTenantID, tenant.GetTenantID())
			header.Set(transport.ImpersonateUserExtraHeaderPrefix+multitenancy.UserExtraInfoWorkspaceID, tenant.GetWorkspaceID())
			header.Set(transport.ImpersonateUserExtraHeaderPrefix+multitenancy.UserExtraInfoClusterID, tenant.GetClusterID())
		}, delegate)
}

func NewHeaderTwistedRoundTripper(headerTransformer HTTPHeaderTransformer, delegate http.RoundTripper) http.RoundTripper {
	if headerRoundTripper, ok := delegate.(*headerTwistedRoundTripper); ok {
		// TODO(zouxiu.jm): remove this hack
		delegate = headerRoundTripper.WrappedRoundTripper()
	}
	return &headerTwistedRoundTripper{headerTransformer, delegate}
}

type headerTwistedRoundTripper struct {
	HTTPHeaderTransformer HTTPHeaderTransformer
	delegate              http.RoundTripper
}

func (rt *headerTwistedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.HTTPHeaderTransformer(req.Header)
	return rt.delegate.RoundTrip(req)
}

func (rt *headerTwistedRoundTripper) CancelRequest(req *http.Request) {
	if canceler, ok := rt.delegate.(requestCanceler); ok {
		canceler.CancelRequest(req)
	} else {
		glog.Errorf("CancelRequest not implemented")
	}
}

func (rt *headerTwistedRoundTripper) WrappedRoundTripper() http.RoundTripper { return rt.delegate }

func transformTenantInfoToUser(tenant multitenancy.TenantInfo, cfg *transport.ImpersonationConfig) error {
	if cfg.Extra == nil {
		cfg.Extra = make(map[string][]string)
	}
	if len(tenant.GetTenantID()) > 0 {
		cfg.Extra[multitenancy.UserExtraInfoTenantID] = []string{tenant.GetTenantID()}
	}
	if len(tenant.GetWorkspaceID()) > 0 {
		cfg.Extra[multitenancy.UserExtraInfoWorkspaceID] = []string{tenant.GetWorkspaceID()}
	}
	if len(tenant.GetClusterID()) > 0 {
		cfg.Extra[multitenancy.UserExtraInfoClusterID] = []string{tenant.GetClusterID()}
	}
	return nil
}
