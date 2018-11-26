/*
Copyright 2018 The Kubernetes Authors.

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

package cm

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
)

// Update updates pod level cgroup
func (m *podContainerManagerImpl) Update(pod *v1.Pod) error {
	podContainerName, _ := m.GetPodContainerName(pod)
	// If podContainerName is nil, it means invalid custom cgroup parent is applied.
	// Just return invalid custom cgroup parent error
	if podContainerName == nil {
		return fmt.Errorf("Invalid custom cgroup parent")
	}

	containerConfig := &CgroupConfig{
		Name:               podContainerName,
		ResourceParameters: ResourceConfigForPod(pod, m.enforceCPULimits, m.cpuCFSQuotaPeriod),
	}
	if err := m.cgroupManager.Update(containerConfig); err != nil {
		return fmt.Errorf("failed to update container for %v : %v", podContainerName, err)
	}

	return nil
}

// GetCgroupParentFromAnnotation extracts cgroup parent from pod annotations
func GetCgroupParentFromAnnotation(pod *v1.Pod) string {
	podAllocSpecJSON, ok := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
	if !ok {
		return ""
	}

	podAllocSpec := &sigmak8sapi.AllocSpec{}
	// unmarshal pod allocation spec
	if err := json.Unmarshal([]byte(podAllocSpecJSON), podAllocSpec); err != nil {
		glog.Errorf("could not get cgroup parent, unmarshal alloc spec err: %v", err)
		return ""
	}
	for _, containerInfo := range podAllocSpec.Containers {
		// since cgroup parent is set on sandbox container, different container must have
		// same cgroup parent config, we could validate this in apiserver
		if len(containerInfo.HostConfig.CgroupParent) != 0 {
			return containerInfo.HostConfig.CgroupParent
		}
	}
	return ""
}