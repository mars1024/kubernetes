package cluster

import (
	"context"
	"log"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Validate checks that an instance of Bucket is well formed
func (BucketStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*Bucket)
	log.Printf("Validating fields for Bucket %s\n", o.Name)
	// perform validation here and add to errors using field.Invalid
	errors := ValidateBucket(o)
	return errors
}

func (BucketStrategy) NamespaceScoped() bool { return false }

func (BucketStatusStrategy) NamespaceScoped() bool { return false }
