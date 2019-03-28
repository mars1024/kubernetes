package router

/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package routerinject contains an admission controller that checks and modifies every new Pod

import (
	"io"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
)

const (
	// PluginName indicates name of admission plugin.
	PluginName = "AlipayRouterInjector"

	// Inject env.
	RouterInjectEnvKey   = "ali_run_mode"
	RouterInjectEnvValue = "alipay_container"

	// Inject label
	RouterInjectLabel = "ali.EnableDefaultRoute"

	// Router Volume info, readonly.
	RouterVolumeName         = "router-volume"
	RouterVolumePublicSource = "/opt/ali-iaas/env_create/alipay_route.public.tmpl"
	RouterVolumeSource       = "/opt/ali-iaas/env_create/alipay_route.tmpl"
	RouterVolumeDestination  = "/etc/route.tmpl"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewRouter(), nil
	})
}

// NewRouter creates a new router admission control handler
func NewRouter() *AlipayRouterInjector {
	return &AlipayRouterInjector{
		Handler: admission.NewHandler(admission.Create),
	}
}

// Armory is an implementation of admission.Interface.
// It validates labels of pods which must meet sigma policy.
type AlipayRouterInjector struct {
	*admission.Handler
}

var _ admission.MutationInterface = &AlipayRouterInjector{}

// Admit makes an admission decision based on the request attributes
func (a *AlipayRouterInjector) Admit(attributes admission.Attributes) (err error) {
	if shouldIgnore(attributes) {
		return nil
	}

	pod, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	op := attributes.GetOperation()
	switch op {
	case admission.Create:
		return a.injectRouter(pod)
	}
	return apierrors.NewBadRequest("Alipay Router Injector Admission only handles Create event")
}

func (a *AlipayRouterInjector) injectRouter(pod *api.Pod) error {
	// add router volumes to pod.
	publicRouter := hasPublicRouterLabel(pod.Labels)
	for i := range pod.Spec.Containers {
		if hasInjectorEnv(pod.Spec.Containers[i].Env) {
			if !hasVolumeMount(pod.Spec.Containers[i].VolumeMounts, RouterVolumeName) {
				pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, getVolumeMount(RouterVolumeDestination))
				if !hasVolume(pod.Spec.Volumes, RouterVolumeName) {
					if publicRouter {
						pod.Spec.Volumes = append(pod.Spec.Volumes, getVolume(RouterVolumePublicSource))
					} else {
						pod.Spec.Volumes = append(pod.Spec.Volumes, getVolume(RouterVolumeSource))
					}
				}
			}
		}
	}
	return nil
}

func hasInjectorEnv(envs []api.EnvVar) bool {
	for _, env := range envs {
		if env.Name == RouterInjectEnvKey && env.Value == RouterInjectEnvValue {
			return true
		}
	}

	return false
}

func hasPublicRouterLabel(labels map[string]string) bool {
	if labels == nil {
		return false
	}
	if value, ok := labels[RouterInjectLabel]; ok && value == "true" {
		return true
	}
	return false
}

func hasVolumeMount(mounts []api.VolumeMount, volumeName string) bool {
	for _, m := range mounts {
		if m.Name == volumeName {
			return true
		}
	}

	return false
}

func hasVolume(volumes []api.Volume, volumeName string) bool {
	for _, v := range volumes {
		if v.Name == volumeName {
			return true
		}
	}

	return false
}

func getVolume(path string) api.Volume {
	hostPathFile := api.HostPathFile
	return api.Volume{
		Name: RouterVolumeName,
		VolumeSource: api.VolumeSource{
			HostPath: &api.HostPathVolumeSource{
				Path: path,
				Type: &hostPathFile,
			},
		},
	}
}

func getVolumeMount(dest string) api.VolumeMount {
	return api.VolumeMount{
		Name:      RouterVolumeName,
		MountPath: RouterVolumeDestination,
		ReadOnly:  true,
	}
}

func shouldIgnore(attributes admission.Attributes) bool {
	// Ignore all calls to subresources or resources other than pods.
	if len(attributes.GetSubresource()) != 0 || attributes.GetResource().GroupResource() != api.Resource("pods") {
		return true
	}

	return false
}
