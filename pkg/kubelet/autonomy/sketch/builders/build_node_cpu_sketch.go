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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sketchapi "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/api/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
)

var (
	nodeCPUUsageAggregations = aggregationMatrix{
		"min1": {
			"max":     core.NodeCPUUsageMaxOver1min,
			"min":     core.NodeCPUUsageMinOver1min,
			"avg":     core.NodeCPUUsageAvgOver1min,
			"p99":     core.NodeCPUUsageP99Over1min,
			"predict": core.NodeCPUUsagePredict1min,
		},
		"min5": {
			"max":     core.NodeCPUUsageMaxOver5min,
			"min":     core.NodeCPUUsageMinOver5min,
			"avg":     core.NodeCPUUsageAvgOver5min,
			"p99":     core.NodeCPUUsageP99Over5min,
			"predict": core.NodeCPUUsagePredict5min,
		},
		"min15": {
			"max":     core.NodeCPUUsageMaxOver15min,
			"min":     core.NodeCPUUsageMinOver15min,
			"avg":     core.NodeCPUUsageAvgOver15min,
			"p99":     core.NodeCPUUsageP99Over15min,
			"predict": core.NodeCPUUsagePredict15min,
		},
	}
)

func buildNodeCPUSketch(valueSet *core.MetricValueSet) *sketchapi.NodeCPUSketch {
	var sketch sketchapi.NodeCPUSketch
	sketch.Usage = buildSketchData(valueSet, &core.NodeCPUUsage, nodeCPUUsageAggregations)
	if sketch.Usage != nil {
		sketch.Time = metav1.NewTime(valueSet.Timestamp)
		return &sketch
	}
	return nil
}
