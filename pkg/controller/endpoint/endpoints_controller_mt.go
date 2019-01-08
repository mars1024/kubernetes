package endpoint

import (
	multitenancy "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	meta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	kubernetes "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
)

func (c *EndpointController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.client = c.client.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.serviceLister = c.serviceLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.ServiceLister)
	copied.podLister = c.podLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.PodLister)
	copied.endpointsLister = c.endpointsLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.EndpointsLister)
	return &copied
}
