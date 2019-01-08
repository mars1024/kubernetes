package resize

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	pvlister "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	storagelisters "k8s.io/kubernetes/pkg/client/listers/storage/internalversion"
)

func (pvcr *persistentVolumeClaimResize) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *pvcr
	copied.pvLister = pvcr.pvLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(pvlister.PersistentVolumeLister)
	copied.scLister = pvcr.scLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(storagelisters.StorageClassLister)
	return &copied
}
