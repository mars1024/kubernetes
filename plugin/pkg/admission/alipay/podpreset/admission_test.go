package podpreset

import (
	"testing"
	"time"

	"strconv"

	"github.com/stretchr/testify/assert"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	"k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	kubeadmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
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
		admission.Update:  false,
		admission.Connect: false,
		admission.Delete:  false,
	} {
		handler := NewAlipayPodPreset()
		if e, a := shouldHandle, handler.Handles(op); e != a {
			t.Errorf("%v: shouldHandle=%t, handles=%t", op, e, a)
		}
	}
}

func TestValidate(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	for _, test := range []struct {
		name     string
		admit    bool
		cm       *core.ConfigMap
		initfunc func(*core.ConfigMap)
		validate func(*core.ConfigMap)
	}{
		{
			name:  "admit success, not preset",
			admit: true,
			initfunc: func(cm *core.ConfigMap) {
				cm.Labels = nil
			},
		},
		{
			name:  "admit success, not default",
			admit: true,
			cm:    newPodPresetConfigMap(false),
			initfunc: func(cm *core.ConfigMap) {
				cm.Labels = map[string]string{alipaysigmak8sapi.LabelDefaultPodPreset: "false"}
			},
		},
		{
			name:  "admit success, default",
			admit: true,
			cm:    newPodPresetConfigMap(false),
			initfunc: func(cm *core.ConfigMap) {
				cm.Labels = map[string]string{alipaysigmak8sapi.LabelDefaultPodPreset: "true"}
			},
		},
		{
			name:  "admit failed, default conflict",
			admit: false,
			cm:    newPodPresetConfigMap(true),
			initfunc: func(cm *core.ConfigMap) {
				cm.Labels = map[string]string{alipaysigmak8sapi.LabelDefaultPodPreset: "true"}
			},
		},
		{
			name:  "admit failed, invalid format",
			admit: false,
			initfunc: func(cm *core.ConfigMap) {
				cm.Labels = map[string]string{alipaysigmak8sapi.LabelDefaultPodPreset: "false"}
				cm.Data = map[string]string{"metadata": "abcdefg"}
			},
		},
	} {
		mockClient := &fake.Clientset{}
		if test.cm != nil {
			mockClient = fake.NewSimpleClientset(test.cm)
		}

		handler, f, err := newHandlerForTest(mockClient)
		if err != nil {
			t.Errorf("unexpected error initializing handler: %v", err)
		}
		f.Start(stopCh)

		cm := newPodPresetConfigMap(false)
		if test.initfunc != nil {
			test.initfunc(cm)
		}

		a := admission.NewAttributesRecord(cm, nil, core.Kind("ConfigMap").WithVersion("version"), cm.Namespace, cm.Name, core.Resource("configmaps").WithVersion("version"), "", admission.Create, false, nil)
		err = handler.Validate(a)

		if test.admit {
			assert.True(t, err == nil, "[%s] admit true: %v", test.name, err)
		} else {
			assert.True(t, err != nil, "[%s] expect error: %v", test.name, err)
		}
		if test.validate != nil {
			test.validate(cm)
		}
	}
}

func TestAdmit(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	for _, test := range []struct {
		name     string
		admit    bool
		cm       *core.ConfigMap
		initfunc func(*core.Pod, *core.ConfigMap)
		validate func(*core.Pod)
	}{
		{
			name:  "admit success",
			admit: true,
			cm:    newPodPresetConfigMap(false),
			validate: func(pod *core.Pod) {
				assert.Equal(t, "GZ00C", pod.Labels[alipaysigmak8sapi.LabelZone])
			},
		},
		{
			name:  "admit success, not overwrite",
			admit: true,
			cm:    newPodPresetConfigMap(false),
			initfunc: func(pod *core.Pod, cm *core.ConfigMap) {
				pod.Labels[alipaysigmak8sapi.LabelZone] = "GZ00X"
			},
			validate: func(pod *core.Pod) {
				assert.Equal(t, "GZ00X", pod.Labels[alipaysigmak8sapi.LabelZone])
			},
		},
		{
			name:  "admit success, default preset",
			admit: true,
			cm:    newPodPresetConfigMap(true),
			initfunc: func(pod *core.Pod, cm *core.ConfigMap) {
				pod.Labels = nil
			},
			validate: func(pod *core.Pod) {
				assert.Equal(t, "GZ00C", pod.Labels[alipaysigmak8sapi.LabelZone])
			},
		},
		{
			name:  "admit success, no default",
			admit: true,
			cm:    newPodPresetConfigMap(false),
			initfunc: func(pod *core.Pod, cm *core.ConfigMap) {
				pod.Labels = nil
			},
			validate: func(pod *core.Pod) {
				_, exists := pod.Labels[alipaysigmak8sapi.LabelZone]
				assert.False(t, exists)
			},
		},
		{
			name:  "admit failed, cm not found",
			admit: false,
			cm:    newPodPresetConfigMap(false),
			initfunc: func(pod *core.Pod, cm *core.ConfigMap) {
				pod.Labels[alipaysigmak8sapi.LabelPodPresetName] = "non-exists-cm"
			},
		},
		{
			name:  "admit failed, cm not found with diff namespace",
			admit: false,
			cm:    newPodPresetConfigMap(false),
			initfunc: func(pod *core.Pod, cm *core.ConfigMap) {
				pod.Namespace = "anotherns"
				pod.Labels[alipaysigmak8sapi.LabelPodPresetName] = "non-exists-cm"
			},
		},
		{
			name:  "admit failed, cm is not podpreset type",
			admit: false,
			cm:    newConfigMap(),
		},
	} {
		mockClient := &fake.Clientset{}
		if test.cm != nil {
			mockClient = fake.NewSimpleClientset(test.cm)
		}

		handler, f, err := newHandlerForTest(mockClient)
		if err != nil {
			t.Errorf("unexpected error initializing handler: %v", err)
		}
		f.Start(stopCh)

		pod := newPod()
		if test.initfunc != nil {
			test.initfunc(pod, test.cm)
		}

		a := admission.NewAttributesRecord(pod, nil, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Create, false, nil)
		err = handler.Admit(a)

		if test.admit {
			assert.True(t, err == nil, "[%s] admit true: %v", test.name, err)
		} else {
			assert.True(t, err != nil, "[%s] expect error: %v", test.name, err)
		}
		if test.validate != nil {
			test.validate(pod)
		}
	}
}

func TestAdmitOtherResources(t *testing.T) {
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
			name:        "non-pod object",
			kind:        "Pod",
			resource:    "pods",
			object:      &core.Service{},
			expectError: true,
		},
	}

	for _, tc := range tests {
		handler := NewAlipayPodPreset()

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
	}
}

func TestValidateOtherResources(t *testing.T) {
	cm := newPodPresetConfigMap(false)

	tests := []struct {
		name        string
		kind        string
		resource    string
		subresource string
		object      runtime.Object
		expectError bool
	}{
		{
			name:     "non-cm resource",
			kind:     "Foo",
			resource: "foos",
			object:   cm,
		},
		{
			name:     "non-cm resource",
			kind:     "Pod",
			resource: "pods",
			object:   newPod(),
		},
		{
			name:     "non-cm object",
			kind:     "ConfigMap",
			resource: "configmaps",
			object:   newPod(),
		},
	}

	for _, tc := range tests {
		handler := NewAlipayPodPreset()

		err := handler.Validate(admission.NewAttributesRecord(tc.object, nil, core.Kind(tc.kind).WithVersion("version"), cm.Namespace, cm.Name, core.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Create, false, nil))

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
			Name:      "test-podpreset-pod",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				alipaysigmak8sapi.LabelPodPresetName: "test-podpreset-cm",
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{Image: "pause:2.0"},
			},
		},
	}
}

func newConfigMap() *core.ConfigMap {
	preset := newPodPresetConfigMap(false)
	preset.Labels = nil
	return preset
}

func newPodPresetConfigMap(isDefault bool) *core.ConfigMap {
	return &core.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-podpreset-cm",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				alipaysigmak8sapi.LabelDefaultPodPreset: strconv.FormatBool(isDefault),
			},
		},
		Data: map[string]string{
			"metadata": `
labels:
  meta.k8s.alipay.com/zone: GZ00C
`,
		},
	}
}

func newHandlerForTest(c internalclientset.Interface) (*AlipayPodPreset, internalversion.SharedInformerFactory, error) {
	f := internalversion.NewSharedInformerFactory(c, 5*time.Minute)
	handler := NewAlipayPodPreset()
	pluginInitializer := kubeadmission.NewPluginInitializer(c, f, nil, nil, nil)
	pluginInitializer.Initialize(handler)
	err := admission.ValidateInitialization(handler)
	return handler, f, err
}
