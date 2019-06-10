/*
Copyright 2016 The Kubernetes Authors.

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

package antiaffinity

import (
	"fmt"
	"github.com/golang/glog"
	cafelabels "gitlab.alipay-inc.com/antstack/cafe-k8s-api/pkg"
	"io"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	PluginName         = "Monotype"
	DefaultTopologyKey = "kubernetes.io/hostname"
)

var (
	MonotypeValues = []string{
		cafelabels.MonotypeLabelValueHard,
		cafelabels.MonotypeLabelValueSoft,
		cafelabels.MonotypeLabelValueNone,
	}
)
// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewMonotypeInjector(), nil
	})
}

// Plugin contains the client used by the admission controller
type Plugin struct {
	*admission.Handler
}

var _ admission.MutationInterface = &Plugin{}

// NewMonotypeInjector creates a new instance of the LimitPodHardAntiAffinityTopology admission controller
func NewMonotypeInjector() *Plugin {
	return &Plugin{
		Handler: admission.NewHandler(admission.Create),
	}
}

// Admit will check pod anti-affinity based on label monotype=hard
// if no affinity inject it
func (p *Plugin) Admit(attributes admission.Attributes) (err error) {
	// Ignore all calls to subresources or resources other than pods.
	if len(attributes.GetSubresource()) != 0 || attributes.GetResource().GroupResource() != api.Resource("pods") {
		return nil
	}
	pod, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}
	labels := pod.GetLabels()
	if labels == nil {
		return nil
	}
	if v, ok := labels[cafelabels.MonotypeLabelKey]; ok {
		if !sets.NewString(MonotypeValues...).Has(v) {
			return fmt.Errorf("invalid monotype value specified: %q,  allowed are %v", v, MonotypeValues)
		}
		switch v {
		case cafelabels.MonotypeLabelValueHard:
			err := CheckResource(pod)
			if err != nil {
				return err
			}
			injectHostNetwork(pod)
			return injectHardAffinity(pod)
		case cafelabels.MonotypeLabelValueSoft:
			return injectSoftAffinity(pod)
		default:
			return nil
		}
	}
	return nil
}

func getLabelSelectorRequirement(value string) metav1.LabelSelectorRequirement {
	return metav1.LabelSelectorRequirement{
		Key:      cafelabels.MonotypeLabelKey,
		Operator: metav1.LabelSelectorOpIn,
		Values:   []string{value},
	}
}

func injectHardAffinity(pod *api.Pod) error {
	// inject the pod anti-affinity
	if pod.Spec.Affinity != nil && pod.Spec.Affinity.PodAntiAffinity != nil {
		required := pod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
		if len(required) > 0 {
			topologyKeyMatched := false
			for _, term := range required {
				if term.TopologyKey == DefaultTopologyKey {
					glog.V(4).Infof("topologyKey is %q matched, mark and continue", term.TopologyKey)
					topologyKeyMatched = true
				}
				for _, requirement := range term.LabelSelector.MatchExpressions {
					if requirement.Key == cafelabels.MonotypeLabelKey && requirement.Operator == metav1.LabelSelectorOpIn {
						r := sets.NewString(requirement.Values...).Has(cafelabels.MonotypeLabelValueHard)
						if r == true && topologyKeyMatched == true {
							return nil
						}
						if r == true && topologyKeyMatched == false {
							return fmt.Errorf("matchExpressions matches, but topologyKey is unwanted %q, not injecting anything", term.TopologyKey)
						}
						return fmt.Errorf("expected %q=%q in the matchExpressions", cafelabels.MonotypeLabelKey, cafelabels.MonotypeLabelValueHard)
					}
				}
			}
		}
		// append expression to the first PodAffinityTerm
		if len(required) > 0 {
			required[0].LabelSelector.MatchExpressions = append(required[0].LabelSelector.MatchExpressions, getLabelSelectorRequirement(cafelabels.MonotypeLabelValueHard))
			return nil
		}
	}
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &api.Affinity{}
	}
	if pod.Spec.Affinity.PodAntiAffinity == nil {
		pod.Spec.Affinity.PodAntiAffinity = &api.PodAntiAffinity{}
	}
	pod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = []api.PodAffinityTerm{
		{
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					getLabelSelectorRequirement(cafelabels.MonotypeLabelValueHard),
				},
			},
			TopologyKey: DefaultTopologyKey,
		},
	}
	return nil
}

func injectHostNetwork(pod *api.Pod) {
	if pod.Spec.SecurityContext == nil {
		pod.Spec.SecurityContext = &api.PodSecurityContext{
			HostNetwork: true,
		}
	}
	if pod.Spec.SecurityContext.HostNetwork == false {
		pod.Spec.SecurityContext.HostNetwork = true
	}
}

func injectSoftAffinity(pod *api.Pod) error {
	// inject the pod anti-affinity
	if pod.Spec.Affinity != nil && pod.Spec.Affinity.PodAntiAffinity != nil {
		preferred := pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
		if len(preferred) > 0 {
			topologyKeyMatched := false
			for _, weight := range preferred {
				term := weight.PodAffinityTerm
				if term.TopologyKey == DefaultTopologyKey {
					glog.V(4).Infof("topologyKey is %q matched, mark and continue", term.TopologyKey)
					topologyKeyMatched = true
				}
				for _, requirement := range term.LabelSelector.MatchExpressions {
					if requirement.Key == cafelabels.MonotypeLabelKey && requirement.Operator == metav1.LabelSelectorOpIn {
						r := sets.NewString(requirement.Values...).Has(cafelabels.MonotypeLabelValueSoft)
						if r == true && topologyKeyMatched == true {
							return nil
						}
						if r == true && topologyKeyMatched == false {
							return fmt.Errorf("matchExpressions matches, but topologyKey is unwanted %q, not injecting anything", term.TopologyKey)
						}
						return fmt.Errorf("expected %q=%q in the matchExpressions", cafelabels.MonotypeLabelKey, cafelabels.MonotypeLabelValueHard)
					}
				}
			}
		}
		// append a new PodAffinityTerm
		term := api.WeightedPodAffinityTerm{
			Weight: 100, // set to extraordinary high, threat it as best choice
			PodAffinityTerm: api.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						getLabelSelectorRequirement(cafelabels.MonotypeLabelValueSoft),
					},
				},
			},
		}
		preferred = append(preferred, term)
		pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferred
		return nil
	}
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &api.Affinity{}
	}
	if pod.Spec.Affinity.PodAntiAffinity == nil {
		pod.Spec.Affinity.PodAntiAffinity = &api.PodAntiAffinity{}
	}
	pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = []api.WeightedPodAffinityTerm{
		{
			Weight: 100, // set to extraordinary high, threat it as best choice
			PodAffinityTerm: api.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						getLabelSelectorRequirement(cafelabels.MonotypeLabelValueSoft),
					},
				},
			},
		},}

	return nil
}

func CheckResource(pod *api.Pod) error {
	cpu := resource.NewMilliQuantity(0, resource.DecimalSI)
	memory := resource.NewMilliQuantity(0, resource.BinarySI)
	for _, container := range pod.Spec.Containers {
		if v, ok := container.Resources.Requests[api.ResourceCPU]; ok {
			cpu.Add(v)
		}
		if v, ok := container.Resources.Requests[api.ResourceMemory]; ok {
			memory.Add(v)
		}
	}
	if cpu.MilliValue() <= 0 {
		return fmt.Errorf("[monotype]total CPU request must be larger than 0")
	}
	if memory.MilliValue() <= 0 {
		return fmt.Errorf("[monotype]total Memory request must be larger than 0")
	}
	return nil
}
