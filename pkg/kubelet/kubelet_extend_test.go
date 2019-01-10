/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubelet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"
)

func TestSyncPodWithProtectionFinalizer(t *testing.T) {
	testKubelet := newTestKubelet(t, false /* controllerAttachDetachEnabled */)
	defer testKubelet.Cleanup()
	kl := testKubelet.kubelet
	deletionTime := metav1.NewTime(time.Now())
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:               "12345678",
			Name:              "bar",
			Namespace:         "foo",
			Finalizers:        []string{"protection.pod.beta1.sigma.ali/vip-removed"},
			DeletionTimestamp: &deletionTime,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "bar",
					Image: "beep",
				},
			},
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name: "bar",
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
				},
			},
		},
	}
	pods := []*v1.Pod{pod}
	kl.podManager.SetPods(pods)
	err := kl.syncPod(syncPodOptions{
		pod: pod,
		podStatus: &kubecontainer.PodStatus{
			ContainerStatuses: []*kubecontainer.ContainerStatus{
				{
					Name:  "bar",
					State: kubecontainer.ContainerStateRunning,
				},
			},
		},
		updateType: kubetypes.SyncPodUpdate,
	})
	require.NoError(t, err)

	// Check pod status stored in the status map.
	checkPodStatus(t, kl, pod, v1.PodRunning)
}

func TestSkipAdmit(t *testing.T) {
	testKubelet := newTestKubelet(t, false /* controllerAttachDetachEnabled */)
	defer testKubelet.Cleanup()
	kl := testKubelet.kubelet

	testCase := []struct {
		name         string
		pod          *v1.Pod
		expectResult bool
	}{
		{
			name:         "pod status is failed, skip",
			pod:          createPod(v1.PodFailed),
			expectResult: true,
		},
		{
			name:         "pod status is succeed, skip",
			pod:          createPod(v1.PodSucceeded),
			expectResult: true,
		},
		{
			name: "pod have cni allocate finalizer, skip",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			expectResult: true,
		},
		{
			name: "pod not update status, not skip",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationContainerStateSpec: "test",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "bar2",
							Image: "beep",
						},
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			},
			expectResult: false,
		},
		{
			name: "pod update status, skip",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodUpdateStatus: "test",
					},
				},
			},
			expectResult: true,
		},
		{
			name: "pod have rebuild info, skip",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationRebuildContainerInfo: "test",
					},
				},
			},
			expectResult: true,
		},
	}
	for _, ts := range testCase {
		t.Run(t.Name(), func(t *testing.T) {
			skip := kl.skipAdmit(ts.pod)
			assert.Equal(t, skip, ts.expectResult)
		})
	}
}

func createPod(phase v1.PodPhase) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bar",
			Namespace: "foo",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "bar",
					Image: "beep",
				},
			},
		},
		Status: v1.PodStatus{
			Phase: phase,
		},
	}
}

func TestGetAnnotationValue(t *testing.T) {
	testes := []struct {
		name        string
		annotation  map[string]string
		key         string
		expectValue string
	}{
		{
			name:        "annotation is nil, so value is nil ",
			annotation:  nil,
			key:         "testKey",
			expectValue: "",
		},
		{
			name:        "annotation is empty, so value is nil ",
			annotation:  make(map[string]string, 2),
			key:         "testKey",
			expectValue: "",
		},
		{
			name:        "key not exist in map, so value is nil ",
			annotation:  map[string]string{"testKey2": "testValue2"},
			key:         "testKey",
			expectValue: "",
		},
		{
			name:        "key exist in map, so everything is ok",
			annotation:  map[string]string{"testKey": "testValue"},
			key:         "testKey",
			expectValue: "testValue",
		},
	}

	for _, tt := range testes {
		t.Run(tt.name, func(t *testing.T) {
			value := getAnnotationValue(tt.annotation, tt.key)
			assert.Equal(t, value, tt.expectValue)
		})
	}
}

func TestSkipPodBecausePending(t *testing.T) {
	tests := []struct {
		name string
		pod  *v1.Pod
		skip bool
	}{
		{
			name: "pod is nil,should not skip and kill",
			pod:  nil,
			skip: false,
		},
		{
			name: "time out is zero,should skip and not kill",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodPendingTimeSeconds: "0",
					},
				},
			},
			skip: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			skip := skipPodBecausePending(test.pod)
			assert.Equal(t, skip, test.skip)
		})
	}
}
