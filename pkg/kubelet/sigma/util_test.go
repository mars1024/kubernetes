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
