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

	sketchapi "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/api/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
)

// time dimension -> aggregate method -> core.Metric
// e.g: min1 -> max -> core.ContainerCPUUsageLimitMaxOver1min
type aggregationMatrix map[string]map[string]core.Metric

type sketchDataBuilder struct {
	valueSet *core.MetricValueSet
	factory  sketchDataFactory
	finder   metricValueFinder
}

func (b *sketchDataBuilder) build(metric *core.Metric, matrix aggregationMatrix) {
	if value, ok := b.valueSet.Values[metric.Name]; ok {
		data := b.factory.newSketchData(metric, &value)
		if data == nil {
			return
		}
		data.Latest = float64(value)
		b.updateSketchData(data, matrix, &value)

	} else if labeledValues, ok := b.valueSet.LabeledValues[metric.Name]; ok {
		for _, v := range labeledValues {
			v := v
			data := b.factory.newSketchData(metric, &v)
			if data == nil {
				continue
			}

			data.Latest = float64(v.MetricValue)
			b.updateSketchData(data, matrix, &v)
		}
	}
}

func (b *sketchDataBuilder) updateSketchData(
	data *sketchapi.SketchData,
	matrix aggregationMatrix, param interface{}) {

	if b.finder == nil {
		return
	}

	tables := map[string]*sketchapi.SketchCumulation{
		"min1":  &data.Min1,
		"min5":  &data.Min5,
		"min15": &data.Min15,
	}

	for k, v := range tables {
		aggregationObjects, ok := matrix[k]
		if !ok {
			continue
		}
		b.buildSketchCumulation(v, aggregationObjects, param)
	}
}

func (b *sketchDataBuilder) buildSketchCumulation(
	s *sketchapi.SketchCumulation,
	aggregationObjects map[string]core.Metric, param interface{}) {

	tables := map[string]*float64{
		"max":     &s.Max,
		"min":     &s.Min,
		"avg":     &s.Avg,
		"p99":     &s.P99,
		"predict": &s.Predict,
	}

	for k, v := range tables {
		metric, ok := aggregationObjects[k]
		if !ok {
			continue
		}

		value, ok := b.finder.findMetricValue(b.valueSet, &metric, param)
		if !ok {
			continue
		}
		*v = float64(value)
	}
}

type metricValueFinder interface {
	findMetricValue(valueSet *core.MetricValueSet, metric *core.Metric, param interface{}) (core.MetricValue, bool)
}

type metricValueFinderFunc func(valueSet *core.MetricValueSet, metric *core.Metric, param interface{}) (core.MetricValue, bool)

func (f metricValueFinderFunc) findMetricValue(valueSet *core.MetricValueSet, metric *core.Metric, param interface{}) (core.MetricValue, bool) {
	return f(valueSet, metric, param)
}

type sketchDataFactory interface {
	newSketchData(metric *core.Metric, value interface{}) *sketchapi.SketchData
}

type sketchDataFactoryFunc func(metric *core.Metric, value interface{}) *sketchapi.SketchData

func (f sketchDataFactoryFunc) newSketchData(metric *core.Metric, value interface{}) *sketchapi.SketchData {
	return f(metric, value)
}

func buildSketchData(
	valueSet *core.MetricValueSet, metric *core.Metric,
	matrix aggregationMatrix) *sketchapi.SketchData {

	var data sketchapi.SketchData

	builder := sketchDataBuilder{
		valueSet: valueSet,
		factory: sketchDataFactoryFunc(
			func(metric *core.Metric, _ interface{}) *sketchapi.SketchData {
				return &data
			}),
		finder: metricValueFinderFunc(
			func(valueSet *core.MetricValueSet, metric *core.Metric, _ interface{}) (core.MetricValue, bool) {
				value, ok := valueSet.Values[metric.Name]
				return value, ok
			}),
	}
	builder.build(metric, matrix)

	var empty sketchapi.SketchData
	if reflect.DeepEqual(&empty, &data) {
		return nil
	}

	return &data
}
