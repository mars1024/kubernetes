// +build linux

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
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetCustomCgroupParentsFromConfigmap(t *testing.T) {

	for caseName, testCase := range map[string]struct {
		configMap           *v1.ConfigMap
		expectCgroupParents []string
		isErrorOccurs       bool
	}{
		"configMap has custom-cgroup-parents key": {
			configMap: &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "custom-cgroup-parents",
					Namespace: "kube-system",
				},
				Data: map[string]string{
					configMapKeyCustomCgroupParent: "/cgroup/parent1;/cgroup/parent2",
				},
			},
			expectCgroupParents: []string{"/cgroup/parent1", "/cgroup/parent2"},
			isErrorOccurs:       false,
		},
		"configMap doesn't has custom-cgroup-parents key": {
			configMap: &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:       "12345678",
					Name:      "custom-cgroup-parents",
					Namespace: "kube-system",
				},
				Data: map[string]string{
					"testkey1": "/cgroup/parent1;/cgroup/parent2",
				},
			},
			expectCgroupParents: []string{},
			isErrorOccurs:       true,
		},
		"configMap is nil": {
			configMap:           nil,
			expectCgroupParents: []string{},
			isErrorOccurs:       true,
		},
	} {
		customCgroupParents, err := getCustomCgroupParentsFromConfigmap(testCase.configMap)
		if testCase.isErrorOccurs && err == nil || !testCase.isErrorOccurs && err != nil {
			t.Errorf("Testcase %s: expected error happens: %v, but got: %v", caseName, testCase.isErrorOccurs, err)
		}
		if !reflect.DeepEqual(customCgroupParents, testCase.expectCgroupParents) {
			t.Errorf("Testcase %s: expect cgroup parents %v, bug got %v", caseName, testCase.expectCgroupParents, customCgroupParents)
		}
	}
}