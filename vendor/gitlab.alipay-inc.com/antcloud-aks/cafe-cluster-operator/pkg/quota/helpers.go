/*
Copyright 2019 The Alipay.com Inc Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package quota

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"

	clusterapi "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
)

func GetObjectMatcher(selector clusterapi.ClusterResourceQuotaSelector) (func(obj metav1.Object) (bool, error), error) {
	var annotationSelector map[string]string
	if len(selector.AnnotationSelector) > 0 {
		// ensure our matcher has a stable copy of the map
		annotationSelector = make(map[string]string, len(selector.AnnotationSelector))
		for k, v := range selector.AnnotationSelector {
			annotationSelector[k] = v
		}
	}

	return func(obj metav1.Object) (bool, error) {
		if annotationSelector != nil {
			objAnnotations := obj.GetAnnotations()
			for k, v := range annotationSelector {
				if objValue, exists := objAnnotations[k]; !exists || objValue != v {
					return false, nil
				}
			}
		}

		return true, nil
	}, nil
}

func GetResourceQuotasStatusByNamespace(status clusterapi.ClusterResourceQuotaStatus, namespace string) (corev1.ResourceQuotaStatus, bool) {
	for i := range status.Namespaces {
		curr := status.Namespaces[i]
		if curr.Name == namespace {
			return curr.Status, true
		}
	}
	return corev1.ResourceQuotaStatus{}, false
}

func UpdateResourceQuotaStatus(status *clusterapi.ClusterResourceQuotaStatus, newStatus clusterapi.NamespaceResourceQuotaStatus) {
	var newNamespaceStatuses []clusterapi.NamespaceResourceQuotaStatus
	found := false
	for i := range status.Namespaces {
		curr := status.Namespaces[i]
		if curr.Name == newStatus.Name {
			newNamespaceStatuses = append(newNamespaceStatuses, newStatus)
			found = true
			continue
		}
		newNamespaceStatuses = append(newNamespaceStatuses, curr)
	}
	if !found {
		newNamespaceStatuses = append(newNamespaceStatuses, newStatus)
	}
	status.Namespaces = newNamespaceStatuses
}
