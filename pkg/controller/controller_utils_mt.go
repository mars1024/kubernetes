package controller

import (
	clientset "k8s.io/client-go/kubernetes"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
)

func (r RealPodControl) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copyRealPodControl := r
	copyRealPodControl.KubeClient = r.KubeClient.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(clientset.Interface)
	// TODO: multi tenancy event recorder
	return copyRealPodControl
}

func (r RealRSControl) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copyRealRSControl := r
	copyRealRSControl.KubeClient = r.KubeClient.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(clientset.Interface)
	// TODO: multi tenancy event recorder
	return copyRealRSControl
}
