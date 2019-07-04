/*
Copyright 2019 The Kubernetes Authors.

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

package node

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInPlaceControlPod(t *testing.T) {
	controller := true
	for caseName, testCase := range map[string]struct {
		pod            *corev1.Pod
		inPlaceControl bool
	}{
		"pod has owner reference": {
			pod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "foo",
					Namespace: "new",
					OwnerReferences: []metav1.OwnerReference{
						{
							Controller: &controller,
							Kind:       "InPlaceSet",
						},
					},
				},
			},
			inPlaceControl: true,
		},
		"pod has owner reference for serverless": {
			pod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "foo",
					Namespace: "new",
					OwnerReferences: []metav1.OwnerReference{
						{
							Controller: &controller,
							Kind:       "CafeServerlessSet",
						},
					},
				},
			},
			inPlaceControl: true,
		},
		"snake pod": {
			pod: &corev1.Pod{
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
			inPlaceControl: false,
		},
	} {
		inPlaceControl := inPlaceControlPod(testCase.pod)
		if inPlaceControl != testCase.inPlaceControl {
			t.Errorf("Case %s: expect inPlaceControl %t bug got %t", caseName, testCase.inPlaceControl, inPlaceControl)
		}

	}
}
