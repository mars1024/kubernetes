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

// Package ContainerState contains an admission controller that checks and modifies every new Pod
package containerstate

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/golang/glog"
	sigmaapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
)

// PluginName indicates name of admission plugin.
const PluginName = "ContainerState"

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewContainerState(), nil
	})
}

// ContainerState is an implementation of admission.Interface.
// It validates annotation `sigmaapi.AnnotationContainerStateSpec`.
type ContainerState struct {
	*admission.Handler
}

var _ admission.ValidationInterface = &ContainerState{}

// Validate makes sure that all containers are set to correct ContainerState labels
func (n *ContainerState) Validate(attributes admission.Attributes) (err error) {
	if shouldIgnore(attributes) {
		return nil
	}

	pod, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	op := attributes.GetOperation()
	if op != admission.Create && op != admission.Update {
		return apierrors.NewBadRequest("ContainerState Admission only handles Update event")
	}

	glog.V(10).Infof("ContainerState validateUpdate pod: %#v", pod)
	if pod.Annotations == nil {
		return nil
	}

	stateBytes, stateExist := pod.Annotations[sigmaapi.AnnotationContainerStateSpec]
	updateStatusBytes, updateStatusExist := pod.Annotations[sigmaapi.AnnotationPodUpdateStatus]

	if stateExist {
		var states sigmaapi.ContainerStateSpec
		err := json.Unmarshal([]byte(stateBytes), &states)
		if err != nil {
			return admission.NewForbidden(attributes,
				fmt.Errorf("annotation %s can not %s due to json unmarshal error `%s`", sigmaapi.AnnotationPodUpdateStatus, op, err))
		}
		for name, state := range states.States {
			if err := validateName(attributes, pod, name); err != nil {
				return err
			}
			if err := validateState(attributes, state, pod.Spec.RestartPolicy); err != nil {
				return err
			}
		}
	}

	if updateStatusExist {
		var statuses sigmaapi.ContainerStateStatus
		err := json.Unmarshal([]byte(updateStatusBytes), &statuses)
		if err != nil {
			return admission.NewForbidden(attributes,
				fmt.Errorf("annotation %s can not %s due to json unmarshal error `%s`", sigmaapi.AnnotationPodUpdateStatus, op, err))
		}
		for name, status := range statuses.Statuses {
			if err := validateName(attributes, pod, name); err != nil {
				return err
			}
			if err := validateState(attributes, status.CurrentState, ""); err != nil {
				return err
			}
			if err := validateState(attributes, status.LastState, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateName(attributes admission.Attributes, pod *api.Pod, name sigmaapi.ContainerInfo) error {
	op := attributes.GetOperation()
	var found bool
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == name.Name {
			found = true
		}
	}
	if !found {
		err := fmt.Errorf("container named %s not found", name.Name)
		return admission.NewForbidden(attributes,
			fmt.Errorf("annotation %s can not %s due to %s", sigmaapi.AnnotationPodUpdateStatus, op, err))
	}
	return nil
}

func validateState(attributes admission.Attributes, status sigmaapi.ContainerState, restartPolicy api.RestartPolicy) error {
	op := attributes.GetOperation()
	switch status {
	case sigmaapi.ContainerStateCreated, sigmaapi.ContainerStatePaused,
		sigmaapi.ContainerStateExited, sigmaapi.ContainerStateUnknown:
	case sigmaapi.ContainerStateRunning:
		if restartPolicy == api.RestartPolicyNever {
			err := fmt.Errorf("pod restart policy is never, so container can't be started")
			return admission.NewForbidden(attributes,
				fmt.Errorf("annotation %s can not %s due to %s", sigmaapi.AnnotationPodUpdateStatus, op, err))
		}
	default:
		err := fmt.Errorf("container state %s is not valid", status)
		return admission.NewForbidden(attributes,
			fmt.Errorf("annotation %s can not %s due to %s", sigmaapi.AnnotationPodUpdateStatus, op, err))
	}
	return nil
}

func shouldIgnore(attributes admission.Attributes) bool {
	// Ignore all calls to subresources or resources other than pods.
	if len(attributes.GetSubresource()) != 0 || attributes.GetResource().GroupResource() != api.Resource("pods") {
		return true
	}

	return false
}

// NewContainerState creates a new ContainerState admission control handler
func NewContainerState() *ContainerState {
	return &ContainerState{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}
}
