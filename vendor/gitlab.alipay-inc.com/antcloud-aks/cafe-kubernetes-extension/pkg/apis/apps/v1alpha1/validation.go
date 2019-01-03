package v1alpha1

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	unversionedvalidation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	utilfeature "k8s.io/apiserver/pkg/util/feature"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/apps"
)

const (
	AppArmor utilfeature.Feature = "AppArmor"
)

func validateInPlaceSetSpec(spec *apps.InPlaceSetSpec) field.ErrorList {
	errors := field.ErrorList{}

	p := field.NewPath("spec")
	if spec.Replicas < 0 {
		errors = append(errors, field.Invalid(p.Child("replicas"), spec.Replicas, fmt.Sprintf("replicas (%d) should not be less than 0", spec.Replicas)))
	}

	if spec.MinReadySeconds < 0 {
		errors = append(errors, field.Invalid(p.Child("minReadySeconds"), spec.MinReadySeconds, fmt.Sprintf("minReadySeconds (%d) should not be less than 0", spec.MinReadySeconds)))
	}

	errors = append(errors, validateInPlaceSetStrategy(&spec.Strategy, p)...)

	if len(spec.Selector.MatchLabels) == 0 && len(spec.Selector.MatchExpressions) == 0 {
		errors = append(errors, field.Invalid(p.Child("selector"), spec.Selector, fmt.Sprintf("selector should be provided")))
	}
	selector, err := metav1.LabelSelectorAsSelector(&spec.Selector)
	if err != nil {
		errors = append(errors, field.Invalid(p.Child("selector"), spec.Selector, fmt.Sprintf("invalid label selector: %s", err)))
	}

	errors = append(errors, ValidatePodTemplateSpecForApps(&spec.Template, selector, spec.Replicas, p.Child("template"))...)

	return errors
}

// Validates the given template and ensures that it is in accordance with the desired selector and replicas.
func ValidatePodTemplateSpecForApps(template *corev1.PodTemplateSpec, selector labels.Selector, replicas int32, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if template == nil {
		allErrs = append(allErrs, field.Required(fldPath, ""))
	} else {
		if !selector.Empty() {
			// Verify that the ReplicaSet selector matches the labels in template.
			labels := labels.Set(template.Labels)
			if !selector.Matches(labels) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("metadata", "labels"), template.Labels, "`selector` does not match template `labels`"))
			}
		}
		allErrs = append(allErrs, ValidatePodTemplateSpec(template, fldPath)...)
		if replicas > 1 {
			allErrs = append(allErrs, ValidateReadOnlyPersistentDisks(template.Spec.Volumes, fldPath.Child("spec", "volumes"))...)
		}
		// RestartPolicy has already been first-order validated as per ValidatePodTemplateSpec().
		if template.Spec.RestartPolicy != corev1.RestartPolicyAlways {
			allErrs = append(allErrs, field.NotSupported(fldPath.Child("spec", "restartPolicy"), template.Spec.RestartPolicy, []string{string(corev1.RestartPolicyAlways)}))
		}
		if template.Spec.ActiveDeadlineSeconds != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("spec", "activeDeadlineSeconds"), template.Spec.ActiveDeadlineSeconds, "must not be specified"))
		}
	}
	return allErrs
}

func ValidateReadOnlyPersistentDisks(volumes []corev1.Volume, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for i := range volumes {
		vol := &volumes[i]
		idxPath := fldPath.Index(i)
		if vol.GCEPersistentDisk != nil {
			if vol.GCEPersistentDisk.ReadOnly == false {
				allErrs = append(allErrs, field.Invalid(idxPath.Child("gcePersistentDisk", "readOnly"), false, "must be true for replicated pods > 1; GCE PD can only be mounted on multiple machines if it is read-only"))
			}
		}
		// TODO: What to do for AWS?  It doesn't support replicas
	}
	return allErrs
}

// ValidatePodTemplateSpec validates the spec of a pod template
func ValidatePodTemplateSpec(spec *corev1.PodTemplateSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, unversionedvalidation.ValidateLabels(spec.Labels, fldPath.Child("labels"))...)
	allErrs = append(allErrs, ValidateAnnotations(spec.Annotations, fldPath.Child("annotations"))...)
	allErrs = append(allErrs, ValidatePodSpecificAnnotations(spec.Annotations, &spec.Spec, fldPath.Child("annotations"))...)
	//allErrs = append(allErrs, ValidatePodSpec(&spec.Spec, fldPath.Child("spec"))...)
	return allErrs
}

func podSpecHasContainer(spec *corev1.PodSpec, containerName string) bool {
	for _, c := range spec.InitContainers {
		if c.Name == containerName {
			return true
		}
	}
	for _, c := range spec.Containers {
		if c.Name == containerName {
			return true
		}
	}
	return false
}

func ValidatePodSpecificAnnotations(annotations map[string]string, spec *corev1.PodSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if value, isMirror := annotations[corev1.MirrorPodAnnotationKey]; isMirror {
		if len(spec.NodeName) == 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Key(corev1.MirrorPodAnnotationKey), value, "must set spec.nodeName if mirror pod annotation is set"))
		}
	}

	if annotations[corev1.TolerationsAnnotationKey] != "" {
		allErrs = append(allErrs, ValidateTolerationsInPodAnnotations(annotations, fldPath)...)
	}

	allErrs = append(allErrs, ValidateSeccompPodAnnotations(annotations, fldPath)...)
	//allErrs = append(allErrs, ValidateAppArmorPodAnnotations(annotations, spec, fldPath)...)

	return allErrs
}

// This validate will make sure targetPath:
// 1. is not abs path
// 2. does not have any element which is ".."
func validateLocalDescendingPath(targetPath string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if path.IsAbs(targetPath) {
		allErrs = append(allErrs, field.Invalid(fldPath, targetPath, "must be a relative path"))
	}

	allErrs = append(allErrs, validatePathNoBacksteps(targetPath, fldPath)...)

	return allErrs
}

// validatePathNoBacksteps makes sure the targetPath does not have any `..` path elements when split
//
// This assumes the OS of the apiserver and the nodes are the same. The same check should be done
// on the node to ensure there are no backsteps.
func validatePathNoBacksteps(targetPath string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	parts := strings.Split(filepath.ToSlash(targetPath), "/")
	for _, item := range parts {
		if item == ".." {
			allErrs = append(allErrs, field.Invalid(fldPath, targetPath, "must not contain '..'"))
			break // even for `../../..`, one error is sufficient to make the point
		}
	}
	return allErrs
}

func ValidateSeccompProfile(p string, fldPath *field.Path) field.ErrorList {
	if p == corev1.SeccompProfileRuntimeDefault || p == corev1.DeprecatedSeccompProfileDockerDefault {
		return nil
	}
	if p == "unconfined" {
		return nil
	}
	if strings.HasPrefix(p, "localhost/") {
		return validateLocalDescendingPath(strings.TrimPrefix(p, "localhost/"), fldPath)
	}
	return field.ErrorList{field.Invalid(fldPath, p, "must be a valid seccomp profile")}
}

func ValidateSeccompPodAnnotations(annotations map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if p, exists := annotations[corev1.SeccompPodAnnotationKey]; exists {
		allErrs = append(allErrs, ValidateSeccompProfile(p, fldPath.Child(corev1.SeccompPodAnnotationKey))...)
	}
	for k, p := range annotations {
		if strings.HasPrefix(k, corev1.SeccompContainerAnnotationKeyPrefix) {
			allErrs = append(allErrs, ValidateSeccompProfile(p, fldPath.Child(k))...)
		}
	}

	return allErrs
}

// ValidateTolerationsInPodAnnotations tests that the serialized tolerations in Pod.Annotations has valid data
func ValidateTolerationsInPodAnnotations(annotations map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	tolerations, err := GetTolerationsFromPodAnnotations(annotations)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath, corev1.TolerationsAnnotationKey, err.Error()))
		return allErrs
	}

	if len(tolerations) > 0 {
		allErrs = append(allErrs, ValidateTolerations(tolerations, fldPath.Child(corev1.TolerationsAnnotationKey))...)
	}

	return allErrs
}

// ValidateTolerations tests if given tolerations have valid data.
func ValidateTolerations(tolerations []corev1.Toleration, fldPath *field.Path) field.ErrorList {
	allErrors := field.ErrorList{}
	for i, toleration := range tolerations {
		idxPath := fldPath.Index(i)
		// validate the toleration key
		if len(toleration.Key) > 0 {
			allErrors = append(allErrors, unversionedvalidation.ValidateLabelName(toleration.Key, idxPath.Child("key"))...)
		}

		// empty toleration key with Exists operator and empty value means match all taints
		if len(toleration.Key) == 0 && toleration.Operator != corev1.TolerationOpExists {
			allErrors = append(allErrors, field.Invalid(idxPath.Child("operator"), toleration.Operator,
				"operator must be Exists when `key` is empty, which means \"match all values and all keys\""))
		}

		if toleration.TolerationSeconds != nil && toleration.Effect != corev1.TaintEffectNoExecute {
			allErrors = append(allErrors, field.Invalid(idxPath.Child("effect"), toleration.Effect,
				"effect must be 'NoExecute' when `tolerationSeconds` is set"))
		}

		// validate toleration operator and value
		switch toleration.Operator {
		// empty operator means Equal
		case corev1.TolerationOpEqual, "":
			if errs := validation.IsValidLabelValue(toleration.Value); len(errs) != 0 {
				allErrors = append(allErrors, field.Invalid(idxPath.Child("operator"), toleration.Value, strings.Join(errs, ";")))
			}
		case corev1.TolerationOpExists:
			if len(toleration.Value) > 0 {
				allErrors = append(allErrors, field.Invalid(idxPath.Child("operator"), toleration, "value must be empty when `operator` is 'Exists'"))
			}
		default:
			validValues := []string{string(corev1.TolerationOpEqual), string(corev1.TolerationOpExists)}
			allErrors = append(allErrors, field.NotSupported(idxPath.Child("operator"), toleration.Operator, validValues))
		}

		// validate toleration effect, empty toleration effect means match all taint effects
		if len(toleration.Effect) > 0 {
			allErrors = append(allErrors, validateTaintEffect(&toleration.Effect, true, idxPath.Child("effect"))...)
		}
	}
	return allErrors
}

func validateTaintEffect(effect *corev1.TaintEffect, allowEmpty bool, fldPath *field.Path) field.ErrorList {
	if !allowEmpty && len(*effect) == 0 {
		return field.ErrorList{field.Required(fldPath, "")}
	}

	allErrors := field.ErrorList{}
	switch *effect {
	// TODO: Replace next line with subsequent commented-out line when implement TaintEffectNoScheduleNoAdmit.
	case corev1.TaintEffectNoSchedule, corev1.TaintEffectPreferNoSchedule, corev1.TaintEffectNoExecute:
		// case core.TaintEffectNoSchedule, core.TaintEffectPreferNoSchedule, core.TaintEffectNoScheduleNoAdmit, core.TaintEffectNoExecute:
	default:
		validValues := []string{
			string(corev1.TaintEffectNoSchedule),
			string(corev1.TaintEffectPreferNoSchedule),
			string(corev1.TaintEffectNoExecute),
			// TODO: Uncomment this block when implement TaintEffectNoScheduleNoAdmit.
			// string(core.TaintEffectNoScheduleNoAdmit),
		}
		allErrors = append(allErrors, field.NotSupported(fldPath, *effect, validValues))
	}
	return allErrors
}

// GetTolerationsFromPodAnnotations gets the json serialized tolerations data from Pod.Annotations
// and converts it to the []Toleration type in core.
func GetTolerationsFromPodAnnotations(annotations map[string]string) ([]corev1.Toleration, error) {
	var tolerations []corev1.Toleration
	if len(annotations) > 0 && annotations[corev1.TolerationsAnnotationKey] != "" {
		err := json.Unmarshal([]byte(annotations[corev1.TolerationsAnnotationKey]), &tolerations)
		if err != nil {
			return tolerations, err
		}
	}
	return tolerations, nil
}

// ValidateAnnotations validates that a set of annotations are correctly defined.
func ValidateAnnotations(annotations map[string]string, fldPath *field.Path) field.ErrorList {
	return apimachineryvalidation.ValidateAnnotations(annotations, fldPath)
}

func validateInPlaceSetStrategy(strategy *apps.UpgradeStrategy, path *field.Path) field.ErrorList {
	errors := field.ErrorList{}

	if strategy.Partition < 0 {
		errors = append(errors, field.Invalid(path.Child("partition"), strategy.Partition, fmt.Sprintf("partition (%d) should not be less than 0", strategy.Partition)))
	}

	return errors
}

func validateInPlaceSetStatus(status *apps.InPlaceSetStatus) field.ErrorList {
	errors := field.ErrorList{}

	if status.Replicas < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "replicas"), status.Replicas, fmt.Sprintf("replicas (%d) should not be less than 0", status.Replicas)))
	}

	if status.AvailableReplicas < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "availableReplicas"), status.AvailableReplicas, fmt.Sprintf("availableReplicas (%d) should not be less than 0", status.AvailableReplicas)))
	}

	if status.FullyLabeledReplicas < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "fullyLabeledReplicas"), status.FullyLabeledReplicas, fmt.Sprintf("fullyLabeledReplicas (%d) should not be less than 0", status.FullyLabeledReplicas)))
	}

	if status.ObservedGeneration < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "observedGeneration"), status.ObservedGeneration, fmt.Sprintf("observedGeneration (%d) should not be less than 0", status.ObservedGeneration)))
	}

	if status.UpdatedReadyReplicas < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "updatedReadyReplicas"), status.UpdatedReadyReplicas, fmt.Sprintf("updatedReadyReplicas (%d) should not be less than 0", status.UpdatedReadyReplicas)))
	}

	if status.UpdatedAvailableReplicas < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "updatedAvailableReplicas"), status.UpdatedAvailableReplicas, fmt.Sprintf("updatedAvailableReplicas (%d) should not be less than 0", status.UpdatedAvailableReplicas)))
	}

	if status.UpdatedReplicas < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "updatedReplicas"), status.UpdatedReplicas, fmt.Sprintf("updatedReplicas (%d) should not be less than 0", status.UpdatedReplicas)))
	}

	return errors
}

func validateCafeDeploymentSpec(spec *apps.CafeDeploymentSpec) field.ErrorList {
	errors := field.ErrorList{}
	p := field.NewPath("spec")

	if spec.Replicas < 0 {
		errors = append(errors, field.Invalid(p.Child("replicas"), spec.Replicas, fmt.Sprintf("replicas (%d) should not be less than 0", spec.Replicas)))
	}

	if spec.HistoryLimit < 0 || spec.HistoryLimit > 50 {
		errors = append(errors, field.Invalid(p.Child("historyLimit"), spec.HistoryLimit, fmt.Sprintf("historyLimit (%d) should be between 0 and 50", spec.HistoryLimit)))
	}

	if len(spec.Selector.MatchLabels) == 0 && len(spec.Selector.MatchExpressions) == 0 {
		errors = append(errors, field.Invalid(p.Child("selector"), spec.Selector, fmt.Sprintf("selector should be provided")))
	}
	selector, err := metav1.LabelSelectorAsSelector(&spec.Selector)
	if err != nil {
		errors = append(errors, field.Invalid(p.Child("selector"), spec.Selector, fmt.Sprintf("invalid label selector: %s", err)))
	}

	errors = append(errors, ValidatePodTemplateSpecForApps(&spec.Template, selector, spec.Replicas, p.Child("template"))...)

	errors = append(errors, validateCafeDeploymentStrategy(&spec.Strategy, p.Child("strategy"))...)

	errors = append(errors, validateCafeDeploymentTopology(spec, p.Child("topology"))...)
	return errors
}

func validateCafeDeplloymentTopologyUnitReplicas(spec *apps.CafeDeploymentSpec, path *field.Path) field.ErrorList {
	errors := field.ErrorList{}

	unitReplicas := spec.Topology.UnitReplicas
	if len(unitReplicas) == 0 {
		return errors
	}

	unitIds := sets.String{}
	for _, unitId := range spec.Topology.Values {
		unitIds.Insert(unitId)
	}

	unitNums := map[string]int32{}
	for unitId, intOrStr := range unitReplicas {
		if !unitIds.Has(unitId) {
			errors = append(errors, field.Invalid(path.Child("unitReplicas"), spec.Topology.UnitReplicas, fmt.Sprintf("undeclared unit id %s", unitId)))
			return errors
		}
		unitIds.Delete(unitId)

		replicas, err := ParseUnitReplicas(spec.Replicas, intOrStr)
		if err != nil {
			errors = append(errors, field.Invalid(path.Child("unitReplicas"), spec.Topology.UnitReplicas, fmt.Sprintf("wrong unit replicas for unit %s: %s", unitId, err.Error())))
			return errors
		}

		if replicas < 0 {
			errors = append(errors, field.Invalid(path.Child("unitReplicas"), spec.Topology.UnitReplicas, fmt.Sprintf("wrong unit replicas for unit %s: replicas should not less than 0", unitId)))
			return errors
		}

		unitNums[unitId] = replicas
	}

	var summary int32 = 0
	for _, replicas := range unitNums {
		summary += replicas
	}

	if summary > spec.Replicas {
		errors = append(errors, field.Invalid(path.Child("unitReplicas"), spec.Topology.UnitReplicas, fmt.Sprintf("the summary of each unit replicas exceeds the spec.replicas")))
		return errors
	}

	fullyConfiged := unitIds.Len() == 0
	if fullyConfiged {
		if summary < spec.Replicas {
			errors = append(errors, field.Invalid(path.Child("unitReplicas"), spec.Topology.UnitReplicas, fmt.Sprintf("the summary of each unit replicas is less than the spec.replicas")))
			return errors
		}
	}

	return errors
}

func validateCafeDeploymentStrategy(strategy *apps.CafeDeploymentUpgradeStrategy, path *field.Path) field.ErrorList {
	errors := field.ErrorList{}

	if strategy.MinReadySeconds < 0 {
		errors = append(errors, field.Invalid(path.Child("minReadySeconds"), strategy.MinReadySeconds, fmt.Sprintf("minReadySeconds (%d) should not be less than 0", strategy.MinReadySeconds)))
	}

	if strategy.BatchSize != nil {
		if *strategy.BatchSize < 0 {
			errors = append(errors, field.Invalid(path.Child("batchSize"), strategy.BatchSize, fmt.Sprintf("batchSize (%d) should not be less than 0", strategy.BatchSize)))
		}
	}

	if strategy.UpgradeType != apps.UpgradeBeta && strategy.UpgradeType != apps.UpgradeBatch {
		errors = append(errors, field.Invalid(path.Child("upgradeType"), strategy.UpgradeType, fmt.Sprintf("upgradeType (%s) should be one of (%s)", strategy.UpgradeType, UpgradeBeta + ", " + UpgradeBatch)))
	}

	return errors
}

func validateCafeDeploymentTopology(spec *apps.CafeDeploymentSpec, path *field.Path) field.ErrorList {
	errors := field.ErrorList{}
	topo := spec.Topology

	for _, unitName := range topo.Values {
		errors = append(errors, ValidateUnitName(unitName, path.Child("values"))...)
	}

	if topo.UnitType != apps.UnitTypeCell && topo.UnitType != apps.UnitTypeZone {
		errors = append(errors, field.Invalid(path.Child("UnitType"), topo.UnitType, fmt.Sprintf("unitType (%s) should be one of (%s)", topo.UnitType, UnitTypeCell + ", " + UnitTypeZone)))
	}

	errors = append(errors, validateCafeDeplloymentTopologyUnitReplicas(spec, path)...)

	if spec.Topology.AutoReschedule != nil {
		errors = append(errors, validateAutoRescheduleConfig(spec.Topology.AutoReschedule, path.Child("autoReschedule"))...)
	}

	return errors
}

func validateAutoRescheduleConfig(config *apps.AutoScheduleConfig, path *field.Path) field.ErrorList {
	errors := field.ErrorList{}

	if config.InitialDelaySeconds != nil {
		if *config.InitialDelaySeconds < 0 {
			errors = append(errors, field.Invalid(path.Child("initialDelaySeconds"), config.InitialDelaySeconds, fmt.Sprintf("initialDelaySeconds (%d) should not be less than 0", *config.InitialDelaySeconds)))
		}
	}

	return errors
}

func ValidateUnitName(value string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if strings.Trim(value, " ") == "" {
		allErrs = append(allErrs, field.Invalid(fldPath, value, "unit name should not be empty"))
	}
	return allErrs
}

func validateCafeDeploymentStatus(status *apps.CafeDeploymentStatus) field.ErrorList {
	errors := field.ErrorList{}

	if status.Replicas < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "replicas"), status.Replicas, fmt.Sprintf("replicas (%d) should not be less than 0", status.Replicas)))
	}

	if status.AvailableReplicas < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "availableReplicas"), status.AvailableReplicas, fmt.Sprintf("availableReplicas (%d) should not be less than 0", status.AvailableReplicas)))
	}

	if status.FullyLableledReplicas < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "fullyLabeledReplicas"), status.FullyLableledReplicas, fmt.Sprintf("fullyLabeledReplicas (%d) should not be less than 0", status.FullyLableledReplicas)))
	}

	if status.ObservedGeneration < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "observedGeneration"), status.ObservedGeneration, fmt.Sprintf("observedGeneration (%d) should not be less than 0", status.ObservedGeneration)))
	}

	if status.UpdatedReadyReplicas < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "updatedReadyReplicas"), status.UpdatedReadyReplicas, fmt.Sprintf("updatedReadyReplicas (%d) should not be less than 0", status.UpdatedReadyReplicas)))
	}

	if status.UpdatedAvailableReplicas < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "updatedAvailableReplicas"), status.UpdatedAvailableReplicas, fmt.Sprintf("updatedAvailableReplicas (%d) should not be less than 0", status.UpdatedAvailableReplicas)))
	}

	if status.UpdatedReplicas < 0 {
		errors = append(errors, field.Invalid(field.NewPath("status", "updatedReplicas"), status.UpdatedReplicas, fmt.Sprintf("updatedReplicas (%d) should not be less than 0", status.UpdatedReplicas)))
	}

	return errors
}
