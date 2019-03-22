package persistentvolume

import (
	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1"
)

func (ctrl *PersistentVolumeController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.Infof("PersistentVolumeController ShallowCopyWithTenant with tenant info %#v", tenant)
	ctrlCloned := *ctrl
	ctrlCloned.kubeClient = ctrl.kubeClient.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(clientset.Interface)
	ctrlCloned.classLister = ctrl.classLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(storagelisters.StorageClassLister)
	// note: scheduler did not initialize following fields
	if ctrl.claimLister != nil {
		ctrlCloned.claimLister = ctrl.claimLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.PersistentVolumeClaimLister)
	}
	if ctrl.volumeLister != nil {
		ctrlCloned.volumeLister = ctrl.volumeLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.PersistentVolumeLister)
	}
	return &ctrlCloned
}

func getVolumeNameFromPVC(pvc *v1.PersistentVolumeClaim) string {
	tenantInfo, err := util.TransformTenantInfoFromAnnotations(pvc.Annotations)
	volumeName := pvc.Spec.VolumeName
	if err == nil {
		return multitenancyutil.TransformTenantInfoToJointString(tenantInfo, "/") + "/" + volumeName
	}
	return volumeName
}
