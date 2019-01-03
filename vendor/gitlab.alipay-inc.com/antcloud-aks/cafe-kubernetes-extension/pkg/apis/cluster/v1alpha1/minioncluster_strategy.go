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

	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/cluster"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/cluster/v1alpha1/validation"
	"k8s.io/apimachinery/pkg/api/equality"
)

// NamespaceScoped is false for MinionCluster.
func (MinionClusterStrategy) NamespaceScoped() bool {
	return false
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (MinionClusterStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	minionCluster := obj.(*cluster.MinionCluster)
	minionCluster.Status = cluster.MinionClusterStatus{
		Phase: cluster.ClusterInitializing,
	}
	minionCluster.Generation = 1
}

// Validate checks that an instance of MinionCluster is well formed
func (MinionClusterStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*cluster.MinionCluster)
	allErrs := field.ErrorList{}

	// perform validation here and add to errors using field.Invalid
	allErrs = append(allErrs, validation.ValidateMinionCluster(o)...)
	return allErrs
}

// Canonicalize normalizes the object after validation.
func (MinionClusterStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for MinionCluster.
func (MinionClusterStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (MinionClusterStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	oldMinionCluster := old.(*cluster.MinionCluster)
	newMinionCluster := obj.(*cluster.MinionCluster)
	if !equality.Semantic.DeepEqual(oldMinionCluster.Spec, newMinionCluster.Spec) {
		newMinionCluster.Generation++
	}
}

func (MinionClusterStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	allErrs := field.ErrorList{}

	return allErrs
}

func (MinionClusterStatusStrategy) NamespaceScoped() bool {
	return false
}
