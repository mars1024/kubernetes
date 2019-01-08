package replication

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"

	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1client "k8s.io/client-go/kubernetes/typed/core/v1"
	v1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/controller"
)

func (l conversionLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := l
	copied.rcLister = l.rcLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1listers.ReplicationControllerLister)
	return copied
}

func (l clientsetAdapter) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copy := l
	copy.Interface = l.Interface.(meta.TenantWise).ShallowCopyWithTenant(tenant).(clientset.Interface)
	return copy
}

func (l informerAdapter) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copy := l
	copy.rcInformer = l.rcInformer.(meta.TenantWise).ShallowCopyWithTenant(tenant).(coreinformers.ReplicationControllerInformer)
	return copy
}

func (l conversionInformer) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copy := l
	copy.SharedIndexInformer = l.SharedIndexInformer.(meta.TenantWise).ShallowCopyWithTenant(tenant).(cache.SharedIndexInformer)
	return copy
}

func (l conversionNamespaceLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copy := l
	copy.rcLister = l.rcLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1listers.ReplicationControllerNamespaceLister)
	return copy
}

func (l conversionEventHandler) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copy := l
	copy.handler = l.handler.(meta.TenantWise).ShallowCopyWithTenant(tenant).(cache.ResourceEventHandler)
	return copy
}

func (l conversionAppsV1Client) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copy := l
	copy.clientset = l.clientset.(meta.TenantWise).ShallowCopyWithTenant(tenant).(clientset.Interface)
	copy.AppsV1Interface = l.AppsV1Interface.(meta.TenantWise).ShallowCopyWithTenant(tenant).(appsv1client.AppsV1Interface)
	return copy
}

func (l conversionClient) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copy := l
	copy.ReplicationControllerInterface = l.ReplicationControllerInterface.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1client.ReplicationControllerInterface)
	return copy
}

func (l podControlAdapter) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copy := l
	copy.PodControlInterface = l.PodControlInterface.(meta.TenantWise).ShallowCopyWithTenant(tenant).(controller.PodControlInterface)
	return copy
}
