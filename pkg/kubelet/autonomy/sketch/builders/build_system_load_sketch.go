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
	systemLoadMin1Aggregations = aggregationMatrix{
		"min1": {
			"max":     core.NodeLoad1mMaxOver1min,
			"min":     core.NodeLoad1mMinOver1min,
			"avg":     core.NodeLoad1mAvgOver1min,
			"p99":     core.NodeLoad1mP99Over1min,
			"predict": core.NodeLoad1mPredict1min,
		},
		"min5": {
			"max":     core.NodeLoad1mMaxOver5min,
			"min":     core.NodeLoad1mMinOver5min,
			"avg":     core.NodeLoad1mAvgOver5min,
			"p99":     core.NodeLoad1mP99Over5min,
			"predict": core.NodeLoad1mPredict5min,
		},
		"min15": {
			"max":     core.NodeLoad1mMaxOver15min,
			"min":     core.NodeLoad1mMinOver15min,
			"avg":     core.NodeLoad1mAvgOver15min,
			"p99":     core.NodeLoad1mP99Over15min,
			"predict": core.NodeLoad1mPredict15min,
		},
	}

	systemLoadMin5Aggregations = aggregationMatrix{
		"min1": {
			"max":     core.NodeLoad5mMaxOver1min,
			"min":     core.NodeLoad5mMinOver1min,
			"avg":     core.NodeLoad5mAvgOver1min,
			"p99":     core.NodeLoad5mP99Over1min,
			"predict": core.NodeLoad5mPredict1min,
		},
		"min5": {
			"max":     core.NodeLoad5mMaxOver5min,
			"min":     core.NodeLoad5mMinOver5min,
			"avg":     core.NodeLoad5mAvgOver5min,
			"p99":     core.NodeLoad5mP99Over5min,
			"predict": core.NodeLoad5mPredict5min,
		},
		"min15": {
			"max":     core.NodeLoad5mMaxOver15min,
			"min":     core.NodeLoad5mMinOver15min,
			"avg":     core.NodeLoad5mAvgOver15min,
			"p99":     core.NodeLoad5mP99Over15min,
			"predict": core.NodeLoad5mPredict15min,
		},
	}

	systemLoadMin15Aggregations = aggregationMatrix{
		"min1": {
			"max":     core.NodeLoad15mMaxOver1min,
			"min":     core.NodeLoad15mMinOver1min,
			"avg":     core.NodeLoad15mAvgOver1min,
			"p99":     core.NodeLoad15mP99Over1min,
			"predict": core.NodeLoad15mPredict1min,
		},
		"min5": {
			"max":     core.NodeLoad15mMaxOver5min,
			"min":     core.NodeLoad15mMinOver5min,
			"avg":     core.NodeLoad15mAvgOver5min,
			"p99":     core.NodeLoad15mP99Over5min,
			"predict": core.NodeLoad15mPredict5min,
		},
		"min15": {
			"max":     core.NodeLoad15mMaxOver15min,
			"min":     core.NodeLoad15mMinOver15min,
			"avg":     core.NodeLoad15mAvgOver15min,
			"p99":     core.NodeLoad15mP99Over15min,
			"predict": core.NodeLoad15mPredict15min,
		},
	}
)

func buildSystemLoad(valueSet *core.MetricValueSet) *sketchapi.NodeSystemLoadSketch {
	var sketch sketchapi.NodeSystemLoadSketch
	sketch.Min1 = buildSketchData(valueSet, &core.NodeLoad1m, systemLoadMin1Aggregations)
	sketch.Min5 = buildSketchData(valueSet, &core.NodeLoad5m, systemLoadMin5Aggregations)
	sketch.Min15 = buildSketchData(valueSet, &core.NodeLoad15m, systemLoadMin15Aggregations)
	if sketch.Min1 != nil || sketch.Min5 != nil || sketch.Min15 != nil {
		sketch.Time = metav1.NewTime(valueSet.Timestamp)
		return &sketch
	}
	return nil
}
