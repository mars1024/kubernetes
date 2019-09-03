package predicates

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
)

const (
	LabelSubCluster = "cafe.sofastack.io/sub-cluster"
)

const (
	// CheckSubClusterPred defines the name of predicate SubCluster.
	CheckSubClusterPred = "CheckSubCluster"
)

var (
	// ErrSubClusterNotMatch is used for SubClusterNotMatch predicate error.
	ErrSubClusterNotMatch = newPredicateFailureError("SubClusterNotMatch", "the pod and the node are not of the same sub-cluster")
)

// CheckSubClusterPredicate checks if the pod and the node are in the same sub cluster
func CheckSubClusterPredicate(pod *v1.Pod, meta algorithm.PredicateMetadata, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	if nodeInfo == nil || nodeInfo.Node() == nil {
		return false, []algorithm.PredicateFailureReason{ErrNodeUnknownCondition}, nil
	}

	// a DaemonSet is considered a cluster level resource, so sub-cluster checking should be exempted.
	// To enforce sub-cluster checking for DaemonSet, use a nodeSelector in the DS's spec instead.
	if podBelongsToDaemonSet(pod) {
		return true, nil, nil
	}

	podSubClusterName := getSubClusterNameFromMetadata(pod.ObjectMeta)
	nodeSubClusterName := getSubClusterNameFromMetadata(nodeInfo.Node().ObjectMeta)

	// if they both have no sub-cluster-name annotation, then the predicate passes, in order to keep it backward-compatible
	if podSubClusterName == "" && nodeSubClusterName == "" {
		return true, nil, nil
	}

	// if the node is in a sub-cluster but the pod is not, the predicate should fail,
	// in order to keep the node exclusive to pods of its same sub-cluster.
	if podSubClusterName == "" && len(nodeSubClusterName) > 0 {
		return false, []algorithm.PredicateFailureReason{ErrSubClusterNotMatch}, nil
	}

	// if the pod is in a sub-cluster but the node is not, the predicate should fail,
	// so that the pod can only be scheduled to a node in its sub-cluster.
	if len(podSubClusterName) > 0 && nodeSubClusterName == "" {
		return false, []algorithm.PredicateFailureReason{ErrSubClusterNotMatch}, nil
	}

	// if they both have sub-cluster-name annotation, then the values should match.
	if podSubClusterName == nodeSubClusterName {
		return true, nil, nil
	} else {
		return false, []algorithm.PredicateFailureReason{ErrSubClusterNotMatch}, nil
	}

}

func podBelongsToDaemonSet(pod *v1.Pod) bool {
	controllerRef := metav1.GetControllerOf(pod)
	if controllerRef == nil {
		return false
	}
	return controllerRef.Kind == "DaemonSet"
}

func getSubClusterNameFromMetadata(metadata metav1.ObjectMeta) string {
	return metadata.Labels[LabelSubCluster]
}
