/*
Copyright 2019 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied
See the License for the specific language governing permissions and
limitations under the License.
*/
package populator

import (
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/volume"
)

func podContainsVolume(pod *v1.Pod, spec *volume.Spec) bool {
	if len(pod.Spec.Volumes) == 0 {
		glog.V(5).Infof("Pod(%s) contains no volume(%s), mount will not be requested again", pod.Name, spec.Name())
		return false
	}
	if spec.PersistentVolume != nil { // Remove PV from pod spec is dangerous, do not support for now
		return true
	}
	for _, podVolume := range pod.Spec.Volumes {
		if podVolume.Name == spec.Name() {
			return true
		}
	}
	glog.V(5).Infof("Pod(%s) no longer contains volume(%s), mount will not be requested again", pod.Name, spec.Name())
	return false
}
