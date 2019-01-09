package podgc

import (
	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/listers/core/v1"
)

func (c *PodGCController) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.kubeClient = c.kubeClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface)
	copied.podLister = c.podLister.(meta.TenantWise).ShallowCopyWithTenant(tenant).(v1.PodLister)

	// manually generated
	copied.deletePod = func(namespace, name string) error {
		glog.Infof("PodGC is force deleting Pod: %s/%s/%s/%v/%v",
			tenant.GetTenantID(), tenant.GetWorkspaceID(),
			tenant.GetClusterID(), namespace, name)
		return copied.kubeClient.CoreV1().Pods(namespace).Delete(name, metav1.NewDeleteOptions(0))
	}
	return &copied
}
