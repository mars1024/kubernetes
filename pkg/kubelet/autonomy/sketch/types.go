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
	"errors"

	"k8s.io/apimachinery/pkg/types"
	sketchapi "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/api/v1alpha1"
)

// Definition of error
var (
	ErrEmpty    = errors.New("sketch: empty sketch")
	ErrNotFound = errors.New("sketch: not found sketch")
)

// Provider provide resouce sketch(1/5/15 miniute metrics of node/pod/container)
type Provider interface {
	Start() error
	Stop()

	GetSketch() Snapshoter
}

// Snapshoter is a interface that support to query sketch summary
type Snapshoter interface {
	GetSummary() (*sketchapi.SketchSummary, error)
	GetNodeSketch() (*sketchapi.NodeSketch, error)
	GetPodSketch(namespace, name string, uid types.UID) (*sketchapi.PodSketch, error)
	GetContainerSketchByName(namepsace, podName string, podUID types.UID, containerName string) (*sketchapi.ContainerSketch, error)
	GetContainerSketchByID(namespace, podName string, podUID types.UID, containerID string) (*sketchapi.ContainerSketch, error)
}
