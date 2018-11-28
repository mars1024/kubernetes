package inclusterkube

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
		admission.Update:  false,
		admission.Connect: false,
		admission.Delete:  false,
	} {
		handler := NewAlipayInClusterKubernetes()
		if e, a := shouldHandle, handler.Handles(op); e != a {
			t.Errorf("%v: shouldHandle=%t, handles=%t", op, e, a)
		}
	}
}

const (
	testKubeConfig = `
apiVersion: v1
clusters:
  - cluster:
      certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN5RENDQWJDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRFNE1EZ3dPVEEzTXpneU1Wb1hEVEk0TURnd05qQTNNemd5TVZvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTndyCjVSbE5IcUJYdGVhNis5T2hhbmVpVldTUVdUL0xvMHJSanF3ZHY3Mkl6VUcyWEFMT21uaWtxU1R5Wit0bDJxN20Ka1dhTXA2VlR4elZ6aFFsYTVYOS9lQm9URDQvelVRS2pkOXgvNXdzUDZraUpqaW04aEZ4bkxVdHBLZ3dvQUJFbAo5K2RCVWVSb0hsNVpBTUMwUDhhY2JnbHlsb0FCYWdkQ0FIVmU5cmZvbnI5THVNMCtFKy9Qcmo2Q29wTUR4cE56CnNBMlFXRFZqaHRKYW1PYmhBNE5IWktKQVRQejltUTA5ckFXdTdocFY3Yy9LMXR0SGZ2RisxQ1QvdlQrWTVKS1YKSVZOQXl5RW40UnFRMXRyUFBkOGpuMjBJSWdOQjdUa0wrcE5QYzljdzZ1UjBoUTRvK0t1c0ZtcmxKOFl1dzRjNworcUd4eFFDRDQyOURFR0FBeHJrQ0F3RUFBYU1qTUNFd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFHUEQrR0t0cDN2MDBReHd2SldEZ0lpd0xsYSsKTUc5Y2tadmRLaVFrSGthVXhhSkZGWlY0NHZKcUhnNXhtYSt6b0t2bUNxWDAzM0lRNlFyeUtoOThONEZleVBEZAo3QXA1TUtsWEpZdk5oOXJvRW5mZnVWQkt0V2w1UkJMb01iWjFyMDNPOWxBejdpSGt4SHBGUTUxZW5xVHhGcGRXCjRXTjV5YUZiN09tRFlBZERURlI4TVFVajVjUDdTMHpNR2M2ZTc0VWZNNTFXaHlPTktsTDBvNHdnODdvZno1Nk8KK2FBVHQzK1M2bWIzQTcwYmhZY1dDajFnd2d1K2lOZGFEQWVNZ1pycUFHMWQ5U0ZlMnlKS1cvd2FWY09FVHlObApRWjdFbXJmUHM3UnFPU0FaVE02aklEVzRTOWYzY2pxK1BoanVTYm1LSHR3QmFFZWJtakhiV3p4NlFGQT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
      server: https://apiserver.alipay-dev.svc.alipay.net:6443
    name: ""
contexts: []
current-context: ""
kind: Config
preferences: {}
users: []
`
)

func TestAdmit(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	for _, test := range []struct {
		name     string
		admit    bool
		cm       *core.ConfigMap
		initpod  func(*core.Pod)
		validate func(*core.Pod)
	}{
		{
			name:  "admit success",
			admit: true,
			cm: &core.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kube-public", Name: "cluster-info"},
				Data:       map[string]string{"kubeconfig": testKubeConfig},
			},
			validate: func(pod *core.Pod) {
				assert.Equal(t, pod.Spec.Containers[0].Env[0].Name, kubernetesInClusterServiceHost)
				assert.Equal(t, pod.Spec.Containers[0].Env[0].Value, "apiserver.alipay-dev.svc.alipay.net")
				assert.Equal(t, pod.Spec.Containers[0].Env[1].Name, kubernetesInClusterServicePort)
				assert.Equal(t, pod.Spec.Containers[0].Env[1].Value, "6443")
			},
		},
		{
			name:  "admit success, cluster-info not found",
			admit: true,
			validate: func(pod *core.Pod) {
				assert.Len(t, pod.Spec.Containers[0].Env, 0)
			},
		},
		{
			name:  "admit failed, invalid configmap",
			admit: false,
			cm: &core.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kube-public", Name: "cluster-info"},
				Data:       map[string]string{"kubeconfig": "abcdefg"},
			},
		},
		{
			name:  "admit success, pod already define in-cluster env",
			admit: true,
			cm: &core.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kube-public", Name: "cluster-info"},
				Data:       map[string]string{"kubeconfig": testKubeConfig},
			},
			initpod: func(pod *core.Pod) {
				pod.Spec.Containers[0].Env = []core.EnvVar{
					{Name: kubernetesInClusterServiceHost, Value: "apiserver.alipay-dev-2.svc.alipay.net"},
				}
			},
			validate: func(pod *core.Pod) {
				assert.Len(t, pod.Spec.Containers[0].Env, 1)
				assert.Equal(t, pod.Spec.Containers[0].Env[0].Name, kubernetesInClusterServiceHost)
				assert.Equal(t, pod.Spec.Containers[0].Env[0].Value, "apiserver.alipay-dev-2.svc.alipay.net")
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

		pod := newPod()
		if test.initpod != nil {
			test.initpod(pod)
		}

		a := admission.NewAttributesRecord(pod, nil, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Create, nil)
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

// TestOtherResources ensures that this admission controller is a no-op for other resources,
// subresources, and non-pods.
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
			name:        "non-pod object",
			kind:        "Pod",
			resource:    "pods",
			object:      &core.Service{},
			expectError: true,
		},
	}

	for _, tc := range tests {
		handler := NewAlipayInClusterKubernetes()

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
			Name: "test-inclusterkube-pod",
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{Image: "pause:2.0"},
			},
		},
	}
}

func newHandlerForTest(c internalclientset.Interface) (*AlipayInClusterKubernetes, internalversion.SharedInformerFactory, error) {
	f := internalversion.NewSharedInformerFactory(c, 5*time.Minute)
	handler := NewAlipayInClusterKubernetes()
	pluginInitializer := kubeadmission.NewPluginInitializer(c, f, nil, nil, nil)
	pluginInitializer.Initialize(handler)
	err := admission.ValidateInitialization(handler)
	return handler, f, err
}
