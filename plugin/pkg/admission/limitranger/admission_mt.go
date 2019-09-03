package limitranger

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/apiserver/pkg/util/feature"
	"k8s.io/apiserver/pkg/admission"
	"fmt"
)

func (l *LimitRanger) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *l
	copied.client = l.client.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(internalclientset.Interface)
	copied.lister = l.lister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.LimitRangeLister)
	return &copied
}

func namespaceGetterMultiTenancyWrapper(a admission.Attributes) string {
	if feature.DefaultFeatureGate.Enabled(multitenancy.FeatureName) {
		tenant, err := multitenancyutil.TransformTenantInfoFromUser(a.GetUserInfo())
		if err != nil {
			panic(fmt.Errorf("missing tenant info in the request: %v", err))
		}
		return multitenancyutil.TransformTenantInfoToJointString(tenant, "/") + "/" + a.GetNamespace()
	}
	return a.GetNamespace()
}
