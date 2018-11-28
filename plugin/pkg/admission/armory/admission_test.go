/*
Copyright 2015 The Kubernetes Authors.

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

package armory

import (
	"testing"

	sigmaapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"strings"
)

func TestRegister(t *testing.T) {
	plugins := admission.NewPlugins()
	Register(plugins)
	registered := plugins.Registered()
	if len(registered) == 1 && registered[0] == PluginName {
		return
	} else {
		t.Errorf("Register failed")
	}
}

func TestNewArmory(t *testing.T) {
	NewArmory()
}

// TestAdmitCreate verifies all create requests for pods result in every container's labels
func TestAdmitCreate(t *testing.T) {
	namespace := "test"
	handler := &Armory{}
	//sn := "test-sn"
	pod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
		},
	}
	err := handler.Admit(admission.NewAttributesRecord(&pod, nil, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Create, nil))
	if err != nil {
		t.Errorf("Unexpected error returned from admission handler")
	}
	if sn, ok := pod.Labels[sigmaapi.LabelPodSn]; !ok {
		t.Errorf("Sn must be set by armory admission")
	} else if sn == "" {
		t.Errorf("Sn must be not empty")
	}
}

func TestAdmitUpdate(t *testing.T) {
	namespace := "test"
	handler := &Armory{}
	//sn := "test-sn"
	pod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
			Labels: map[string]string{
				sigmaapi.LabelPodSn: "test-sn",
			},
		},
	}
	err := handler.Admit(admission.NewAttributesRecord(&pod, nil, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Update, nil))
	if err != nil {
		t.Errorf("Unexpected error returned from admission handler")
	}
	if sn, ok := pod.Labels[sigmaapi.LabelPodSn]; !ok {
		t.Errorf("Sn must be set by armory admission")
	} else if sn == "" {
		t.Errorf("Sn must be not empty")
	}
}

func TestValidateCreate(t *testing.T) {
	namespace := "test"
	handler := &Armory{}
	pod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
			Labels: map[string]string{
				sigmaapi.LabelPodSn: "test-sn",
			},
		},
	}
	expectedError := `must be set`
	err := handler.Validate(admission.NewAttributesRecord(&pod, nil, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Create, nil))
	if err == nil {
		t.Errorf("missing expected error")
	}
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf(err.Error())
	}

	pod.ObjectMeta.Labels = nil
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Create, nil))
	if err == nil {
		t.Errorf("missing expected error")
	}
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf(err.Error())
	}

	pod.ObjectMeta.Labels = map[string]string{
		sigmaapi.LabelPodSn:   "test-sn",
		sigmaapi.LabelAppName: "app1",
		sigmaapi.LabelSite:    "site0",
	}
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Create, nil))
	if err != nil {
		t.Errorf("Unexpected error returned from admission handler")
	}
}

func TestValidateUpdate(t *testing.T) {
	namespace := "test"
	handler := &Armory{}
	pod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
			Labels: map[string]string{
				sigmaapi.LabelPodSn: "test-update-sn",
			},
		},
	}
	oldPod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
			Labels: map[string]string{
				sigmaapi.LabelPodSn: "test-sn",
			},
		},
	}
	expectedError := `pods "123" is forbidden: labels sigma.ali/sn can not update`
	err := handler.Validate(admission.NewAttributesRecord(&pod, &oldPod, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Update, nil))
	if err == nil {
		t.Error("missing expected error")
	}
	if err.Error() != expectedError {
		t.Error(err)
	}

	pod.ObjectMeta.Labels = map[string]string{
		"a": "b",
		"sigma.ali/instance-group": "g1",
	}
	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Update, nil))
	if err != nil {
		t.Fatal("must not expected error")
	}

	pod.ObjectMeta.Labels = map[string]string{
		"a": "b",
		"sigma.alibaba-inc.com/app-unit": "g1",
	}
	expectedError = `pods "123" is forbidden: labels sigma.alibaba-inc.com/app-unit can not update`
	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Update, nil))
	if err == nil {
		t.Error("missing expected error")
	}
	if err.Error() != expectedError {
		t.Error(err)
	}
}

// TestOtherResources ensures that this admission controller is a no-op for other resources,
// subresources, and non-pods.
func TestOtherResources(t *testing.T) {
	namespace := "testnamespace"
	name := "testname"
	pod := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{Name: "ctr2", Image: "image", ImagePullPolicy: api.PullNever},
			},
		},
	}
	tests := []struct {
		name        string
		kind        string
		resource    string
		subresource string
		object      runtime.Object
		expectError bool
	}{
		{
			name:     "non-pod resource",
			kind:     "Foo",
			resource: "foos",
			object:   pod,
		},
		{
			name:        "pod subresource",
			kind:        "Pod",
			resource:    "pods",
			subresource: "exec",
			object:      pod,
		},
		{
			name:        "non-pod object",
			kind:        "Pod",
			resource:    "pods",
			object:      &api.Service{},
			expectError: true,
		},
	}

	for _, tc := range tests {
		handler := &Armory{}

		err := handler.Admit(admission.NewAttributesRecord(tc.object, nil, api.Kind(tc.kind).WithVersion("version"), namespace, name, api.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Create, nil))

		if tc.expectError {
			if err == nil {
				t.Errorf("%s: unexpected nil error", tc.name)
			}
			continue
		}

		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
			continue
		}

		err = handler.Validate(admission.NewAttributesRecord(tc.object, nil, api.Kind(tc.kind).WithVersion("version"), namespace, name, api.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Create, nil))
		if tc.expectError {
			if err == nil {
				t.Errorf("%s: unexpected nil error", tc.name)
			}
			continue
		}

		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
			continue
		}
	}
}

// TestAdmissionError verifies all create requests for pods result in every container's labels
func TestAdmissionError(t *testing.T) {
	namespace := "test"
	handler := &Armory{}
	pod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
			Labels: map[string]string{
				sigmaapi.LabelPodSn: "test-sn",
			},
		},
	}

	oldPod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
			Labels: map[string]string{
				sigmaapi.LabelPodSn: "test-sn",
			},
		},
	}
	expectedError := `pods "123" is forbidden: labels sigma.ali/app-name, sigma.ali/site must be set`
	err := handler.Validate(admission.NewAttributesRecord(&pod, &oldPod, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Create, nil))
	if err == nil {
		t.Errorf("missing expected error")
	}
	if err.Error() != expectedError {
		t.Error(err)
	}

	expectedError = `Armory Admission only handles Create or Update event`
	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Delete, nil))
	if err == nil {
		t.Errorf("missing expected error")
	}
	if err.Error() != expectedError {
		t.Error(err)
	}

	expectedError = `Armory Admission only handles Create or Update event`
	err = handler.Admit(admission.NewAttributesRecord(&pod, &oldPod, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Delete, nil))
	if err == nil {
		t.Errorf("missing expected error")
	}
	if err.Error() != expectedError {
		t.Error(err)
	}

	expectedError = `Resource was marked with kind Pod but was unable to be converted`
	err = handler.Validate(admission.NewAttributesRecord(&extensions.Deployment{}, &extensions.Deployment{}, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Create, nil))
	if err == nil {
		t.Errorf("missing expected error")
	}
	if err.Error() != expectedError {
		t.Error(err)
	}

	pod.ObjectMeta.Labels = map[string]string{
		sigmaapi.LabelPodSn:   "test-sn",
		sigmaapi.LabelAppName: "app1",
		sigmaapi.LabelSite:    "site0",
	}
	expectedError = `Resource was marked with kind Pod but was unable to be converted`
	err = handler.Validate(admission.NewAttributesRecord(&pod, &extensions.Deployment{}, api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, api.Resource("pods").WithVersion("version"), "", admission.Update, nil))
	if err == nil {
		t.Fatalf("missing expected error: %s", expectedError)
	}
	if err.Error() != expectedError {
		t.Error(err)
	}
}
