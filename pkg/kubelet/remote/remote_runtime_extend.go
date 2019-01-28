/*
Copyright 2016 The Kubernetes Authors.

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

package remote

import (
	"strings"

	"github.com/golang/glog"

	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

const (
	dockerPauseContainerError = "already paused"
	pouchPauseContainerError  = "not running: paused"
)

// PauseContainer pauses the container.
func (r *RemoteRuntimeService) PauseContainer(containerID string) error {
	ctx, cancel := getContextWithTimeout(r.timeout)
	defer cancel()

	_, err := r.runtimeClient.PauseContainer(ctx, &runtimeapi.PauseContainerRequest{
		ContainerId: containerID,
	})
	if err != nil {
		if strings.Contains(err.Error(), dockerPauseContainerError) || strings.Contains(err.Error(), pouchPauseContainerError) {
			return nil
		}
		glog.Errorf("PauseContainer %q from runtime service failed: %v", containerID, err)
		return err
	}

	return nil
}

// UnpauseContainer unpauses the container.
func (r *RemoteRuntimeService) UnpauseContainer(containerID string) error {
	ctx, cancel := getContextWithTimeout(r.timeout)
	defer cancel()

	_, err := r.runtimeClient.UnpauseContainer(ctx, &runtimeapi.UnpauseContainerRequest{
		ContainerId: containerID,
	})
	if err != nil {
		glog.Errorf("UnpauseContainer %q from runtime service failed: %v", containerID, err)
		return err
	}

	return nil
}
