package app

import (
	"k8s.io/kubernetes/pkg/scheduler"
)

/*
   To use convertible MostRequestedPriority / LeastRequestedPriority,
   they must both be loaded (specified in scheduler config) at start.
*/
func enableResourceAllocationPrioritiesConvertible(config *scheduler.Config) bool {
	leastRequestedPriorityLoaded := false
	mostRequestedPriorityLoaded := false
	for _, prioritizer := range config.Algorithm.Prioritizers() {
		if prioritizer.Name == "LeastRequestedPriority" {
			leastRequestedPriorityLoaded = true
		}
		if prioritizer.Name == "MostRequestedPriority" {
			mostRequestedPriorityLoaded = true
		}
	}
	return leastRequestedPriorityLoaded && mostRequestedPriorityLoaded
}
