/*
Copyright 2015 The Kubernetes Authors.

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

// Package SigmaScheduling contains an admission controller that checks and modifies every new Pod
package sigmascheduling

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/golang/glog"
	sigmaapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	core "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	unversionedvalidation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/core/validation"
	kubeletapis "k8s.io/kubernetes/pkg/kubelet/apis"
)

// PluginName indicates name of admission plugin.
const PluginName = "SigmaScheduling"

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewContainerState(), nil
	})
}

// SigmaScheduling is an implementation of admission.Interface.
// It admits and validate Pod AllocSpec, Node LocalInfos and LogicInfos.
type SigmaScheduling struct {
	*admission.Handler
}

var _ admission.MutationInterface = &SigmaScheduling{}
var _ admission.ValidationInterface = &SigmaScheduling{}

// Admit makes an admission decision based on the request attributes
func (a *SigmaScheduling) Admit(attributes admission.Attributes) (err error) {
	if shouldIgnore(attributes) {
		return nil
	}

	op := attributes.GetOperation()
	if op != admission.Create && op != admission.Update {
		return apierrors.NewBadRequest("SigmaScheduling Admission only handles Create and Update event")
	}

	r := attributes.GetResource().GroupResource()
	if r == api.Resource("pods") {
		return admitPod(attributes)
	}

	return nil
}

// Validate makes sure that all containers are set to correct SigmaScheduling labels
func (n *SigmaScheduling) Validate(attributes admission.Attributes) (err error) {
	if shouldIgnore(attributes) {
		return nil
	}

	op := attributes.GetOperation()
	if op != admission.Create && op != admission.Update {
		return apierrors.NewBadRequest("SigmaScheduling Admission only handles Create and Update event")
	}

	r := attributes.GetResource().GroupResource()
	if r == api.Resource("pods") {
		return validatePod(attributes)
	} else if r == api.Resource("nodes") {
		return validateNode(attributes)
	}

	return apierrors.NewBadRequest("Resource was marked with kind Pod or Node but was unable to be converted")
}

func admitPod(attributes admission.Attributes) error {
	op := attributes.GetOperation()
	pod, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	if priorityString, _ := pod.Annotations[sigmaapi.AnnotationNetPriority]; priorityString == "" {
		pod.Annotations[sigmaapi.AnnotationNetPriority] = "5"
	}
	allocSpecString, ok := pod.Annotations[sigmaapi.AnnotationPodAllocSpec]
	if !ok {
		// annotation not found
		return nil
	}

	var allocSpec sigmaapi.AllocSpec
	if err := json.Unmarshal([]byte(allocSpecString), &allocSpec); err != nil {
		return admission.NewForbidden(attributes,
			fmt.Errorf("can not %s due to annotation %s json unmarshal error `%s`", op, sigmaapi.AnnotationPodAllocSpec, err))
	}
	for idx, c := range allocSpec.Containers {
		var found bool
		for _, ctx := range pod.Spec.Containers {
			if ctx.Name == c.Name {
				found = true
				break
			}
		}
		if !found {
			err := fmt.Errorf("container %s not found", c.Name)
			return admission.NewForbidden(attributes,
				fmt.Errorf("can not %s due to annotation %s json unmarshal error `%s`", op, sigmaapi.AnnotationPodAllocSpec, err))
		}
		if c.Resource.CPU.CPUSet != nil && c.Resource.CPU.CPUSet.SpreadStrategy == "" {
			allocSpec.Containers[idx].Resource.CPU.CPUSet.SpreadStrategy = sigmaapi.SpreadStrategySpread
		}
		if c.Resource.GPU.ShareMode == "" {
			allocSpec.Containers[idx].Resource.GPU.ShareMode = sigmaapi.GPUShareModeExclusive
		}
	}
	allocSpecBytes, err := json.Marshal(&allocSpec)
	if err != nil {
		return admission.NewForbidden(attributes,
			fmt.Errorf("can not %s due to annotation %s json unmarshal error `%s`", op, sigmaapi.AnnotationPodAllocSpec, err))
	}
	pod.Annotations[sigmaapi.AnnotationPodAllocSpec] = string(allocSpecBytes)
	return nil
}

func validatePod(attributes admission.Attributes) error {
	op := attributes.GetOperation()
	pod, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	glog.V(10).Infof("SigmaScheduling validateUpdate pod: %#v", pod)
	if pod.Annotations == nil {
		return nil
	}

	allocSpecString, ok := pod.Annotations[sigmaapi.AnnotationPodAllocSpec]
	if !ok {
		return nil
	}
	var allocSpec sigmaapi.AllocSpec
	if err := json.Unmarshal([]byte(allocSpecString), &allocSpec); err != nil {
		return admission.NewForbidden(attributes,
			fmt.Errorf("can not %s due to annotation %s json unmarshal error `%s`", op, sigmaapi.AnnotationPodAllocSpec, err))
	}

	// start to validate
	allErrs := field.ErrorList{}

	// Validate inplace update annotation.
	if isInInplaceUpdateProcess(pod) {
		state, _ := pod.Annotations[sigmaapi.AnnotationPodInplaceUpdateState]
		switch state {
		case sigmaapi.InplaceUpdateStateCreated, sigmaapi.InplaceUpdateStateAccepted:
		case sigmaapi.InplaceUpdateStateFailed, sigmaapi.InplaceUpdateStateSucceeded:
		default:
			fld := field.NewPath("annotations").Child("inplace-update-state")
			expectValues := fmt.Sprintf("[%s, %s, %s, %s]", sigmaapi.InplaceUpdateStateCreated, sigmaapi.InplaceUpdateStateAccepted,
				sigmaapi.InplaceUpdateStateFailed, sigmaapi.InplaceUpdateStateSucceeded)
			allErrs = append(allErrs, field.Invalid(fld, sigmaapi.AnnotationPodInplaceUpdateState, expectValues))
		}
	}

	if af := allocSpec.Affinity; af != nil {
		if af.PodAntiAffinity != nil {
			fldPath := field.NewPath("allocSpec").Child("affinity").Child("podAntiAffinity")
			// TODO: Uncomment below code once RequiredDuringSchedulingRequiredDuringExecution is implemented.
			// if podAntiAffinity.RequiredDuringSchedulingRequiredDuringExecution != nil {
			//	allErrs = append(allErrs, validatePodAffinityTerms(podAntiAffinity.RequiredDuringSchedulingRequiredDuringExecution, false,
			//		fldPath.Child("requiredDuringSchedulingRequiredDuringExecution"))...)
			//}
			if af.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
				allErrs = append(allErrs, validatePodAffinityTerms(af.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
					fldPath.Child("requiredDuringSchedulingIgnoredDuringExecution"))...)
			}

			for _, v := range af.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
				if r := utilvalidation.IsInRange(int(v.MaxPercent), 0, 100); r != nil {
					fld := fldPath.Child("requiredDuringSchedulingIgnoredDuringExecution").Child("maxPercent")
					allErrs = append(allErrs, field.Invalid(fld, v.MaxPercent, r[0]))
				}
			}

			if af.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
				allErrs = append(allErrs, validateWeightedPodAffinityTerms(af.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
					fldPath.Child("preferredDuringSchedulingIgnoredDuringExecution"))...)
			}

			for _, v := range af.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
				if r := utilvalidation.IsInRange(int(v.MaxPercent), 0, 100); r != nil {
					fld := fldPath.Child("preferredDuringSchedulingIgnoredDuringExecution").Child("maxPercent")
					allErrs = append(allErrs, field.Invalid(fld, v.MaxPercent, r[0]))
				}
			}
		}
		if af.CPUAntiAffinity != nil {
			fldPath := field.NewPath("allocSpec").Child("affinity").Child("cpuAntiAffinity")
			if af.CPUAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
				allErrs = append(allErrs, validateCPUWeightedPodAffinityTerms(af.CPUAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
					fldPath.Child("preferredDuringSchedulingIgnoredDuringExecution"))...)
			}
		}
	}

	containerField := field.NewPath("allocSpec").Child("containers")
	for i, container := range allocSpec.Containers {
		var found bool
		var idx int
		for i, c1 := range pod.Spec.Containers {
			if container.Name == c1.Name {
				found = true
				idx = i
			}
		}
		if !found {
			allErrs = append(allErrs, field.Invalid(containerField.Index(i), container.Name, "container not found"))
		}

		if container.Resource.CPU.CPUSet != nil {
			// validate that if the pod is created with CPUIDs, the NodeName must be specified.
			if len(container.Resource.CPU.CPUSet.CPUIDs) > 0 && op == admission.Create {
				if len(pod.Spec.NodeName) == 0 {
					fld := containerField.Index(i).Child("resource").Child("cpu").Child("cpuset").Child("cpuIDs")
					allErrs = append(allErrs, field.Invalid(fld, fmt.Sprintf("%v", container.Resource.CPU.CPUSet.CPUIDs),
						fmt.Sprintf("the pod is created with specified CPUIDs, but the NodeName of this pod is not specified")))
				}
			}

			// validate cpuIDs is not duplicated
			if ids := findDuplicatedCPUIDs(container.Resource.CPU.CPUSet); len(ids) > 0 {
				fld := containerField.Index(i).Child("resource").Child("cpu").Child("cpuset").Child("cpuIDs")
				allErrs = append(allErrs, field.Invalid(fld, fmt.Sprintf("%v", container.Resource.CPU.CPUSet.CPUIDs),
					fmt.Sprintf("duplicity cpuIDs `%s`", strings.Join(ids, ", "))))
			}

			// validate cpuIDs count
			milliValue := pod.Spec.Containers[idx].Resources.Requests.Cpu().MilliValue()
			count := milliValue / 1000
			fractionalValue := milliValue % 1000
			if fractionalValue == 0 {
				c := len(container.Resource.CPU.CPUSet.CPUIDs)
				if c > 0 && c != int(count) && !isInInplaceUpdateProcess(pod) {
					fld := containerField.Index(i).Child("resource").Child("cpu").Child("cpuset").Child("cpuIDs")
					allErrs = append(allErrs, field.Invalid(fld, fmt.Sprintf("%v", container.Resource.CPU.CPUSet.CPUIDs),
						fmt.Sprintf("the count of cpuIDs is not match pod spec and this pod is not in inplace update process")))
				}
			} else {
				fld := field.NewPath("spec").Child("containers").Index(i).Child("resources").Child("requests").Child("cpu")
				allErrs = append(allErrs, field.Invalid(fld, fmt.Sprintf("%v", pod.Spec.Containers[idx].Resources.Requests.Cpu().String()),
					fmt.Sprintf("pod spec is invalid, must be integer")))
			}

			switch container.Resource.CPU.CPUSet.SpreadStrategy {
			case sigmaapi.SpreadStrategySameCoreFirst, sigmaapi.SpreadStrategySpread:
			default:
				fld := containerField.Index(i).Child("resource").Child("cpu").Child("cpuset").Child("spreadStrategy")
				expectValues := fmt.Sprintf("[%s, %s]", sigmaapi.SpreadStrategySameCoreFirst, sigmaapi.SpreadStrategySpread)
				allErrs = append(allErrs, field.Invalid(fld, string(container.Resource.CPU.CPUSet.SpreadStrategy), expectValues))
			}
		}

		switch container.Resource.GPU.ShareMode {
		case sigmaapi.GPUShareModeExclusive:
		default:
			fld := containerField.Index(i).Child("resource").Child("gpu").Child("shareMode")
			allErrs = append(allErrs, field.Invalid(fld, string(container.Resource.GPU.ShareMode), ""))
		}
	}

	if priorityString, _ := pod.Annotations[sigmaapi.AnnotationNetPriority]; priorityString != "" {
		if priority, err := strconv.Atoi(priorityString); err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("net-priority"), priorityString, "net-priority must be integer"))
		} else if priority < 0 || priority > 15 {
			allErrs = append(allErrs, field.Invalid(field.NewPath("net-priority"), priorityString, "net-priority must be with range of 0-15"))
		}
	}

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(attributes.GetKind().GroupKind(), pod.ObjectMeta.GetName(), allErrs)
	}

	// skip for create
	if op == admission.Create {
		return nil
	}

	// all fields can not update except cpuID
	oldPod, ok := attributes.GetOldObject().(*api.Pod)
	if !ok || oldPod == nil {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	oldAllocSpecString, ok := oldPod.Annotations[sigmaapi.AnnotationPodAllocSpec]
	if !ok {
		return nil
	}
	var oldAllocSpec sigmaapi.AllocSpec
	if err := json.Unmarshal([]byte(oldAllocSpecString), &oldAllocSpec); err != nil {
		return admission.NewForbidden(attributes,
			fmt.Errorf("can not %s annotation %s due to json unmarshal error `%s`", op, sigmaapi.AnnotationPodAllocSpec, err))
	}

	for i, container := range oldAllocSpec.Containers {
		if container.Resource.CPU.CPUSet != nil && allocSpec.Containers[i].Resource.CPU.CPUSet != nil {
			oldAllocSpec.Containers[i].Resource.CPU.CPUSet.CPUIDs = allocSpec.Containers[i].Resource.CPU.CPUSet.CPUIDs
		}
	}

	if !apiequality.Semantic.DeepEqual(allocSpec, oldAllocSpec) {
		// TODO: Pinpoint the specific field that causes the invalid error after we have strategic merge diff
		return admission.NewForbidden(attributes,
			fmt.Errorf("can not %s annotation %s due to only cpuIDs can update", op, sigmaapi.AnnotationPodAllocSpec))
	}

	return nil
}

func findDuplicatedCPUIDs(cpusets *sigmaapi.CPUSetSpec) []string {
	var ids []string
	tmp := make(map[int]struct{})
	for _, v := range cpusets.CPUIDs {
		if _, ok := tmp[v]; ok {
			ids = append(ids, fmt.Sprintf("%d", v))
		} else {
			tmp[v] = struct{}{}
		}
	}
	return ids
}

var validateLabels = map[string]struct{}{
	sigmaapi.LabelCPUOverQuota:  {},
	sigmaapi.LabelMemOverQuota:  {},
	sigmaapi.LabelDiskOverQuota: {},
}

func validateNodeLabels(node *api.Node) []error {
	var errs []error
	for label, value := range node.Labels {
		if _, ok := validateLabels[label]; ok {
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				errs = append(errs, err)
			}
			if v < 1.0 {
				errs = append(errs, fmt.Errorf("%s must greater then 1.0", label))
			}
		}
	}
	return errs
}

func isInInplaceUpdateProcess(pod *api.Pod) bool {
	_, ok := pod.Annotations[sigmaapi.AnnotationPodInplaceUpdateState]
	return ok
}

func validateNode(attributes admission.Attributes) error {
	op := attributes.GetOperation()

	var oldNode *api.Node
	var oldLocalInfoExist bool
	var node *api.Node

	var exist bool

	if attributes.GetOldObject() != nil {
		oldNode, exist = attributes.GetOldObject().(*api.Node)
		if !exist {
			return apierrors.NewBadRequest("Resource was marked with kind Node but was unable to be converted")
		}

		if oldNode.Annotations != nil {
			var oldLocalInfoString string
			oldLocalInfoString, oldLocalInfoExist = oldNode.Annotations[sigmaapi.AnnotationLocalInfo]

			var oldLocalInfo sigmaapi.LocalInfo
			if err := json.Unmarshal([]byte(oldLocalInfoString), &oldLocalInfo); err != nil {
				return admission.NewForbidden(attributes,
					fmt.Errorf("can not %s due to old annotation %s json unmarshal error `%s`", op, sigmaapi.AnnotationLocalInfo, err))
			}
		}
	}

	node, exist = attributes.GetObject().(*api.Node)
	if !exist {
		return apierrors.NewBadRequest("Resource was marked with kind Node but was unable to be converted")
	}
	errs := validateNodeLabels(node)
	if len(errs) > 0 {
		var errString []string
		for _, e := range errs {
			errString = append(errString, e.Error())
		}
		return admission.NewForbidden(attributes,
			fmt.Errorf("can not %s due to labels %s %s", op, sigmaapi.AnnotationLocalInfo, strings.Join(errString, ", ")))
	}

	glog.V(10).Infof("SigmaScheduling validateUpdate Node: %#v", node)
	if node.Annotations == nil && !oldLocalInfoExist {
		return nil
	} else if node.Annotations == nil && oldLocalInfoExist {
		return admission.NewForbidden(attributes,
			fmt.Errorf("can not %s to remove annotation %s", op, sigmaapi.AnnotationLocalInfo))
	}

	localInfoString, exist := node.Annotations[sigmaapi.AnnotationLocalInfo]
	// 旧的 node 信息中包含了，新的没有，报错
	if !exist && oldLocalInfoExist {
		return admission.NewForbidden(attributes,
			fmt.Errorf("can not %s to remove annotation %s", op, sigmaapi.AnnotationLocalInfo))
	} else if !exist { // 旧的 node 信息中没有，新的也没有
		return nil
	}

	var localInfo sigmaapi.LocalInfo
	if err := json.Unmarshal([]byte(localInfoString), &localInfo); err != nil {
		return admission.NewForbidden(attributes,
			fmt.Errorf("can not %s due to annotation %s json unmarshal error %s", op, sigmaapi.AnnotationLocalInfo, err))
	}
	return nil
}

// validatePodAffinityTerms tests that the specified podAffinityTerms fields have valid data
// copy from k8s.io/api/core/validation
func validatePodAffinityTerms(podAffinityTerms []sigmaapi.PodAffinityTerm, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for i, podAffinityTerm := range podAffinityTerms {
		allErrs = append(allErrs, validatePodAffinityTerm(podAffinityTerm.PodAffinityTerm, fldPath.Index(i))...)
	}
	return allErrs
}

// validatePodAffinityTerm tests that the specified podAffinityTerm fields have valid data
// copy from k8s.io/api/core/validation
func validatePodAffinityTerm(podAffinityTerm core.PodAffinityTerm, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, unversionedvalidation.ValidateLabelSelector(podAffinityTerm.LabelSelector, fldPath.Child("matchExpressions"))...)
	for _, name := range podAffinityTerm.Namespaces {
		for _, msg := range validation.ValidateNamespaceName(name, false) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("namespace"), name, msg))
		}
	}

	// TODO(wade.lwd): support other topology key.
	if podAffinityTerm.TopologyKey != kubeletapis.LabelHostname {
		allErrs = append(allErrs, field.Required(fldPath.Child("topologyKey"), fmt.Sprintf("has topologyKey %q but only key %q is allowed", podAffinityTerm.TopologyKey, kubeletapis.LabelHostname)))
	}

	return append(allErrs, unversionedvalidation.ValidateLabelName(podAffinityTerm.TopologyKey, fldPath.Child("topologyKey"))...)
}

// validateWeightedPodAffinityTerms tests that the specified weightedPodAffinityTerms fields have valid data
// copy from k8s.io/api/core/validation
func validateWeightedPodAffinityTerms(weightedPodAffinityTerms []sigmaapi.WeightedPodAffinityTerm, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for j, weightedTerm := range weightedPodAffinityTerms {
		if weightedTerm.Weight <= 0 || weightedTerm.Weight > 100 {
			allErrs = append(allErrs, field.Invalid(fldPath.Index(j).Child("weight"), weightedTerm.Weight, "must be in the range 1-100"))
		}
		allErrs = append(allErrs, validatePodAffinityTerm(weightedTerm.PodAffinityTerm, fldPath.Index(j).Child("podAffinityTerm"))...)
	}
	return allErrs
}

func shouldIgnore(attributes admission.Attributes) bool {
	// Ignore all calls to subresources or resources other than pods and nodes.
	if len(attributes.GetSubresource()) != 0 || (attributes.GetResource().GroupResource() != api.Resource("pods") &&
		attributes.GetResource().GroupResource() != api.Resource("nodes")) {
		return true
	}

	return false
}

// NewContainerState creates a new SigmaScheduling admission control handler
func NewContainerState() *SigmaScheduling {
	return &SigmaScheduling{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}
}

func validateCPUWeightedPodAffinityTerms(terms []core.WeightedPodAffinityTerm, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for j, weightedTerm := range terms {
		if weightedTerm.Weight <= 0 || weightedTerm.Weight > 100 {
			allErrs = append(allErrs, field.Invalid(fldPath.Index(j).Child("weight"), weightedTerm.Weight, "must be in the range 1-100"))
		}
		allErrs = append(allErrs, validateCPUPodAffinityTerm(weightedTerm.PodAffinityTerm, fldPath.Index(j).Child("podAffinityTerm"))...)
	}
	return allErrs
}

func validateCPUPodAffinityTerm(podAffinityTerm core.PodAffinityTerm, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, unversionedvalidation.ValidateLabelSelector(podAffinityTerm.LabelSelector, fldPath.Child("matchExpressions"))...)
	for _, name := range podAffinityTerm.Namespaces {
		for _, msg := range validation.ValidateNamespaceName(name, false) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("namespace"), name, msg))
		}
	}

	// Only allow specific topology keys
	switch tk := podAffinityTerm.TopologyKey; tk {
	case sigmaapi.TopologyKeyLogicalCore, sigmaapi.TopologyKeyPhysicalCore:
	default:
		allErrs = append(allErrs, field.Required(fldPath.Child("topologyKey"), fmt.Sprintf("has topologyKey %q but only key [%q, %q] is allowed", podAffinityTerm.TopologyKey, sigmaapi.TopologyKeyLogicalCore, sigmaapi.TopologyKeyPhysicalCore)))
	}

	return append(allErrs, unversionedvalidation.ValidateLabelName(podAffinityTerm.TopologyKey, fldPath.Child("topologyKey"))...)
}
