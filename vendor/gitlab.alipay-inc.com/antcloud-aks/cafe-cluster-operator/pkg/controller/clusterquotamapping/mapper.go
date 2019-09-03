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

package clusterquotamapping

import (
	"sync"
	"reflect"

	"k8s.io/apimachinery/pkg/util/sets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/equality"

	clusterapi "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
)

const (
	AnnotationCafeMinionCluster = "aks.cafe.sofastack.io/mc"
)

type ClusterQuotaMapper interface {
	// GetClusterQuotasFor returns the list of clusterquota names that this namespace matches.  It also
	// returns the selectionFields associated with the namespace for the check so that callers can determine staleness
	GetClusterQuotasFor(namespaceName string) ([]string, SelectionFields)
	// GetNamespacesFor returns the list of namespace names that this cluster quota matches.  It also
	// returns the selector associated with the clusterquota for the check so that callers can determine staleness
	GetNamespacesFor(quotaName string) ([]string, clusterapi.ClusterResourceQuotaSelector)

	AddListener(listener MappingChangeListener)
}

type MappingChangeListener interface {
	AddMapping(quotaName, namespaceName string)
	RemoveMapping(quotaName, namespaceName string)
}

type SelectionFields struct {
	Annotations map[string]string
}

type clusterQuotaMapper struct {
	lock sync.RWMutex

	// requiredQuotaToSelector indicates the latest label selector this controller has observed for a quota
	requiredQuotaToSelector map[string]clusterapi.ClusterResourceQuotaSelector
	// requiredNamespaceToLabels indicates the latest selectionFields this controller has observed for a namespace
	requiredNamespaceToLabels map[string]SelectionFields
	// completedQuotaToSelector indicates the latest label selector this controller has scanned against namespaces
	completedQuotaToSelector map[string]clusterapi.ClusterResourceQuotaSelector
	// completedNamespaceToLabels indicates the latest selectionFields this controller has scanned against cluster quotas
	completedNamespaceToLabels map[string]SelectionFields

	quotaToNamespaces map[string]sets.String
	namespaceToQuota  map[string]sets.String

	listeners []MappingChangeListener
}

func NewClusterQuotaMapper() *clusterQuotaMapper {
	return &clusterQuotaMapper{
		requiredQuotaToSelector:    map[string]clusterapi.ClusterResourceQuotaSelector{},
		requiredNamespaceToLabels:  map[string]SelectionFields{},
		completedQuotaToSelector:   map[string]clusterapi.ClusterResourceQuotaSelector{},
		completedNamespaceToLabels: map[string]SelectionFields{},

		quotaToNamespaces: map[string]sets.String{},
		namespaceToQuota:  map[string]sets.String{},
	}
}

func (m *clusterQuotaMapper) GetClusterQuotasFor(namespaceName string) ([]string, SelectionFields) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	quotas, ok := m.namespaceToQuota[namespaceName]
	if !ok {
		return []string{}, m.completedNamespaceToLabels[namespaceName]
	}
	return quotas.List(), m.completedNamespaceToLabels[namespaceName]
}

func (m *clusterQuotaMapper) GetNamespacesFor(quotaName string) ([]string, clusterapi.ClusterResourceQuotaSelector) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	namespaces, ok := m.quotaToNamespaces[quotaName]
	if !ok {
		return []string{}, m.completedQuotaToSelector[quotaName]
	}
	return namespaces.List(), m.completedQuotaToSelector[quotaName]
}

func (m *clusterQuotaMapper) AddListener(listener MappingChangeListener) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.listeners = append(m.listeners, listener)
}

func (m *clusterQuotaMapper) requireQuota(quota *clusterapi.ClusterResourceQuota) bool {
	m.lock.RLock()
	selector, exists := m.requiredQuotaToSelector[quota.Name]
	m.lock.RUnlock()

	if selectorMatches(selector, exists, quota) {
		return false
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	selector, exists = m.requiredQuotaToSelector[quota.Name]
	if selectorMatches(selector, exists, quota) {
		return false
	}

	m.requiredQuotaToSelector[quota.Name] = quota.Spec.Selector
	return true
}

// completeQuota updates the latest selector used to generate the mappings for this quota.  The value is returned
// by the Get methods for the mapping so that callers can determine staleness
func (m *clusterQuotaMapper) completeQuota(quota *clusterapi.ClusterResourceQuota) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.completedQuotaToSelector[quota.Name] = quota.Spec.Selector
}

// removeQuota deletes a quota from all mappings
func (m *clusterQuotaMapper) removeQuota(quotaName string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.requiredQuotaToSelector, quotaName)
	delete(m.completedQuotaToSelector, quotaName)
	delete(m.quotaToNamespaces, quotaName)
	for namespaceName, quotas := range m.namespaceToQuota {
		if quotas.Has(quotaName) {
			quotas.Delete(quotaName)
			for _, listener := range m.listeners {
				listener.RemoveMapping(quotaName, namespaceName)
			}
		}
	}
}

// requireNamespace updates the label requirements for the given namespace.  This prevents stale updates to the mapping itself.
// returns true if a modification was made
func (m *clusterQuotaMapper) requireNamespace(namespace metav1.Object) bool {
	m.lock.RLock()
	fullName := getNamespaceFullName(namespace)
	selectionFields, exists := m.requiredNamespaceToLabels[fullName]
	m.lock.RUnlock()

	if selectionFieldsMatch(selectionFields, exists, namespace) {
		return false
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	selectionFields, exists = m.requiredNamespaceToLabels[fullName]
	if selectionFieldsMatch(selectionFields, exists, namespace) {
		return false
	}

	m.requiredNamespaceToLabels[fullName] = GetSelectionFields(namespace)
	return true
}

// completeNamespace updates the latest selectionFields used to generate the mappings for this namespace.  The value is returned
// by the Get methods for the mapping so that callers can determine staleness
func (m *clusterQuotaMapper) completeNamespace(namespace metav1.Object) {
	m.lock.Lock()
	defer m.lock.Unlock()
	fullName := getNamespaceFullName(namespace)
	m.completedNamespaceToLabels[fullName] = GetSelectionFields(namespace)
}

// removeNamespace deletes a namespace from all mappings
func (m *clusterQuotaMapper) removeNamespace(namespaceName string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.requiredNamespaceToLabels, namespaceName)
	delete(m.completedNamespaceToLabels, namespaceName)
	delete(m.namespaceToQuota, namespaceName)
	for quotaName, namespaces := range m.quotaToNamespaces {
		if namespaces.Has(namespaceName) {
			namespaces.Delete(namespaceName)
			for _, listener := range m.listeners {
				listener.RemoveMapping(quotaName, namespaceName)
			}
		}
	}
}

func selectorMatches(selector clusterapi.ClusterResourceQuotaSelector, exists bool, quota *clusterapi.ClusterResourceQuota) bool {
	return exists && equality.Semantic.DeepEqual(selector, quota.Spec.Selector)
}

func selectionFieldsMatch(selectionFields SelectionFields, exists bool, namespace metav1.Object) bool {
	return exists && reflect.DeepEqual(selectionFields, GetSelectionFields(namespace))
}

// setMapping maps (or removes a mapping) between a clusterquota and a namespace
// It returns whether the action worked, whether the quota is out of date, whether the namespace is out of date
// This allows callers to decide whether to pull new information from the cache or simply skip execution
func (m *clusterQuotaMapper) setMapping(quota *clusterapi.ClusterResourceQuota, namespace metav1.Object) (bool /*added*/ , bool /*quota matches*/ , bool /*namespace matches*/) {
	fullName := getNamespaceFullName(namespace)
	m.lock.RLock()
	selector, selectorExists := m.requiredQuotaToSelector[quota.Name]
	selectionFields, selectionFieldsExist := m.requiredNamespaceToLabels[fullName]
	m.lock.RUnlock()

	if !selectorMatches(selector, selectorExists, quota) {
		return false, false, selectionFieldsMatch(selectionFields, selectionFieldsExist, namespace)
	}
	if !selectionFieldsMatch(selectionFields, selectionFieldsExist, namespace) {
		return false, true, false
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	selector, selectorExists = m.requiredQuotaToSelector[quota.Name]
	selectionFields, selectionFieldsExist = m.requiredNamespaceToLabels[fullName]
	if !selectorMatches(selector, selectorExists, quota) {
		return false, false, selectionFieldsMatch(selectionFields, selectionFieldsExist, namespace)
	}
	if !selectionFieldsMatch(selectionFields, selectionFieldsExist, namespace) {
		return false, true, false
	}

	mutated := false

	namespaces, ok := m.quotaToNamespaces[quota.Name]
	if !ok {
		mutated = true
		m.quotaToNamespaces[quota.Name] = sets.NewString(fullName)
	} else {
		mutated = !namespaces.Has(fullName)
		namespaces.Insert(fullName)
	}

	quotas, ok := m.namespaceToQuota[fullName]
	if !ok {
		mutated = true
		m.namespaceToQuota[fullName] = sets.NewString(quota.Name)
	} else {
		mutated = mutated || !quotas.Has(quota.Name)
		quotas.Insert(quota.Name)
	}

	if mutated {
		for _, listener := range m.listeners {
			listener.AddMapping(quota.Name, fullName)
		}
	}

	return true, true, true
}

func GetSelectionFields(namespace metav1.Object) SelectionFields {
	mc, _ := namespace.GetAnnotations()[AnnotationCafeMinionCluster]
	return SelectionFields{Annotations: map[string]string{
		AnnotationCafeMinionCluster: mc,
	}}
}
