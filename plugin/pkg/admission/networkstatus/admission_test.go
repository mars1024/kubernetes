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

package networkstatus

import (
	"encoding/json"
	"testing"

	sigmaapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/extensions"
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

func TestNewNetworkStatus(t *testing.T) {
	NewNetworkStatus()
}

// TestAdmission verifies all update requests for pods result in every container's labels
func TestAdmission(t *testing.T) {
	namespace := "test"
	handler := &NetworkStatus{}
	status := sigmaapi.NetworkStatus{
		VlanID:              "701",
		NetworkPrefixLength: 24,
		Gateway:             "100.88.23.25",
		MACAddress:          "02:42:64:58:17:19",
	}

	oldStatus := sigmaapi.NetworkStatus{
		VlanID:              "701",
		NetworkPrefixLength: 24,
		Gateway:             "100.88.23.25",
		MACAddress:          "02:42:64:58:17:19",
	}
	statusBytes, err := json.Marshal(status)
	pod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
		},
	}
	oldPod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
		},
	}

	// error event
	expectedError := `NetworkStatus Admission only handles Update event`

	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Create, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	// nil Annotations
	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err != nil {
		t.Errorf("Unexpected error returned from admission handler: %s", err)
	}

	// without Annotations
	pod.ObjectMeta.Annotations = map[string]string{}
	oldPod.ObjectMeta.Annotations = map[string]string{}

	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err != nil {
		t.Errorf("Unexpected error returned from admission handler: %s", err)
	}

	// with new Annotations, without old Annotations
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)

	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err != nil {
		t.Errorf("Unexpected error returned from admission handler: %s", err)
	}

	// with new Annotations, with old Annotations
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)
	oldPod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)

	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err != nil {
		t.Errorf("Unexpected error returned from admission handler: %s", err)
	}

	// with new Annotations, with old Annotations, sandboxID changed
	oldPod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)
	changedStatus := status
	changedStatus.SandboxId = "another-sb-id"
	changedStatusBytes, _ := json.Marshal(changedStatus)
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(changedStatusBytes)

	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err != nil {
		t.Errorf("Unexpected error returned from admission handler: %s", err)
	}

	// with new Annotations, with old Annotations, immutable field changed
	oldPod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)
	changedStatus = status
	changedStatus.VlanID = "703"
	changedStatusBytes, _ = json.Marshal(changedStatus)
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(changedStatusBytes)

	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Errorf("expected error is missing")
	}

	// without new Annotations, with old Annotations
	delete(pod.ObjectMeta.Annotations, sigmaapi.AnnotationPodNetworkStats)
	oldPod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)

	expectedError = `pods "123" is forbidden: annotation pod.beta1.sigma.ali/network-status can not update`
	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err != nil {
		t.Fatalf("Unexpected expected error: %v", err)
	}

	// error format with new status 1
	status = sigmaapi.NetworkStatus{
		VlanID:              "8192",
		NetworkPrefixLength: 33,
		Gateway:             "225.222.222.256",
		MACAddress:          "02:42:64:58:17:19x",
	}
	statusBytes, err = json.Marshal(status)
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)
	expectedError = `pods "123" is forbidden: annotation pod.beta1.sigma.ali/network-status can not update due to invalid field: vlan must be between 1 and 4095, inclusive, invalid field: networkPrefixLen must be between 0 and 32, inclusive, invalid field: gateway must be a valid IP address, (e.g. 10.9.8.7), invalid field: macAddress address 02:42:64:58:17:19x: invalid MAC address`
	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	// error format with new status 2
	status = sigmaapi.NetworkStatus{
		VlanID:              "vlan0",
		NetworkPrefixLength: 33,
		Gateway:             "225.222.222.256",
		MACAddress:          "02:42:64:58:17:19x",
	}
	statusBytes, err = json.Marshal(status)
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)
	expectedError = `pods "123" is forbidden: annotation pod.beta1.sigma.ali/network-status can not update due to invalid field: vlan strconv.Atoi: parsing "vlan0": invalid syntax, invalid field: networkPrefixLen must be between 0 and 32, inclusive, invalid field: gateway must be a valid IP address, (e.g. 10.9.8.7), invalid field: macAddress address 02:42:64:58:17:19x: invalid MAC address`
	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	// error format with old status
	status = sigmaapi.NetworkStatus{
		VlanID:              "701",
		NetworkPrefixLength: 24,
		Gateway:             "100.88.23.25",
		MACAddress:          "02:42:64:58:17:19",
	}
	statusBytes, err = json.Marshal(status)
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)

	oldStatus = sigmaapi.NetworkStatus{
		VlanID:              "8192",
		NetworkPrefixLength: 33,
		Gateway:             "225.222.222.256",
		MACAddress:          "02:42:64:58:17:19x",
	}
	oldStatusBytes, err := json.Marshal(oldStatus)
	oldPod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(oldStatusBytes)
	expectedError = "pods \"123\" is forbidden: annotation pod.beta1.sigma.ali/network-status can not update due to invalid field: vlan must be between 1 and 4095, inclusive, invalid field: networkPrefixLen must be between 0 and 32, inclusive, invalid field: gateway must be a valid IP address, (e.g. 10.9.8.7), invalid field: macAddress address 02:42:64:58:17:19x: invalid MAC address"
	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	// empty MACAddress is valid
	status = sigmaapi.NetworkStatus{
		VlanID:              "701",
		NetworkPrefixLength: 24,
		Gateway:             "100.88.23.25",
	}
	statusBytes, err = json.Marshal(status)
	pod.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)
	oldPod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)
	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err != nil {
		t.Fatalf("uexpected error: %v", err)
	}

	// empty VlanID is valid
	status = sigmaapi.NetworkStatus{
		Gateway:    "100.88.23.25",
		VPortToken: "vportToken",
		VPortID:    "vportID",
		VSwitchID:  "vswitchID",
	}
	statusBytes, err = json.Marshal(status)
	pod.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)
	oldPod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = string(statusBytes)
	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err != nil {
		t.Fatalf("uexpected error: %v", err)
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
		handler := &NetworkStatus{}

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

// TestAdmission verifies all update requests for pods result in every container's labels
func TestAdmissionError(t *testing.T) {
	namespace := "test"
	handler := &NetworkStatus{}
	status := sigmaapi.NetworkStatus{
		VlanID:              "701",
		NetworkPrefixLength: 24,
		Gateway:             "100.88.23.25",
		MACAddress:          "02:42:64:58:17:19",
	}
	statusBytes, err := json.Marshal(status)
	pod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
			Annotations: map[string]string{
				sigmaapi.AnnotationPodNetworkStats: string(statusBytes),
			},
		},
	}
	oldPod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "123",
			Namespace: namespace,
			Annotations: map[string]string{
				sigmaapi.AnnotationPodNetworkStats: string(statusBytes),
			},
		},
	}

	// old object is not Pod
	expectedError := `Resource was marked with kind Pod but was unable to be converted`
	err = handler.Validate(admission.NewAttributesRecord(&pod, &extensions.Deployment{},
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	// new annotations sigmaapi.AnnotationPodNetworkStats is not json string
	pod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = "error network status"
	expectedError = "pods \"123\" is forbidden: annotation pod.beta1.sigma.ali/network-status can not update due to json unmarshal error `invalid character 'e' looking for beginning of value`"
	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}

	// old annotations sigmaapi.AnnotationPodNetworkStats is not json string
	oldPod.ObjectMeta.Annotations[sigmaapi.AnnotationPodNetworkStats] = "error network status"
	expectedError = "pods \"123\" is forbidden: annotation pod.beta1.sigma.ali/network-status can not update due to json unmarshal error `invalid character 'e' looking for beginning of value`"
	err = handler.Validate(admission.NewAttributesRecord(&pod, &oldPod,
		api.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name,
		api.Resource("pods").WithVersion("version"), "", admission.Update, false, nil))
	if err == nil {
		t.Fatal("missing expected error")
	}
	if err.Error() != expectedError {
		t.Fatal(err)
	}
}
