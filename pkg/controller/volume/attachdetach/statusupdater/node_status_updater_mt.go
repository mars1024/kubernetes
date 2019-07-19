package statusupdater

import (
	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancycache "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/cache"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
)

func (nsu *nodeStatusUpdater) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	glog.V(5).Info("nodeStatusUpdater.ShallowCopyWithTenant: %v", tenant)
	tenantNsu := *nsu
	tenantNsu.kubeClient = nsu.kubeClient.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(clientset.Interface)
	tenantNsu.nodeLister = nsu.nodeLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.NodeLister)
	return &tenantNsu
}

func (nsu *nodeStatusUpdater) tenantFromNodeName(nodeName types.NodeName) (multitenancy.TenantInfo, error) {
	nodeString := string(nodeName)
	tenant, _, simpleNode, err := multitenancycache.MultiTenancySplitKeyWrapper(func(key string) (string, string, error) {
		return "", key, nil
	})(nodeString)
	if err != nil {
		glog.V(5).Infof("no tenant info from node name %s, simpleNode:%s", nodeString, simpleNode)
	}
	return tenant, err
}
func extractNodeName(nodeName types.NodeName) types.NodeName {
	node := string(nodeName)
	_, _, simpleNode, err := multitenancycache.MultiTenancySplitKeyWrapper(func(key string) (string, string, error) {
		return "", key, nil
	})(node)
	if err == nil {
		node = simpleNode
	}
	return types.NodeName(node)
}
