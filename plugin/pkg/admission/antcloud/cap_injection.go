package antcloud

/*
Copyright 2019 The Kubernetes Authors.

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
import (
	"github.com/golang/glog"
	"io"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
)

const (
	PluginName  = "CapInjection"
	SysResource = "sys_resource"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewCapInjector(), nil
	})
}

// Plugin contains the client used by the admission controller
type Plugin struct {
	*admission.Handler
}

var _ admission.MutationInterface = &Plugin{}

// NewSysResourceInjector creates a new instance of the NewSysResourceInjector admission controller
func NewCapInjector() *Plugin {
	return &Plugin{
		Handler: admission.NewHandler(admission.Create),
	}
}

// Admit will check pod cap drop and cap add
// and drop sys_resource if needed
func (p *Plugin) Admit(attributes admission.Attributes) (err error) {
	// Ignore all calls to subresources or resources other than pods.
	if len(attributes.GetSubresource()) != 0 || attributes.GetResource().GroupResource() != api.Resource("pods") {
		return nil
	}
	pod, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}
	return InjectPodCap(pod)
}

func InjectPodCap(pod *api.Pod) error {
	containers := pod.Spec.Containers
	for idx, container := range containers {
		needed := true
		if container.SecurityContext == nil {
			container.SecurityContext = &api.SecurityContext{
				Capabilities: &api.Capabilities{
					Drop: []api.Capability{
						api.Capability(SysResource),
					},
				},
			}
			containers[idx] = container
			continue
		}
		if container.SecurityContext.Privileged != nil {
			if *container.SecurityContext.Privileged == true {
				continue
			}
		}
		if container.SecurityContext.Capabilities == nil {
			container.SecurityContext.Capabilities = &api.Capabilities{
				Drop: []api.Capability{
					api.Capability(SysResource),
				},
			}
			containers[idx] = container
			continue
		}
		caps := container.SecurityContext.Capabilities
		capsAdd := caps.Add
		capsDrop := caps.Drop
		for _, capAdd := range capsAdd {
			if string(capAdd) == SysResource {
				glog.V(5).Infof("user already adds sys_resource, not dropping it automatically")
				needed = false
			}
		}
		for _, capDrop := range capsDrop {
			if string(capDrop) == SysResource {
				glog.V(5).Infof("user already drops sys_resource, not dropping it automatically")
				needed = false
			}
		}
		if needed {
			capsDrop = append(capsDrop, api.Capability(SysResource))
			caps.Drop = capsDrop
			container.SecurityContext.Capabilities = caps
		}
		pod.Spec.Containers[idx] = container

	}
	return nil
}
