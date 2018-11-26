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
	containerCPUUsageLimitAggregations = aggregationMatrix{
		"min1": {
			"max":     core.ContainerCPUUsageLimitMaxOver1min,
			"min":     core.ContainerCPUUsageLimitMinOver1min,
			"avg":     core.ContainerCPUUsageLimitAvgOver1min,
			"p99":     core.ContainerCPUUsageLimitP99Over1min,
			"predict": core.ContainerCPUUsageLimitPredict1min,
		},
		"min5": {
			"max":     core.ContainerCPUUsageLimitMaxOver5min,
			"min":     core.ContainerCPUUsageLimitMinOver5min,
			"avg":     core.ContainerCPUUsageLimitAvgOver5min,
			"p99":     core.ContainerCPUUsageLimitP99Over5min,
			"predict": core.ContainerCPUUsageLimitPredict5min,
		},
		"min15": {
			"max":     core.ContainerCPUUsageLimitMaxOver15min,
			"min":     core.ContainerCPUUsageLimitMinOver15min,
			"avg":     core.ContainerCPUUsageLimitAvgOver15min,
			"p99":     core.ContainerCPUUsageLimitP99Over15min,
			"predict": core.ContainerCPUUsageLimitPredict15min,
		},
	}

	containerCPUUsageRequestAggregations = aggregationMatrix{
		"min1": {
			"max":     core.ContainerCPUUsageRequestMaxOver1min,
			"min":     core.ContainerCPUUsageRequestMinOver1min,
			"avg":     core.ContainerCPUUsageRequestAvgOver1min,
			"p99":     core.ContainerCPUUsageRequestP99Over1min,
			"predict": core.ContainerCPUUsageRequestPredict1min,
		},
		"min5": {
			"max":     core.ContainerCPUUsageRequestMaxOver5min,
			"min":     core.ContainerCPUUsageRequestMinOver5min,
			"avg":     core.ContainerCPUUsageRequestAvgOver5min,
			"p99":     core.ContainerCPUUsageRequestP99Over5min,
			"predict": core.ContainerCPUUsageRequestPredict5min,
		},
		"min15": {
			"max":     core.ContainerCPUUsageRequestMaxOver15min,
			"min":     core.ContainerCPUUsageRequestMinOver15min,
			"avg":     core.ContainerCPUUsageRequestAvgOver15min,
			"p99":     core.ContainerCPUUsageRequestP99Over15min,
			"predict": core.ContainerCPUUsageRequestPredict15min,
		},
	}

	containerCPULoadAvgAggregations = aggregationMatrix{
		"min1": {
			"max":     core.ContainerCPULoadAverage10sMaxOver1min,
			"min":     core.ContainerCPULoadAverage10sMinOver1min,
			"avg":     core.ContainerCPULoadAverage10sAvgOver1min,
			"p99":     core.ContainerCPULoadAverage10sP99Over1min,
			"predict": core.ContainerCPULoadAverage10sPredict1min,
		},
		"min5": {
			"max":     core.ContainerCPULoadAverage10sMaxOver5min,
			"min":     core.ContainerCPULoadAverage10sMinOver5min,
			"avg":     core.ContainerCPULoadAverage10sAvgOver5min,
			"p99":     core.ContainerCPULoadAverage10sP99Over5min,
			"predict": core.ContainerCPULoadAverage10sPredict5min,
		},
		"min15": {
			"max":     core.ContainerCPULoadAverage10sMaxOver15min,
			"min":     core.ContainerCPULoadAverage10sMinOver15min,
			"avg":     core.ContainerCPULoadAverage10sAvgOver15min,
			"p99":     core.ContainerCPULoadAverage10sP99Over15min,
			"predict": core.ContainerCPULoadAverage10sPredict15min,
		},
	}
)

func buildContainerCPUSketch(valueSet *core.MetricValueSet) *sketchapi.ContainerCPUSketch {
	var sketch sketchapi.ContainerCPUSketch
	sketch.UsageInLimit = buildSketchData(valueSet, &core.ContainerCPUUsageLimit, containerCPUUsageLimitAggregations)
	sketch.UsageInRequest = buildSketchData(valueSet, &core.ContainerCPUUsageRequest, containerCPUUsageRequestAggregations)
	sketch.LoadAverage = buildSketchData(valueSet, &core.ContainerCPULoadAverage10s, containerCPULoadAvgAggregations)
	if sketch.UsageInLimit != nil || sketch.UsageInRequest != nil || sketch.LoadAverage != nil {
		sketch.Time = metav1.NewTime(valueSet.Timestamp)
		return &sketch
	}
	return nil
}
