package util

import (
	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
)

func TenantNodeNameFromPod(pod *v1.Pod) types.NodeName {
	nodeName := pod.Spec.NodeName
	if utilfeature.DefaultFeatureGate.Enabled(multitenancy.FeatureName) {
		tenant, err := multitenancyutil.TransformTenantInfoFromAnnotations(pod.Annotations)
		if err == nil && nodeName != "" {
			nodeName = multitenancyutil.TransformTenantInfoToJointString(tenant, "/") + "/" + nodeName
			glog.V(5).Infof("transform nodeName to tenant based: %s", nodeName)
		}
	}
	return types.NodeName(nodeName)
}
