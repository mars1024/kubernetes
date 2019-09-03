package cache

import (
	"fmt"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
	"strings"
)

const (
	TenantIndex          string = "tenant"
	TenantNamespaceIndex string = "tenant_namespace"
)

var MultiTenancyKeyFuncWrapper = cache.MultiTenancyKeyFuncWrapper

func MultiTenancyPodKey(pod *v1.Pod) string {
	tenant, err := util.TransformTenantInfoFromAnnotations(pod.Annotations)
	if err != nil {
		// This line should never reach
		panic(err)
	}
	return util.TransformTenantInfoToJointString(tenant, "/") + "/" + pod.Namespace + "/" + pod.Name
}

func MetaTenantNamespaceIndexFunc(obj interface{}) ([]string, error) {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return []string{""}, fmt.Errorf("object has no meta: %v", err)
	}
	tenantWrappedKeyFunc := MultiTenancyKeyFuncWrapper(func(obj interface{}) (string, error) {
		return meta.GetNamespace(), nil
	})
	namespaceWithTenant, err := tenantWrappedKeyFunc(obj)
	if err != nil {
		return []string{""}, err
	}
	return []string{namespaceWithTenant}, nil
}

func MetaTenantIndexFunc(obj interface{}) ([]string, error) {
	key, err := MultiTenancyKeyFuncWrapper(func(obj interface{}) (string, error) {
		return "", nil
	})(obj)
	if err != nil {
		return []string{""}, err
	}
	return []string{key}, nil
}

func MultiTenancySplitKeyWrapper(splitKeyFunc func(string) (string, string, error)) func(key string) (multitenancy.TenantInfo, string, string, error) {
	return func(key string) (multitenancy.TenantInfo, string, string, error) {
		parts := strings.SplitN(key, "/", 4)
		if len(parts) != 4 {
			namespace, name, err := splitKeyFunc(key)
			return nil, namespace, name, err
		}
		tenant := multitenancy.NewTenantInfo(parts[0], parts[1], parts[2])
		namespace, name, err := splitKeyFunc(parts[3])
		return tenant, namespace, name, err
	}
}
