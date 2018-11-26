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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SketchSummary is a top-level container for holding NodeSketch and PodSketch.
type SketchSummary struct {
	// Overall node sketch.
	Node NodeSketch `json:"node"`
	// Per-pod sketch.
	Pods []PodSketch `json:"pods"`
}

// NodeSketch holds node-level unprocessed sample sketch.
type NodeSketch struct {
	// Name is the name of the measured Node.
	Name string `json:"name"`
	// CPU resources related Sketch.
	// +optional
	CPU *NodeCPUSketch `json:"cpu,omitempty"`
	// System load related sketch.
	// +optional
	Load *NodeSystemLoadSketch `json:"load,omitempty"`
	// Memory resources related Sketch.
	// +optional
	Memory *NodeMemorySketch `json:"memory,omitempty"`
}

// PodSketch holds pod-level unprocessed sample sketch.
type PodSketch struct {
	// Reference to the measured Pod.
	PodRef PodReference `json:"podRef"`
	// Sketch of containers in the measured pod.
	// +patchMergeKey=name
	// +patchStrategy=merge
	Containers []*ContainerSketch `json:"containers" patchStrategy:"merge" patchMergeKey:"name"`
}

// ContainerSketch holds container-level unprocessed sample sketch.
type ContainerSketch struct {
	// Name is the name of the measured container.
	Name string `json:"name"`
	// ID is the id of the measured container.
	ID string `json:"id"`
	// CPU resources related Sketch.
	// +optional
	CPU *ContainerCPUSketch `json:"cpu,omitempty"`
	// Memory resources related Sketch.
	// +optional
	Memory *ContainerMemorySketch `json:"memory,omitempty"`
}

// PodReference contains enough information to locate the referenced pod.
type PodReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	UID       string `json:"uid"`
}

// NodeCPUSketch contains data about node-level CPU usage
type NodeCPUSketch struct {
	// The time at which these sketch were updated.
	Time metav1.Time `json:"time"`
	// Total CPU Usage
	Usage *SketchData `json:"usage,omitempty"`
}

// NodeSystemLoadSketch contains data about system-level load
type NodeSystemLoadSketch struct {
	// The time at which these sketch were updated.
	Time  metav1.Time `json:"time"`
	Min1  *SketchData `json:"min1,omitempty"`
	Min5  *SketchData `json:"min5,omitempty"`
	Min15 *SketchData `json:"min15,omitempty"`
}

// ContainerCPUSketch contains data about container-level CPU usage
type ContainerCPUSketch struct {
	// The time at which these sketch were updated.
	Time metav1.Time `json:"time"`
	// Total Usage relative to Resource.Limit.CPU
	UsageInLimit *SketchData `json:"usageInLimit,omitempty"`
	// Total Usage relative to Resource.Request.CPU
	UsageInRequest *SketchData `json:"usageInRequest,omitempty"`
	// CPU Load over the last 10 seconds.
	LoadAverage *SketchData `json:"loadAverage,omitempty"`
}

// NodeMemorySketch contains data about node-level memory usage
type NodeMemorySketch struct {
	MemorySketch `json:",inline"`
}

// ContainerMemorySketch contains data about container-level memory usage
type ContainerMemorySketch struct {
	MemorySketch `json:",inline"`
}

// MemorySketch contains data about memory usage
type MemorySketch struct {
	// The time at which these sketch were updated.
	Time metav1.Time `json:"time"`
	// Available memory for use.  This is defined as the memory limit - workingSetBytes.
	// If memory limit is undefined, the available bytes is omitted.
	// +optional
	AvailableBytes uint64 `json:"availableBytes,omitempty"`
	// Total memory in use. This includes all memory regardless of when it was accessed.
	// +optional
	UsageBytes uint64 `json:"usageBytes,omitempty"`
	// The amount of working set memory. This includes recently accessed memory,
	// dirty memory, and kernel memory. WorkingSetBytes is <= UsageBytes
	// +optional
	WorkingSetBytes uint64 `json:"workingSetBytes,omitempty"`
}

// SketchData is a buffer and result for each metrics
type SketchData struct {
	Latest float64 `json:"latest"`
	// cumulation value for 1 minute
	Min1 SketchCumulation `json:"min1"`
	// cumulation value for 5 minute
	Min5 SketchCumulation `json:"min5"`
	// cumulation value for 15 minute
	Min15 SketchCumulation `json:"min15"`
}

// SketchCumulation can be used as min/max/avg value of some period
type SketchCumulation struct {
	Max float64 `json:"max,omitempty"`
	Min float64 `json:"min,omitempty"`
	Avg float64 `json:"avg,omitempty"`
	P99 float64 `json:"p99,omitempty"`
	// Predict represents the value in future periods that depends on the needs of the specific metric.
	// For example, calculate based the values of a minute from the past of the system CPU usage
	// to predict next a minute; and maybe predict next five minutes.
	Predict float64 `json:"predict,omitempty"`
}
