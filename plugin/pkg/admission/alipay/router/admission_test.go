package router

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
		handler := NewRouter()
		assert.Equal(shouldHandle, handler.Handles(op))
	}
}

func TestAdmit(t *testing.T) {
	assert := assert.New(t)

	type TestCase struct {
		name          string
		env           bool
		volumeExist   bool
		injectLabel   bool
		injectPublic  bool
		injectDefault bool
	}

	handler := NewRouter()

	tcs := []*TestCase{
		{
			name:          "env exist, volume/volumeMount already mounted, expect no update.",
			env:           true,
			volumeExist:   true,
			injectLabel:   false,
			injectPublic:  false,
			injectDefault: true,
		},
		{
			name:          "env and label exist, volume/volumeMount already mounted, expect no update.",
			env:           true,
			volumeExist:   true,
			injectLabel:   true,
			injectPublic:  true,
			injectDefault: false,
		},
		{
			name:          "env exist, volume/volumeMount does not mounted, expect update default volume.",
			env:           true,
			volumeExist:   false,
			injectLabel:   false,
			injectPublic:  false,
			injectDefault: true,
		},
		{
			name:          "env and label exist, volume/volumeMount does not mounted, expect update public volume.",
			env:           true,
			volumeExist:   false,
			injectLabel:   true,
			injectPublic:  true,
			injectDefault: false,
		},
		{
			name:          "env and label does not exist, volume/volumeMount mounted, expect not update public volume.",
			env:           false,
			volumeExist:   true,
			injectLabel:   false,
			injectPublic:  false,
			injectDefault: true,
		},
		{
			name:          "env and label does not exist, volume/volumeMount does not mounted, expect not update public volume.",
			env:           false,
			volumeExist:   false,
			injectLabel:   false,
			injectPublic:  false,
			injectDefault: false,
		},
	}
	for _, tc := range tcs {
		t.Logf("test case: %v", tc.name)
		pod := newPod(tc.env, tc.volumeExist, tc.injectLabel)

		attr := admission.NewAttributesRecord(pod, nil, core.Kind("Pod").WithVersion("version"), pod.Namespace, pod.Name, core.Resource("pods").WithVersion("version"), "", admission.Create, false, nil)
		err := handler.Admit(attr)
		assert.Nil(err)

		if tc.injectDefault || tc.injectPublic {
			// assert volume is successfully injected
			assert.Equal(1, len(pod.Spec.Volumes))
			assert.Equal(RouterVolumeName, pod.Spec.Volumes[0].Name)
			if tc.injectPublic {
				assert.Equal(RouterVolumePublicSource, pod.Spec.Volumes[0].HostPath.Path)
			} else {
				assert.Equal(RouterVolumeSource, pod.Spec.Volumes[0].HostPath.Path)
			}
			assert.Equal(core.HostPathFile, *pod.Spec.Volumes[0].HostPath.Type)

			// assert container env and volumeMounts are successfully injected
			for i, container := range pod.Spec.Containers {
				if i == 0 {
					if tc.env {
						assert.Equal(1, len(container.Env))
						assert.Equal(RouterInjectEnvKey, container.Env[0].Name)
						assert.Equal(RouterInjectEnvValue, container.Env[0].Value)
					}

					assert.Equal(1, len(container.VolumeMounts))
					assert.Equal(RouterVolumeName, container.VolumeMounts[0].Name)
					assert.Equal(RouterVolumeDestination, container.VolumeMounts[0].MountPath)
					assert.Equal(true, container.VolumeMounts[0].ReadOnly)
				} else {
					assert.Equal(0, len(container.Env))
				}
			}
		} else {
			assert.Equal(0, len(pod.Spec.Volumes))
			assert.Equal(0, len(pod.Spec.Containers[0].VolumeMounts))
		}
		t.Logf("Admitted Pod:%#v", pod.Spec)
	}

}

func newPod(env, volumeExist, injectLabel bool) *core.Pod {
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-setdefault-pod",
			Namespace:   metav1.NamespaceDefault,
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:  "javaweb",
					Image: "pause:2.0",
				},
				{
					Name:  "sidecar",
					Image: "pause:2.0",
				},
			},
		},
	}
	if env {
		pod.Spec.Containers[0].Env = []core.EnvVar{
			core.EnvVar{
				Name:  RouterInjectEnvKey,
				Value: RouterInjectEnvValue,
			},
		}
	}
	if injectLabel {
		pod.Labels[RouterInjectLabel] = "true"
	}
	if volumeExist {
		if injectLabel {
			pod.Spec.Volumes = []core.Volume{
				getVolume(RouterVolumePublicSource),
			}
		} else {
			pod.Spec.Volumes = []core.Volume{
				getVolume(RouterVolumeSource),
			}
		}
		pod.Spec.Containers[0].VolumeMounts = []core.VolumeMount{
			getVolumeMount(RouterVolumeDestination),
		}
	}
	return pod
}
