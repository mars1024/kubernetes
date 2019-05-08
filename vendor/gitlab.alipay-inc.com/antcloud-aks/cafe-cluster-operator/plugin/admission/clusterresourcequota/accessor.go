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

package clusterresourcequota

import (
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apiserver/pkg/storage/etcd"

	quotalister "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/listers_generated/cluster/v1alpha1"
	clusterapi "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	clustertypedclient "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
	utilquota "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/quota"

	"github.com/hashicorp/golang-lru"
	"github.com/golang/glog"
)

type QuotaAccessor interface {
	// UpdateQuotaStatus is called to persist final status.  This method should write to persistent storage.
	// An error indicates that write didn't complete successfully.
	UpdateQuotaStatus(newQuota *v1.ResourceQuota) error
	// GetQuota returns the matched cluster resource quota
	GetQuotas(cluster, namespace string) ([]v1.ResourceQuota, error)
}

type clusterQuotaAccessor struct {
	client kubernetes.Interface

	clusterQuotaLister quotalister.ClusterResourceQuotaLister
	clusterQuotaClient clustertypedclient.ClusterResourceQuotasGetter

	liveLookupCache *lru.Cache
	liveTTL         time.Duration
	// updatedQuotas holds a cache of quotas that we've updated.  This is used to pull the "really latest" during back to
	// back quota evaluations that touch the same quota doc.  This only works because we can compare etcd resourceVersions
	// for the same resource as integers.  Before this change: 22 updates with 12 conflicts.  after this change: 15 updates with 0 conflicts
	updatedClusterQuotas *lru.Cache
}

// newQuotaAccessor creates an object that conforms to the QuotaAccessor interface to be used to retrieve quota objects.
func newQuotaAccessor(
	clusterQuotaLister quotalister.ClusterResourceQuotaLister,
	clusterQuotaClient clustertypedclient.ClusterResourceQuotasGetter,
) *clusterQuotaAccessor {
	updatedCache, err := lru.New(100)
	if err != nil {
		// this should never happen
		panic(err)
	}

	return &clusterQuotaAccessor{
		clusterQuotaLister:   clusterQuotaLister,
		clusterQuotaClient:   clusterQuotaClient,
		updatedClusterQuotas: updatedCache,
	}
}

// UpdateQuotaStatus the newQuota coming in will be incremented from the original.  The difference between the original
// and the new is the amount to add to the namespace total, but the total status is the used value itself
func (e *clusterQuotaAccessor) UpdateQuotaStatus(newQuota *v1.ResourceQuota) error {
	clusterQuota, err := e.clusterQuotaLister.Get(newQuota.Name)
	if err != nil {
		return err
	}
	clusterQuota = e.checkCache(clusterQuota)

	// re-assign objectmeta
	// make a copy
	clusterQuota = clusterQuota.DeepCopy()
	clusterQuota.ObjectMeta = newQuota.ObjectMeta
	clusterQuota.Namespace = ""

	// determine change in usage
	usageDiff := utilquota.Subtract(newQuota.Status.Used, clusterQuota.Status.Total.Used)

	// update aggregate usage
	clusterQuota.Status.Total.Used = newQuota.Status.Used

	// update per namespace totals
	oldNamespaceTotals, _ := utilquota.GetResourceQuotasStatusByNamespace(clusterQuota.Status, newQuota.Namespace)
	namespaceTotalCopy := oldNamespaceTotals.DeepCopy()
	newNamespaceTotals := *namespaceTotalCopy
	newNamespaceTotals.Used = utilquota.Add(oldNamespaceTotals.Used, usageDiff)
	utilquota.UpdateResourceQuotaStatus(&clusterQuota.Status, clusterapi.NamespaceResourceQuotaStatus{
		Name:   newQuota.Namespace,
		Status: newNamespaceTotals,
	})

	glog.Infof("update cluster quota %s stats %+v", clusterQuota.Name, clusterQuota)
	updatedQuota, err := e.clusterQuotaClient.ClusterResourceQuotas().UpdateStatus(clusterQuota)
	if err != nil {
		return err
	}

	e.updatedClusterQuotas.Add(clusterQuota.Name, updatedQuota)
	return nil
}

var etcdVersioner = etcd.APIObjectVersioner{}

// checkCache compares the passed quota against the value in the look-aside cache and returns the newer
// if the cache is out of date, it deletes the stale entry.  This only works because of etcd resourceVersions
// being monotonically increasing integers
func (e *clusterQuotaAccessor) checkCache(clusterQuota *clusterapi.ClusterResourceQuota) *clusterapi.ClusterResourceQuota {
	uncastCachedQuota, ok := e.updatedClusterQuotas.Get(clusterQuota.Name)
	if !ok {
		return clusterQuota
	}
	cachedQuota := uncastCachedQuota.(*clusterapi.ClusterResourceQuota)

	if etcdVersioner.CompareResourceVersion(clusterQuota, cachedQuota) >= 0 {
		e.updatedClusterQuotas.Remove(clusterQuota.Name)
		return clusterQuota
	}
	return cachedQuota
}

func (e *clusterQuotaAccessor) GetQuotas(cluster, namespace string) ([]v1.ResourceQuota, error) {
	clusterQuota, err := e.clusterQuotaLister.Get(cluster)
	if err != nil {
		return nil, err
	}

	clusterQuota = e.checkCache(clusterQuota)

	// now convert to ResourceQuota
	convertedQuota := v1.ResourceQuota{}
	convertedQuota.ObjectMeta = clusterQuota.ObjectMeta
	convertedQuota.Namespace = namespace
	convertedQuota.Spec = clusterQuota.Spec.Quota
	convertedQuota.Status = clusterQuota.Status.Total
	return []v1.ResourceQuota{convertedQuota}, nil
}
