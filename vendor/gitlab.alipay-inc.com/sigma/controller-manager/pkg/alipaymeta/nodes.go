package alipaymeta

import (
	"k8s.io/api/core/v1"

	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

func GetNodeSN(n *v1.Node) string {
	if v, exists := n.Labels[api.LabelNodeSN]; exists {
		return v
	}
	return ""
}

func GetNodeIP(n *v1.Node) string {
	if v, exists := n.Labels[api.LabelNodeIP]; exists {
		return v
	}

	for _, addr := range n.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

func GetNodeSite(n *v1.Node) string {
	if v, exists := n.Labels[api.LabelSite]; exists {
		return v
	}
	return ""
}
