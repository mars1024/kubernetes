package factory

import (
	"reflect"

	"k8s.io/api/core/v1"
)

func nodeSchedulingPropertiesChanged(newNode *v1.Node, oldNode *v1.Node) bool {
	if nodeSpecUnschedulableChanged(newNode, oldNode) {
		return true
	}
	if nodeAllocatableChanged(newNode, oldNode) {
		return true
	}
	if nodeLabelsChanged(newNode, oldNode) {
		return true
	}
	if nodeTaintsChanged(newNode, oldNode) {
		return true
	}
	if nodeConditionsChanged(newNode, oldNode) {
		return true
	}

	return false
}

func nodeAllocatableChanged(newNode *v1.Node, oldNode *v1.Node) bool {
	return !reflect.DeepEqual(oldNode.Status.Allocatable, newNode.Status.Allocatable)
}

func nodeLabelsChanged(newNode *v1.Node, oldNode *v1.Node) bool {
	return !reflect.DeepEqual(oldNode.GetLabels(), newNode.GetLabels())
}

func nodeTaintsChanged(newNode *v1.Node, oldNode *v1.Node) bool {
	return !reflect.DeepEqual(newNode.Spec.Taints, oldNode.Spec.Taints)
}

func nodeConditionsChanged(newNode *v1.Node, oldNode *v1.Node) bool {
	strip := func(conditions []v1.NodeCondition) map[v1.NodeConditionType]v1.ConditionStatus {
		conditionStatuses := make(map[v1.NodeConditionType]v1.ConditionStatus, len(conditions))
		for i := range conditions {
			conditionStatuses[conditions[i].Type] = conditions[i].Status
		}
		return conditionStatuses
	}
	return !reflect.DeepEqual(strip(oldNode.Status.Conditions), strip(newNode.Status.Conditions))
}

func nodeSpecUnschedulableChanged(newNode *v1.Node, oldNode *v1.Node) bool {
	return newNode.Spec.Unschedulable != oldNode.Spec.Unschedulable && newNode.Spec.Unschedulable == false
}