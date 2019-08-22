package convertible

import (
	"os"

	"k8s.io/api/core/v1"
)

// ResourceAllocationPriority Convertible gives the ability to use
// either MostRequestedPriority or LeastRequestedPriority
// according to the pod's annotation: aks.cafe.sofastack.io/scheduling-policy

var defaultAllocationPriority = getDefaultAllocationPriority()

func getDefaultAllocationPriority() string {
	if len(os.Getenv(envDefaultResourceAllocationPriority)) > 0 {
		return os.Getenv(envDefaultResourceAllocationPriority)
	} else {
		return resourceAllocationPriorityLeastRequested
	}
}

func ShouldSkipResourceAllocationPriority(priorityName string, pod *v1.Pod, node *v1.Node) bool {
	convertible, ok := ResourceAllocationPrioritiesConvertible.Load().(bool)
	if !ok || !convertible {
		// not in convertible mode, fallback to normal behavior
		return false
	}

	annotations, err := metadataAccessor.Annotations(pod)
	if err != nil {
		return false
	}

	var chosenAllocationPriority string
	userSpecifiedAllocationPriority, hasSchedulingPolicyAnnotation := annotations[annotationSchedulingPolicy]
	if hasSchedulingPolicyAnnotation {
		chosenAllocationPriority = userSpecifiedAllocationPriority
	} else {
		chosenAllocationPriority = defaultAllocationPriority
	}

	switch priorityName {
	case "LeastResourceAllocation", "BalancedResourceAllocation":
		return chosenAllocationPriority != resourceAllocationPriorityLeastRequested
	case "MostResourceAllocation":
		return chosenAllocationPriority != resourceAllocationPriorityMostRequested
	case "RequestedToCapacityRatioResourceAllocationPriority":
		return chosenAllocationPriority != resourceAllocationRequestToCapacityRatio
	default:
		return hasSchedulingPolicyAnnotation
	}
}
