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

package walle

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
)

var _ core.MetricsScraper = &metricsScraper{}

type metricsScraper struct {
	api   v1.API
	group *core.MetricGroup
}

// NewMetricsScraper constructs core.MetricsSource
func NewMetricsScraper(api v1.API, group core.MetricGroup) core.MetricsScraper {
	return &metricsScraper{
		api:   api,
		group: &group,
	}
}

func (s *metricsScraper) Name() string { return "walle" }

func (s *metricsScraper) Scrape(ctx context.Context, timestamp time.Time) *core.DataBatch {
	batch := &core.DataBatch{
		Timestamp:       timestamp,
		MetricValueSets: make(map[string]*core.MetricValueSet),
	}

	for _, metric := range s.group.Metrics {
		s.scrapeMetric(ctx, metric, batch)
	}

	return batch
}

func (s *metricsScraper) scrapeMetric(ctx context.Context, metric core.Metric, batch *core.DataBatch) {
	if metric.Expr == "" {
		return
	}

	sampleValue, err := s.api.Query(ctx, metric.Expr, time.Time{})
	if err != nil {
		return
	}

	switch value := sampleValue.(type) {
	case model.Vector:
		parseVector(value, batch, s.group, &metric)
	case model.Matrix:
		parseMatrix(value, batch, s.group, &metric)
	}
}

func parseVector(vector model.Vector, batch *core.DataBatch, group *core.MetricGroup, metric *core.Metric) {
	if len(vector) == 0 {
		return
	}

	for _, sample := range vector {
		parseSample(sample, batch, group, metric)
	}
}

func parseSample(sample *model.Sample, batch *core.DataBatch, group *core.MetricGroup, metric *core.Metric) {
	set := newMetricValueSet(sample.Metric, batch, group, metric)
	if set == nil {
		return
	}

	value := core.MetricValue(sample.Value)

	if metric.HasLabel() {
		var labeledValue core.LabeledMetricValue
		labeledValue.MetricValue = value
		labeledValue.Labels = extractLabel(sample.Metric, metric.Labels)
		set.LabeledValues[metric.Name] = append(set.LabeledValues[metric.Name], labeledValue)
	} else {
		set.Values[metric.Name] = value
	}
}

func parseMatrix(matrix model.Matrix, batch *core.DataBatch, group *core.MetricGroup, metric *core.Metric) {
	if len(matrix) == 0 {
		return
	}

	for _, sampleStream := range matrix {
		parseSampleStream(sampleStream, batch, group, metric)
	}
}

func parseSampleStream(sample *model.SampleStream, batch *core.DataBatch, group *core.MetricGroup, metric *core.Metric) {
	latest := findLatestSamplePair(sample.Values)
	if latest == -1 {
		return
	}
	valuePair := sample.Values[latest]

	set := newMetricValueSet(sample.Metric, batch, group, metric)
	if set == nil {
		return
	}

	value := core.MetricValue(valuePair.Value)

	if metric.HasLabel() {
		var labeledValue core.LabeledMetricValue
		labeledValue.MetricValue = value
		labeledValue.Labels = extractLabel(sample.Metric, metric.Labels)
		set.LabeledValues[metric.Name] = append(set.LabeledValues[metric.Name], labeledValue)
	} else {
		set.Values[metric.Name] = value
	}
}

func newMetricValueSet(sampleMetric model.Metric, batch *core.DataBatch, group *core.MetricGroup, metric *core.Metric) *core.MetricValueSet {
	var key string
	switch group.Type {
	case core.NodeMetricType:
		key = core.NodeKey()
	case core.PodMetricType:
		namespace := sampleMetric[model.LabelName(core.NamespaceLabel)]
		pod := sampleMetric[model.LabelName(core.PodNameLabel)]
		if namespace != "" && pod != "" {
			key = core.PodKey(string(namespace), string(pod))
		}
	case core.ContainerMetricType:
		id := sampleMetric[model.LabelName(core.ContainerIDLabel)]
		if id != "" {
			key = core.ContainerKey(string(id))
		}
	}
	if key == "" {
		return nil
	}

	set := batch.MetricValueSets[key]
	if set == nil {
		set = core.NewMetricValueSet()
		set.Timestamp = batch.Timestamp
		set.CommonLabels[core.TypeLabel] = group.Type
		batch.MetricValueSets[key] = set
	}

	updateCommonLables(set, group, sampleMetric)

	return set
}

func updateCommonLables(set *core.MetricValueSet, group *core.MetricGroup, metric model.Metric) {
	if group.HasLabel() {
		labels := extractLabel(metric, group.Labels)
		set.CommonLabels = mergeLabel(set.CommonLabels, labels)
	}
}

func extractLabel(metric model.Metric, labels []string) map[string]string {
	var m map[string]string
	for _, label := range labels {
		if value, ok := metric[model.LabelName(label)]; ok {
			if m == nil {
				m = make(map[string]string)
			}
			m[label] = string(value)
		}
	}

	return m
}

func mergeLabel(r map[string]string, l map[string]string) map[string]string {
	for k1, v1 := range l {
		if _, ok := r[k1]; !ok {
			r[k1] = v1
		}
	}
	return r
}

func findLatestSamplePair(pairs []model.SamplePair) int {
	if len(pairs) == 0 {
		return -1
	}

	var index int
	var lastTimestamp model.Time
	for i, v := range pairs {
		if v.Timestamp > lastTimestamp {
			index, lastTimestamp = i, v.Timestamp
		}
	}

	return index
}
