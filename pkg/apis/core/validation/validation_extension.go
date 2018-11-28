/*
Copyright 2018 The Kubernetes Authors.

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

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/core/helper/qos"
)

// ValidateContainerQoSUpdate tests to see if QoS class changed in update process. For each of Burstable and the Best-effort
// class, kubelet maintains a class-level cgroup under the ‘kubepods’ cgroup, e.g kubepods/besteffort is the parent cgroup of all
// BestEffort pods. So in order to change the QoS class of a pod, it needs not only to update resources, but also needs to change
// its parent cgroup properly.
// But once a pod is created, its parent cgroup cannot be changed with the current Docker API. So the QoS class of a pod cannot
// be changed through resource update.
func ValidateContainerQoSUpdate(newPod, oldPod *core.Pod, fldPath *field.Path) (allErrs field.ErrorList, stop bool) {
	allErrs = field.ErrorList{}
	oldQOSClass := qos.GetPodQOS(oldPod)
	newQOSClass := qos.GetPodQOS(newPod)
	if string(oldQOSClass) != string(newQOSClass) {
		allErrs = append(allErrs, field.Forbidden(fldPath, "container resource updates must not change pod QoS class"))
		return allErrs, true
	}

	return allErrs, false
}

// handle updatable fields by munging those fields prior to deep equal comparison
// TODO: handle more updatable fields
func copyUpdatableContainerFields(dst, src *core.Container) {
	oldResourceRequirement := &src.Resources
	dst.Resources = *oldResourceRequirement.DeepCopy()
}
