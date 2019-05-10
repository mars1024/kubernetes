package objectmetareconcile

import (
	"io"

	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apimachinery/pkg/api/meta"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	clusterv1alpha1 "gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/cluster/v1alpha1"
)

const (
	PluginName = "ObjectMetaReconcile"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewObjectMetaReconcileAdmissionController(), nil
	})
}

var _ admission.Interface = &ObjectMetaReconcile{}

type ObjectMetaReconcile struct {
	*admission.Handler
	reconcilingAnnotationKeys []string
	reconcilingLabelKeys      []string
}

func NewObjectMetaReconcileAdmissionController() *ObjectMetaReconcile {
	return &ObjectMetaReconcile{
		reconcilingLabelKeys: []string{
			clusterv1alpha1.LabelTenantName,
			clusterv1alpha1.LabelWorkspaceName,
			clusterv1alpha1.LabelClusterName,
			clusterv1alpha1.LabelProvider,
			clusterv1alpha1.LabelCloud,
		},
		reconcilingAnnotationKeys: []string{
			multitenancy.MultiTenancyAnnotationKeyTenantID,
			multitenancy.MultiTenancyAnnotationKeyWorkspaceID,
			multitenancy.MultiTenancyAnnotationKeyClusterID,
			multitenancy.AnnotationCafeMinionClusterID,
		},
		Handler: admission.NewHandler(
			admission.Update,
		),
	}
}

func (r *ObjectMetaReconcile) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return r
}

func (r *ObjectMetaReconcile) Admit(a admission.Attributes) error {
	accessor, err := meta.Accessor(a.GetObject())
	if err != nil {
		return err
	}
	oldAccessor, err := meta.Accessor(a.GetOldObject())
	if err != nil {
		return err
	}

	// reconcile annotations
	annotations := accessor.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	for _, key := range r.reconcilingAnnotationKeys {
		existsInOldObject := false
		if oldAccessor.GetAnnotations() != nil {
			_, existsInOldObject = oldAccessor.GetAnnotations()[key]
		}
		if existsInOldObject {
			annotations[key] = oldAccessor.GetAnnotations()[key]
		} else {
			// Hack: disable aks.cafe.sofastack.io/mc reconcile for data rectification
			if key != multitenancy.AnnotationCafeMinionClusterID {
				delete(annotations, key)
			}
		}
	}
	accessor.SetAnnotations(annotations)

	// reconcile labels
	labels := accessor.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	for _, key := range r.reconcilingLabelKeys {
		existsInOldObject := false
		if oldAccessor.GetLabels() != nil {
			_, existsInOldObject = oldAccessor.GetLabels()[key]
		}
		if existsInOldObject {
			labels[key] = oldAccessor.GetLabels()[key]
		} else {
			delete(labels, key)
		}
	}
	accessor.SetLabels(labels)

	return nil
}
