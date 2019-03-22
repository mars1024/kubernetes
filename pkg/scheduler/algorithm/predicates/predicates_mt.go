package predicates

import (
	corelisters "k8s.io/client-go/listers/core/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1"
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	"k8s.io/kubernetes/pkg/scheduler/volumebinder"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"

	"github.com/golang/glog"
)

var _ multitenancymeta.TenantWise = &PodAffinityChecker{}
var _ multitenancymeta.TenantWise = &ServiceAffinity{}
var _ multitenancymeta.TenantWise = &VolumeBindingChecker{}
var _ multitenancymeta.TenantWise = &VolumeZoneChecker{}

func (c *PodAffinityChecker) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("check pod affinity with tenant %#v", tenant)
	tenantPodAffinityChecker := *c
	tenantPodLister := tenantPodAffinityChecker.podLister.(multitenancymeta.TenantWise)
	tenantPodAffinityChecker.podLister = tenantPodLister.ShallowCopyWithTenant(tenant).(algorithm.PodLister)
	tenantNodeInfo := tenantPodAffinityChecker.info.(multitenancymeta.TenantWise)
	tenantPodAffinityChecker.info = tenantNodeInfo.ShallowCopyWithTenant(tenant).(NodeInfo)
	return &tenantPodAffinityChecker
}

func (s *ServiceAffinity) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("check service affinity with tenant %#v", tenant)
	tenantServiceAffinity := *s
	tenantPodLister := tenantServiceAffinity.podLister.(multitenancymeta.TenantWise)
	tenantServiceAffinity.podLister = tenantPodLister.ShallowCopyWithTenant(tenant).(algorithm.PodLister)
	tenantServiceLister := tenantServiceAffinity.serviceLister.(multitenancymeta.TenantWise)
	tenantServiceAffinity.serviceLister = tenantServiceLister.ShallowCopyWithTenant(tenant).(algorithm.ServiceLister)
	tenantNodeInfo := tenantServiceAffinity.nodeInfo.(multitenancymeta.TenantWise)
	tenantServiceAffinity.nodeInfo = tenantNodeInfo.ShallowCopyWithTenant(tenant).(NodeInfo)
	return &tenantServiceAffinity
}

func (c *VolumeBindingChecker) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("check volume binding with tenant %#v", tenant)
	tenantVolumeBindingChecker := *c
	tenantVolumeBindingChecker.binder = c.binder.ShallowCopyWithTenant(tenant).(*volumebinder.VolumeBinder)
	return &tenantVolumeBindingChecker
}

func (c *CachedNodeInfo) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("cached node info copy with tenant %#v", tenant)
	tenantCachedNodeInfo := *c
	tenantNodeLister := tenantCachedNodeInfo.NodeLister.(multitenancymeta.TenantWise)
	tenantCachedNodeInfo.NodeLister = tenantNodeLister.ShallowCopyWithTenant(tenant).(corelisters.NodeLister)
	return &tenantCachedNodeInfo
}

func (c *VolumeZoneChecker) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("cached volume zone checker copy with tenant #%v", tenant)
	tenantVolumeZoneChecker := *c
	tenantPvInfo := tenantVolumeZoneChecker.pvInfo.(multitenancymeta.TenantWise)
	tenantVolumeZoneChecker.pvInfo = tenantPvInfo.ShallowCopyWithTenant(tenant).(PersistentVolumeInfo)
	tenantPvcInfo := tenantVolumeZoneChecker.pvcInfo.(multitenancymeta.TenantWise)
	tenantVolumeZoneChecker.pvcInfo = tenantPvcInfo.ShallowCopyWithTenant(tenant).(PersistentVolumeClaimInfo)
	tenantClassInfo := tenantVolumeZoneChecker.classInfo.(multitenancymeta.TenantWise)
	tenantVolumeZoneChecker.classInfo = tenantClassInfo.ShallowCopyWithTenant(tenant).(StorageClassInfo)
	return &tenantVolumeZoneChecker
}

func (c *CachedPersistentVolumeInfo) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("cached pv info copy with tenant #%v", tenant)
	tenantCachedPvInfo := *c
	tenantPersistentVolumeLister := tenantCachedPvInfo.PersistentVolumeLister.(multitenancymeta.TenantWise)
	tenantCachedPvInfo.PersistentVolumeLister = tenantPersistentVolumeLister.ShallowCopyWithTenant(tenant).(corelisters.PersistentVolumeLister)
	return &tenantCachedPvInfo
}

func (c *CachedPersistentVolumeClaimInfo) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("cached pvc info copy with tenant #%v", tenant)
	tenantCachedPvcInfo := *c
	tenantPersistentVolumeClaimLister := tenantCachedPvcInfo.PersistentVolumeClaimLister.(multitenancymeta.TenantWise)
	tenantCachedPvcInfo.PersistentVolumeClaimLister = tenantPersistentVolumeClaimLister.ShallowCopyWithTenant(tenant).(corelisters.PersistentVolumeClaimLister)
	return &tenantCachedPvcInfo
}

func (c *CachedStorageClassInfo) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("cached storage class info copy with tenant #%v", tenant)
	tenantCachedScInfo := *c
	tenantStorageClassLister := tenantCachedScInfo.StorageClassLister.(multitenancymeta.TenantWise)
	tenantCachedScInfo.StorageClassLister = tenantStorageClassLister.ShallowCopyWithTenant(tenant).(storagelisters.StorageClassLister)
	return &tenantCachedScInfo
}
func (c *CSIMaxVolumeLimitChecker) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("CSIMaxVolumeLimitChecker copy with tenant #%v", tenant)
	tenantCSIMaxVolumeLimitChecker := *c
	tenantPvcInfo := tenantCSIMaxVolumeLimitChecker.pvcInfo.(multitenancymeta.TenantWise)
	tenantPvInfo := tenantCSIMaxVolumeLimitChecker.pvInfo.(multitenancymeta.TenantWise)
	tenantCSIMaxVolumeLimitChecker.pvcInfo = tenantPvcInfo.ShallowCopyWithTenant(tenant).(*CachedPersistentVolumeClaimInfo)
	tenantCSIMaxVolumeLimitChecker.pvInfo = tenantPvInfo.ShallowCopyWithTenant(tenant).(*CachedPersistentVolumeInfo)
	return &tenantCSIMaxVolumeLimitChecker
}

func (c *MaxPDVolumeCountChecker) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("MaxPDVolumeCountChecker copy with tenant #%v", tenant)
	tenantChecker := *c
	tenantPvcInfo := tenantChecker.pvcInfo.(multitenancymeta.TenantWise)
	tenantPvInfo := tenantChecker.pvInfo.(multitenancymeta.TenantWise)
	tenantChecker.pvcInfo = tenantPvcInfo.ShallowCopyWithTenant(tenant).(*CachedPersistentVolumeClaimInfo)
	tenantChecker.pvInfo = tenantPvInfo.ShallowCopyWithTenant(tenant).(*CachedPersistentVolumeInfo)
	return &tenantChecker
}
