package webhookcainjector

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/admissionregistration"
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
		handler := NewWebhookCAInjector()
		if e, a := shouldHandle, handler.Handles(op); e != a {
			t.Errorf("%v: shouldHandle=%t, handles=%t", op, e, a)
		}
	}
}

var (
	ca = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN5RENDQWJDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRFNE1EZ3dPVEEzTXpneU1Wb1hEVEk0TURnd05qQTNNemd5TVZvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTndyCjVSbE5IcUJYdGVhNis5T2hhbmVpVldTUVdUL0xvMHJSanF3ZHY3Mkl6VUcyWEFMT21uaWtxU1R5Wit0bDJxN20Ka1dhTXA2VlR4elZ6aFFsYTVYOS9lQm9URDQvelVRS2pkOXgvNXdzUDZraUpqaW04aEZ4bkxVdHBLZ3dvQUJFbAo5K2RCVWVSb0hsNVpBTUMwUDhhY2JnbHlsb0FCYWdkQ0FIVmU5cmZvbnI5THVNMCtFKy9Qcmo2Q29wTUR4cE56CnNBMlFXRFZqaHRKYW1PYmhBNE5IWktKQVRQejltUTA5ckFXdTdocFY3Yy9LMXR0SGZ2RisxQ1QvdlQrWTVKS1YKSVZOQXl5RW40UnFRMXRyUFBkOGpuMjBJSWdOQjdUa0wrcE5QYzljdzZ1UjBoUTRvK0t1c0ZtcmxKOFl1dzRjNworcUd4eFFDRDQyOURFR0FBeHJrQ0F3RUFBYU1qTUNFd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFHUEQrR0t0cDN2MDBReHd2SldEZ0lpd0xsYSsKTUc5Y2tadmRLaVFrSGthVXhhSkZGWlY0NHZKcUhnNXhtYSt6b0t2bUNxWDAzM0lRNlFyeUtoOThONEZleVBEZAo3QXA1TUtsWEpZdk5oOXJvRW5mZnVWQkt0V2w1UkJMb01iWjFyMDNPOWxBejdpSGt4SHBGUTUxZW5xVHhGcGRXCjRXTjV5YUZiN09tRFlBZERURlI4TVFVajVjUDdTMHpNR2M2ZTc0VWZNNTFXaHlPTktsTDBvNHdnODdvZno1Nk8KK2FBVHQzK1M2bWIzQTcwYmhZY1dDajFnd2d1K2lOZGFEQWVNZ1pycUFHMWQ5U0ZlMnlKS1cvd2FWY09FVHlObApRWjdFbXJmUHM3UnFPU0FaVE02aklEVzRTOWYzY2pxK1BoanVTYm1LSHR3QmFFZWJtakhiV3p4NlFGQT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="

	testKubeConfig = fmt.Sprintf(`
apiVersion: v1
clusters:
  - cluster:
      certificate-authority-data: %s
      server: https://apiserver.alipay-dev.svc.alipay.net:6443
    name: ""
contexts: []
current-context: ""
kind: Config
preferences: {}
users: []
`, ca)
)

const (
	nameEmptyCA = "empty-ca.test.alipay.com"
	nameNilCA   = "nil-ca.test.alipay.com"
	nameHasCA   = "has-ca.test.alipay.com"
)

var (
	shouldMutateNames = map[string]struct{}{
		nameEmptyCA: {},
		nameNilCA:   {},
	}
)

func TestAdmitValidating(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	for _, test := range []struct {
		name     string
		admit    bool
		cm       *core.ConfigMap
		obj      *admissionregistration.ValidatingWebhookConfiguration
		validate func(*admissionregistration.ValidatingWebhookConfiguration)
	}{
		{
			name:  "admit success",
			admit: true,
			cm: &core.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kube-public", Name: "cluster-info"},
				Data:       map[string]string{"kubeconfig": testKubeConfig},
			},
			obj: newValidating(),
			validate: func(o *admissionregistration.ValidatingWebhookConfiguration) {
				for i := range o.Webhooks {
					if _, ok := shouldMutateNames[o.Webhooks[i].Name]; ok {
						assert.Equal(t, ca, base64.StdEncoding.EncodeToString(o.Webhooks[i].ClientConfig.CABundle))
					} else {
						assert.NotEqual(t, ca, base64.StdEncoding.EncodeToString(o.Webhooks[i].ClientConfig.CABundle))
					}
				}
			},
		},
		{
			name:  "admit success, cluster-info not found",
			admit: true,
			obj:   newValidating(),
			validate: func(o *admissionregistration.ValidatingWebhookConfiguration) {
				for i := range o.Webhooks {
					if _, ok := shouldMutateNames[o.Webhooks[i].Name]; ok {
						assert.Equal(t, "", base64.StdEncoding.EncodeToString(o.Webhooks[i].ClientConfig.CABundle))
					} else {
						assert.NotEqual(t, "", base64.StdEncoding.EncodeToString(o.Webhooks[i].ClientConfig.CABundle))
					}
				}
			},
		},
		{
			name:  "admit failed, invalid configmap",
			admit: false,
			cm: &core.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kube-public", Name: "cluster-info"},
				Data:       map[string]string{"kubeconfig": "abcdefg"},
			},
			obj: newValidating(),
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

		a := admission.NewAttributesRecord(test.obj, nil,
			admissionregistration.Kind("ValidatingWebhookConfiguration").WithVersion("version"),
			test.obj.Namespace, test.obj.Name,
			admissionregistration.Resource("validatingwebhookconfigurations").WithVersion("version"), "",
			admission.Create, false, nil)
		err = handler.Admit(a)

		if test.admit {
			assert.True(t, err == nil, "[%s] admit true: %v", test.name, err)
		} else {
			assert.True(t, err != nil, "[%s] expect error: %v", test.name, err)
		}
		if test.validate != nil {
			test.validate(test.obj)
		}
	}
}

func TestAdmitMutating(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	for _, test := range []struct {
		name     string
		admit    bool
		cm       *core.ConfigMap
		obj      *admissionregistration.MutatingWebhookConfiguration
		validate func(*admissionregistration.MutatingWebhookConfiguration)
	}{
		{
			name:  "admit success",
			admit: true,
			cm: &core.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kube-public", Name: "cluster-info"},
				Data:       map[string]string{"kubeconfig": testKubeConfig},
			},
			obj: newMutating(),
			validate: func(o *admissionregistration.MutatingWebhookConfiguration) {
				for i := range o.Webhooks {
					if _, ok := shouldMutateNames[o.Webhooks[i].Name]; ok {
						assert.Equal(t, ca, base64.StdEncoding.EncodeToString(o.Webhooks[i].ClientConfig.CABundle))
					} else {
						assert.NotEqual(t, ca, base64.StdEncoding.EncodeToString(o.Webhooks[i].ClientConfig.CABundle))
					}
				}
			},
		},
		{
			name:  "admit success, cluster-info not found",
			admit: true,
			obj:   newMutating(),
			validate: func(o *admissionregistration.MutatingWebhookConfiguration) {
				for i := range o.Webhooks {
					if _, ok := shouldMutateNames[o.Webhooks[i].Name]; ok {
						assert.Equal(t, "", base64.StdEncoding.EncodeToString(o.Webhooks[i].ClientConfig.CABundle))
					} else {
						assert.NotEqual(t, "", base64.StdEncoding.EncodeToString(o.Webhooks[i].ClientConfig.CABundle))
					}
				}
			},
		},
		{
			name:  "admit failed, invalid configmap",
			admit: false,
			cm: &core.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "kube-public", Name: "cluster-info"},
				Data:       map[string]string{"kubeconfig": "abcdefg"},
			},
			obj: newMutating(),
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

		a := admission.NewAttributesRecord(test.obj, nil,
			admissionregistration.Kind("MutatingWebhookConfiguration").WithVersion("version"),
			test.obj.Namespace, test.obj.Name,
			admissionregistration.Resource("mutatingwebhookconfigurations").WithVersion("version"), "",
			admission.Create, false, nil)
		err = handler.Admit(a)

		if test.admit {
			assert.True(t, err == nil, "[%s] admit true: %v", test.name, err)
		} else {
			assert.True(t, err != nil, "[%s] expect error: %v", test.name, err)
		}
		if test.validate != nil {
			test.validate(test.obj)
		}
	}
}

// TestOtherResources ensures that this admission controller is a no-op for other resources,
// subresources, and non-pods.
func TestOtherResources(t *testing.T) {
	namespace := "testnamespace"
	name := "testname"

	tests := []struct {
		name        string
		kind        string
		resource    string
		subresource string
		object      runtime.Object
		expectError bool
	}{
		{
			name:     "non-MutatingWebhookConfiguration resource",
			kind:     "Foo",
			resource: "foos",
			object:   newMutating(),
		},
		{
			name:     "non-ValidatingWebhookConfiguration resource",
			kind:     "Foo",
			resource: "foos",
			object:   newValidating(),
		},
		{
			name:        "non-MutatingWebhookConfiguration object",
			kind:        "MutatingWebhookConfiguration",
			resource:    "mutatingwebhookconfigurations",
			object:      &core.Service{},
			expectError: true,
		},
		{
			name:        "non-ValidatingWebhookConfiguration object",
			kind:        "ValidatingWebhookConfiguration",
			resource:    "validatingwebhookconfigurations",
			object:      &core.Service{},
			expectError: true,
		},
	}

	for _, tc := range tests {
		handler := NewWebhookCAInjector()

		err := handler.Admit(admission.NewAttributesRecord(
			tc.object, nil,
			admissionregistration.Kind(tc.kind).WithVersion("version"),
			namespace, name,
			admissionregistration.Resource(tc.resource).WithVersion("version"), tc.subresource,
			admission.Create, false, nil))

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

func newMutating() *admissionregistration.MutatingWebhookConfiguration {
	return &admissionregistration.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ca-injector-mutating",
		},
		Webhooks: []admissionregistration.Webhook{
			{
				Name:         nameNilCA,
				ClientConfig: admissionregistration.WebhookClientConfig{},
			},
			{
				Name:         nameEmptyCA,
				ClientConfig: admissionregistration.WebhookClientConfig{},
			},
			{
				Name: nameHasCA,
				ClientConfig: admissionregistration.WebhookClientConfig{
					CABundle: []byte("hello"),
				},
			},
		},
	}
}

func newValidating() *admissionregistration.ValidatingWebhookConfiguration {
	return &admissionregistration.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ca-injector-mutating",
		},
		Webhooks: []admissionregistration.Webhook{
			{
				Name:         nameNilCA,
				ClientConfig: admissionregistration.WebhookClientConfig{},
			},
			{
				Name: nameEmptyCA,
				ClientConfig: admissionregistration.WebhookClientConfig{
					CABundle: []byte{},
				},
			},
			{
				Name: nameHasCA,
				ClientConfig: admissionregistration.WebhookClientConfig{
					CABundle: []byte("hello"),
				},
			},
		},
	}
}

func newHandlerForTest(c internalclientset.Interface) (*WebhookCAInjector, internalversion.SharedInformerFactory, error) {
	f := internalversion.NewSharedInformerFactory(c, 5*time.Minute)
	handler := NewWebhookCAInjector()
	pluginInitializer := kubeadmission.NewPluginInitializer(c, f, nil, nil, nil)
	pluginInitializer.Initialize(handler)
	err := admission.ValidateInitialization(handler)
	return handler, f, err
}
