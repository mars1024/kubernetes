package registry

import (
	"fmt"
	"strings"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"

	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/validation/path"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"reflect"
	"context"
)

// MultiTenancyKeyRootFuncForNamespaced is a function for constructing etcd storage path with
// tenant info for namespaced resource. It will respect namespace from request context.
func MultiTenancyKeyRootFuncForNamespaced(ctx context.Context, prefix string) string {
	key, isGlobal := constructMultiTenancyKeyForPrefix(ctx, prefix)
	if isGlobal {
		return key
	}
	ns, ok := genericapirequest.NamespaceFrom(ctx)
	if ok && len(ns) > 0 {
		key = key + "/" + ns
	}
	return key
}

// MultiTenancyKeyRootFuncForNonNamespaced is a function for constructing etcd storage path with
// tenant info for namespaced resource. It will respect namespace from request context.
func MultiTenancyKeyRootFuncForNonNamespaced(ctx context.Context, prefix string) string {
	key, _ := constructMultiTenancyKeyForPrefix(ctx, prefix)
	return key
}

// MultiTenancyKeyFuncForNamespaced is a function for constructing etcd storage path with
// tenant info for namespaced resource. It will respect namespace from request context.
// TODO(zuoxiu.jm): String concatenation by "+" operator is costy for performance. Might do an AB test to testify.
func MultiTenancyKeyFuncForNamespaced(ctx context.Context, prefix string, name string, strict bool) (string, error) {
	key, isGlobal := constructMultiTenancyKeyForPrefix(ctx, prefix)
	if isGlobal && strict {
		return "", kubeerr.NewBadRequest("admin tenant is only allowed to list/watch")
	}
	ns, ok := genericapirequest.NamespaceFrom(ctx)
	if !ok || len(ns) == 0 {
		return "", kubeerr.NewBadRequest("Namespace parameter required.")
	}
	if len(name) == 0 {
		return "", kubeerr.NewBadRequest("Name parameter required.")
	}
	if msgs := path.IsValidPathSegmentName(name); len(msgs) != 0 {
		return "", kubeerr.NewBadRequest(fmt.Sprintf("Name parameter invalid: %q: %s", name, strings.Join(msgs, ";")))
	}
	key = key + "/" + ns + "/" + name
	return key, nil
}

// MultiTenancyKeyFuncForNonNamespaced is a function for constructing etcd storage path with
// tenant info for non-namespaced resource. It will not respect namespace from request context.
func MultiTenancyKeyFuncForNonNamespaced(ctx context.Context, prefix string, name string, strict bool) (string, error) {
	key, isGlobal := constructMultiTenancyKeyForPrefix(ctx, prefix)
	if isGlobal && strict {
		return "", kubeerr.NewBadRequest("admin tenant is only allowed to list/watch")
	}
	if len(name) == 0 {
		return "", kubeerr.NewBadRequest("Name parameter required.")
	}
	if msgs := path.IsValidPathSegmentName(name); len(msgs) != 0 {
		return "", kubeerr.NewBadRequest(fmt.Sprintf("Name parameter invalid: %q: %s", name, strings.Join(msgs, ";")))
	}
	key = key + "/" + name
	return key, nil
}

func constructMultiTenancyKeyForPrefix(ctx context.Context, prefix string) (string, bool) {
	user, userExists := genericapirequest.UserFrom(ctx)
	var tenant multitenancy.TenantInfo
	var err error
	if userExists {
		tenant, err = util.TransformTenantInfoFromUser(user)
	}
	switch {
	case !userExists:
		// we're hitting this line b/c of direct storage call from any kind of registry
		return constructGlobalKeyWithPrefix(prefix), true
	case err != nil:
		glog.Warningf("abortted storage call due to malformed user info: %#v", user)
		fallthrough
	case reflect.DeepEqual(tenant, multitenancy.GlobalAdminTenant):
		return constructGlobalKeyWithPrefix(prefix), true
	default:
		return constructMultiTenancyKeyWithPrefix(tenant, prefix), false
	}
}

func constructMultiTenancyKeyWithPrefix(tenant multitenancy.TenantInfo, prefix string) string {
	return prefix + "/" + util.TransformTenantInfoToJointString(tenant, "/")
}

func constructGlobalKeyWithPrefix(prefix string) string {
	return prefix
}
