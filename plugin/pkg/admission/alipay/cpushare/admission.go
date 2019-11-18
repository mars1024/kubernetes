package cpushare

import (
	"io"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"

	"k8s.io/kubernetes/pkg/apis/core"
)

const (
	// PluginName is the name for current plugin, it should be unique among all plugins
	PluginName = "AlipayCPUShareInject"

	cpushareModeEnvName             = "ALIPAY_SIGMA_CPUMODE"
	cpushareModeEnvValue            = "cpushare"
	cpushareMaxProcessorEnvName     = "SIGMA_MAX_PROCESSORS_LIMIT"
	cpushareAJDKMaxProcessorEnvName = "AJDK_MAX_PROCESSORS_LIMIT"

	cpushareVolumeName = "cpushare-volume"
	cpusharePatchFile  = "/lib/libsysconf-alipay.so"
)

var (
	_ admission.MutationInterface = &AlipayCPUShareAdmission{}
)

// Register is used to register current plugin to APIServer
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return newAlipayCPUShareAdmission(), nil
	})
}

// AlipayCPUShareAdmission is the main struct to inject cpushare related fields to pod.
type AlipayCPUShareAdmission struct {
	*admission.Handler
}

func newAlipayCPUShareAdmission() *AlipayCPUShareAdmission {
	return &AlipayCPUShareAdmission{
		Handler: admission.NewHandler(admission.Create),
	}
}

// Admit makes an admission decision based on the request attributes.
func (a *AlipayCPUShareAdmission) Admit(attr admission.Attributes) (err error) {
	if shouldIgnore(attr) {
		return nil
	}

	pod := attr.GetObject().(*core.Pod)
	if err := a.injectCPUShare(pod); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}

func hasEnv(envs []core.EnvVar, envName string) bool {
	for _, env := range envs {
		if env.Name == envName {
			return true
		}
	}

	return false
}

func hasVolumeMount(mounts []core.VolumeMount, volumeName string) bool {
	for _, m := range mounts {
		if m.Name == volumeName {
			return true
		}
	}

	return false
}

func hasVolume(volumes []core.Volume, volumeName string) bool {
	for _, v := range volumes {
		if v.Name == volumeName {
			return true
		}
	}

	return false
}

func (a *AlipayCPUShareAdmission) injectCPUShare(pod *core.Pod) error {
	// add volumes to pod.
	// All pods should mount hostPath `/lib/libsysconf-alipay.so` to containers

	if !hasVolume(pod.Spec.Volumes, cpushareVolumeName) {
		hostPathFile := core.HostPathFile
		pod.Spec.Volumes = append(pod.Spec.Volumes, core.Volume{
			Name: cpushareVolumeName,
			VolumeSource: core.VolumeSource{
				HostPath: &core.HostPathVolumeSource{
					Path: cpusharePatchFile,
					Type: &hostPathFile,
				},
			},
		})
	}

	for i := range pod.Spec.Containers {
		// FIXME: this clearly adds env and volume mounts to all containers in the pod,
		// should we skip some containers? such as sidecar container?

		// add container env: cpusetmode=cpushare
		if !hasEnv(pod.Spec.Containers[i].Env, cpushareModeEnvName) {
			pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, core.EnvVar{
				Name:  cpushareModeEnvName,
				Value: cpushareModeEnvValue,
			})
		}

		//add container env: SIGMA_MAX_PROCESSORS_LIMIT=cpu request
		// For cpuset container, it should be cpuset cores number;
		// For cpushare container, it will be cpu request(after cpushare admission injection)
		if !hasEnv(pod.Spec.Containers[i].Env, cpushareMaxProcessorEnvName) {
			cpuNum := pod.Spec.Containers[i].Resources.Limits.Cpu()
			// round up to integer, so 1.5C would become 2C.
			// A zero cpu request would end up at least 1.
			cpuNum.RoundUp(0)

			pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, core.EnvVar{
				Name:  cpushareMaxProcessorEnvName,
				Value: cpuNum.String(),
			})
		}

		if !hasEnv(pod.Spec.Containers[i].Env, cpushareAJDKMaxProcessorEnvName) {
			cpuNum := pod.Spec.Containers[i].Resources.Limits.Cpu()
			// round up to integer, so 1.5C would become 2C.
			// A zero cpu request would end up at least 1.
			cpuNum.RoundUp(0)

			pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, core.EnvVar{
				Name:  cpushareAJDKMaxProcessorEnvName,
				Value: cpuNum.String(),
			})
		}

		// add volumeMounts
		if !hasVolumeMount(pod.Spec.Containers[i].VolumeMounts, cpushareVolumeName) {
			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, core.VolumeMount{
				Name:      cpushareVolumeName,
				MountPath: cpusharePatchFile,
				ReadOnly:  true,
			})
		}
	}
	return nil
}

func shouldIgnore(a admission.Attributes) bool {
	resource := a.GetResource().GroupResource()
	if resource != core.Resource("pods") {
		return true
	}

	if a.GetSubresource() != "" {
		return true
	}

	_, ok := a.GetObject().(*core.Pod)
	if !ok {
		glog.Errorf("expected pod but got %s", a.GetKind().Kind)
		return true
	}

	if a.GetOperation() != admission.Create {
		// only admit created pod
		return true
	}
	return false
}
