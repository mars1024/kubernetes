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

package containerstate

import (
	"encoding/json"
	"testing"

	sigmaapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
)

func TestRegister(t *testing.T) {
	plugins := admission.NewPlugins()
	Register(plugins)
	registered := plugins.Registered()
	if len(registered) == 1 && registered[0] == PluginName {
		return
	}
	t.Errorf("Register failed")
}

func TestNewContainerState(t *testing.T) {
	NewContainerState()
}

// TestAdmission verifies all update requests for pods result in every container's labels
func TestAdmission(t *testing.T) {
	namespace := "test"
	handler := &ContainerState{}
	state := sigmaapi.ContainerStateSpec{
		States: map[sigmaapi.ContainerInfo]sigmaapi.ContainerState{
			sigmaapi.ContainerInfo{Name: "name1"}: sigmaapi.ContainerStateCreated,
			sigmaapi.ContainerInfo{Name: "name2"}: sigmaapi.ContainerStateExited,
			sigmaapi.ContainerInfo{Name: "name3"}: sigmaapi.ContainerStateRunning,
			sigmaapi.ContainerInfo{Name: "name4"}: sigmaapi.ContainerStatePaused,
			sigmaapi.ContainerInfo{Name: "name5"}: sigmaapi.ContainerStateSuspended},
	}
	stateBytes, err := json.Marshal(state)
	status := sigmaapi.ContainerStateStatus{
		Statuses: map[sigmaapi.ContainerInfo]sigmaapi.ContainerStatus{
			sigmaapi.ContainerInfo{Name: "name1"}: {
				CurrentState: sigmaapi.ContainerStateCreated,
				LastState:    sigmaapi.ContainerStateCreated,
			},
			sigmaapi.ContainerInfo{Name: "name2"}: {
				CurrentState: sigmaapi.ContainerStateCreated,
				LastState:    sigmaapi.ContainerStateCreated,
			},
		},
	}
	statusBytes, err := json.Marshal(status)
	pod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
			Annotations: map[string]string{
				sigmaapi.AnnotationContainerStateSpec: string(stateBytes),
				sigmaapi.AnnotationPodUpdateStatus:    string(statusBytes),
			},
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{Name: "name1"},
				{Name: "name2"},
				{Name: "name3"},
				{Name: "name4"},
				{Name: "name5"},
			},
		},
	}
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err != nil {
		t.Errorf("Unexpected error returned from admission handler: %s", err)
	}

	pod.Spec.Containers = []api.Container{
		{Name: "name1"},
		{Name: "name2"},
		{Name: "name4"},
		{Name: "name5"},
	}
	state.States[sigmaapi.ContainerInfo{Name: "name3"}] = sigmaapi.ContainerStateCreated
	stateBytes, err = json.Marshal(state)
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationContainerStateSpec] = string(stateBytes)

	expectedError := `pods "123" is forbidden: annotation pod.beta1.sigma.ali/update-status can not UPDATE due to container named name3 not found`
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	delete(state.States, sigmaapi.ContainerInfo{Name: "name3"})
	state.States[sigmaapi.ContainerInfo{Name: "name2"}] = sigmaapi.ContainerState("wrong")
	stateBytes, err = json.Marshal(state)
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationContainerStateSpec] = string(stateBytes)
	expectedError = `pods "123" is forbidden: annotation pod.beta1.sigma.ali/update-status can not UPDATE due to container state wrong is not valid`
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	// reset
	state = sigmaapi.ContainerStateSpec{
		States: map[sigmaapi.ContainerInfo]sigmaapi.ContainerState{
			sigmaapi.ContainerInfo{Name: "name1"}: sigmaapi.ContainerStateCreated,
			sigmaapi.ContainerInfo{Name: "name2"}: sigmaapi.ContainerStateCreated,
		},
	}
	stateBytes, err = json.Marshal(state)
	status = sigmaapi.ContainerStateStatus{
		Statuses: map[sigmaapi.ContainerInfo]sigmaapi.ContainerStatus{
			sigmaapi.ContainerInfo{Name: "name1"}: {
				CurrentState: sigmaapi.ContainerStateCreated,
				LastState:    sigmaapi.ContainerStateCreated,
			},
			sigmaapi.ContainerInfo{Name: "name2"}: {
				CurrentState: sigmaapi.ContainerStateCreated,
				LastState:    sigmaapi.ContainerStateCreated,
			},
			sigmaapi.ContainerInfo{Name: "name3"}: {
				CurrentState: sigmaapi.ContainerStateCreated,
				LastState:    sigmaapi.ContainerStateCreated,
			},
		},
	}
	statusBytes, err = json.Marshal(status)
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationContainerStateSpec] = string(stateBytes)
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationPodUpdateStatus] = string(statusBytes)

	expectedError = `pods "123" is forbidden: annotation pod.beta1.sigma.ali/update-status can not UPDATE due to container named name3 not found`
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	//
	status = sigmaapi.ContainerStateStatus{
		Statuses: map[sigmaapi.ContainerInfo]sigmaapi.ContainerStatus{
			sigmaapi.ContainerInfo{Name: "name1"}: {
				CurrentState: sigmaapi.ContainerStateCreated,
				LastState:    sigmaapi.ContainerStateCreated,
			},
			sigmaapi.ContainerInfo{Name: "name2"}: {
				CurrentState: sigmaapi.ContainerStateCreated,
				LastState:    sigmaapi.ContainerState("wrong"),
			},
		},
	}

	statusBytes, err = json.Marshal(status)
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationPodUpdateStatus] = string(statusBytes)
	expectedError = `pods "123" is forbidden: annotation pod.beta1.sigma.ali/update-status can not UPDATE due to container state wrong is not valid`
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	//
	status = sigmaapi.ContainerStateStatus{
		Statuses: map[sigmaapi.ContainerInfo]sigmaapi.ContainerStatus{
			sigmaapi.ContainerInfo{Name: "name1"}: {
				CurrentState: sigmaapi.ContainerStateCreated,
				LastState:    sigmaapi.ContainerStateCreated,
			},
			sigmaapi.ContainerInfo{Name: "name2"}: {
				CurrentState: sigmaapi.ContainerState("wrong"),
				LastState:    sigmaapi.ContainerStateCreated,
			},
		},
	}

	statusBytes, err = json.Marshal(status)
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationPodUpdateStatus] = string(statusBytes)
	expectedError = `pods "123" is forbidden: annotation pod.beta1.sigma.ali/update-status can not UPDATE due to container state wrong is not valid`
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	//
	pod.ObjectMeta.Annotations = nil
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err != nil {
		t.Errorf("Unexpected error returned from admission handler: %s", err)
	}

	// test if pod restart policy is never, can't set expect status to running
	state = sigmaapi.ContainerStateSpec{
		States: map[sigmaapi.ContainerInfo]sigmaapi.ContainerState{
			sigmaapi.ContainerInfo{Name: "name1"}: sigmaapi.ContainerStateRunning,
			sigmaapi.ContainerInfo{Name: "name2"}: sigmaapi.ContainerStateCreated,
		},
	}
	stateBytes, err = json.Marshal(state)
	status = sigmaapi.ContainerStateStatus{
		Statuses: map[sigmaapi.ContainerInfo]sigmaapi.ContainerStatus{
			sigmaapi.ContainerInfo{Name: "name1"}: {
				CurrentState: sigmaapi.ContainerStateCreated,
				LastState:    sigmaapi.ContainerStateCreated,
			},
			sigmaapi.ContainerInfo{Name: "name2"}: {
				CurrentState: sigmaapi.ContainerStateCreated,
				LastState:    sigmaapi.ContainerStateCreated,
			},
		},
	}
	statusBytes, err = json.Marshal(status)
	pod = api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
			Annotations: map[string]string{
				sigmaapi.AnnotationContainerStateSpec: string(stateBytes),
				sigmaapi.AnnotationPodUpdateStatus:    string(statusBytes),
			},
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{Name: "name1"},
				{Name: "name2"},
			},
			RestartPolicy: api.RestartPolicyNever,
		},
	}

	expectedError = `pods "123" is forbidden: annotation pod.beta1.sigma.ali/update-status can not UPDATE due to pod restart policy is never, so container can't be started`
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
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
		handler := &ContainerState{}

		err := handler.Validate(admission.NewAttributesRecord(tc.object, nil, api.Kind(tc.kind).WithVersion("version"), namespace, name, api.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Update, false, nil))

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

func TestAdmissionError(t *testing.T) {
	namespace := "test"
	handler := &ContainerState{}
	state := sigmaapi.ContainerStateSpec{
		States: map[sigmaapi.ContainerInfo]sigmaapi.ContainerState{
			sigmaapi.ContainerInfo{Name: "name1"}: sigmaapi.ContainerStateCreated,
			sigmaapi.ContainerInfo{Name: "name2"}: sigmaapi.ContainerStateCreated,
		},
	}
	stateBytes, err := json.Marshal(state)
	status := sigmaapi.ContainerStateStatus{
		Statuses: map[sigmaapi.ContainerInfo]sigmaapi.ContainerStatus{
			sigmaapi.ContainerInfo{Name: "name1"}: {
				CurrentState: sigmaapi.ContainerStateCreated,
				LastState:    sigmaapi.ContainerStateCreated,
			},
			sigmaapi.ContainerInfo{Name: "name2"}: {
				CurrentState: sigmaapi.ContainerStateCreated,
				LastState:    sigmaapi.ContainerStateCreated,
			},
		},
	}
	statusBytes, err := json.Marshal(status)
	pod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
			Annotations: map[string]string{
				sigmaapi.AnnotationContainerStateSpec: string(stateBytes),
				sigmaapi.AnnotationPodUpdateStatus:    string(statusBytes),
			},
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{Name: "name1"},
				{Name: "name2"},
			},
		},
	}

	pod.ObjectMeta.Annotations[sigmaapi.AnnotationContainerStateSpec] = "error json"
	expectedError := "pods \"123\" is forbidden: annotation pod.beta1.sigma.ali/update-status can not UPDATE due to json unmarshal error `invalid character 'e' looking for beginning of value`"
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	pod.ObjectMeta.Annotations[sigmaapi.AnnotationContainerStateSpec] = string(stateBytes)
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationPodUpdateStatus] = "error json"
	expectedError = "pods \"123\" is forbidden: annotation pod.beta1.sigma.ali/update-status can not UPDATE due to json unmarshal error `invalid character 'e' looking for beginning of value`"
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	expectedError = `ContainerState Admission only handles Update event`
	err = handler.Validate(admission.NewAttributesRecord(&pod, nil,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Delete, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}
}
