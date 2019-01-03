package v1alpha1

import (
	"context"
	"fmt"
	"log"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/apps"
)

func (s *CafeDeploymentStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	allErrs := field.ErrorList{}
	oldCd := old.(*apps.CafeDeployment)
	newCd := obj.(*apps.CafeDeployment)
	log.Printf("Validating fields for CafeDeployment %s\n", newCd.Name)

	allErrs = append(allErrs, validateCafeDeploymentSpec(&newCd.Spec)...)

	allErrs = append(allErrs, validateCafeDeploymentStatus(&newCd.Status)...)

	allErrs = append(allErrs, s.validatePodSpecUpdate(&oldCd.Spec.Template.Spec, &newCd.Spec.Template.Spec, field.NewPath("spec", "template", "spec"))...)
	return allErrs
}

// reference https://yuque.antfin-inc.com/sigma.pouch/sigma3.x/vwl7ue#api-validation-changes
func (CafeDeploymentStrategy) validatePodSpecUpdate(oldSpec, newSpec *corev1.PodSpec, p *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if !reflect.DeepEqual(oldSpec.NodeSelector, newSpec.NodeSelector) {
		allErrs = append(allErrs, field.Invalid(p.Child("nodeSelector"), newSpec.NodeSelector, fmt.Sprintf("nodeSelector (from %v to %v) should not be updated", oldSpec.NodeSelector, newSpec.NodeSelector)))
	}

	if oldSpec.NodeName != newSpec.NodeName {
		allErrs = append(allErrs, field.Invalid(p.Child("nodeName"), newSpec.NodeName, fmt.Sprintf("nodeName (from %s to %s) should not be updated", oldSpec.NodeName, newSpec.NodeName)))
	}

	if oldSpec.HostNetwork != newSpec.HostNetwork {
		allErrs = append(allErrs, field.Invalid(p.Child("hostNetwork"), newSpec.HostNetwork, fmt.Sprintf("hostNetwork (from %t to %t) should not be updated", oldSpec.HostNetwork, newSpec.HostNetwork)))
	}

	if oldSpec.HostIPC != newSpec.HostIPC {
		allErrs = append(allErrs, field.Invalid(p.Child("hostIPC"), newSpec.HostIPC, fmt.Sprintf("hostIPC (from %t to %t) should not be updated", oldSpec.HostIPC, newSpec.HostIPC)))
	}

	if oldSpec.HostPID != newSpec.HostPID {
		allErrs = append(allErrs, field.Invalid(p.Child("hostPID"), newSpec.HostPID, fmt.Sprintf("hostPID (from %t to %t) should not be updated", oldSpec.HostPID, newSpec.HostPID)))
	}

	if oldSpec.ShareProcessNamespace != nil && newSpec.ShareProcessNamespace != nil {
		if *oldSpec.ShareProcessNamespace != *newSpec.ShareProcessNamespace {
			allErrs = append(allErrs, field.Invalid(p.Child("shareProcessNamespace"), newSpec.ShareProcessNamespace, fmt.Sprintf("shareProcessNamespace (from %t to %t) should not be updated", *oldSpec.ShareProcessNamespace, *newSpec.ShareProcessNamespace)))
		}
	} else if oldSpec.ShareProcessNamespace != nil || newSpec.ShareProcessNamespace != nil {
		allErrs = append(allErrs, field.Invalid(p.Child("shareProcessNamespace"), newSpec.ShareProcessNamespace, fmt.Sprintf("shareProcessNamespace should not be updated")))
	}

	if oldSpec.Hostname != newSpec.Hostname {
		allErrs = append(allErrs, field.Invalid(p.Child("hostname"), newSpec.Hostname, fmt.Sprintf("hostname (from %s to %s) should not be updated", oldSpec.Hostname, newSpec.Hostname)))
	}

	if oldSpec.Subdomain != newSpec.Subdomain {
		allErrs = append(allErrs, field.Invalid(p.Child("subdomain"), newSpec.Subdomain, fmt.Sprintf("subdomain (from %s to %s) should not be updated", oldSpec.Subdomain, newSpec.Subdomain)))
	}

	newContainers := map[string]*corev1.Container{}

	for _, container := range newSpec.Containers {
		newContainers[container.Name] = &container
	}

	for _, oldContainer := range oldSpec.Containers {
		if _, exist := newContainers[oldContainer.Name]; !exist {
			allErrs = append(allErrs, field.Invalid(p.Child("containers"), newSpec.Containers, fmt.Sprintf("containers should not remove old container %s", oldContainer.Name)))
		}
	}

	return allErrs
}

func (CafeDeploymentStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	cd := obj.(*apps.CafeDeployment)
	oldCd := old.(*apps.CafeDeployment)
	// remove affinity
	cd.Spec.Template.Spec.Affinity = nil

	if !equality.Semantic.DeepEqual(oldCd.Spec, cd.Spec) {
		cd.Generation = oldCd.Generation + 1
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (CafeDeploymentStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	cd := obj.(*apps.CafeDeployment)

	// remove affinity
	cd.Spec.Template.Spec.Affinity = nil

	cd.Status = apps.CafeDeploymentStatus{
		Replicas: cd.Spec.Replicas,
	}
	cd.Generation = 1
}

// Validate checks that an instance of CafeDeployment is well formed
func (CafeDeploymentStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*apps.CafeDeployment)
	log.Printf("Validating fields for CafeDeployment %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	errors = append(errors, validateCafeDeploymentSpec(&o.Spec)...)
	errors = append(errors, validateCafeDeploymentStatus(&o.Status)...)
	return errors
}
