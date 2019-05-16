package filter

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"

	"k8s.io/apimachinery/pkg/runtime/schema"
	authnuser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
)

var (
	// See complete list/doc at: https://yuque.antfin-inc.com/antcloud-paas/aks/wpc0po
	AKSSupportedResources = []schema.GroupResource{
		// namespace-scoped
		{"", "pods"},
		{"", "services"},
		{"", "endpoints"},
		{"", "secrets"},
		{"", "configmaps"},
		{"", "replicationcontrollers"},
		{"", "serviceaccounts"},
		{"", "resourcequotas"},
		{"", "persistentvolumeclaims"},

		{"extensions", "ingresses"},
		{"extensions", "deployments"},
		{"extensions", "replicasets"},
		{"extensions", "daemonsets"},
		{"extensions", "statefulsets"},

		{"apps", "replicasets"},
		{"apps", "deployments"},
		{"apps", "daemonsets"},
		{"apps", "statefulsets"},
		{"apps", "controllerrevisions"},

		{"batch", "jobs"},
		{"batch", "cronjobs"},

		{"rbac.authorization.k8s.io", "roles"},
		{"rbac.authorization.k8s.io", "rolebindings"},

		// cluster-scoped
		{"", "nodes"},
		{"", "namespaces"},
		{"", "persistentvolumes"},
		{"", "persistentvolumes"},

		{"rbac.authorization.k8s.io", "clusterroles"},
		{"rbac.authorization.k8s.io", "clusterrolebindings"},

		{"cafe.sofastack.io", "serviceinstances"},
		{"cafe.sofastack.io", "servicebindings"},

		{"apiextensions.k8s.io", "customresourcedefinitions"},
	}
	AKSSupportedGroupSuffixes = []string{
		".istio.io",
		".knative.dev",
	}
)

func WithResourceWhiteList(delegate http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		reqInfo, _ := request.RequestInfoFrom(req.Context())
		user, _ := request.UserFrom(req.Context())
		tenant, _ := multitenancyutil.TransformTenantInfoFromUser(user)

		for _, group := range user.GetGroups() {
			if group == authnuser.NodesGroup {
				delegate.ServeHTTP(w, req)
				return
			}
		}

		if user.GetName() == authnuser.KubeProxy {
			delegate.ServeHTTP(w, req)
			return
		}

		// non-resource requests are always allowed, such as discovery
		if !reqInfo.IsResourceRequest {
			delegate.ServeHTTP(w, req)
			return
		}

		// backward-compatibility for empty-tenant admin certificates
		if multitenancyutil.IsMultiTenancyWiseAdmin(user.GetName()) && reflect.DeepEqual(tenant, multitenancy.GlobalAdminTenant) {
			delegate.ServeHTTP(w, req)
			return
		}

		// matching admin tenant with special prefix
		if multitenancyutil.IsMultiTenancyWiseTenant(tenant) {
			delegate.ServeHTTP(w, req)
			return
		}

		// checking whether the tenant is impersonated by an admin
		for _, group := range user.GetGroups() {
			if group == multitenancy.UserGroupMultiTenancyImpersonated {
				delegate.ServeHTTP(w, req)
				return
			}
		}

		// supported group suffix
		for _, groupSuffix := range AKSSupportedGroupSuffixes {
			if strings.HasSuffix(reqInfo.APIGroup, groupSuffix) {
				delegate.ServeHTTP(w, req)
				return
			}
		}

		// for normal tenants, only whitelisted resources is permitted
		targetGR := schema.GroupResource{reqInfo.APIGroup, reqInfo.Resource}
		for _, supportedGR := range AKSSupportedResources {
			if reflect.DeepEqual(supportedGR, targetGR) {
				delegate.ServeHTTP(w, req)
				return
			}
		}

		// rejects
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(fmt.Sprintf("resource %v/%v is currently not supported", reqInfo.APIGroup, reqInfo.Resource)))
		return
	})
}
