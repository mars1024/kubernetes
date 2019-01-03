package util

import (
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
)

type TenantGroupVersion struct {
	multitenancyutil.TenantHash
	schema.GroupVersion
}

func (tgv TenantGroupVersion) TenantGroup() TenantGroup {
	return TenantGroup{
		tgv.TenantHash,
		tgv.Group,
	}
}

type TenantGroup struct {
	multitenancyutil.TenantHash
	Group string
}

func CRDKeyFunc(crd *apiextensions.CustomResourceDefinition) string {
	tenant, _ := multitenancyutil.TransformTenantInfoFromAnnotations(crd.Annotations)
	return multitenancyutil.TransformTenantInfoToJointString(tenant, "/") + "/" + crd.Name
}
