package cache

import (
	"fmt"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/apimachinery/pkg/api/meta"
)

func MultiTenancyKeyFuncWrapper(keyFunc KeyFunc) KeyFunc {
	return func(obj interface{}) (string, error) {
		if key, ok := obj.(ExplicitKey); ok {
			return string(key), nil
		}
		key, err := keyFunc(obj)
		if err != nil {
			return key, err
		}
		if d, ok := obj.(DeletedFinalStateUnknown); ok {
			obj = d.Obj
		}
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return "", fmt.Errorf("fail to extract tenant info from %#v: %s", obj, err.Error())
		}
		tenantInfo, err := util.TransformTenantInfoFromAnnotations(accessor.GetAnnotations())
		if err == nil {
			return util.TransformTenantInfoToJointString(tenantInfo, "/") + "/" + key, nil
		}
		return key, nil
	}
}
