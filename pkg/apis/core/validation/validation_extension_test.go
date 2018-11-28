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

package validation

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func getResourceRequirements(requests, limits core.ResourceList) core.ResourceRequirements {
	res := core.ResourceRequirements{}
	res.Requests = requests
	res.Limits = limits
	return res
}

func TestValidateContainerResourceUpdate(t *testing.T) {
	tests := []struct {
		new  core.Pod
		old  core.Pod
		err  string
		test string
	}{
		{
			core.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: core.PodSpec{
					NodeName: "node1",
					Containers: []core.Container{
						{
							Name:      "bar",
							Image:     "foo:V2",
							Resources: getResourceRequirements(getResourceList("8", "8Gi"), getResourceList("8", "8Gi")),
						},
					},
				},
			},
			core.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: core.PodSpec{
					NodeName: "node1",
					Containers: []core.Container{
						{
							Name:      "bar",
							Image:     "foo:V2",
							Resources: getResourceRequirements(getResourceList("4", "6Gi"), getResourceList("4", "6Gi")),
						},
					},
				},
			},
			"",
			"normal pod resource update",
		},
		{
			core.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: core.PodSpec{
					NodeName: "node1",
					Containers: []core.Container{
						{
							Name:      "bar",
							Image:     "foo:V2",
							Resources: getResourceRequirements(getResourceList("8", "8Gi"), getResourceList("8", "8Gi")),
						},
					},
				},
			},
			core.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: core.PodSpec{
					NodeName: "node1",
					Containers: []core.Container{
						{
							Name:      "bar",
							Image:     "foo:V3",
							Resources: getResourceRequirements(getResourceList("4", "6Gi"), getResourceList("8", "8Gi")),
						},
					},
				},
			},
			"spec.containers: Forbidden: container resource updates must not change pod QoS class",
			"bad pod resource update",
		},
	}
	for _, test := range tests {
		test.new.ObjectMeta.ResourceVersion = "1"
		test.old.ObjectMeta.ResourceVersion = "1"
		errs := ValidatePodUpdate(&test.new, &test.old)
		if test.err == "" {
			if len(errs) != 0 {
				t.Errorf("unexpected invalid: %s (%+v)\nA: %+v\nB: %+v", test.test, errs, test.new, test.old)
			}
		} else {
			if len(errs) == 0 {
				t.Errorf("unexpected valid: %s\nA: %+v\nB: %+v", test.test, test.new, test.old)
			} else if actualErr := errs.ToAggregate().Error(); !strings.Contains(actualErr, test.err) {
				t.Errorf("unexpected error message: %s\nExpected error: %s\nActual error: %s", test.test, test.err, actualErr)
			}
		}
	}
}
