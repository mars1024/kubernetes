package csi

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancycache "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/cache"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func (c *csiAttacher) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	clonedC := *c
	clonedC.k8s = c.k8s.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	return &clonedC
}

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