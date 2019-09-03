package attachdetach

import (
	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancycache "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/cache"
	utilfeature "k8s.io/apiserver/pkg/util/feature"

	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	csiclient "k8s.io/csi-api/pkg/client/clientset/versioned"
)

func (b *attachDetachController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	tenantB := *b
	tenantB.kubeClient = b.kubeClient.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(clientset.Interface)
	tenantB.pvLister = b.pvLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.PersistentVolumeLister)
	tenantB.pvcLister = b.pvcLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.PersistentVolumeClaimLister)
	tenantB.podLister = b.podLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.PodLister)
	tenantB.csiClient = b.csiClient.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(csiclient.Interface)
	tenantB.nodeLister = b.nodeLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.NodeLister)
	return &tenantB
}

func (b *attachDetachController) tenantNodeName(node *v1.Node) types.NodeName {
	nodeName := node.Name
	if utilfeature.DefaultFeatureGate.Enabled(multitenancy.FeatureName) {
		tenant, err := multitenancyutil.TransformTenantInfoFromAnnotations(node.Annotations)
		if err == nil {
			nodeName = multitenancyutil.TransformTenantInfoToJointString(tenant, "/") + "/" + nodeName
			glog.V(5).Infof("transform nodeName to tenant based: %s", nodeName)
		}
	}
	return types.NodeName(nodeName)
}
func (b *attachDetachController) extractNodeName(nodeName types.NodeName) string {
	node := string(nodeName)
	_, _, simpleNode, err := multitenancycache.MultiTenancySplitKeyWrapper(func(key string) (string, string, error) {
		return "", key, nil
	})(node)
	if err == nil {
		node = simpleNode
	}
	return node
}
