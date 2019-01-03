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
	"log"
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1/validation"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// Bucket
// +k8s:openapi-gen=true
// +resource:path=buckets,strategy=BucketStrategy
type Bucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BucketSpec   `json:"spec,omitempty"`
	Status BucketStatus `json:"status,omitempty"`
}

type PriorityBand string

// BucketSpec defines the desired state of Bucket
type BucketSpec struct {
	ReservedQuota int          `json:"reservedQuota"`
	SharedQuota   int          `json:"sharedQuota"`
	Priority      PriorityBand `json:"priority"`
	Weight        int          `json:"weight"`
}

// BucketStatus defines the observed state of Bucket
type BucketStatus struct {
	Phase string `json:"phase,omitempty"`
}

// Validate checks that an instance of Bucket is well formed
func (BucketStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*cluster.Bucket)
	log.Printf("Validating fields for Bucket %s\n", o.Name)
	// perform validation here and add to errors using field.Invalid
	errors := validation.ValidateBucket(o)
	return errors
}

func (BucketStrategy) NamespaceScoped() bool { return false }

func (BucketStatusStrategy) NamespaceScoped() bool { return false }

// DefaultingFunction sets default Bucket field values
func (BucketSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*Bucket)
	// set default field values here
	obj.Spec.Priority = SystemLowPriorityBand
	log.Printf("Defaulting fields for Bucket %s\n", obj.Name)
}
