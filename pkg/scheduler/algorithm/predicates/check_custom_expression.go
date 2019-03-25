package predicates

import (
	"fmt"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
	"k8s.io/kubernetes/pkg/scheduler/evaluateexpression"
	"math"
)

const (
	// CustomExpressionPred defines the name of predicate CustomExpression.
	CustomExpressionPred = "CustomExpression"
)

var (
	// ErrCheckCustomExpressionFailed is used for CheckCustomExpressionPredicate predicate error.
	ErrCheckCustomExpressionFailed = newPredicateFailureError("CheckCustomExpressionFailed", "custom expression predicate evaluated to a falsy value")
)

// CheckCustomExpressionPredicate checks if a custom expression evaluates to true.
func CheckCustomExpressionPredicate(pod *v1.Pod, meta algorithm.PredicateMetadata, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	node := nodeInfo.Node()

	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}

	var expressionString string
	if pod.Annotations != nil {
		expressionString, _ = pod.Annotations[evaluateexpression.PodPredicateExpressionAnnotationKey]
	}

	// let the predicate pass for pods that don't use custom expression
	if len(expressionString) == 0 {
		return true, nil, nil
	}

	value, err := evaluateexpression.EvaluateExpression(expressionString, node, pod)

	if err != nil {
		return false, nil, err
	}

	reasons := []algorithm.PredicateFailureReason{ErrCheckCustomExpressionFailed}
	switch value.(type) {
	case float64:
		return math.Abs(value.(float64)*10000000000) > 1, reasons, nil
	case bool:
		return value.(bool), reasons, nil
	case string:
		return !(value.(string) == "" || value.(string) == "false" || value.(string) == "0" || value.(string) == "null" || value.(string) == "nil"), reasons, nil
	default:
		return false, nil, fmt.Errorf("invalid return type of custom expression (boolean expected)")
	}
}
