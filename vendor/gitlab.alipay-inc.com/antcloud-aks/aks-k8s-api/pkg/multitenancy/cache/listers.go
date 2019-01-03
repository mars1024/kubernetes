// +build multitenancy

package cache

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

func ListAllWithTenant(index cache.Indexer, selector labels.Selector, tenant multitenancy.TenantInfo, appendFunc cache.AppendFunc) error {
	items, err := index.Index(TenantIndex, &metav1.ObjectMeta{Annotations: util.TransformTenantInfoToAnnotations(tenant)})
	if err != nil {
		return err
	}
	for _, m := range items {
		metadata, err := meta.Accessor(m)
		if err != nil {
			return err
		}
		if selector.Matches(labels.Set(metadata.GetLabels())) {
			appendFunc(m)
		}
	}
	return nil
}

func ListAllByNamespaceWithTenant(index cache.Indexer, namespace string, selector labels.Selector, tenant multitenancy.TenantInfo, appendFunc cache.AppendFunc) error {
	items, err := index.Index(TenantNamespaceIndex, &metav1.ObjectMeta{
		Namespace:   namespace,
		Annotations: util.TransformTenantInfoToAnnotations(tenant),
	})
	if err != nil {
		return err
	}
	for _, m := range items {
		metadata, err := meta.Accessor(m)
		if err != nil {
			return err
		}
		if selector.Matches(labels.Set(metadata.GetLabels())) {
			appendFunc(m)
		}
	}
	return nil
}
