package priorities

import (
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schedulerapi "k8s.io/kubernetes/pkg/scheduler/api"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
	"k8s.io/kubernetes/pkg/scheduler/evaluateexpression"
	schedulertesting "k8s.io/kubernetes/pkg/scheduler/testing"
	"reflect"
	"testing"
)

func TestCustomExpressionPriority(t *testing.T) {
	labels2 := map[string]string{
		"abc": "true",
	}
	zone1Spec := v1.PodSpec{
		NodeName: "machine1",
	}

	tests := []struct {
		pod          *v1.Pod
		pods         []*v1.Pod
		nodes        []string
		rcs          []*v1.ReplicationController
		rss          []*apps.ReplicaSet
		services     []*v1.Service
		sss          []*apps.StatefulSet
		expectedList schedulerapi.HostPriorityList
		name         string
		param        string
		expectError  bool
	}{
		{
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{}},
			pods: []*v1.Pod{
				{Spec: zone1Spec, ObjectMeta: metav1.ObjectMeta{Labels: labels2}},
			},
			nodes:        []string{"machine1"},
			services:     []*v1.Service{{Spec: v1.ServiceSpec{Selector: labels2}}},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 0}},
			name:         "nothing",
			param:        "",
		},
		{
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{}},
			pods: []*v1.Pod{
				{Spec: zone1Spec, ObjectMeta: metav1.ObjectMeta{Labels: labels2}},
			},
			nodes:        []string{"machine1"},
			services:     []*v1.Service{{Spec: v1.ServiceSpec{Selector: labels2}}},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 1}},
			name:         "hardcoded",
			param:        "1",
		},
		{
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{
				Labels:      map[string]string{"abc": "5"},
				Annotations: map[string]string{"abc": "2"},
			}},
			pods: []*v1.Pod{
				{Spec: zone1Spec, ObjectMeta: metav1.ObjectMeta{Labels: labels2}},
			},
			nodes:        []string{"machine1"},
			services:     []*v1.Service{{Spec: v1.ServiceSpec{Selector: labels2}}},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 7}},
			name:         "from pod label and annotation",
			param:        "pod.labels['abc'] + pod.annotations['abc']",
		},
		{
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{
				Labels:      map[string]string{"abc": "5"},
				Annotations: map[string]string{"abc": "2"},
			}},
			pods: []*v1.Pod{
				{Spec: zone1Spec, ObjectMeta: metav1.ObjectMeta{Labels: labels2}},
			},
			nodes:        []string{"machine1"},
			services:     []*v1.Service{{Spec: v1.ServiceSpec{Selector: labels2}}},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 7}},
			name:         "from node label and annotation",
			param:        "node.labels['abc'] + node.annotations['abc']",
		},
		{
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{}},
			pods: []*v1.Pod{
				{Spec: zone1Spec, ObjectMeta: metav1.ObjectMeta{Labels: labels2}},
			},
			nodes:        []string{"machine1"},
			services:     []*v1.Service{{Spec: v1.ServiceSpec{Selector: labels2}}},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 6}},
			name:         "basic arithmetic",
			param:        "1 + 5",
		},
		{
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{}},
			pods: []*v1.Pod{
				{Spec: zone1Spec, ObjectMeta: metav1.ObjectMeta{Labels: labels2}},
			},
			nodes:        []string{"machine1"},
			services:     []*v1.Service{{Spec: v1.ServiceSpec{Selector: labels2}}},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 10}},
			name:         "above max",
			param:        "1 + 10",
		},
		{
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{}},
			pods: []*v1.Pod{
				{Spec: zone1Spec, ObjectMeta: metav1.ObjectMeta{Labels: labels2}},
			},
			nodes:        []string{"machine1"},
			services:     []*v1.Service{{Spec: v1.ServiceSpec{Selector: labels2}}},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 0}},
			name:         "below min",
			param:        "1 - 10",
		},
		{
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{}},
			pods: []*v1.Pod{
				{Spec: zone1Spec, ObjectMeta: metav1.ObjectMeta{Labels: labels2}},
			},
			nodes:        []string{"machine1"},
			services:     []*v1.Service{{Spec: v1.ServiceSpec{Selector: labels2}}},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 0}},
			name:         "invalid",
			param:        "\"a\"",
			expectError:  true,
		},
		{
			pod: &v1.Pod{ObjectMeta: metav1.ObjectMeta{}},
			pods: []*v1.Pod{
				{Spec: zone1Spec, ObjectMeta: metav1.ObjectMeta{Labels: labels2}},
			},
			nodes:        []string{"machine1"},
			services:     []*v1.Service{{Spec: v1.ServiceSpec{Selector: labels2}}},
			expectedList: []schedulerapi.HostPriority{{Host: "machine1", Score: 6}},
			name:         "invalid",
			param:        "\"\" / 2",
			expectError:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if len(evaluateexpression.PodPriorityExpressionAnnotationKey) > 0 {
				if test.pod.Annotations == nil {
					test.pod.Annotations = make(map[string]string)
				}

				test.pod.Annotations[evaluateexpression.PodPriorityExpressionAnnotationKey] = test.param
			}

			nodeNameToInfo := schedulercache.CreateNodeNameToInfoMap(test.pods, makeNodeList2(test.nodes))

			calculateCustomExpressionPriorityMap, _ := NewCustomExpressionPriority(
				schedulertesting.FakeServiceLister(test.services),
				schedulertesting.FakeControllerLister(test.rcs),
				schedulertesting.FakeReplicaSetLister(test.rss),
				schedulertesting.FakeStatefulSetLister(test.sss),
			)

			metaDataProducer := NewPriorityMetadataFactory(
				schedulertesting.FakeServiceLister(test.services),
				schedulertesting.FakeControllerLister(test.rcs),
				schedulertesting.FakeReplicaSetLister(test.rss),
				schedulertesting.FakeStatefulSetLister(test.sss))
			metaData := metaDataProducer(test.pod, nodeNameToInfo)

			ttp := priorityFunction(calculateCustomExpressionPriorityMap, nil, metaData)
			list, err := ttp(test.pod, nodeNameToInfo, makeNodeList2(test.nodes))
			if err != nil && test.expectError {
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v \n", err)
			}
			if !reflect.DeepEqual(test.expectedList, list) {
				t.Errorf("expected %#v, got %#v", test.expectedList, list)
			}
		})
	}
}

func makeNodeList2(nodeNames []string) []*v1.Node {
	nodes := make([]*v1.Node, 0, len(nodeNames))
	for _, nodeName := range nodeNames {
		nodes = append(nodes, &v1.Node{ObjectMeta: metav1.ObjectMeta{
			Name:        nodeName,
			Labels:      map[string]string{"abc": "5"},
			Annotations: map[string]string{"abc": "2"},
		}})
	}
	return nodes
}
