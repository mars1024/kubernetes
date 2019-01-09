package priorities

import (
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/scheduler/algorithm/predicates"
)

var _ multitenancymeta.TenantWise = &InterPodAffinity{}

func (ipa *InterPodAffinity) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("priority inter pod affinity with tenant %#v", tenant)
	tenantInterPodAffinity := *ipa
	tenantNodeInfo := tenantInterPodAffinity.info.(multitenancymeta.TenantWise)
	tenantInterPodAffinity.info = tenantNodeInfo.ShallowCopyWithTenant(tenant).(predicates.NodeInfo)
	tenantPodLister := tenantInterPodAffinity.podLister.(multitenancymeta.TenantWise)
	tenantInterPodAffinity.podLister = tenantPodLister.ShallowCopyWithTenant(tenant).(algorithm.PodLister)
	tenantNodeLister := tenantInterPodAffinity.nodeLister.(multitenancymeta.TenantWise)
	tenantInterPodAffinity.nodeLister = tenantNodeLister.ShallowCopyWithTenant(tenant).(algorithm.NodeLister)
	return &tenantInterPodAffinity
}