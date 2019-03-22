package persistentvolume

import (
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/apimachinery/pkg/api/meta"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"github.com/golang/glog"
)

var MultiTenancyKeyFuncWrapper = cache.MultiTenancyKeyFuncWrapper

func MetaTenantNamespaceIndexFunc(obj interface{}) ([]string, error) {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return []string{""}, fmt.Errorf("object has no meta: %v", err)
	}
	tenantWrappedKeyFunc := MultiTenancyKeyFuncWrapper(func(obj interface{}) (string, error) {
		return metadata.GetNamespace(), nil
	})
	namespaceWithTenant, err := tenantWrappedKeyFunc(obj)
	if err != nil {
		return []string{""}, err
	}
	return []string{namespaceWithTenant}, nil
}

func (c *pvcAssumeCache) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return &tenantPVCAssumeCache{
		c,
		tenant,
	}
}

func (c *pvAssumeCache) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return &tenantPVAssumeCache{
		c,
		tenant,
	}
}

var _ PVCAssumeCache = &tenantPVCAssumeCache{}
var _ PVAssumeCache = &tenantPVAssumeCache{}

type tenantPVCAssumeCache struct {
	*pvcAssumeCache
	tenant multitenancy.TenantInfo
}

func (c *tenantPVCAssumeCache) GetPVC(pvcKey string) (*v1.PersistentVolumeClaim, error) {
	fullPvcKey := util.TransformTenantInfoToJointString(c.tenant, "/") + "/" + pvcKey
	obj, err := c.Get(fullPvcKey)
	if err != nil {
		return nil, err
	}

	pvc, ok := obj.(*v1.PersistentVolumeClaim)
	if !ok {
		return nil, &errWrongType{"v1.PersistentVolumeClaim", obj}
	}
	return pvc, nil
}

type tenantPVAssumeCache struct {
	*pvAssumeCache
	tenant multitenancy.TenantInfo
}

func (c *tenantPVAssumeCache) GetPV(pvName string) (*v1.PersistentVolume, error) {
	fullPvName := util.TransformTenantInfoToJointString(c.tenant, "/") + "/" + pvName
	obj, err := c.Get(fullPvName)
	if err != nil {
		return nil, err
	}

	pv, ok := obj.(*v1.PersistentVolume)
	if !ok {
		return nil, &errWrongType{"v1.PersistentVolume", obj}
	}
	return pv, nil
}

func (c *tenantPVAssumeCache) ListPVs(storageClassName string) []*v1.PersistentVolume {
	objs := c.List(&v1.PersistentVolume{
		Spec: v1.PersistentVolumeSpec{
			StorageClassName: storageClassName,
		},
	})
	pvs := []*v1.PersistentVolume{}
	for _, obj := range objs {
		pv, ok := obj.(*v1.PersistentVolume)
		if !ok {
			glog.Errorf("ListPVs: %v", &errWrongType{"v1.PersistentVolume", obj})
		}
		pvs = append(pvs, pv)
	}
	return pvs
}
