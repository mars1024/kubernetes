/*
Copyright 2018 The Alipay.com Inc Authors.

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

package v1alpha1

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1/validation"
)

// NamespaceScoped is false for ClusterResourceQuota.
func (ClusterResourceQuotaStrategy) NamespaceScoped() bool {
	return false
}

func (ClusterResourceQuotaStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (ClusterResourceQuotaStrategy) AllowUnconditionalUpdate() bool {
	return false
}

// Canonicalize normalizes the object after validation.
func (ClusterResourceQuotaStrategy) Canonicalize(obj runtime.Object) {
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (ClusterResourceQuotaStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	quota := obj.(*cluster.ClusterResourceQuota)
	quota.Status = cluster.ClusterResourceQuotaStatus{}
}

// Validate checks that an instance of ClusterResourceQuota is well formed
func (ClusterResourceQuotaStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*cluster.ClusterResourceQuota)
	allErrs := field.ErrorList{}

	// perform validation here and add to errors using field.Invalid
	allErrs = append(allErrs, validation.ValidateClusterResourceQuota(o)...)
	return allErrs
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (ClusterResourceQuotaStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	curr := obj.(*cluster.ClusterResourceQuota)
	prev := old.(*cluster.ClusterResourceQuota)

	curr.Status = prev.Status
}

func (ClusterResourceQuotaStatusStrategy) NamespaceScoped() bool {
	return false
}

func (ClusterResourceQuotaStatusStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (ClusterResourceQuotaStatusStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (ClusterResourceQuotaStatusStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (ClusterResourceQuotaStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	curr := obj.(*cluster.ClusterResourceQuota)
	prev := obj.(*cluster.ClusterResourceQuota)

	curr.Spec = prev.Spec
}

func (ClusterResourceQuotaStatusStrategy) Canonicalize(obj runtime.Object) {
}
