package generic

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/namespace"
	"k8s.io/apiserver/pkg/util/webhook"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestObjectLabelSelectorParsing(t *testing.T) {
	testCases := []struct {
		name             string
		webhook          runtime.Object
		expectedSelector labels.Selector
	}{
		{
			name: "normal mutating webhook w/o annotation should work",
			webhook: &admissionregistrationv1beta1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			expectedSelector: labels.Everything(),
		},
		{
			name: "normal validating webhook w/o annotation should work",
			webhook: &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			expectedSelector: labels.Everything(),
		},
		{
			name: "normal webhook w/ foo/bar selector should work 1",
			webhook: &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ObjectLabelSelectorAnnotationQualifiedPrefix + "foo": "bar",
					},
				},
			},
			expectedSelector: labels.SelectorFromValidatedSet(
				map[string]string{
					"foo": "bar",
				},
			),
		},
		{
			name: "normal webhook w/ foo/bar selector should work 2",
			webhook: &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ObjectLabelSelectorAnnotationQualifiedPrefix + "foo": "bar",
						"tik": "tok",
					},
				},
			},
			expectedSelector: labels.SelectorFromValidatedSet(
				map[string]string{
					"foo": "bar",
				},
			),
		},
		{
			name: "normal webhook w/ merely prefix should work",
			webhook: &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ObjectLabelSelectorAnnotationQualifiedPrefix: "bar",
					},
				},
			},
			expectedSelector: labels.Everything(),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, GetObjectLabelSelector(testCase.webhook), testCase.expectedSelector)
		})
	}
}

type mockSource struct {
	getter func() []admissionregistrationv1beta1.Webhook
}

func (s *mockSource) Webhooks() []admissionregistrationv1beta1.Webhook {
	return s.getter()
}

func (s *mockSource) HasSynced() bool {
	return true
}

func TestNewWebhookProxyConstructor(t *testing.T) {

	fakeClient := fake.NewSimpleClientset()
	fakeInformer := informers.NewSharedInformerFactory(fakeClient, 0)
	stopCh := make(chan struct{})
	defer close(stopCh)

	mockConstructor := func(handler *admission.Handler, configFile io.Reader, sourceFactory sourceFactory, dispatcherFactory dispatcherFactory) (*Webhook, error) {
		return &Webhook{
			Handler:          admission.NewHandler(),
			namespaceMatcher: &namespace.Matcher{},
			sourceFactory:    sourceFactory,
		}, nil
	}

	h1 := admissionregistrationv1beta1.Webhook{Name: "h1"}
	h2 := admissionregistrationv1beta1.Webhook{Name: "h2"}
	h3 := admissionregistrationv1beta1.Webhook{Name: "h3"}
	h4 := admissionregistrationv1beta1.Webhook{Name: "h4"}

	w, err := NewWebhookWithObjectSelectorProxy(mockConstructor, nil, nil,
		func(f informers.SharedInformerFactory) Source {
			return &mockSource{
				getter: func() []admissionregistrationv1beta1.Webhook {
					return []admissionregistrationv1beta1.Webhook{h1, h2}
				},
			}
		},
		func(cm *webhook.ClientManager) Dispatcher {
			return nil
		})
	require.NoError(t, err)

	w.SetExternalKubeClientSet(fakeClient)
	w.SetExternalKubeInformerFactory(fakeInformer)

	mh := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test1",
			Annotations: map[string]string{
				ObjectLabelSelectorAnnotationQualifiedPrefix + "m-foo": "m-bar",
			},
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{h1, h2},
	}
	require.NoError(t, err)
	fakeInformer.Start(stopCh)

	// create test1
	_, err = fakeClient.Admissionregistration().MutatingWebhookConfigurations().Create(mh)
	// hold a sec for informer loading caches..
	time.Sleep(time.Second * 1)
	actual := w.selectorGetter(&h1)
	assert.Equal(t, labels.SelectorFromValidatedSet(map[string]string{
		"m-foo": "m-bar",
	}), actual)

	vh := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test2",
			Annotations: map[string]string{
				ObjectLabelSelectorAnnotationQualifiedPrefix + "v-foo": "v-bar",
			},
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{h3, h4},
	}

	// create test2
	_, err = fakeClient.Admissionregistration().MutatingWebhookConfigurations().Create(vh)
	// hold a sec for informer loading caches..
	time.Sleep(time.Second * 1)
	actual = w.selectorGetter(&h3)
	assert.Equal(t, labels.SelectorFromValidatedSet(map[string]string{
		"v-foo": "v-bar",
	}), actual)

	// delete test1
	err = fakeClient.Admissionregistration().MutatingWebhookConfigurations().Delete(mh.Name, nil)
	require.NoError(t, err)
	// hold a sec for informer loading caches..
	time.Sleep(time.Second * 1)
	actual = w.selectorGetter(&h1)
	assert.Equal(t, labels.Everything(), actual)
}
