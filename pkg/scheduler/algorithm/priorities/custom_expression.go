package priorities

import (
	"fmt"
	"math"
	"strconv"

	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	schedulerapi "k8s.io/kubernetes/pkg/scheduler/api"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
	"k8s.io/kubernetes/pkg/scheduler/evaluateexpression"
)

type CustomExpression struct {
	serviceLister     algorithm.ServiceLister
	controllerLister  algorithm.ControllerLister
	replicaSetLister  algorithm.ReplicaSetLister
	statefulSetLister algorithm.StatefulSetLister
}

// NewCustomExpressionPriority creates a CustomExpression.
func NewCustomExpressionPriority(
	serviceLister algorithm.ServiceLister,
	controllerLister algorithm.ControllerLister,
	replicaSetLister algorithm.ReplicaSetLister,
	statefulSetLister algorithm.StatefulSetLister) (algorithm.PriorityMapFunction, algorithm.PriorityReduceFunction) {
	fasSameZoneSpread := &CustomExpression{
		serviceLister:     serviceLister,
		controllerLister:  controllerLister,
		replicaSetLister:  replicaSetLister,
		statefulSetLister: statefulSetLister,
	}
	return fasSameZoneSpread.CalculateCustomExpressionPriorityMap, nil
}

func (s *CustomExpression) CalculateCustomExpressionPriorityMap(pod *v1.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (schedulerapi.HostPriority, error) {
	node := nodeInfo.Node()
	if node == nil {
		return schedulerapi.HostPriority{}, fmt.Errorf("node not found")
	}

	var expressionString string
	if pod.Annotations != nil {
		expressionString, _ = pod.Annotations[evaluateexpression.PodPriorityExpressionAnnotationKey]
	}

	// return score 0 for pods that don't use custom expression
	if len(expressionString) == 0 {
		return schedulerapi.HostPriority{
			Host:  node.Name,
			Score: 0,
		}, nil
	}

	value, err := evaluateexpression.EvaluateExpression(expressionString, node, pod)

	if err != nil {
		return schedulerapi.HostPriority{}, err
	}

	switch value.(type) {
	case float64:
		return schedulerapi.HostPriority{
			Host:  node.Name,
			Score: normalizeFloat64HostPriorityScore(value.(float64)),
		}, nil
	default:
		return schedulerapi.HostPriority{}, fmt.Errorf("invalid return type of custom expression (number expected)")
	}
}

func normalizeFloat64HostPriorityScore(value float64) int {
	if value <= 0 {
		return 0
	} else if value >= 10 {
		return 10
	} else {
		i, _ := strconv.Atoi(fmt.Sprintf("%.0f", math.Round(value)))
		return i
	}
}
