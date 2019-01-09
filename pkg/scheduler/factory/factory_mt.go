package factory

import (
	"reflect"

	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
	"k8s.io/apimachinery/pkg/labels"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
)

func (n *nodeLister) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	tenantNodeLister := n.NodeLister.(multitenancymeta.TenantWise)
	return &nodeLister{
		NodeLister: tenantNodeLister.ShallowCopyWithTenant(tenant).(corelisters.NodeLister),
	}
}

func (b *binder) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	tenantClient := b.Client.(multitenancymeta.TenantWise)
	return &binder{
		Client: tenantClient.ShallowCopyWithTenant(tenant).(clientset.Interface),
	}
}

var tenantPodFilterGetter = func(tenant multitenancy.TenantInfo) schedulercache.PodFilter {
	return func(pod *v1.Pod) bool {
		tenantInfo, _ := multitenancyutil.TransformTenantInfoFromAnnotations(pod.Annotations)
		return reflect.DeepEqual(tenant, tenantInfo)
	}
}

func (p *podListerAdapter) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return &podListerAdapter{
		filteredListFunc: func(podFilter schedulercache.PodFilter, selector labels.Selector) ([]*v1.Pod, error) {
			return p.filteredListFunc(func(pod *v1.Pod) bool {
				return tenantPodFilterGetter(tenant)(pod) && podFilter(pod)
			}, selector)
		},
	}
}

func (p *podConditionUpdater) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	tenantClient := p.Client.(multitenancymeta.TenantWise)
	return &podConditionUpdater{
		Client: tenantClient.ShallowCopyWithTenant(tenant).(clientset.Interface),
	}
}

func (p *podPreemptor) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	tenantClient := p.Client.(multitenancymeta.TenantWise)
	return &podPreemptor{
		Client: tenantClient.ShallowCopyWithTenant(tenant).(clientset.Interface),
	}
}
