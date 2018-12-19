package zappinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
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

func TestHandles(t *testing.T) {
	for op, shouldHandle := range map[admission.Operation]bool{
		admission.Create:  true,
		admission.Update:  true,
		admission.Connect: false,
		admission.Delete:  false,
	} {
		handler := NewAlipayZAppInfo()
		if e, a := shouldHandle, handler.Handles(op); e != a {
			t.Errorf("%v: shouldHandle=%t, handles=%t", op, e, a)
		}
	}
}

func TestAdmit(t *testing.T) {
	for _, test := range []struct {
		name     string
		admit    bool
		getPods  func() (*core.Pod, *core.Pod)
		validate func(*core.Pod)
	}{
		{
			name:  "admit create success",
			admit: true,
			getPods: func() (*core.Pod, *core.Pod) {
				pod := newPod()
				pod.Annotations = make(map[string]string)
				return pod, nil
			},
			validate: func(pod *core.Pod) {
				info, err := getPodZAppInfo(pod)
				assert.NoError(t, err)
				assert.Equal(t, info.Spec.AppName, pod.Labels[sigmak8sapi.LabelAppName])
				assert.Equal(t, info.Spec.ServerType, "DOCKER")
				assert.Equal(t, info.Spec.Zone, pod.Labels[alipaysigmak8sapi.LabelZone])
			},
		},
		{
			name:  "admit update success",
			admit: true,
			getPods: func() (*core.Pod, *core.Pod) {
				pod := newPod()
				pod.Annotations = make(map[string]string)
				return pod, pod.DeepCopy()
			},
		},
	} {
		handler := NewAlipayZAppInfo()
		new, old := test.getPods()

		op := admission.Create
		if old != nil {
			op = admission.Update
		}

		a := admission.NewAttributesRecord(new, old, core.Kind("Pod").WithVersion("version"), new.Namespace, new.Name, core.Resource("pods").WithVersion("version"), "", op, false, nil)
		err := handler.Admit(a)

		if test.admit {
			assert.True(t, err == nil, "[%s] admit true: %v", test.name, err)
		} else {
			assert.True(t, err != nil, "[%s] expect error: %v", test.name, err)
		}
		if test.validate != nil {
			test.validate(new)
		}
	}
}

func TestValidate(t *testing.T) {
	for _, test := range []struct {
		name     string
		admit    bool
		getPods  func() (*core.Pod, *core.Pod)
		validate func(*core.Pod)
	}{
		{
			name:  "admit create success",
			admit: true,
			getPods: func() (*core.Pod, *core.Pod) {
				pod := newPod()
				return pod, nil
			},
		},
		{
			name:  "admit create success, registered true",
			admit: true,
			getPods: func() (*core.Pod, *core.Pod) {
				pod := newPod()
				pod.Labels = make(map[string]string)
				pod.Annotations = map[string]string{
					alipaysigmak8sapi.AnnotationZappinfo: `{"status":{"registered":true}}`,
				}
				return pod, nil
			},
		},
		{
			name:  "admit update success",
			admit: true,
			getPods: func() (*core.Pod, *core.Pod) {
				pod := newPod()
				return pod, pod.DeepCopy()
			},
		},
		{
			name:  "admit create false, no zone",
			admit: false,
			getPods: func() (*core.Pod, *core.Pod) {
				pod := newPod()
				pod.Annotations[alipaysigmak8sapi.AnnotationZappinfo] = `{
					"spec": {
						"appName": "app1",
						"serverType": "DOCKER"
					}
				}`
				return pod, nil
			},
		},
		{
			name:  "admit create false, no appName",
			admit: false,
			getPods: func() (*core.Pod, *core.Pod) {
				pod := newPod()
				pod.Annotations[alipaysigmak8sapi.AnnotationZappinfo] = `{
					"spec": {
						"zone": "GZ00C",
						"serverType": "DOCKER"
					}
				}`
				return pod, nil
			},
		},
		{
			name:  "admit update fail, spec cannot change",
			admit: false,
			getPods: func() (*core.Pod, *core.Pod) {
				pod := newPod()
				old := pod.DeepCopy()
				old.Annotations[alipaysigmak8sapi.AnnotationZappinfo] = `{
					"spec": {
						"appName": "app-2",
						"zone": "GZ00C",
						"serverType": "DOCKER"
					}
				}`
				return pod, old
			},
		},
	} {
		handler := NewAlipayZAppInfo()
		new, old := test.getPods()

		op := admission.Create
		if old != nil {
			op = admission.Update
		}

		a := admission.NewAttributesRecord(new, old, core.Kind("Pod").WithVersion("version"), new.Namespace, new.Name, core.Resource("pods").WithVersion("version"), "", op, false, nil)
		err := handler.Validate(a)

		if test.admit {
			assert.True(t, err == nil, "[%s] admit true: %v", test.name, err)
		} else {
			assert.True(t, err != nil, "[%s] expect error: %v", test.name, err)
		}
		if test.validate != nil {
			test.validate(new)
		}
	}
}

func TestOtherResources(t *testing.T) {
	pod := newPod()

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
		handler := NewAlipayZAppInfo()

		err := handler.Admit(admission.NewAttributesRecord(tc.object, nil, core.Kind(tc.kind).WithVersion("version"), pod.Namespace, pod.Name, core.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Create, false, nil))

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

		err = handler.Validate(admission.NewAttributesRecord(tc.object, nil, core.Kind(tc.kind).WithVersion("version"), pod.Namespace, pod.Name, core.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Create, false, nil))

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
			Name: "test-zappinfo-pod",
			Labels: map[string]string{
				sigmak8sapi.LabelAppName:    "app1",
				alipaysigmak8sapi.LabelZone: "GZ00C",
			},
			Annotations: map[string]string{
				alipaysigmak8sapi.AnnotationZappinfo: `{
					"spec": {
						"appName": "app1",
						"zone": "GZ00C",
						"serverType": "DOCKER"
					}
				}`,
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{Image: "pause:2.0"},
			},
		},
	}
}
