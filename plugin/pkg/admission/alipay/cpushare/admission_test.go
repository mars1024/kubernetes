package cpushare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"

	"k8s.io/kubernetes/pkg/apis/core"
)

func TestRegister(t *testing.T) {
	assert := assert.New(t)

	plugins := admission.NewPlugins()
	Register(plugins)
	registered := plugins.Registered()

	assert.Equal(len(registered), 1, "plugin should be registered")
	assert.Equal(registered[0], PluginName, "plugin should be registered")
}

func TestHandles(t *testing.T) {
	assert := assert.New(t)

	testCases := map[admission.Operation]bool{
		admission.Create:  true,
		admission.Update:  false,
		admission.Connect: false,
		admission.Delete:  false,
	}

	for op, shouldHandle := range testCases {
		handler := newAlipayCPUShareAdmission()
		assert.Equal(shouldHandle, handler.Handles(op))
	}
}

func TestAdmit(t *testing.T) {
	assert := assert.New(t)

	handler := newAlipayCPUShareAdmission()
	pod := newPod()
	attr := admission.NewAttributesRecord(pod, nil, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Create, false, nil)
	err := handler.Admit(attr)
	assert.Nil(err)

	// assert volume is successfully injected
	assert.Equal(1, len(pod.Spec.Volumes))
	assert.Equal(cpushareVolumeName, pod.Spec.Volumes[0].Name)
	assert.Equal(cpusharePatchFile, pod.Spec.Volumes[0].HostPath.Path)
	assert.Equal(core.HostPathFile, *pod.Spec.Volumes[0].HostPath.Type)

	// assert container env and volumeMounts are successfully injected
	for _, container := range pod.Spec.Containers {
		assert.Equal(1, len(container.Env))
		assert.Equal(cpushareModeEnvName, container.Env[0].Name)
		assert.Equal(cpushareModeEnvValue, container.Env[0].Value)

		assert.Equal(1, len(container.VolumeMounts))
		assert.Equal(cpushareVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(cpusharePatchFile, container.VolumeMounts[0].MountPath)
		assert.Equal(true, container.VolumeMounts[0].ReadOnly)
	}
}

func newPod() *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-setdefault-pod",
			Namespace:   metav1.NamespaceDefault,
			Annotations: map[string]string{},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:  "javaweb",
					Image: "pause:2.0",
					Env: []core.EnvVar{
						core.EnvVar{
							Name:  cpushareModeEnvName,
							Value: cpushareModeEnvValue,
						},
					},
				},
				{
					Name:  "sidecar",
					Image: "pause:2.0",
				},
			},
		},
	}
}
