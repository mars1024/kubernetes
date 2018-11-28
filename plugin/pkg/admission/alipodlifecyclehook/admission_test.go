package alipodlifecyclehook

import (
	"reflect"
	"testing"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	"k8s.io/kubernetes/pkg/controller"
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

func NewTestAdmission(t *testing.T, f informers.SharedInformerFactory, objects ...runtime.Object) (internalclientset.Interface, admission.MutationInterface) {
	// Build a test client that the admission plugin can use to look up the service account missing from its cache
	client := fake.NewSimpleClientset(objects...)

	p := NewPlugin()
	if p.ValidateInitialization() == nil {
		t.Fatalf("plugin ValidateInitialization should return error")
	}
	p.SetInternalKubeClientSet(client)
	p.SetInternalKubeInformerFactory(f)
	if p.ValidateInitialization() != nil {
		t.Fatalf("plugin ValidateInitialization should return error")
	}
	return client, p
}

func TestAdmit(t *testing.T) {
	informerFactory := informers.NewSharedInformerFactory(nil, controller.NoResyncPeriodFunc())
	_ = informerFactory.Core().InternalVersion().ConfigMaps().Informer().GetStore().Add(&api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigName,
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"probe-timeout-seconds":          "60",
			"volume-sigmalogs-name":          "vol-sigmalogs-test",
			"probe-period-seconds-specified": `{"jiuzhu-test": 1800}`,
		},
	})
	_ = informerFactory.Core().InternalVersion().Secrets().Informer().GetStore().Add(&api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "registry-user-aone",
			Namespace: "kube-system",
			Labels: map[string]string{
				"username": "aone",
				"usage":    "ali-registry-user-account",
			},
		},
		Data: map[string][]byte{
			"password": []byte("abcd"),
		},
	})
	cm, err := informerFactory.Core().InternalVersion().ConfigMaps().Lister().ConfigMaps("kube-system").Get(ConfigName)
	if err != nil {
		t.Fatalf("store configmap and get failed: %v", err)
	} else if cm == nil {
		t.Fatalf("store configmap and get nil")
	}
	clientset, plugin := NewTestAdmission(t, informerFactory)

	pod0 := api.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod01",
			Namespace: "ns01",
			Labels: map[string]string{
				sigmak8sapi.LabelAppName: "jiuzhu-test",
			},
			Annotations: make(map[string]string),
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				api.Container{
					Name:  "ctn01",
					Image: "reg.docker.alibaba-inc.com/ali/os:7u2",
				},
			},
		},
	}

	expectPod := pod0.DeepCopy()
	_ = plugin.Admit(admission.NewAttributesRecord(
		&pod0,
		nil,
		api.Kind("Pod").WithVersion("version"),
		pod0.Namespace,
		"",
		api.Resource("pods").WithVersion("version"),
		"",
		admission.Update,
		&user.DefaultInfo{},
	))

	if !reflect.DeepEqual(pod0.Spec, expectPod.Spec) {
		t.Fatalf("pod not equal, origPod: %+v got: %+v", expectPod, pod0)
	}
	_ = plugin.Admit(admission.NewAttributesRecord(
		&pod0,
		nil,
		api.Kind("Pod").WithVersion("version"),
		pod0.Namespace,
		"",
		api.Resource("pods").WithVersion("version"),
		"",
		admission.Create,
		&user.DefaultInfo{},
	))

	if !reflect.DeepEqual(pod0.Spec, expectPod.Spec) {
		t.Fatalf("pod not equal, origPod: %+v got: %+v", expectPod, pod0)
	}

	pod1 := pod0.DeepCopy()
	pod1.Labels[sigmak8sapi.LabelPodContainerModel] = "dockervm"
	pod1.Annotations["pod.beta1.sigma.ali/disable-lifecycle-hook"] = "true"
	pod1.Spec.Containers = []api.Container{
		api.Container{
			Name:  "ctn01",
			Image: "reg.docker.alibaba-inc.com/ali/os:7u2",
			Lifecycle: &api.Lifecycle{
				PostStart: &api.Handler{
					Exec: &api.ExecAction{
						Command: []string{"/bin/sh", "-c", "test.sh"},
					},
				},
			},
		},
	}
	expectPod = &api.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod01",
			Namespace: "ns01",
			Annotations: map[string]string{
				"pod.beta1.sigma.ali/disable-lifecycle-hook": "true",
			},
			Labels: map[string]string{
				sigmak8sapi.LabelPodContainerModel: "dockervm",
			},
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				api.Container{
					Name:  "ctn01",
					Image: "reg.docker.alibaba-inc.com/ali/os:7u2",
				},
			},
		},
	}
	_ = plugin.Admit(admission.NewAttributesRecord(
		pod1,
		nil,
		api.Kind("Pod").WithVersion("version"),
		pod1.Namespace,
		"",
		api.Resource("pods").WithVersion("version"),
		"",
		admission.Create,
		&user.DefaultInfo{},
	))

	if !reflect.DeepEqual(pod1.Spec, expectPod.Spec) {
		t.Fatalf("pod not equal, origPod: %+v got: %+v", expectPod.Spec, pod1.Spec)
	}

	pod2 := pod1.DeepCopy()
	pod2.Annotations["pod.beta1.sigma.ali/disable-lifecycle-hook"] = "false"
	pod2.Spec.Containers = []api.Container{
		api.Container{
			Name:  "ctn01",
			Image: "reg.docker.alibaba-inc.com/aone/sigma-boss:20180731145943_daily",
		},
	}
	pod2.Labels[sigmak8sapi.LabelPodSn] = "pod01-sn"
	expectPod = &api.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod01",
			Namespace: "ns01",
			Annotations: map[string]string{
				"pod.beta1.sigma.ali/disable-lifecycle-hook": "false",
			},
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				api.Container{
					Name:  "ctn01",
					Image: "reg.docker.alibaba-inc.com/aone/sigma-boss:20180731145943_daily",
					Lifecycle: &api.Lifecycle{
						PostStart: &api.Handler{
							Exec: &api.ExecAction{
								Command: []string{
									"/bin/sh",
									"-c",
									"for i in $(seq 1 60); do [ -x /home/admin/.start ] && break ; sleep 5 ; done; sudo -u admin /home/admin/.start>/var/log/sigma/start.log 2>&1 && sudo -u admin /home/admin/health.sh>>/var/log/sigma/start.log 2>&1",
								},
							},
						},
						PreStop: &api.Handler{
							Exec: &api.ExecAction{
								Command: []string{
									"/bin/sh",
									"-c",
									"sudo -u admin /home/admin/stop.sh>/var/log/sigma/stop.log 2>&1",
								},
							},
						},
					},
					ReadinessProbe: &api.Probe{
						Handler: api.Handler{
							Exec: &api.ExecAction{
								Command: []string{
									"/bin/sh",
									"-c",
									"sudo -u admin /home/admin/health.sh>/var/log/sigma/health.log 2>&1",
								},
							},
						},
						InitialDelaySeconds: 20,
						TimeoutSeconds:      60,
						PeriodSeconds:       1800,
					},
				},
			},
			ImagePullSecrets: []api.LocalObjectReference{
				{Name: "aone-image-secret"},
			},
		},
	}
	_ = plugin.Admit(admission.NewAttributesRecord(
		pod2,
		nil,
		api.Kind("Pod").WithVersion("version"),
		pod2.Namespace,
		"",
		api.Resource("pods").WithVersion("version"),
		"",
		admission.Update,
		&user.DefaultInfo{},
	))

	secrets, err := clientset.Core().Secrets("ns01").List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(secrets.Items) == 0 {
		t.Fatalf("list secret empty")
	} else if len(secrets.Items) > 1 {
		t.Fatalf("list secret more than 1: %#v", secrets.Items)
	}
	if string(secrets.Items[0].Data[api.DockerConfigJsonKey]) != `{"auths":{"reg.docker.alibaba-inc.com":{"username":"aone","password":"abcd","email":"sigma.ali","auth":"YW9uZTphYmNk"}}}` {
		t.Fatalf("secret %s not equal to expected", api.DockerConfigJsonKey)
	}
	expectPod.Spec.ImagePullSecrets = []api.LocalObjectReference{
		{Name: secrets.Items[0].Name},
	}

	if !reflect.DeepEqual(pod2.Spec, expectPod.Spec) {
		t.Fatalf("pod not equal, origPod: %+v got: %+v", expectPod.Spec, pod2.Spec)
	}
}

func TestGenerateSecretDockerRegistryCfg(t *testing.T) {
	client := fake.NewSimpleClientset()

	secret, err := generateSecretDockerRegistryCfg("fake-secret", "aone", "abcd", "reg.docker.alibaba-inc.com", "sigma.ali")
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
