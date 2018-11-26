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
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sketchapi "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/api/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
)

const (
	availableBytes  = "availableBytes"
	usageBytes      = "usageBytes"
	workingSetBytes = "workingSetBytes"
)

var (
	containerMemoryMetrics = map[string]core.Metric{
		availableBytes:  core.ContainerMemoryAvailableBytes,
		usageBytes:      core.ContainerMemoryUsageBytes,
		workingSetBytes: core.ContainerMemoryWorkingSetBytes,
	}

	nodeMemoryMetrics = map[string]core.Metric{
		availableBytes:  core.NodeMemoryAvailableBytes,
		usageBytes:      core.NodeMemoryUsedBytes,
		workingSetBytes: core.NodeMemoryWorkingsetBytes,
	}
)

func buildMemorySketch(sketch *sketchapi.MemorySketch, valueSet *core.MetricValueSet, metrics map[string]core.Metric) bool {

	tables := map[string]*uint64{
		availableBytes:  &sketch.AvailableBytes,
		usageBytes:      &sketch.UsageBytes,
		workingSetBytes: &sketch.WorkingSetBytes,
	}

	for k, v := range tables {
		m, ok := metrics[k]
		if !ok {
			continue
		}
		value, ok := valueSet.Values[m.Name]
		if ok {
			*v = uint64(value)
		}
	}

	var empty sketchapi.MemorySketch
	if reflect.DeepEqual(&empty, sketch) {
		return false
	}

	sketch.Time = metav1.NewTime(valueSet.Timestamp)
	return true
}
