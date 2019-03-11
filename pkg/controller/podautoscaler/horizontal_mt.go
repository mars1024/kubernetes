package podautoscaler

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	autoscalinglisters "k8s.io/client-go/listers/autoscaling/v1"
	autoscalingclient "k8s.io/client-go/kubernetes/typed/autoscaling/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/scale"
)

func (a *HorizontalController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *a
	copied.hpaLister = a.hpaLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(autoscalinglisters.HorizontalPodAutoscalerLister)
	copied.podLister = a.podLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(corelisters.PodLister)
	copied.hpaNamespacer = a.hpaNamespacer.(meta.TenantWise).ShallowCopyWithTenant(tenant).(autoscalingclient.HorizontalPodAutoscalersGetter)
	copied.scaleNamespacer = a.scaleNamespacer.(meta.TenantWise).ShallowCopyWithTenant(tenant).(scale.ScalesGetter)
	copied.replicaCalc = a.ShallowCopyWithTenant(tenant).(*ReplicaCalculator)
	return &copied
}
