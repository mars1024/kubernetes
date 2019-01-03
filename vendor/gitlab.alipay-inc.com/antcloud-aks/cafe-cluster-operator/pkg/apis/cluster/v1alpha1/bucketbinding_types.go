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

// BucketBinding
// +k8s:openapi-gen=true
// +resource:path=bucketbindings,strategy=BucketBindingStrategy
type BucketBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BucketBindingSpec   `json:"spec,omitempty"`
	Status BucketBindingStatus `json:"status,omitempty"`
}

// BucketBindingSpec defines the desired state of BucketBinding
type BucketBindingSpec struct {
	Rules     []*BucketBindingRule `json:"rules,omitempty"`
	BucketRef *BucketReference     `json:"bucketRef,omitempty"`
}

type BucketReference struct {
	Name string `json:"name"`
}

type BucketBindingRule struct {
	Field  string   `json:"field,omitempty"`
	Values []string `json:"values,omitempty"`
}

// BucketBindingStatus defines the observed state of BucketBinding
type BucketBindingStatus struct {
	Phase string `json:"phase,omitempty"`
}

// Validate checks that an instance of BucketBinding is well formed
func (BucketBindingStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*cluster.BucketBinding)
	log.Printf("Validating fields for BucketBinding %s\n", o.Name)
	// perform validation here and add to errors using field.Invalid
	errors := validation.ValidateBucketBinding(o)
	return errors
}

func (BucketBindingStrategy) NamespaceScoped() bool { return false }

func (BucketBindingStatusStrategy) NamespaceScoped() bool { return false }

// DefaultingFunction sets default BucketBinding field values
func (BucketBindingSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*BucketBinding)
	// set default field values here
	log.Printf("Defaulting fields for BucketBinding %s\n", obj.Name)
}
