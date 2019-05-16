package predicates

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
	"testing"
)

func TestCheckSubClusterPredicate(t *testing.T) {
	_true := true
	tests := []struct {
		name         string
		pod          *corev1.Pod
		node         *corev1.Node
		expectResult bool
	}{
		{
			name: "no subcluster",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testpod",
				},
			},
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testnode",
				},
			},
			expectResult: true,
		},
		{
			name: "unknown node",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testpod",
				},
			},
			node:         nil,
			expectResult: false,
		},
		{
			name: "pod has but node hasnt",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "testpod",
					Labels: map[string]string{LabelSubCluster: "a"},
				},
			},
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testnode",
				},
			},
			expectResult: false,
		},
		{
			name: "node has but pod hasnt",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testpod",
				},
			},
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "testnode",
					Labels: map[string]string{LabelSubCluster: "a"},
				},
			},
			expectResult: false,
		},
		{
			name: "not equal",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "testpod",
					Labels: map[string]string{LabelSubCluster: "a"},
				},
			},
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "testnode",
					Labels: map[string]string{LabelSubCluster: "b"},
				},
			},
			expectResult: false,
		},
		{
			name: "equal",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "testpod",
					Labels: map[string]string{LabelSubCluster: "a"},
				},
			},
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "testnode",
					Labels: map[string]string{LabelSubCluster: "a"},
				},
			},
			expectResult: true,
		},
		{
			name: "daemonset",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testpod",
					OwnerReferences: []metav1.OwnerReference{
						{
							Controller: &_true,
							Kind:       "DaemonSet",
						},
					},
				},
			},
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "testnode",
					Labels: map[string]string{LabelSubCluster: "a"},
				},
			},
			expectResult: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nodeInfo := schedulercache.NewNodeInfo(test.pod)
			if test.node != nil {
				nodeInfo.SetNode(test.node)
			}
			fits, _, err := CheckSubClusterPredicate(test.pod, nil, nodeInfo)

			if err != nil {
				t.Errorf("Error calling CheckSubClusterPredicate: %v", err)
			}

			if fits != test.expectResult {
				t.Errorf("CheckSubClusterPredicate (%s): expected %v, got %v", test.name, test.expectResult, fits)
			}
		})
	}
}
