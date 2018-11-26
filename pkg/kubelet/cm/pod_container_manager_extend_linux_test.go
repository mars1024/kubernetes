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

package cm

import (
	"encoding/json"
	"testing"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetCgroupParentAnnotation(t *testing.T) {
	allocWithCgroupParent := sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			sigmak8sapi.Container{
				Name: "foo",
				HostConfig: sigmak8sapi.HostConfigInfo{
					CgroupParent: "/cgrouptest",
				},
			},
		},
	}
	allocWithCgroupParentBytes, _ := json.Marshal(allocWithCgroupParent)

	allocWithCgroupEmptyParent := sigmak8sapi.AllocSpec{
		Containers: []sigmak8sapi.Container{
			sigmak8sapi.Container{
				Name: "foo",
				HostConfig: sigmak8sapi.HostConfigInfo{
					CgroupParent: "",
				},
			},
		},
	}
	allocWithCgroupEmptyParentBytes, _ := json.Marshal(allocWithCgroupEmptyParent)

	for caseName, testCase := range map[string]struct {
		pod                *v1.Pod
		expectCgroupParent string
	}{
		"pod has cgroup parent": {
			pod: &v1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "foo",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(allocWithCgroupParentBytes),
					},
				},
			},
			expectCgroupParent: "/cgrouptest",
		},
		"pod has empty cgroup parent": {
			pod: &v1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "foo",
					Namespace: "new",
					Annotations: map[string]string{
						sigmak8sapi.AnnotationPodAllocSpec: string(allocWithCgroupEmptyParentBytes),
					},
				},
			},
			expectCgroupParent: "",
		},
		"pod has no annotation": {
			pod: &v1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "foo",
					Namespace: "new",
				},
			},
			expectCgroupParent: "",
		},
	} {
		cgroupParent := GetCgroupParentFromAnnotation(testCase.pod)
		if cgroupParent != testCase.expectCgroupParent {
			t.Errorf("Failed to test case %s: expected cgroup parent %s, bug got %s", caseName, testCase.expectCgroupParent, cgroupParent)
		}
	}
}