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

package sketch

import (
	"k8s.io/apimachinery/pkg/types"
	sketchapi "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/api/v1alpha1"
)

var _ Snapshoter = &snapshoterImpl{}

type snapshoterImpl struct {
	summary interface{}
}

func newSnapshotImpl(x interface{}) *snapshoterImpl {
	return &snapshoterImpl{
		summary: x,
	}
}

func (s *snapshoterImpl) GetSummary() (*sketchapi.SketchSummary, error) {
	summary, ok := s.summary.(*sketchapi.SketchSummary)
	if !ok {
		return nil, ErrEmpty
	}
	return summary, nil
}

func (s *snapshoterImpl) GetNodeSketch() (*sketchapi.NodeSketch, error) {
	summary, ok := s.summary.(*sketchapi.SketchSummary)
	if !ok {
		return nil, ErrEmpty
	}

	return &summary.Node, nil
}

func (s *snapshoterImpl) GetPodSketch(namespace, podName string, podUID types.UID) (*sketchapi.PodSketch, error) {
	summary, ok := s.summary.(*sketchapi.SketchSummary)
	if !ok {
		return nil, ErrEmpty
	}

	var podSkech *sketchapi.PodSketch
	for _, v := range summary.Pods {
		if (v.PodRef.Namespace == namespace && v.PodRef.Name == podName) ||
			(v.PodRef.UID != "" && v.PodRef.UID == string(podUID)) {
			podSkech = &v
			break
		}
	}

	var err error
	if podSkech == nil {
		err = ErrNotFound
	}

	return podSkech, err
}

func (s *snapshoterImpl) GetContainerSketchByName(namespace, podName string, podUID types.UID, containerName string) (*sketchapi.ContainerSketch, error) {
	return filterContainerSketch(s, namespace, podName, podUID, func(c *sketchapi.ContainerSketch) bool {
		return c.Name == containerName
	})
}

func (s *snapshoterImpl) GetContainerSketchByID(namespace, podName string, podUID types.UID, containerID string) (*sketchapi.ContainerSketch, error) {
	return filterContainerSketch(s, namespace, podName, podUID, func(c *sketchapi.ContainerSketch) bool {
		return c.ID != "" && c.ID == containerID
	})
}

func filterContainerSketch(s *snapshoterImpl, namespace, podName string, podUID types.UID, f func(c *sketchapi.ContainerSketch) bool) (*sketchapi.ContainerSketch, error) {
	podSketch, err := s.GetPodSketch(namespace, podName, podUID)
	if err != nil {
		return nil, err
	}

	var sketch *sketchapi.ContainerSketch
	for _, v := range podSketch.Containers {
		if f(v) {
			sketch = v
			break
		}
	}

	if sketch == nil {
		err = ErrNotFound
	}

	return sketch, err
}
