package util

import (
	"github.com/golang/glog"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TenantNodeNameFromPod(pod *v1.Pod) types.NodeName {
	tenant, err := multitenancyutil.TransformTenantInfoFromAnnotations(pod.Annotations)
	nodeName := pod.Spec.NodeName
	if err == nil && nodeName != "" {
		nodeName = multitenancyutil.TransformTenantInfoToJointString(tenant, "/") + "/" + nodeName
		glog.V(5).Infof("transform nodeName to tenant based: %s", nodeName)

	}
	return types.NodeName(nodeName)
}
