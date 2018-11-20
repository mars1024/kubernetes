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
	"testing"

	sketchapi "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/api/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
)

type fakeSkechDataFactory struct {
	data map[string]*sketchapi.SketchData
}

func (f *fakeSkechDataFactory) newSketchData(metric *core.Metric, value interface{}) *sketchapi.SketchData {
	lv := value.(*core.LabeledMetricValue)
	id := lv.Labels["id"]
	s := f.data[id]
	if s == nil {
		s = &sketchapi.SketchData{}
		f.data[id] = s
	}
	return s
}

func fakeFindMetricValue(
	valueSet *core.MetricValueSet,
	metric *core.Metric,
	param interface{}) (core.MetricValue, bool) {

	lv := param.(*core.LabeledMetricValue)
	values, ok := valueSet.LabeledValues[metric.Name]
	if ok {
		for _, v := range values {
			if v.Labels["id"] == lv.Labels["id"] {
				return v.MetricValue, true
			}
		}
	}

	return 0, false
}

func TestLabeldMetricValue(t *testing.T) {
	metric := core.Metric{
		Name: "test-metric",
	}

	maxMetric := core.Metric{
		Name: "test-metric-max",
	}

	ff := &fakeSkechDataFactory{
		data: make(map[string]*sketchapi.SketchData),
	}
	builder := sketchDataBuilder{
		valueSet: &core.MetricValueSet{
			LabeledValues: map[string][]core.LabeledMetricValue{
				metric.Name: []core.LabeledMetricValue{
					{
						MetricValue: 1.1,
						Labels: map[string]string{
							"id": "test-1",
						},
					},
					{
						MetricValue: 1.2,
						Labels: map[string]string{
							"id": "test-2",
						},
					},
					{
						MetricValue: 1.3,
						Labels: map[string]string{
							"id": "test-3",
						},
					},
				},
				maxMetric.Name: []core.LabeledMetricValue{
					{
						MetricValue: 2.1,
						Labels: map[string]string{
							"id": "test-1",
						},
					},
					{
						MetricValue: 2.2,
						Labels: map[string]string{
							"id": "test-2",
						},
					},
				},
			},
		},
		factory: ff,
		finder:  metricValueFinderFunc(fakeFindMetricValue),
	}

	matrix := aggregationMatrix{
		"min1": {
			"max": maxMetric,
		},
	}

	builder.build(&metric, matrix)

	expect := map[string]*sketchapi.SketchData{
		"test-1": &sketchapi.SketchData{
			Latest: 1.1,
			Min1: sketchapi.SketchCumulation{
				Max: 2.1,
			},
		},
		"test-2": &sketchapi.SketchData{
			Latest: 1.2,
			Min1: sketchapi.SketchCumulation{
				Max: 2.2,
			},
		},
		"test-3": &sketchapi.SketchData{
			Latest: 1.3,
		},
	}
	if !reflect.DeepEqual(expect, ff.data) {
		t.Errorf("builder.build = %#v, want %#v", ff.data, expect)
	}
}

func TestLabeldMetricValueWithNoData(t *testing.T) {
	metric := core.Metric{
		Name: "test-metric",
	}

	ff := &fakeSkechDataFactory{}

	builder := sketchDataBuilder{
		valueSet: &core.MetricValueSet{
			LabeledValues: map[string][]core.LabeledMetricValue{
				metric.Name: []core.LabeledMetricValue{
					{
						MetricValue: 1.1,
						Labels: map[string]string{
							"id": "test-1",
						},
					},
				},
			},
		},
		factory: sketchDataFactoryFunc(func(metric *core.Metric, value interface{}) *sketchapi.SketchData {
			return nil
		}),
		finder: nil,
	}

	builder.build(&metric, nil)

	var expect map[string]*sketchapi.SketchData
	if !reflect.DeepEqual(expect, ff.data) {
		t.Errorf("builder.build = %#v, want %#v", ff.data, expect)
	}
}

func TestMetricValueWithNoData(t *testing.T) {
	metric := core.Metric{
		Name: "test-metric",
	}

	builder := sketchDataBuilder{
		valueSet: &core.MetricValueSet{
			Values: map[string]core.MetricValue{
				metric.Name: 1.1,
			},
		},
		factory: sketchDataFactoryFunc(func(metric *core.Metric, value interface{}) *sketchapi.SketchData {
			return nil
		}),
		finder: nil,
	}

	builder.build(&metric, nil)
}

func TestMetricValueWithNoFinder(t *testing.T) {
	metric := core.Metric{
		Name: "test-metric",
	}

	builder := sketchDataBuilder{
		valueSet: &core.MetricValueSet{
			Values: map[string]core.MetricValue{
				metric.Name: 1.1,
			},
		},
		factory: sketchDataFactoryFunc(func(metric *core.Metric, value interface{}) *sketchapi.SketchData {
			return &sketchapi.SketchData{}
		}),
		finder: nil,
	}

	builder.build(&metric, nil)
}
