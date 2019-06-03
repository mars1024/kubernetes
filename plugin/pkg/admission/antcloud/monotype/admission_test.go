package antiaffinity

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	cafelabels "gitlab.alipay-inc.com/antstack/cafe-k8s-api/pkg"

)

func TestNewMonotypeInjector(t *testing.T) {
	handler := NewMonotypeInjector()
	pod := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"monotype": "hard",
			},
		},
		Spec: api.PodSpec{
			Affinity: &api.Affinity{
				PodAntiAffinity: &api.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []api.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "security",
										Operator: metav1.LabelSelectorOpIn,
										Values:   []string{"S2"},
									},
								},
							},
							TopologyKey: "az",
						},
					},
				},
			},
		},
	}

	err := handler.Admit(admission.NewAttributesRecord(pod, nil, api.Kind("Pod").WithVersion("version"), "foo", "name", api.Resource("pods").WithVersion("version"), "", "ignored", false, nil))
	if err != nil {
		t.Errorf("failed to admit pod %v", pod)
	}
	injected := false
	terms := pod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	for _, t := range terms {
		if len(t.LabelSelector.MatchExpressions) < 2 {
			break
		}
		for _, x := range t.LabelSelector.MatchExpressions {

			if x.Key == cafelabels.MonotypeLabelKey && x.Values[0] == cafelabels.MonotypeLabelValueHard {
				if x.Operator == metav1.LabelSelectorOpIn {
					injected = true
				}
			}
		}
	}
	if !injected {
		t.Errorf("expected monotype=hard affinity not found")
	}
}

func TestNewMonotypeInjector_NoLabel(t *testing.T) {
	handler := NewMonotypeInjector()
	pod := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"hello": "test",
			},
		},
		Spec: api.PodSpec{
		},
	}

	err := handler.Admit(admission.NewAttributesRecord(pod, nil, api.Kind("Pod").WithVersion("version"), "foo", "name", api.Resource("pods").WithVersion("version"), "", "ignored", false, nil))
	if err != nil {
		t.Errorf("failed to admit pod %v", pod)
	}
	if pod.Spec.Affinity != nil {
		t.Errorf("affinity should be nil, unexpected pod anti-affinity injection")
	}

}

func TestNewMonotypeInjector_TopologyKey_Unmatched(t *testing.T) {
	handler := NewMonotypeInjector()
	pod := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"monotype": "hard",
			},
		},
		Spec: api.PodSpec{
			Affinity: &api.Affinity{
				PodAntiAffinity: &api.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []api.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      cafelabels.MonotypeLabelKey,
										Operator: metav1.LabelSelectorOpIn,
										Values:   []string{cafelabels.MonotypeLabelValueHard},
									},
								},
							},
							TopologyKey: "az",
						},
					},
				},
			},
		},
	}

	err := handler.Admit(admission.NewAttributesRecord(pod, nil, api.Kind("Pod").WithVersion("version"), "foo", "name", api.Resource("pods").WithVersion("version"), "", "ignored", false, nil))
	if err == nil {
		t.Errorf("expect failure, actual success for pod %v", pod)
	}
	injected := false
	terms := pod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	for _, t := range terms {
		if len(t.LabelSelector.MatchExpressions) >= 2 {
			break
		}
		for _, x := range t.LabelSelector.MatchExpressions {

			if x.Key == cafelabels.MonotypeLabelKey && x.Values[0] == cafelabels.MonotypeLabelValueHard {
				if x.Operator == metav1.LabelSelectorOpIn {
					injected = true
				}
			}
		}
		injected = injected && t.TopologyKey == DefaultTopologyKey
	}
	if injected {
		t.Errorf("expected no monotype=hard affinity inject")
	}
}

func TestMonotypeEnum(t *testing.T) {
	valueUnderTest1 := "hard1"
	handler := NewMonotypeInjector()

	pod := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"monotype": valueUnderTest1,
			},
		},}
	err := handler.Admit(admission.NewAttributesRecord(pod, nil, api.Kind("Pod").WithVersion("version"), "foo", "name", api.Resource("pods").WithVersion("version"), "", "ignored", false, nil))
	if err == nil {
		t.Errorf("got unexpected success, should be rejected")
	}

	pod1 := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"monotype": "hard",
			},
		},}
	err1 := handler.Admit(admission.NewAttributesRecord(pod1, nil, api.Kind("Pod").WithVersion("version"), "foo", "name", api.Resource("pods").WithVersion("version"), "", "ignored", false, nil))
	if err1 != nil {
		t.Errorf("should be a valid value, but rejected: %s", err1.Error())
	}
}

func TestNewMonotypeInjector_Soft(t *testing.T) {
	handler := NewMonotypeInjector()
	pod := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"monotype": "soft",
			},
		},
		Spec: api.PodSpec{
			Affinity: &api.Affinity{
				PodAntiAffinity: &api.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []api.WeightedPodAffinityTerm{
						{
							Weight: 100,
							PodAffinityTerm: api.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{
										{
											Key:      "security",
											Operator: metav1.LabelSelectorOpIn,
											Values:   []string{"S2"},
										},
									},
								},
								TopologyKey: "az",
							},
						},
					},
				},
			},
		},}

	err := handler.Admit(admission.NewAttributesRecord(pod, nil, api.Kind("Pod").WithVersion("version"), "foo", "name", api.Resource("pods").WithVersion("version"), "", "ignored", false, nil))
	if err != nil {
		t.Errorf("failed to admit pod %v", pod)
	}
	injected := false
	terms := pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
	if len(terms) >= 2 {
		for _, t := range terms {
			aterm := t.PodAffinityTerm
			for _, x := range aterm.LabelSelector.MatchExpressions {
				if x.Key == cafelabels.MonotypeLabelKey && x.Values[0] == cafelabels.MonotypeLabelValueSoft {
					if x.Operator == metav1.LabelSelectorOpIn {
						injected = true
					}
				}
			}

		}
	}
	if !injected {
		t.Errorf("expected monotype=soft affinity not found")
	}
}
