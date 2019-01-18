package imagepullsecret

import (
	"reflect"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	"k8s.io/kubernetes/staging/src/k8s.io/apiserver/pkg/authentication/user"
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

func NewTestAdmission(t *testing.T, client internalclientset.Interface, f informers.SharedInformerFactory) admission.MutationInterface {
	p := NewPlugin()

	if p.ValidateInitialization() == nil {
		t.Fatalf("plugin ValidateInitialization should return error")
	}

	p.SetInternalKubeClientSet(client)
	p.SetInternalKubeInformerFactory(f)

	if p.ValidateInitialization() != nil {
		t.Fatalf("plugin ValidateInitialization should not return error")
	}
	return p
}

func TestAdmit(t *testing.T) {
	client := fake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(client, 10*time.Second)
	plugin := NewTestAdmission(t, client, informerFactory)

	secret, err := generateSecretDockerRegistryCfg(DefaultImagePullSecret, "user", "pass", "", DefaultImageRegistryServer)
	if err != nil {
		t.Errorf("generateSecretDockerRegistryCfg failed: %v", err)
	}

	secret.Namespace = "kube-system"
	_ = informerFactory.Core().InternalVersion().Secrets().Informer().GetStore().Add(secret)

	testCases := []struct {
		name                   string
		pod                    *api.Pod
		action                 admission.Operation
		expectImagePullSecrets []api.LocalObjectReference
	}{
		{
			name: "Create: Pod has no ImagePullSecrets",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						api.Container{
							Name:  "container1",
							Image: "image1",
						},
					},
				},
			},
			action: admission.Create,
			expectImagePullSecrets: []api.LocalObjectReference{
				{Name: DefaultImagePullSecret},
			},
		},
		{
			name: "Create: Pod has DefaultImagePullSecret",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo3",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						api.Container{
							Name:  "container1",
							Image: "image1",
						},
					},
					ImagePullSecrets: []api.LocalObjectReference{
						{Name: DefaultImagePullSecret},
					},
				},
			},
			action: admission.Create,
			expectImagePullSecrets: []api.LocalObjectReference{
				{Name: DefaultImagePullSecret},
			},
		},
		{
			name: "Create: Pod has other ImagePullSecret",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo5",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						api.Container{
							Name:  "container1",
							Image: "image1",
						},
					},
					ImagePullSecrets: []api.LocalObjectReference{
						{Name: "other-secret"},
					},
				},
			},
			action: admission.Create,
			expectImagePullSecrets: []api.LocalObjectReference{
				{Name: "other-secret"},
				{Name: DefaultImagePullSecret},
			},
		},
	}
	for _, ts := range testCases {
		pod := ts.pod

		// Add secrect to informerFactory manually.
		secret.Namespace = pod.Namespace
		_ = informerFactory.Core().InternalVersion().Secrets().Informer().GetStore().Add(secret)

		a := admission.NewAttributesRecord(
			pod,
			nil,
			api.Kind("Pod").WithVersion("version"),
			pod.Namespace,
			pod.Name,
			api.Resource("pods").WithVersion("version"),
			"",
			ts.action,
			false,
			&user.DefaultInfo{},
		)
		plugin.Admit(a)
		actualImagePullSecrets := pod.Spec.ImagePullSecrets
		if !reflect.DeepEqual(actualImagePullSecrets, ts.expectImagePullSecrets) {
			t.Errorf("ImagePullSecret injection failed")
		}
	}
}

func TestGenerateSecretDockerRegistryCfg(t *testing.T) {
	client := fake.NewSimpleClientset()

	secret, err := generateSecretDockerRegistryCfg("fake-secret", "aone", "abcd", "sigma.ali", "reg.docker.alibaba-inc.com")
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	gotSecret, err := client.Core().Secrets("default").Create(secret)
	if err != nil {
		t.Fatalf("create secret error: %v", err)
	}

	expectedCfg := `{"auths":{"reg.docker.alibaba-inc.com":{"username":"aone","password":"abcd","email":"sigma.ali","auth":"YW9uZTphYmNk"}}}`
	if string(gotSecret.Data[api.DockerConfigJsonKey]) != expectedCfg {
		t.Fatalf("secret data expected: %v\n got: %v", expectedCfg, string(gotSecret.Data[api.DockerConfigJsonKey]))
	}
}
