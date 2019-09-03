package v1alpha1

import (
	"context"
	"log"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/apps"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (s *InPlaceSetStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	allErrs := field.ErrorList{}
	newIps := obj.(*apps.InPlaceSet)
	log.Printf("Validating fields for InPlaceSet %s\n", newIps.Name)

	allErrs = append(allErrs, validateInPlaceSetSpec(&newIps.Spec)...)
	allErrs = append(allErrs, validateInPlaceSetStatus(&newIps.Status)...)

	return allErrs
}

// Validate checks that an instance of InPlaceSet is well formed
func (InPlaceSetStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*apps.InPlaceSet)
	log.Printf("Validating fields for InPlaceSet %s\n", o.Name)
	errors := field.ErrorList{}

	errors = append(errors, validateInPlaceSetSpec(&o.Spec)...)
	errors = append(errors, validateInPlaceSetStatus(&o.Status)...)
	return errors
}

func (InPlaceSetStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {

}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (InPlaceSetStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	ips := obj.(*apps.InPlaceSet)
	ips.Status = apps.InPlaceSetStatus{
		Replicas: ips.Spec.Replicas,
	}
	ips.Generation = 1
}
