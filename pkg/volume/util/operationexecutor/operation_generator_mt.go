package operationexecutor

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	clientset "k8s.io/client-go/kubernetes"
)

func (og *operationGenerator) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	cloned := *og
	cloned.kubeClient = og.kubeClient.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(clientset.Interface)
	return &cloned
}
