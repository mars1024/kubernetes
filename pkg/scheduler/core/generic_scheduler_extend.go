package core

import (
	"encoding/json"
	"k8s.io/api/core/v1"
)

const(
	PriorityWeightOverrideAnnotationKey = "scheduling.aks.cafe.sofastack.io/priority-weight-override"
)

func getEffectivePriorityWeight(priorityWeightOverrideMap map[string]int, priorityConfigName string, fallbackValue int) int {
	overriddenWeight, ok := priorityWeightOverrideMap[priorityConfigName]
	if ok {
		return overriddenWeight
	}

	return fallbackValue
}

func getPriorityWeightOverrideMap(pod *v1.Pod) map[string]int {
	priorityWeightOverride := make(map[string]int)
	if pod.Annotations != nil {
		priorityWeightOverrideAnnotation, ok := pod.Annotations[PriorityWeightOverrideAnnotationKey]
		if ok {
			_ = json.Unmarshal([]byte(priorityWeightOverrideAnnotation), &priorityWeightOverride)
		}
	}
	return priorityWeightOverride
}
