package util

import (
	"encoding/json"
	"errors"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/kubernetes/test/e2e/framework"
)

// PatchNodeExtendedResource patch the node's status with extended resource.
func PatchNodeExtendedResource(f *framework.Framework, nodes []v1.Node, key string, value string) error {
	for _, node := range nodes {
		oldData, err := json.Marshal(node)
		if err != nil {
			return err
		}

		if node.Status.Capacity == nil {
			node.Status.Capacity = make(v1.ResourceList)
		}
		if node.Status.Allocatable == nil {
			node.Status.Allocatable = make(v1.ResourceList)
		}
		node.Status.Capacity[v1.ResourceName(key)] = resource.MustParse(value)
		node.Status.Allocatable[v1.ResourceName(key)] = resource.MustParse(value)

		newData, err := json.Marshal(node)
		if err != nil {
			return err
		}

		patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, v1.Node{})
		if err != nil {
			return err
		}
		_, err = f.ClientSet.CoreV1().Nodes().PatchStatus(string(node.Name), patchBytes)
		if err != nil {
			return err
		}
	}
	return nil
}

// PatchNodeStatusJsonPathType apply JsonPatchType patch on nodes status.
func PatchNodeStatusJsonPathType(f *framework.Framework, nodes []v1.Node, data []byte) error {
	for _, node := range nodes {
		_, err := f.ClientSet.CoreV1().Nodes().Patch(string(node.Name), types.JSONPatchType, data, "status")
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAffinityNodeSelectorRequirement sets the affinity.
func GetAffinityNodeSelectorRequirement(key string, value []string) *v1.Affinity {
	return &v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: []v1.NodeSelectorTerm{
					{
						MatchExpressions: []v1.NodeSelectorRequirement{
							{
								Key:      key,
								Operator: v1.NodeSelectorOpIn,
								Values:   value,
							},
						},
					},
				},
			},
		},
	}
}

func GetAffinityNodeSelectorRequirementAndMap(config map[string][]string) *v1.Affinity {
	items := make([]v1.NodeSelectorRequirement, 0, len(config))
	for key, value := range config {
		items = append(items, v1.NodeSelectorRequirement{
			Key:      key,
			Operator: v1.NodeSelectorOpIn,
			Values:   value,
		})
	}

	return &v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: []v1.NodeSelectorTerm{
					{
						MatchExpressions: items,
					},
				},
			},
		},
	}
}

// AddMatchExpressionsToFirstNodeSelectorTermsOfAffinity add match expression to the first node selector of affinity.
func AddMatchExpressionsToFirstNodeSelectorTermsOfAffinity(key string, value []string, affinity *v1.Affinity) (*v1.Affinity, error) {
	if len(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) < 1 {
		return affinity, errors.New("first NodeSelectorTerms not found.")
	}
	matchExpressions := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions
	newMatchExpression := v1.NodeSelectorRequirement{
		Key:      key,
		Operator: v1.NodeSelectorOpIn,
		Values:   value,
	}
	matchExpressions = append(matchExpressions, newMatchExpression)
	affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions = matchExpressions
	return affinity, nil
}

// AddMatchExpressionsToNewNodeSelectorTermsOfAffinity add match expression to the new node selector of affinity.
func AddMatchExpressionsToNewNodeSelectorTermsOfAffinity(key string, value []string, affinity *v1.Affinity) (*v1.Affinity, error) {
	nodeSelector := v1.NodeSelectorTerm{
		MatchExpressions: []v1.NodeSelectorRequirement{
			{
				Key:      key,
				Operator: v1.NodeSelectorOpIn,
				Values:   value,
			},
		},
	}
	curSelectorTerms := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	curSelectorTerms = append(curSelectorTerms, nodeSelector)
	affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = curSelectorTerms
	return affinity, nil
}

// GetAffinityNodeSelectorNotInRequirement sets the NotIn match expressions affinity.
func GetAffinityNodeSelectorNotInRequirement(key string, value []string) *v1.Affinity {
	return &v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: []v1.NodeSelectorTerm{
					{
						MatchExpressions: []v1.NodeSelectorRequirement{
							{
								Key:      key,
								Operator: v1.NodeSelectorOpNotIn,
								Values:   value,
							},
						},
					},
				},
			},
		},
	}
}

// AddMatchExpressionsToPerferredScheduleOfAffinity add match expression to the PreferredSchedulingTerm of affinity.
func AddMatchExpressionsToPerferredScheduleOfAffinity(key string, value []string, weight int32, affinity *v1.Affinity) *v1.Affinity {
	preferTerm := v1.PreferredSchedulingTerm{
		Weight: weight,
		Preference: v1.NodeSelectorTerm{
			MatchExpressions: []v1.NodeSelectorRequirement{
				{
					Key:      key,
					Operator: v1.NodeSelectorOpIn,
					Values:   value,
				},
			},
		},
	}
	affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution, preferTerm)
	return affinity
}
