package sigma

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetAndUpdateDanglingPods(t *testing.T) {
	for _, test := range []struct {
		danglingPods []sigmak8sapi.DanglingPod
		node         *v1.Node
		kubeClient   clientset.Interface
		message      string
	}{
		{
			danglingPods: []sigmak8sapi.DanglingPod{
				sigmak8sapi.DanglingPod{
					Name:      "pod1",
					Namespace: "namespace1",
				},
			},
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "host1",
				},
			},
			kubeClient: fake.NewSimpleClientset(),
			message:    "one dangling pod",
		},
		{
			danglingPods: []sigmak8sapi.DanglingPod{
				sigmak8sapi.DanglingPod{
					Name:      "pod1",
					Namespace: "namespace1",
				},
				sigmak8sapi.DanglingPod{
					Name:      "pod2",
					Namespace: "namespace2",
				},
				sigmak8sapi.DanglingPod{
					Name:      "pod2",
					Namespace: "namespace2",
				},
			},
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "host1",
				},
			},
			kubeClient: fake.NewSimpleClientset(),
			message:    "multiple dangling pods",
		},
		{
			danglingPods: []sigmak8sapi.DanglingPod{},
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "host1",
				},
			},
			kubeClient: fake.NewSimpleClientset(),
			message:    "no dangling pods",
		},
	} {
		_, err := test.kubeClient.CoreV1().Nodes().Create(test.node)
		if err != nil {
			t.Errorf("%v", err)
		}

		UpdateDanglingPods(test.kubeClient, test.node.Name, test.danglingPods)

		nodeDanglingPods, err := GetDanglingPods(test.kubeClient, test.node.Name)
		assert.Equal(t, err, nil)
		if !reflect.DeepEqual(test.danglingPods, nodeDanglingPods) {
			t.Errorf("Get wrong dangling pod in case %q: expect: %v, but get: %v", test.message, test.danglingPods, nodeDanglingPods)
		}
	}
}
