/*
Copyright 2017 The Kubernetes Authors.

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

// This file was autogenerated by apiregister-gen. Do not edit it manually!

package apis

import (
	"github.com/kubernetes-incubator/apiserver-builder-alpha/pkg/builders"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster"
	clusterv1alpha1 "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
)

// GetAllApiBuilders returns all known APIGroupBuilders
// so they can be registered with the apiserver
func GetAllApiBuilders() []*builders.APIGroupBuilder {
	return []*builders.APIGroupBuilder{
		GetClusterAPIBuilder(),
	}
}

var clusterApiGroup = builders.NewApiGroupBuilder(
	"cluster.aks.cafe.sofastack.io",
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster").
	WithUnVersionedApi(cluster.ApiVersion).
	WithVersionedApis(
		clusterv1alpha1.ApiVersion,
	).
	WithRootScopedKinds(
		"Bucket",
		"BucketBinding",
		"ClusterResourceQuota",
	)

func GetClusterAPIBuilder() *builders.APIGroupBuilder {
	return clusterApiGroup
}
