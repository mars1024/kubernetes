package cluster

import (
	"log"
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Validate checks that an instance of BucketBinding is well formed
func (BucketBindingStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*BucketBinding)
	log.Printf("Validating fields for BucketBinding %s\n", o.Name)
	// perform validation here and add to errors using field.Invalid
	errors := ValidateBucketBinding(o)
	return errors
}

func (BucketBindingStrategy) NamespaceScoped() bool { return false }

func (BucketBindingStatusStrategy) NamespaceScoped() bool { return false }
