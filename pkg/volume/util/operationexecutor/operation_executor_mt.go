package operationexecutor

import (
	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancycache "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/cache"
	"k8s.io/apimachinery/pkg/types"
)

func extractNodeName(nodeName types.NodeName) string {
	node := string(nodeName)
	_, _, simpleNode, err := multitenancycache.MultiTenancySplitKeyWrapper(func(key string) (string, string, error) {
		return "", key, nil
	})(node)
	if err == nil {
		node = simpleNode
	}
	return node
}
func tenantFromNodeName(nodeName types.NodeName) (multitenancy.TenantInfo, error) {
	nodeString := string(nodeName)
	tenant, _, simpleNode, err := multitenancycache.MultiTenancySplitKeyWrapper(func(key string) (string, string, error) {
		return "", key, nil
	})(nodeString)
	if err != nil {
		glog.V(5).Infof("no tenant info from node name %s, simpleNode:%s", nodeString, simpleNode)
	}
	return tenant, err
}
