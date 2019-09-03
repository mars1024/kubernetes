package persistentvolume

import (
	"k8s.io/api/core/v1"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
)

func (b *volumeBinder) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	tenantVolumeBinder := *b
	tenantPvcCache := b.pvcCache.(multitenancymeta.TenantWise)
	tenantVolumeBinder.pvcCache = tenantPvcCache.ShallowCopyWithTenant(tenant).(PVCAssumeCache)
	tenantPvCache := b.pvCache.(multitenancymeta.TenantWise)
	tenantVolumeBinder.pvCache = tenantPvCache.ShallowCopyWithTenant(tenant).(PVAssumeCache)
	tenantVolumeBinder.ctrl = b.ctrl.ShallowCopyWithTenant(tenant).(*PersistentVolumeController)
	return &tenantVolumeBinder
}

func getPodNameWithCluster(pod *v1.Pod) string {
	tenant, err := util.TransformTenantInfoFromAnnotations(pod.Annotations)
	if err != nil {
		// This line should never reach
		panic(err)
	}
	return util.TransformTenantInfoToJointString(tenant, "/") + "/" + pod.Namespace + "/" + pod.Name
}
