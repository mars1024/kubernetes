package clusterinjection

import (
	"fmt"
	"io"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	cafeadmission "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/admission"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	clusterv1alpha1 "gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/cluster/v1alpha1"
	cafeinformers "gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/client/informers_generated/externalversions"
	listers "gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/client/listers_generated/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/admission"
)

const (
	PluginName = "MinionClusterInjection"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewMinionClusterInjectionAdmissionController(), nil
	})
}

func NewMinionClusterInjectionAdmissionController() *MinionClusterInjection {
	return &MinionClusterInjection{
		Handler: admission.NewHandler(admission.Create),
	}
}

type MinionClusterInjection struct {
	//TODO(zuoxiu.jm): Watch tenant model
	*admission.Handler
	MinionClusterLister listers.MinionClusterLister
}

var _ admission.MutationInterface = &MinionClusterInjection{}
var _ multitenancymeta.TenantWise = &MinionClusterInjection{}

var _ = cafeadmission.WantsCafeExtensionKubeInformerFactory(&MinionClusterInjection{})

func (r *MinionClusterInjection) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return r
}

func (r *MinionClusterInjection) SetCafeExtensionKubeInformerFactory(informer cafeinformers.SharedInformerFactory) {
	r.MinionClusterLister = informer.Cluster().V1alpha1().MinionClusters().Lister().(multitenancymeta.TenantWise).ShallowCopyWithTenant(multitenancy.AKSAdminTenant).(listers.MinionClusterLister)
	informer.Start(make(chan struct{}))
}

func (r *MinionClusterInjection) ValidateInitialization() error {
	if r.MinionClusterLister == nil {
		return fmt.Errorf("%s requires a lister", PluginName)
	}
	return nil
}

// Admit makes an admission decision based on the request attributes
func (r *MinionClusterInjection) Admit(a admission.Attributes) error {
	accessor, err := meta.Accessor(a.GetObject())
	if err != nil {
		return err
	}
	if accessor.GetAnnotations() == nil {
		accessor.SetAnnotations(make(map[string]string))
	}

	tenantInfoFromRequestContext, err := multitenancyutil.TransformTenantInfoFromUser(a.GetUserInfo())
	if err != nil {
		return err
	}
	if err := r.setTenantInfo(accessor, tenantInfoFromRequestContext); err != nil {
		return err
	}

	// HACK
	accessor.GetAnnotations()[multitenancy.AnnotationCafeAKSPopulated] = "true"
	return nil
}

func (r *MinionClusterInjection) setTenantInfo(accessor metav1.Object, tenant multitenancy.TenantInfo) error {
	annotations := accessor.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	for k, v := range multitenancyutil.TransformTenantInfoToAnnotations(tenant) {
		annotations[k] = v
	}

	clusters, err := r.MinionClusterLister.List(labels.Everything())
	if err != nil {
		return err
	}

	foundMinionCluster := false
	for _, cluster := range clusters {
		if cluster.Labels[clusterv1alpha1.LabelTenantName] != tenant.GetTenantID() {
			continue
		}
		if cluster.Labels[clusterv1alpha1.LabelWorkspaceName] != tenant.GetWorkspaceID() {
			continue
		}
		if cluster.Labels[clusterv1alpha1.LabelClusterName] != tenant.GetClusterID() {
			continue
		}

		l := accessor.GetLabels()
		if l == nil {
			l = make(map[string]string)
		}
		l[clusterv1alpha1.LabelTenantName] = cluster.Labels[clusterv1alpha1.LabelTenantName]
		l[clusterv1alpha1.LabelWorkspaceName] = cluster.Labels[clusterv1alpha1.LabelWorkspaceName]
		l[clusterv1alpha1.LabelClusterName] = cluster.Labels[clusterv1alpha1.LabelClusterName]
		if cloud, ok := cluster.Labels[clusterv1alpha1.LabelCloud]; ok && len(cloud) != 0 {
			l[clusterv1alpha1.LabelCloud] = cloud
		}
		if provider, ok := cluster.Labels[clusterv1alpha1.LabelProvider]; ok && len(provider) != 0 {
			l[clusterv1alpha1.LabelProvider] = provider
		}
		annotations[multitenancy.AnnotationCafeMinionClusterID] = cluster.Name
		accessor.SetLabels(l)
		foundMinionCluster = true
	}

	if !foundMinionCluster {
		for _, cluster := range clusters {
			if cluster.Labels["cloud.alipay.com/tenant-name"] != tenant.GetTenantID() {
				continue
			}
			if cluster.Labels["cloud.alipay.com/workspace-name"] != tenant.GetWorkspaceID() {
				continue
			}
			if cluster.Labels["cloud.alipay.com/cluster-name"] != tenant.GetClusterID() {
				continue
			}

			l := accessor.GetLabels()
			if l == nil {
				l = make(map[string]string)
			}
			l[clusterv1alpha1.LabelTenantName] = cluster.Labels["cloud.alipay.com/tenant-name"]
			l[clusterv1alpha1.LabelWorkspaceName] = cluster.Labels["cloud.alipay.com/workspace-name"]
			l[clusterv1alpha1.LabelClusterName] = cluster.Labels["cloud.alipay.com/cluster-name"]
			if cloud, ok := cluster.Labels[clusterv1alpha1.LabelCloud]; ok && len(cloud) != 0 {
				l[clusterv1alpha1.LabelCloud] = cloud
			}
			if provider, ok := cluster.Labels[clusterv1alpha1.LabelProvider]; ok && len(provider) != 0 {
				l[clusterv1alpha1.LabelProvider] = provider
			}
			annotations[multitenancy.AnnotationCafeMinionClusterID] = cluster.Name
			accessor.SetLabels(l)
		}
	}

	accessor.SetAnnotations(annotations)
	return nil
}
