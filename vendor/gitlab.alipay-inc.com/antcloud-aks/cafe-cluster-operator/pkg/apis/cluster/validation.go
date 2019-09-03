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

package cluster

import (
	"regexp"

	"k8s.io/apimachinery/pkg/util/validation/field"
	validationutil "k8s.io/apimachinery/pkg/util/validation"
	genericvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/api/resource"
	corev1 "k8s.io/api/core/v1"
)

// ValidateMinionClusterName can be used to check whether the given MinionCluster
// name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
const (
	minionClusterNameMaxLen = 255
)

const isNotIntegerErrorMsg = `must be an integer`

var (
	ValidateMinionClusterNameMsg   = "minion cluster name must consist of alphanumeric characters or '-'"
	ValidateMinionClusterNameRegex = regexp.MustCompile(validMinionClusterNameFmt)
	validMinionClusterNameFmt      = `^[a-zA-Z0-9\-]+$`
)

func ValidateBucket(bucket *Bucket) field.ErrorList {
	allErrs := genericvalidation.ValidateObjectMeta(&bucket.ObjectMeta, false, ValidateMinionClusterName, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateBucketSpec(&bucket.Spec, field.NewPath("spec"))...)
	return allErrs
}

func ValidateBucketSpec(spec *BucketSpec, fld *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if !IsValidPriority(spec.Priority) {
		var (
			allPriorities = []string{
				string(SystemTopPriorityBand),
				string(SystemHighPriorityBand),
				string(SystemMediumPriorityBand),
				string(SystemNormalPriorityBand),
				string(SystemLowPriorityBand),
				// NOTE(zuoxiu.jm): Lowest priority is unreachable
				// string(cluster.SystemLowestPriorityBand),
			}
		)
		allErrs = append(allErrs, field.NotSupported(fld.Child("priority"), spec.Priority, allPriorities))
	}
	if spec.Weight < 0 {
		allErrs = append(allErrs, field.Invalid(fld.Child("weight"), spec.Weight, "must be greater than 0"))
	}
	if spec.ReservedQuota < 0 {
		allErrs = append(allErrs, field.Invalid(fld.Child("reservedQuota"), spec.ReservedQuota, "must be greater than 0"))
	}
	if spec.SharedQuota < 0 {
		allErrs = append(allErrs, field.Invalid(fld.Child("sharedQuota"), spec.SharedQuota, "must be greater than 0"))
	}
	return allErrs
}

func IsValidPriority(priority PriorityBand) bool {
	valid := false
	for i := range AllPriorities {
		if AllPriorities[i] == priority {
			valid = true
		}
	}
	return valid
}

func IsValidBucketBindingRuleSubject(sub *BucketBindingRule) bool {
	valid := false
	for i := range AllBucketBindingRuleSubjects {
		if AllBucketBindingRuleSubjects[i] == sub.Field {
			valid = true
		}
	}
	return valid
}

func ValidateBucketBinding(binding *BucketBinding) field.ErrorList {
	allErrs := genericvalidation.ValidateObjectMeta(&binding.ObjectMeta, false, ValidateMinionClusterName, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateBucketBindingSpec(&binding.Spec, field.NewPath("spec"))...)
	return allErrs
}

func ValidateBucketBindingSpec(spec *BucketBindingSpec, fld *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if spec.BucketRef != nil {
		if len(spec.BucketRef.Name) == 0 {
			allErrs = append(allErrs, field.Invalid(fld.Child("bucketRef").Child("name"), spec.BucketRef.Name, "must not be empty"))
		}
	} else {
		allErrs = append(allErrs, field.Required(fld.Child("bucketRef"), "must have a bucket reference"))
	}
	for i := range spec.Rules {
		if !IsValidBucketBindingRuleSubject(spec.Rules[i]) {
			allErrs = append(allErrs, field.NotSupported(fld.Child("rules").Index(i).Child("field"), spec.Rules[i].Field, AllBucketBindingRuleSubjects))
		}
		for j := range spec.Rules[i].Values {
			if len(spec.Rules[i].Values[j]) == 0 {
				allErrs = append(allErrs, field.Invalid(fld.Child("rules").Index(j), spec.Rules[i].Values[j], "must not be empty"))
			}
		}
	}
	return allErrs
}

func ValidateClusterResourceQuota(quota *ClusterResourceQuota) field.ErrorList {
	allErrs := genericvalidation.ValidateObjectMeta(&quota.ObjectMeta, false, ValidateMinionClusterName, field.NewPath("metadata"))

	allErrs = append(allErrs, ValidateResourceQuotaSpec(&quota.Spec.Quota, field.NewPath("spec"))...)
	allErrs = append(allErrs, ValidateResourceQuotaStatus(&quota.Status.Total, field.NewPath("status", "total"))...)
	return allErrs
}

func ValidateResourceQuotaSpec(resourceQuotaSpec *corev1.ResourceQuotaSpec, fld *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	fldPath := fld.Child("hard")
	for k, v := range resourceQuotaSpec.Hard {
		resPath := fldPath.Key(string(k))
		allErrs = append(allErrs, validateResourceName(string(k), resPath)...)
		allErrs = append(allErrs, validateResourceQuantityValue(v, resPath)...)
	}
	return allErrs
}
func ValidateResourceQuotaStatus(status *corev1.ResourceQuotaStatus, fld *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	fldPath := fld.Child("hard")
	for k, v := range status.Hard {
		resPath := fldPath.Key(string(k))
		allErrs = append(allErrs, validateResourceName(string(k), resPath)...)
		allErrs = append(allErrs, validateResourceQuantityValue(v, resPath)...)
	}
	fldPath = fld.Child("used")
	for k, v := range status.Used {
		resPath := fldPath.Key(string(k))
		allErrs = append(allErrs, validateResourceName(string(k), resPath)...)
		allErrs = append(allErrs, validateResourceQuantityValue(v, resPath)...)
	}

	return allErrs
}

func ValidateMinionClusterName(name string, prefix bool) (allErrs []string) {
	if !ValidateMinionClusterNameRegex.MatchString(name) {
		allErrs = append(allErrs, validationutil.RegexError(ValidateMinionClusterNameMsg, validMinionClusterNameFmt, "example-com"))
	}
	if len(name) > minionClusterNameMaxLen {
		allErrs = append(allErrs, validationutil.MaxLenError(minionClusterNameMaxLen))
	}
	return allErrs
}

// Validate compute resource typename.
// Refer to docs/design/resources.md for more details.
func validateResourceName(value string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for _, msg := range validationutil.IsQualifiedName(value) {
		allErrs = append(allErrs, field.Invalid(fldPath, value, msg))
	}
	if len(allErrs) != 0 {
		return allErrs
	}

	return allErrs
}

// ValidateResourceQuantityValue enforces that specified quantity is valid for specified resource
func validateResourceQuantityValue(value resource.Quantity, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateNonnegativeQuantity(value, fldPath)...)
	if value.MilliValue()%int64(1000) != int64(0) {
		allErrs = append(allErrs, field.Invalid(fldPath, value, isNotIntegerErrorMsg))
	}
	return allErrs
}

// Validates that a Quantity is not negative
func validateNonnegativeQuantity(value resource.Quantity, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if value.Cmp(resource.Quantity{}) < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, value.String(), genericvalidation.IsNegativeErrorMsg))
	}
	return allErrs
}
