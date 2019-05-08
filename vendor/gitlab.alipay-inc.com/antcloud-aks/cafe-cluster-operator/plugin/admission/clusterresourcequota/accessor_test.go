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
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	clientgotesting "k8s.io/client-go/testing"
	utildiff "k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/tools/cache"

	clusterapi "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	quotalister "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/listers_generated/cluster/v1alpha1"
	fakequotaclient "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/clientset_generated/clientset/fake"
)

func TestUpdateQuotaStatus(t *testing.T) {
	testCases := []struct {
		name            string
		availableQuotas func() []*clusterapi.ClusterResourceQuota
		quotaToUpdate   *corev1.ResourceQuota

		expectedQuota func() *clusterapi.ClusterResourceQuota
		expectedError string
	}{
		{
			name: "update properly",
			availableQuotas: func() []*clusterapi.ClusterResourceQuota {
				user1 := defaultQuota()
				user1.Name = "user-one"
				user1.Status.Total.Hard = user1.Spec.Quota.Hard
				user1.Status.Total.Used = corev1.ResourceList{corev1.ResourcePods: resource.MustParse("15")}

				user2 := defaultQuota()
				user2.Name = "user-two"
				user2.Status.Total.Hard = user2.Spec.Quota.Hard
				user2.Status.Total.Used = corev1.ResourceList{corev1.ResourcePods: resource.MustParse("5")}

				return []*clusterapi.ClusterResourceQuota{user1, user2}
			},
			quotaToUpdate: &corev1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "user-one"},
				Spec: corev1.ResourceQuotaSpec{
					Hard: corev1.ResourceList{
						corev1.ResourcePods:    resource.MustParse("10"),
						corev1.ResourceSecrets: resource.MustParse("5"),
					},
				},
				Status: corev1.ResourceQuotaStatus{
					Hard: corev1.ResourceList{
						corev1.ResourcePods:    resource.MustParse("10"),
						corev1.ResourceSecrets: resource.MustParse("5"),
					},
					Used: corev1.ResourceList{
						corev1.ResourcePods: resource.MustParse("20"),
					},
				}},

			expectedQuota: func() *clusterapi.ClusterResourceQuota {
				user1 := defaultQuota()
				user1.Name = "user-one"
				user1.Status.Total.Hard = user1.Spec.Quota.Hard
				user1.Status.Total.Used = corev1.ResourceList{corev1.ResourcePods: resource.MustParse("20")}
				return user1
			},
		},
	}

	for _, tc := range testCases {
		quotaIndexer := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})
		availableQuotas := tc.availableQuotas()
		var objs []runtime.Object
		for i := range availableQuotas {
			quotaIndexer.Add(availableQuotas[i])
			objs = append(objs, availableQuotas[i])
		}
		quotaLister := quotalister.NewClusterResourceQuotaLister(quotaIndexer)

		client := fakequotaclient.NewSimpleClientset(objs...)

		accessor := newQuotaAccessor(quotaLister, client.ClusterV1alpha1())

		actualErr := accessor.UpdateQuotaStatus(tc.quotaToUpdate)
		switch {
		case len(tc.expectedError) == 0 && actualErr == nil:
		case len(tc.expectedError) == 0 && actualErr != nil:
			t.Errorf("%s: unexpected error: %v", tc.name, actualErr)
			continue
		case len(tc.expectedError) != 0 && actualErr == nil:
			t.Errorf("%s: missing expected error: %v", tc.name, tc.expectedError)
			continue
		case len(tc.expectedError) != 0 && actualErr != nil && !strings.Contains(actualErr.Error(), tc.expectedError):
			t.Errorf("%s: expected %v, got %v", tc.name, tc.expectedError, actualErr)
			continue
		}

		var actualQuota *clusterapi.ClusterResourceQuota
		for _, action := range client.Actions() {
			updateAction, ok := action.(clientgotesting.UpdateActionImpl)
			if !ok {
				continue
			}
			if updateAction.Matches("update", "clusterresourcequotas") && updateAction.Subresource == "status" {
				actualQuota = updateAction.GetObject().(*clusterapi.ClusterResourceQuota)
				break
			}
		}

		if !equality.Semantic.DeepEqual(tc.expectedQuota(), actualQuota) {
			t.Errorf("%s: %v", tc.name, utildiff.ObjectDiff(tc.expectedQuota(), actualQuota))
			continue
		}
	}
}

func defaultQuota() *clusterapi.ClusterResourceQuota {
	return &clusterapi.ClusterResourceQuota{
		ObjectMeta: metav1.ObjectMeta{Name: "foo"},
		Spec: clusterapi.ClusterResourceQuotaSpec{
			Quota: corev1.ResourceQuotaSpec{
				Hard: corev1.ResourceList{
					corev1.ResourcePods:    resource.MustParse("10"),
					corev1.ResourceSecrets: resource.MustParse("5"),
				},
			},
		},
	}
}

func TestGetQuota(t *testing.T) {
	testCases := []struct {
		name             string
		availableQuotas  func() []*clusterapi.ClusterResourceQuota
		requestedCluster string

		expectedQuotas func() []*corev1.ResourceQuota
		expectedError  string
	}{
		{
			name: "no hits",
			availableQuotas: func() []*clusterapi.ClusterResourceQuota {
				return nil
			},
			requestedCluster: "foo",

			expectedQuotas: func() []*corev1.ResourceQuota {
				return nil
			},
			expectedError: "not found",
		},
		{
			name: "correct quota and namespaces",
			availableQuotas: func() []*clusterapi.ClusterResourceQuota {
				user1 := defaultQuota()
				user1.Name = "one"
				user1.Status.Total.Hard = user1.Spec.Quota.Hard
				user1.Status.Total.Used = corev1.ResourceList{corev1.ResourcePods: resource.MustParse("15")}

				user2 := defaultQuota()
				user2.Name = "two"
				user2.Status.Total.Hard = user2.Spec.Quota.Hard
				user2.Status.Total.Used = corev1.ResourceList{corev1.ResourcePods: resource.MustParse("5")}

				return []*clusterapi.ClusterResourceQuota{user1, user2}
			},
			requestedCluster: "one",

			expectedQuotas: func() []*corev1.ResourceQuota {
				return []*corev1.ResourceQuota{
					{
						ObjectMeta: metav1.ObjectMeta{Namespace: "one", Name: "one"},
						Spec: corev1.ResourceQuotaSpec{
							Hard: corev1.ResourceList{
								corev1.ResourcePods:    resource.MustParse("10"),
								corev1.ResourceSecrets: resource.MustParse("5"),
							},
						},
						Status: corev1.ResourceQuotaStatus{
							Hard: corev1.ResourceList{
								corev1.ResourcePods:    resource.MustParse("10"),
								corev1.ResourceSecrets: resource.MustParse("5"),
							},
							Used: corev1.ResourceList{
								corev1.ResourcePods: resource.MustParse("15"),
							},
						},
					},
				}
			},
		},
	}

	for _, tc := range testCases {
		quotaIndexer := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})
		availableQuotas := tc.availableQuotas()
		for i := range availableQuotas {
			quotaIndexer.Add(availableQuotas[i])
		}
		quotaLister := quotalister.NewClusterResourceQuotaLister(quotaIndexer)

		client := fakequotaclient.NewSimpleClientset()

		accessor := newQuotaAccessor(quotaLister, client.ClusterV1alpha1())

		actualQuotas, actualErr := accessor.GetQuotas(tc.requestedCluster)
		switch {
		case len(tc.expectedError) == 0 && actualErr == nil:
		case len(tc.expectedError) == 0 && actualErr != nil:
			t.Errorf("%s: unexpected error: %v", tc.name, actualErr)
			continue
		case len(tc.expectedError) != 0 && actualErr == nil:
			t.Errorf("%s: missing expected error: %v", tc.name, tc.expectedError)
			continue
		case len(tc.expectedError) != 0 && actualErr != nil && !strings.Contains(actualErr.Error(), tc.expectedError):
			t.Errorf("%s: expected %v, got %v", tc.name, tc.expectedError, actualErr)
			continue
		}

		if len(tc.expectedError) != 0 {
			continue
		}

		if tc.expectedQuotas == nil {
			continue
		}

		var actualQuotaPointers []*corev1.ResourceQuota
		for i := range actualQuotas {
			actualQuotaPointers = append(actualQuotaPointers, &actualQuotas[i])
		}

		expectedQuotas := tc.expectedQuotas()
		if !equality.Semantic.DeepEqual(expectedQuotas, actualQuotaPointers) {
			t.Errorf("%s: expectedLen: %v actualLen: %v", tc.name, len(expectedQuotas), len(actualQuotas))
			continue
		}
	}
}
