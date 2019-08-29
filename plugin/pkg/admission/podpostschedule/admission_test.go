package podpostschedule

import (
	"testing"
	"time"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	cafev1alpha1 "gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/apps/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authentication/user"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	kubeletapis "k8s.io/kubernetes/pkg/kubelet/apis"
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

	nodeName := "foo-node"
	node := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   nodeName,
			Labels: map[string]string{},
		},
	}
	_ = informerFactory.Core().InternalVersion().Nodes().Informer().GetStore().Add(node)

	testCases := []struct {
		name   string
		pod    *api.Pod
		action admission.Operation
		az     string
	}{
		{
			name: "Create: Pod has no az label",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar1",
					Namespace: "foo",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "container1",
							Image: "image1",
						},
					},
				},
			},
			action: admission.Create,
			az:     "zone1",
		},
		{
			name: "Update: Pod has no az label",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar2",
					Namespace: "foo3",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "container2",
							Image: "image2",
						},
					},
				},
			},
			action: admission.Update,
			az:     "zone2",
		},
		{
			name: "Update: Pod has no az label",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar2",
					Namespace: "foo3",
					Labels: map[string]string{
						cafev1alpha1.AppServiceNameLabel: "appsvc",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "container2",
							Image: "image2",
						},
					},
				},
			},
			action: admission.Update,
			az:     "zone2",
		},
		{
			name: "Delete: Pod has no az",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar3",
					Namespace: "foo5",
					Labels: map[string]string{
						cafev1alpha1.AppServiceNameLabel: "appsvc",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "container3",
							Image: "image3",
						},
					},
				},
			},
			action: admission.Delete,
			az:     "zone3",
		},
		{
			name: "Update: Pod has no az",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar4",
					Namespace: "foo3",
					Labels: map[string]string{
						cafev1alpha1.AppServiceNameLabel: "appsvc",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "container4",
							Image: "image4",
						},
					},
					NodeName: node.Name,
				},
			},
			action: admission.Update,
			az:     "zone4",
		},
		{
			name: "Update: Pod has no az",
			pod: &api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar5",
					Namespace: "foo3",
					Labels: map[string]string{
						cafev1alpha1.AppServiceNameLabel: "appsvc",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "container5",
							Image: "image5",
							Env: []api.EnvVar{
								{
									Name:  PodAvailabilityZoneEnvKey,
									Value: "other",
								},
							},
						},
					},
					NodeName: node.Name,
				},
			},
			action: admission.Update,
			az:     "zone5",
		},
	}
	for _, ts := range testCases {
		pod := ts.pod

		node.Labels[kubeletapis.LabelZoneFailureDomain] = ts.az
		informerFactory.Core().InternalVersion().Nodes().Informer().GetStore().Add(node)

		userInfo := &user.DefaultInfo{
			Extra: map[string][]string{
				multitenancy.UserExtraInfoTenantID:    {"tenant"},
				multitenancy.UserExtraInfoWorkspaceID: {"workspace"},
				multitenancy.UserExtraInfoClusterID:   {"cluster"},
			},
		}
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
			userInfo,
		)

		hasScheduled := len(pod.Spec.NodeName) > 0
		_, hasAppSvc := pod.Labels[cafev1alpha1.AppServiceNameLabel]
		needInject := hasScheduled && hasAppSvc

		plugin.Admit(a)
		zone, exist := pod.Labels[kubeletapis.LabelZoneFailureDomain]
		if needInject && ts.action == admission.Update && !exist {
			t.Fatalf("has no az label on pod %s", pod.Name)
		}

		if needInject && ts.action == admission.Update && zone != ts.az {
			t.Fatalf("expected az %s, got %s", ts.az, zone)
		}

		for _, c := range pod.Spec.Containers {
			found := false
			val := ""
			for _, env := range c.Env {
				if env.Name == PodAvailabilityZoneEnvKey {
					found = true
					val = env.Value
					break
				}
			}

			if needInject && ts.action == admission.Update && !found {
				t.Fatalf("container %s has no az env", c.Name)
			}

			if needInject && ts.action == admission.Update && val != zone {
				t.Fatalf("expected container %s has env %s, got %s", c.Name, zone, val)
			}
		}
	}
}
