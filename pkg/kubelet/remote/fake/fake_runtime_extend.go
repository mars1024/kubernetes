/*
Copyright 2017 The Kubernetes Authors.

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

package fake

import (
	"context"
	kubeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

// PauseContainer pauses the container.
func (f *RemoteRuntime) PauseContainer(ctx context.Context, req *kubeapi.PauseContainerRequest) (*kubeapi.PauseContainerResponse, error) {
	err := f.RuntimeService.PauseContainer(req.ContainerId)
	if err != nil {
		return nil, err
	}

	return &kubeapi.PauseContainerResponse{}, nil
}

// UnpauseContainer unpauses the container.
func (f *RemoteRuntime) UnpauseContainer(ctx context.Context, req *kubeapi.UnpauseContainerRequest) (*kubeapi.UnpauseContainerResponse, error) {
	err := f.RuntimeService.UnpauseContainer(req.ContainerId)
	if err != nil {
		return nil, err
	}

	return &kubeapi.UnpauseContainerResponse{}, nil
}
