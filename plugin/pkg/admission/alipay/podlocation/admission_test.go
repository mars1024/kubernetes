package podlocation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
		admission.Update:  true,
		admission.Connect: false,
		admission.Delete:  false,
	} {
		handler := NewAlipayPodLocation()
		if e, a := shouldHandle, handler.Handles(op); e != a {
			t.Errorf("%v: shouldHandle=%t, handles=%t", op, e, a)
		}
	}
}

func TestAdmit(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	for _, test := range []struct {
		name       string
		admit      bool
		node       *core.Node
		oldPodFunc func(*core.Pod, *core.Node) *core.Pod
		validate   func(*core.Pod)
	}{
		{
			name:  "admit create success, not scheduled yet",
			admit: true,
			node:  newNode(),
			validate: func(pod *core.Pod) {
				assert.Len(t, pod.Spec.Containers[0].Env, 0)
			},
			oldPodFunc: func(pod *core.Pod, node *core.Node) *core.Pod {
				return nil
			},
		},
		{
			name:  "admit create success, scheduled",
			admit: true,
			node:  newNode(),
			oldPodFunc: func(pod *core.Pod, node *core.Node) *core.Pod {
				pod.Spec.NodeName = node.Name
				return nil
			},
			validate: func(pod *core.Pod) {
				assert.Len(t, pod.Spec.Containers[0].Env, len(topologyKeyMap))
				for _, env := range pod.Spec.Containers[0].Env {
					assert.Equal(t, env.Value, env.Value)
				}
			},
		},
		{
			name:  "admit update success, not scheduled yet",
			admit: true,
			node:  newNode(),
			validate: func(pod *core.Pod) {
				assert.Len(t, pod.Spec.Containers[0].Env, 0)
			},
		},
		{
			name:  "admit update success, scheduled",
			admit: true,
			node:  newNode(),
			oldPodFunc: func(pod *core.Pod, node *core.Node) *core.Pod {
				old := pod.DeepCopy()
				pod.Spec.NodeName = node.Name
				return old
			},
			validate: func(pod *core.Pod) {
				assert.Len(t, pod.Spec.Containers[0].Env, len(topologyKeyMap))
				for _, env := range pod.Spec.Containers[0].Env {
					assert.Equal(t, env.Value, env.Value)
				}
			},
		},
	} {
		mockClient := &fake.Clientset{}
		if test.node != nil {
			mockClient = fake.NewSimpleClientset(test.node)
		}

		handler, f, err := newHandlerForTest(mockClient)
		if err != nil {
			t.Errorf("unexpected error initializing handler: %v", err)
		}
		f.Start(stopCh)

		pod := newPod()
		old := pod.DeepCopy()
		if test.oldPodFunc != nil {
			old = test.oldPodFunc(pod, test.node)
		}

		op := admission.Create
		if old != nil {
			op = admission.Update
		}

		a := admission.NewAttributesRecord(pod, old, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", op, nil)
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

func TestOtherResources(t *testing.T) {
	namespace := "testnamespace"
	name := "testname"
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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
		handler := NewAlipayPodLocation()

		err := handler.Admit(admission.NewAttributesRecord(tc.object, nil, core.Kind(tc.kind).WithVersion("version"), namespace, name, core.Resource(tc.resource).WithVersion("version"), tc.subresource, admission.Create, nil))

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
			Name: "test-podlocation-pod",
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{Image: "pause:2.0"},
			},
		},
	}
}

func newNode() *core.Node {
	n := &core.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-node-1",
			Labels: map[string]string{},
		},
	}
	for _, item := range topologyKeyMap {
		n.Labels[item.label] = item.label
	}
	return n
}

func newHandlerForTest(c internalclientset.Interface) (*AlipayPodLocation, internalversion.SharedInformerFactory, error) {
	f := internalversion.NewSharedInformerFactory(c, 5*time.Minute)
	handler := NewAlipayPodLocation()
	pluginInitializer := kubeadmission.NewPluginInitializer(c, f, nil, nil, nil)
	pluginInitializer.Initialize(handler)
	err := admission.ValidateInitialization(handler)
	return handler, f, err
}
