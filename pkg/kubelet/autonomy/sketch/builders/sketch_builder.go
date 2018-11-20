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

package builders

import (
	"k8s.io/api/core/v1"
	sketchapi "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/api/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
)

// StatsProvider is a interface that wraps GetNode method
type StatsProvider interface {
	GetPodByName(namespace, name string) (*v1.Pod, bool)
	GetNode() (*v1.Node, error)
}

// SketchBuilder is a interface that builds SketchData from DataBatch
type SketchBuilder interface {
	Build(batch *core.DataBatch) *sketchapi.SketchSummary
}

var _ SketchBuilder = &builderImpl{}

type builderImpl struct {
	statsProvider StatsProvider
}

// New constructs core.SketchBuilder instance
func New(statsProvider StatsProvider) SketchBuilder {
	return &builderImpl{
		statsProvider: statsProvider,
	}
}

func (b *builderImpl) Build(batch *core.DataBatch) *sketchapi.SketchSummary {
	var summary sketchapi.SketchSummary

	containerSkethes := b.buildContainerSketches(batch)
	summary.Pods = b.buildPodSketches(batch, containerSkethes)
	b.buildNodeSketch(&summary.Node, batch)

	return &summary
}

func (b *builderImpl) buildContainerSketches(batch *core.DataBatch) map[string][]*sketchapi.ContainerSketch {
	m := make(map[string][]*sketchapi.ContainerSketch)

	for _, valueSet := range batch.MetricValueSets {
		vsType := valueSet.CommonLabels[core.TypeLabel]
		if vsType != core.ContainerMetricType {
			continue
		}

		id := valueSet.CommonLabels[core.ContainerIDLabel]
		if id == "" {
			continue
		}

		var sketch sketchapi.ContainerSketch
		sketch.ID = id
		sketch.Name = b.getContainerName(id, valueSet.CommonLabels)
		sketch.CPU = buildContainerCPUSketch(valueSet)

		var memorySketch sketchapi.ContainerMemorySketch
		if buildMemorySketch(&memorySketch.MemorySketch, valueSet, containerMemoryMetrics) {
			sketch.Memory = &memorySketch
		}

		podName := valueSet.CommonLabels[core.PodNameLabel]
		namespace := valueSet.CommonLabels[core.NamespaceLabel]
		key := core.PodKey(namespace, podName)

		s := m[key]
		s = append(s, &sketch)
		m[key] = s
	}
	return m
}

func (b *builderImpl) getContainerName(id string, labels map[string]string) (name string) {
	if b.statsProvider == nil {
		return
	}

	namespace := labels[core.NamespaceLabel]
	podName := labels[core.PodNameLabel]
	if pod, ok := b.statsProvider.GetPodByName(namespace, podName); ok {
		for _, v := range pod.Status.ContainerStatuses {
			var containerID kubecontainer.ContainerID
			err := containerID.ParseString(v.ContainerID)
			if err == nil {
				if containerID.ID == id {
					name = v.Name
					break
				}
			}
		}
	}
	return
}

func (b *builderImpl) buildPodSketches(batch *core.DataBatch, containers map[string][]*sketchapi.ContainerSketch) []sketchapi.PodSketch {
	var sketches []sketchapi.PodSketch

	for _, valueSet := range batch.MetricValueSets {
		vsType := valueSet.CommonLabels[core.TypeLabel]
		if vsType != core.ContainerMetricType {
			continue
		}

		podName := valueSet.CommonLabels[core.PodNameLabel]
		namespace := valueSet.CommonLabels[core.NamespaceLabel]
		if podName == "" || namespace == "" {
			continue
		}

		var sketch sketchapi.PodSketch
		sketch.PodRef.Name = podName
		sketch.PodRef.Namespace = namespace

		if b.statsProvider != nil {
			pod, exist := b.statsProvider.GetPodByName(sketch.PodRef.Namespace, sketch.PodRef.Name)
			if exist {
				sketch.PodRef.UID = string(pod.UID)
			}
		}

		sketch.Containers = containers[core.PodKey(namespace, podName)]

		sketches = append(sketches, sketch)
	}
	return sketches
}

func (b *builderImpl) buildNodeSketch(sketch *sketchapi.NodeSketch, batch *core.DataBatch) {
	valueSet, ok := batch.MetricValueSets[core.NodeKey()]
	if !ok {
		return
	}

	if b.statsProvider != nil {
		if node, err := b.statsProvider.GetNode(); err == nil {
			sketch.Name = node.Name
		}
	}

	sketch.CPU = buildNodeCPUSketch(valueSet)
	sketch.Load = buildSystemLoad(valueSet)
	var memorySketch sketchapi.NodeMemorySketch
	if buildMemorySketch(&memorySketch.MemorySketch, valueSet, nodeMemoryMetrics) {
		sketch.Memory = &memorySketch
	}
}
