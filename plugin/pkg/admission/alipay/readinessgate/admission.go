package readinessgate

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

// Package readinessGate contains an admission controller that checks and modifies every new Pod
// now only time-sharing pod will be admitted.

import (
	"io"

	antapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
)

const (
	// PluginName indicates name of admission plugin.
	PluginName = "ReadinessGate"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewReadinessGate(), nil
	})
}

// NewReadinessGate creates a new readinessGate admission control handler
func NewReadinessGate() *AlipayReadinessGate {
	return &AlipayReadinessGate{
		Handler: admission.NewHandler(admission.Create),
	}
}

// AlipayReadinessGate is an implementation of admission.Interface.
// It validates readinessGate of pods which must meet sigma policy.
type AlipayReadinessGate struct {
	*admission.Handler
}

var _ admission.MutationInterface = &AlipayReadinessGate{}

// Admit makes an admission decision based on the request attributes
func (a *AlipayReadinessGate) Admit(attributes admission.Attributes) (err error) {
	if shouldIgnore(attributes) {
		return nil
	}

	pod, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}
	if err := a.setDefaultReadinessGate(pod); err != nil {
		return apierrors.NewInternalError(err)
	}
	return nil
}

func (a *AlipayReadinessGate) setDefaultReadinessGate(pod *api.Pod) error {
	// time-sharing pod.
	if value, ok := pod.Labels[antapi.LabelPodPromotionType]; ok && (value == antapi.PodPromotionTypeTaobao.String() || value == antapi.PodPromotionTypeAntMember.String()) {
		AddReadinessGate(pod, antapi.TimeShareSchedulingReadinessGate)
	}
	return nil
}

// Readiness Gate. Add Readiness Gate or Check pod ready.
// AddReadinessGate() add condition into readinessGate.
func AddReadinessGate(pod *api.Pod, readinessGate string) {
	if ReadinessGateExists(pod.Spec.ReadinessGates, readinessGate) {
		return
	}
	if len(pod.Spec.ReadinessGates) == 0 {
		pod.Spec.ReadinessGates = []api.PodReadinessGate{}
	}
	pod.Spec.ReadinessGates = append(pod.Spec.ReadinessGates, api.PodReadinessGate{ConditionType: api.PodConditionType(readinessGate)})
	return
}

// ReadinessGateExists() whether the gate is exists.
func ReadinessGateExists(readinessGates []api.PodReadinessGate, gate string) bool {
	if len(readinessGates) == 0 {
		return false
	}
	for _, readinessGate := range readinessGates {
		if string(readinessGate.ConditionType) == gate {
			return true
		}
	}
	return false
}

func shouldIgnore(attributes admission.Attributes) bool {
	// Ignore all calls to subresources or resources other than pods.
	if len(attributes.GetSubresource()) != 0 || attributes.GetResource().GroupResource() != api.Resource("pods") {
		return true
	}

	return false
}
