package priorities

import (
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"

	"github.com/golang/glog"
)

func (pmf *PriorityMetadataFactory) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("get priority meta with tenant info %#v", tenant)
	tenantServiceLister := pmf.serviceLister.(multitenancymeta.TenantWise)
	tenantControllerLister := pmf.controllerLister.(multitenancymeta.TenantWise)
	tenantReplicaSetLister := pmf.replicaSetLister.(multitenancymeta.TenantWise)
	tenantStatefulSetLister := pmf.statefulSetLister.(multitenancymeta.TenantWise)
	return &PriorityMetadataFactory{
		serviceLister:     tenantServiceLister.ShallowCopyWithTenant(tenant).(algorithm.ServiceLister),
		controllerLister:  tenantControllerLister.ShallowCopyWithTenant(tenant).(algorithm.ControllerLister),
		replicaSetLister:  tenantReplicaSetLister.ShallowCopyWithTenant(tenant).(algorithm.ReplicaSetLister),
		statefulSetLister: tenantStatefulSetLister.ShallowCopyWithTenant(tenant).(algorithm.StatefulSetLister),
	}
}