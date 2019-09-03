package sigma

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHasProtectionFinalizer(t *testing.T) {
	tests := []struct {
		pod                    *v1.Pod
		hasProtectionFinalizer bool
	}{
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"protection.pod.beta1.sigma.ali/vip-removed", "pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			hasProtectionFinalizer: true,
		},
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			hasProtectionFinalizer: false,
		},
	}

	for _, test := range tests {
		assert.Equal(t, HasProtectionFinalizer(test.pod), test.hasProtectionFinalizer)
	}
}

func TestHasUpgradeProtectionFinalizer(t *testing.T) {
	tests := []struct {
		pod                           *v1.Pod
		hasUpgradeProtectionFinalizer bool
	}{
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"protection-upgrade.pod.beta1.sigma.ali/vip-removed", "pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			hasUpgradeProtectionFinalizer: true,
		},
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			hasUpgradeProtectionFinalizer: false,
		},
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"protection-delete.pod.beta1.sigma.ali/vip-removed"},
				},
			},
			hasUpgradeProtectionFinalizer: false,
		},
	}
	for _, test := range tests {
		assert.Equal(t, HasUpgradeProtectionFinalizer(test.pod), test.hasUpgradeProtectionFinalizer)
	}
}

func TestIsPodDockerVMMode(t *testing.T) {
	tests := []struct {
		message        string
		pod            *v1.Pod
		isDockerVMMode bool
	}{
		{
			message: "pod is docker_vm mode",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Labels:    map[string]string{sigmak8sapi.LabelServerType: "DOCKER_VM"},
				},
			},
			isDockerVMMode: true,
		},
		{
			message: "pod is not docker_vm mode, label is nil",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
				},
			},
			isDockerVMMode: false,
		},
		{
			message: "pod is not docker_vm mode, label value is wrong",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Labels:    map[string]string{sigmak8sapi.LabelServerType: "DOCKER_NOT_VM"},
				},
			},
			isDockerVMMode: false,
		},
		{
			message:        "pod is not docker_vm mode, pod is nil",
			pod:            nil,
			isDockerVMMode: false,
		},
	}

	for _, test := range tests {
		t.Logf("case %s", test.message)
		assert.Equal(t, IsPodDockerVMMode(test.pod), test.isDockerVMMode)
	}
}

func TestHasDeleteProtectionFinalizer(t *testing.T) {
	tests := []struct {
		pod                          *v1.Pod
		hasDeleteProtectionFinalizer bool
	}{
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"protection-delete.pod.beta1.sigma.ali/vip-removed", "pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			hasDeleteProtectionFinalizer: true,
		},
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			hasDeleteProtectionFinalizer: false,
		},
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"protection-upgrade.pod.beta1.sigma.ali/vip-removed", "pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			hasDeleteProtectionFinalizer: false,
		},
	}
	for _, test := range tests {
		assert.Equal(t, HasDeleteProtectionFinalizer(test.pod), test.hasDeleteProtectionFinalizer)
	}
}

func TestPodShouldNotDelete(t *testing.T) {
	tests := []struct {
		pod             *v1.Pod
		shouldNotDelete bool
	}{
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"protection-delete.pod.beta1.sigma.ali/vip-removed", "pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			shouldNotDelete: true,
		},
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"protection.pod.beta1.sigma.ali/vip-removed", "pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			shouldNotDelete: true,
		},
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"pod.beta1.sigma.ali/cni-allocated", "protection-upgrade.pod.beta1.sigma.ali/vip-removed"},
				},
			},
			shouldNotDelete: false,
		},
	}
	for _, test := range tests {
		assert.Equal(t, PodShouldNotDelete(test.pod), test.shouldNotDelete)
	}
}

func TestPodShouldNotUpgrade(t *testing.T) {
	tests := []struct {
		pod              *v1.Pod
		shouldNotUpgrade bool
	}{
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"protection-upgrade.pod.beta1.sigma.ali/vip-removed", "pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			shouldNotUpgrade: true,
		},
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"protection.pod.beta1.sigma.ali/vip-removed", "pod.beta1.sigma.ali/cni-allocated"},
				},
			},
			shouldNotUpgrade: true,
		},
		{
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  "foo",
					Finalizers: []string{"pod.beta1.sigma.ali/cni-allocated", "protection-delete.pod.beta1.sigma.ali/vip-removed"},
				},
			},
			shouldNotUpgrade: false,
		},
	}
	for _, test := range tests {
		assert.Equal(t, PodShouldNotUpgrade(test.pod), test.shouldNotUpgrade)
	}
}


func TestIsPodJob(t *testing.T) {
	tests := []struct {
		message  string
		pod      *v1.Pod
		isPodJob bool
	}{
		{
			message: "pod is job",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Labels:    map[string]string{sigmak8sapi.LabelPodIsJob: "true"},
				},
			},
			isPodJob: true,
		},
		{
			message: "pod is not job, label is nil",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
				},
			},
			isPodJob: false,
		},
		{
			message: "pod is not job, label value is wrong",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Labels:    map[string]string{sigmak8sapi.LabelPodIsJob: "something_else"},
				},
			},
			isPodJob: false,
		},
		{
			message:  "pod is nil",
			pod:      nil,
			isPodJob: false,
		},
	}

	for _, test := range tests {
		t.Logf("case %s", test.message)
		assert.Equal(t, IsPodJob(test.pod), test.isPodJob)
	}
}

func TestIsContainerReadyIgnore(t *testing.T) {
	tests := []struct {
		message                string
		container              *v1.Container
		isContainerReadyIgnore bool
	}{
		{
			message:                "container is nil",
			container:              nil,
			isContainerReadyIgnore: false,
		},
		{
			message: "container env is nil",
			container: &v1.Container{
				Name: "bar",
			},
			isContainerReadyIgnore: false,
		},
		{
			message: "container have env, but not have sidecar env",
			container: &v1.Container{
				Name: "bar",
				Env: []v1.EnvVar{
					{
						Name:  "a",
						Value: "b",
					},
				},
			},
			isContainerReadyIgnore: false,
		},
		{
			message: "container have sidecar env, but value is invalid",
			container: &v1.Container{
				Name: "bar",
				Env: []v1.EnvVar{
					{
						Name:  sigmak8sapi.EnvIgnoreReady,
						Value: "b",
					},
				},
			},
			isContainerReadyIgnore: false,
		},
		{
			message: "container have sidecar env, value is true, should ignore",
			container: &v1.Container{
				Name: "bar",
				Env: []v1.EnvVar{
					{
						Name:  sigmak8sapi.EnvIgnoreReady,
						Value: "true",
					},
				},
			},
			isContainerReadyIgnore: true,
		},
	}
	for _, test := range tests {
		t.Logf("case %s", test.message)
		assert.Equal(t, test.isContainerReadyIgnore, IsContainerReadyIgnore(test.container))
	}
}

func TestIsPodDisableServiceLinks(t *testing.T) {
	tests := []struct {
		message                  string
		pod                      *v1.Pod
		isPodDisableServiceLinks bool
	}{
		{
			message: "disalbe service links",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Labels:    map[string]string{sigmak8sapi.LabelDisableServiceLinks: "true"},
				},
			},
			isPodDisableServiceLinks: true,
		},
		{
			message: "enable service links because label is nil",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
				},
			},
			isPodDisableServiceLinks: false,
		},
		{
			message: "enable service links because label value is wrong",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
					Labels:    map[string]string{sigmak8sapi.LabelDisableServiceLinks: "something_else"},
				},
			},
			isPodDisableServiceLinks: false,
		},
		{
			message:                  "pod is nil",
			pod:                      nil,
			isPodDisableServiceLinks: false,
		},
	}

	for _, test := range tests {
		t.Logf("case %s", test.message)
		assert.Equal(t, IsPodDisableServiceLinks(test.pod), test.isPodDisableServiceLinks)
	}
}
