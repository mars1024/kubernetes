package cmdb

import (
	"testing"

	"fmt"
	"strings"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
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

func TestValidateCreate(t *testing.T) {
	handler := NewAlipayCMDB()
	pod := newPod()

	err := handler.Validate(admission.NewAttributesRecord(pod, nil, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Create, nil))
	if err != nil {
		t.Errorf("Validate pod %#v error: %v", pod, err)
	}
}

func TestValidateCreateError(t *testing.T) {
	handler := NewAlipayCMDB()

	for _, k := range mustRequiredCMDBLabels {
		pod := newPod()
		delete(pod.Labels, k)
		err := handler.Validate(admission.NewAttributesRecord(pod, nil, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Create, nil))
		if err == nil {
			t.Errorf("Validate pod %#v expect an error", pod)
		} else {
			if msg := fmt.Sprintf("label %s is required", k); !strings.Contains(err.Error(), msg) {
				t.Errorf("Validate pod %#v error should contain %q", pod, msg)
			}
		}
	}
}

func TestValidateUpdate(t *testing.T) {
	handler := NewAlipayCMDB()
	pod := newPod()

	err := handler.Validate(admission.NewAttributesRecord(pod, pod, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Update, nil))
	if err != nil {
		t.Errorf("Validate pod %#v update error: %v", pod, err)
	}
}

func TestValidateUpdateError(t *testing.T) {
	handler := NewAlipayCMDB()
	old := newPod()

	for _, k := range mustRequiredCMDBLabels {
		pod := old.DeepCopy()
		pod.Labels[k] = old.Labels[k] + " "
		err := handler.Validate(admission.NewAttributesRecord(pod, old, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Update, nil))
		if err == nil {
			t.Errorf("Validate pod %#v expect an error", pod)
		} else {
			if msg := fmt.Sprintf("label %s is immutable", k); !strings.Contains(err.Error(), msg) {
				t.Errorf("Validate pod %#v error should contain %q", pod, msg)
			}
		}
	}
}

func TestHandles(t *testing.T) {
	for op, shouldHandle := range map[admission.Operation]bool{
		admission.Create:  true,
		admission.Update:  true,
		admission.Connect: false,
		admission.Delete:  false,
	} {
		handler := NewAlipayCMDB()
		if e, a := shouldHandle, handler.Handles(op); e != a {
			t.Errorf("%v: shouldHandle=%t, handles=%t", op, e, a)
		}
	}
}

// TestOtherResources ensures that this admission controller is a no-op for other resources,
// subresources, and non-pods.
func TestOtherResources(t *testing.T) {
	namespace := "testnamespace"
	name := "testname"
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{},
		},
	}
	for _, k := range mustRequiredCMDBLabels {
		pod.Labels[k] = k
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
			subresource: "eviction",
			object:      pod,
		},
		{
			name:     "non-pod object",
			kind:     "Pod",
			resource: "pods",
			object:   &core.Service{},
		},
	}

	for _, tc := range tests {
		handler := NewAlipayCMDB()

		err := handler.Validate(admission.NewAttributesRecord(tc.object, nil, core.Kind(tc.kind).WithVersion("version"), namespace, name, core.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Create, nil))

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

func newPod() *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cmdb-pod",
			Labels: map[string]string{
				sigmak8sapi.LabelAppName:       "app-name",
				sigmak8sapi.LabelPodSn:         "pod-sn",
				sigmak8sapi.LabelDeployUnit:    "deploy-unit",
				sigmak8sapi.LabelSite:          "site",
				sigmak8sapi.LabelInstanceGroup: "instance-group",
			},
		},
		Spec: core.PodSpec{},
	}
}
