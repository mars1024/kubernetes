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

package dockershim

import (
	"context"

	"github.com/golang/glog"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

// RemoveVolume removes the volume.
func (ds *dockerService) RemoveVolume(_ context.Context, r *runtimeapi.RemoveVolumeRequest) (*runtimeapi.RemoveVolumeResponse, error) {
	err := ds.client.RemoveVolume(r.VolumeName, false)
	if err != nil {
		glog.Errorf("Failed to remove volume %s", r.VolumeName)
		return &runtimeapi.RemoveVolumeResponse{}, err
	}
	glog.V(0).Infof("Remove volume %s successfully", r.VolumeName)
	return &runtimeapi.RemoveVolumeResponse{}, nil
}
