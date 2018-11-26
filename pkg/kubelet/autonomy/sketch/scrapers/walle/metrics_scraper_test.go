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
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
	. "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/scrapers/walle/testing"
)

func setupMockAPIForQuery(controller *gomock.Controller, group *core.MetricGroup, results []model.Value) *MockAPI {
	mockAPI := NewMockAPI(controller)
	var pre *gomock.Call
	for i, v := range group.Metrics {
		call := mockAPI.EXPECT().Query(gomock.Any(), v.Expr, gomock.Any()).Return(results[i], error(nil))
		if pre != nil {
			call.After(pre)
		}
		pre = call
	}
	return mockAPI
}

func setupFailedMockAPIForQuery(controller *gomock.Controller, agr0, arg1, arg2 interface{}, err error) *MockAPI {
	mockAPI := NewMockAPI(controller)
	mockAPI.EXPECT().Query(agr0, arg1, arg2).Return(nil, err)
	return mockAPI
}

func Test_metricsScraper_Scrape(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	now := time.Now()
	start := now.Truncate(60 * time.Second)
	end := start.Add(60 * time.Second)

	tests := []struct {
		name   string
		group  core.MetricGroup
		values []model.Value
		start  time.Time
		end    time.Time
		want   *core.DataBatch
	}{
		{
			name:  "normal_with_vector",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.ContainerMetricType,
				Metrics: []core.Metric{
					{
						Name: "go_goroutines",
						Expr: "go_goroutines",
					},
				},
			},
			values: []model.Value{
				model.Vector{
					&model.Sample{
						Metric: model.Metric{
							"__name__":            "go_goroutines",
							core.ContainerIDLabel: "test-container",
						},
						Value:     model.SampleValue(69),
						Timestamp: model.Now(),
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey("test-container"): &core.MetricValueSet{
						Timestamp: end,
						Values: map[string]core.MetricValue{
							"go_goroutines": core.MetricValue(69),
						},
						CommonLabels: map[string]string{
							core.TypeLabel: core.ContainerMetricType,
						},
						LabeledValues: map[string][]core.LabeledMetricValue{},
					},
				},
			},
		},

		{
			name:  "normal_with_single_vector_with_label",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.ContainerMetricType,
				Metrics: []core.Metric{
					{
						Name:   "go_goroutines",
						Expr:   "go_goroutines",
						Labels: []string{"labela", "labelb"},
					},
				},
			},
			values: []model.Value{
				model.Vector{
					&model.Sample{
						Metric: model.Metric{
							"__name__":            "go_goroutines",
							core.ContainerIDLabel: "test-container",
							"labela":              "value1",
							"labelb":              "value1",
						},
						Value:     model.SampleValue(69),
						Timestamp: model.Now(),
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey("test-container"): &core.MetricValueSet{
						Timestamp: end,
						Values:    map[string]core.MetricValue{},
						CommonLabels: map[string]string{
							core.TypeLabel: core.ContainerMetricType,
						},
						LabeledValues: map[string][]core.LabeledMetricValue{
							"go_goroutines": []core.LabeledMetricValue{
								{
									MetricValue: core.MetricValue(69),
									Labels: map[string]string{
										"labela": "value1",
										"labelb": "value1",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "normal_with_more_vector_with_label",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.ContainerMetricType,
				Metrics: []core.Metric{
					{
						Name:   "go_goroutines",
						Expr:   "go_goroutines",
						Labels: []string{"labela", "labelb"},
					},
				},
			},
			values: []model.Value{
				model.Vector{
					&model.Sample{
						Metric: model.Metric{
							"__name__":            "go_goroutines",
							"labela":              "value1",
							"labelb":              "value1",
							core.ContainerIDLabel: "test-container",
						},
						Value:     model.SampleValue(69),
						Timestamp: model.Now(),
					},
					&model.Sample{
						Metric: model.Metric{
							"__name__":            "go_goroutines",
							"labela":              "value2",
							"labelb":              "value1",
							core.ContainerIDLabel: "test-container",
						},
						Value:     model.SampleValue(59),
						Timestamp: model.Now(),
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey("test-container"): &core.MetricValueSet{
						Timestamp: end,
						Values:    map[string]core.MetricValue{},
						CommonLabels: map[string]string{
							core.TypeLabel: core.ContainerMetricType,
						},
						LabeledValues: map[string][]core.LabeledMetricValue{
							"go_goroutines": []core.LabeledMetricValue{
								{
									MetricValue: core.MetricValue(69),
									Labels: map[string]string{
										"labela": "value1",
										"labelb": "value1",
									},
								},
								{
									MetricValue: core.MetricValue(59),
									Labels: map[string]string{
										"labela": "value2",
										"labelb": "value1",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "normal_with_matrix",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.ContainerMetricType,
				Metrics: []core.Metric{
					{
						Name: "go_goroutines",
						Expr: "go_goroutines",
					},
				},
			},
			values: []model.Value{
				model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{
							"__name__":            "go_goroutines",
							core.ContainerIDLabel: "test-container",
						},
						Values: []model.SamplePair{
							{
								Value:     model.SampleValue(69),
								Timestamp: model.Now(),
							},
						},
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey("test-container"): &core.MetricValueSet{
						Timestamp: end,
						Values: map[string]core.MetricValue{
							"go_goroutines": core.MetricValue(69),
						},
						CommonLabels: map[string]string{
							core.TypeLabel: core.ContainerMetricType,
						},
						LabeledValues: map[string][]core.LabeledMetricValue{},
					},
				},
			},
		},
		{
			name:  "normal_with_more_matrix",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.ContainerMetricType,
				Metrics: []core.Metric{
					{
						Name: "go_goroutines",
						Expr: "go_goroutines",
					},
				},
			},
			values: []model.Value{
				model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{
							"__name__":            "go_goroutines",
							core.ContainerIDLabel: "test-container",
						},
						Values: []model.SamplePair{
							{
								Value:     model.SampleValue(69),
								Timestamp: model.TimeFromUnixNano(start.UnixNano()),
							},
							{
								Value:     model.SampleValue(59),
								Timestamp: model.TimeFromUnixNano(now.UnixNano()),
							},
						},
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey("test-container"): &core.MetricValueSet{
						Timestamp: end,
						Values: map[string]core.MetricValue{
							"go_goroutines": core.MetricValue(59),
						},
						CommonLabels: map[string]string{
							core.TypeLabel: core.ContainerMetricType,
						},
						LabeledValues: map[string][]core.LabeledMetricValue{},
					},
				},
			},
		},
		{
			name:  "normal_with_single_matrix_with_label",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.ContainerMetricType,
				Metrics: []core.Metric{
					{
						Name:   "go_goroutines",
						Expr:   "go_goroutines",
						Labels: []string{"labela", "labelb"},
					},
				},
			},
			values: []model.Value{
				model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{
							"__name__":            "go_goroutines",
							"labela":              "value1",
							"labelb":              "value1",
							core.ContainerIDLabel: "test-container",
						},
						Values: []model.SamplePair{
							{
								Value:     model.SampleValue(69),
								Timestamp: model.Now(),
							},
						},
					},
					&model.SampleStream{
						Metric: model.Metric{
							"__name__":            "go_goroutines",
							"labela":              "value2",
							"labelb":              "value1",
							core.ContainerIDLabel: "test-container",
						},
						Values: []model.SamplePair{
							{
								Value:     model.SampleValue(59),
								Timestamp: model.Now(),
							},
						},
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey("test-container"): &core.MetricValueSet{
						Timestamp: end,
						Values:    map[string]core.MetricValue{},
						CommonLabels: map[string]string{
							core.TypeLabel: core.ContainerMetricType,
						},
						LabeledValues: map[string][]core.LabeledMetricValue{
							"go_goroutines": []core.LabeledMetricValue{
								{
									MetricValue: core.MetricValue(69),
									Labels: map[string]string{
										"labela": "value1",
										"labelb": "value1",
									},
								},
								{
									MetricValue: core.MetricValue(59),
									Labels: map[string]string{
										"labela": "value2",
										"labelb": "value1",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "normal_with_more_matrix_with_label",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.ContainerMetricType,
				Metrics: []core.Metric{
					{
						Name:   "go_goroutines",
						Expr:   "go_goroutines",
						Labels: []string{"labela", "labelb"},
					},
				},
			},
			values: []model.Value{
				model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{
							"__name__":            "go_goroutines",
							"labela":              "value1",
							"labelb":              "value1",
							core.ContainerIDLabel: "test-container",
						},
						Values: []model.SamplePair{
							{
								Value:     model.SampleValue(69),
								Timestamp: model.TimeFromUnixNano(start.UnixNano()),
							},
							{
								Value:     model.SampleValue(59),
								Timestamp: model.TimeFromUnixNano(now.UnixNano()),
							},
						},
					},
					&model.SampleStream{
						Metric: model.Metric{
							"__name__":            "go_goroutines",
							"labela":              "value2",
							"labelb":              "value1",
							core.ContainerIDLabel: "test-container",
						},
						Values: []model.SamplePair{
							{
								Value:     model.SampleValue(79),
								Timestamp: model.TimeFromUnixNano(now.UnixNano()),
							},
							{
								Value:     model.SampleValue(59),
								Timestamp: model.TimeFromUnixNano(start.UnixNano()),
							},
						},
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey("test-container"): &core.MetricValueSet{
						Timestamp: end,
						Values:    map[string]core.MetricValue{},
						CommonLabels: map[string]string{
							core.TypeLabel: core.ContainerMetricType,
						},
						LabeledValues: map[string][]core.LabeledMetricValue{
							"go_goroutines": []core.LabeledMetricValue{
								{
									MetricValue: core.MetricValue(59),
									Labels: map[string]string{
										"labela": "value1",
										"labelb": "value1",
									},
								},
								{
									MetricValue: core.MetricValue(79),
									Labels: map[string]string{
										"labela": "value2",
										"labelb": "value1",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMetricsScraper(
				setupMockAPIForQuery(controller, &tt.group, tt.values),
				tt.group)

			got := ms.Scrape(context.Background(), tt.end)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("metricsScraper.Scrape() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_metricsScraper_Scrape_Exception(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	now := time.Now()
	start := now.Truncate(60 * time.Second)
	end := start.Add(60 * time.Second)

	tests := []struct {
		name    string
		group   core.MetricGroup
		values  []model.Value
		start   time.Time
		end     time.Time
		api     v1.API
		want    *core.DataBatch
		wantErr bool
	}{
		{
			name:  "server_failed",
			start: start,
			end:   end,
			api:   setupFailedMockAPIForQuery(controller, gomock.Any(), gomock.Any(), gomock.Any(), errors.New("server crashed")),
			group: core.MetricGroup{
				Metrics: []core.Metric{
					{
						Name: "go_goroutines",
						Expr: "go_goroutines",
					},
				},
			},
			want: &core.DataBatch{
				Timestamp:       end,
				MetricValueSets: map[string]*core.MetricValueSet{},
			},
		},
		{
			name:  "vector_with_empty_key_generator",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Metrics: []core.Metric{
					{
						Name: "go_goroutines",
						Expr: "go_goroutines",
					},
				},
			},
			values: []model.Value{
				model.Vector{
					&model.Sample{
						Metric: model.Metric{
							model.MetricNameLabel: "go_goroutines",
						},
					},
				},
			},
			want: &core.DataBatch{
				Timestamp:       end,
				MetricValueSets: map[string]*core.MetricValueSet{},
			},
		},
		{
			name:  "maxtrix_with_empty_metric_type",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Metrics: []core.Metric{
					{
						Name: "go_goroutines",
						Expr: "go_goroutines",
					},
				},
			},
			values: []model.Value{
				model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{
							model.MetricNameLabel: "go_goroutines",
						},
						Values: []model.SamplePair{
							{
								Timestamp: model.Now(),
								Value:     model.SampleValue(1),
							},
						},
					},
				},
			},
			want: &core.DataBatch{
				Timestamp:       end,
				MetricValueSets: map[string]*core.MetricValueSet{},
			},
		},
		{
			name:  "vector_with_empty",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Metrics: []core.Metric{
					{
						Name: "go_goroutines",
						Expr: "go_goroutines",
					},
				},
			},
			values: []model.Value{
				model.Vector{},
			},
			want: &core.DataBatch{
				Timestamp:       end,
				MetricValueSets: map[string]*core.MetricValueSet{},
			},
		},
		{
			name:  "vector_with_empty_metric",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Metrics: []core.Metric{
					{
						Name: "go_goroutines",
						Expr: "go_goroutines",
					},
				},
			},
			values: []model.Value{
				model.Vector{
					&model.Sample{
						Metric: model.Metric{},
					},
				},
			},
			want: &core.DataBatch{
				Timestamp:       end,
				MetricValueSets: map[string]*core.MetricValueSet{},
			},
		},
		{
			name:  "vector_with_empty_expr",
			start: start,
			end:   end,
			api:   NewMockAPI(controller),
			group: core.MetricGroup{
				Metrics: []core.Metric{
					{
						Name: "go_goroutines",
						Expr: "",
					},
				},
			},
			values: []model.Value{
				model.Vector{},
			},
			want: &core.DataBatch{
				Timestamp:       end,
				MetricValueSets: map[string]*core.MetricValueSet{},
			},
		},
		{
			name:  "matrix_with_empty",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Metrics: []core.Metric{
					{
						Name: "go_goroutines",
						Expr: "go_goroutines",
					},
				},
			},
			values: []model.Value{
				model.Matrix{},
			},
			want: &core.DataBatch{
				Timestamp:       end,
				MetricValueSets: map[string]*core.MetricValueSet{},
			},
		},
		{
			name:  "matrix_with_empty_pair",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Metrics: []core.Metric{
					{
						Name: "go_goroutines",
						Expr: "go_goroutines",
					},
				},
			},
			values: []model.Value{
				model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{
							"__name__": "go_goroutines",
						},
					},
				},
			},
			want: &core.DataBatch{
				Timestamp:       end,
				MetricValueSets: map[string]*core.MetricValueSet{},
			},
		},
		{
			name:  "matrix_with_same_pairs_with_same_timestamp",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.ContainerMetricType,
				Metrics: []core.Metric{
					{
						Name: "go_goroutines",
						Expr: "go_goroutines",
					},
				},
			},
			values: []model.Value{
				model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{
							"__name__":            "go_goroutines",
							core.ContainerIDLabel: "test-container",
						},
						Values: []model.SamplePair{
							{
								Timestamp: model.TimeFromUnixNano(start.UnixNano()),
								Value:     model.SampleValue(1),
							},
							{
								Timestamp: model.TimeFromUnixNano(start.UnixNano()),
								Value:     model.SampleValue(2),
							},
						},
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey("test-container"): &core.MetricValueSet{
						Timestamp: end,
						CommonLabels: map[string]string{
							core.TypeLabel: core.ContainerMetricType,
						},
						Values: map[string]core.MetricValue{
							"go_goroutines": core.MetricValue(1),
						},
						LabeledValues: map[string][]core.LabeledMetricValue{},
					},
				},
			},
		},
		{
			name:  "with_more_vector_with_partial_label",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.ContainerMetricType,
				Metrics: []core.Metric{
					{
						Name:   "go_goroutines",
						Expr:   "go_goroutines",
						Labels: []string{"labela", "labelb"},
					},
				},
			},
			values: []model.Value{
				model.Vector{
					&model.Sample{
						Metric: model.Metric{
							"__name__":            "go_goroutines",
							"labela":              "value1",
							"labelb":              "value1",
							core.ContainerIDLabel: "test-container",
						},
						Value:     model.SampleValue(69),
						Timestamp: model.Now(),
					},
					&model.Sample{
						Metric: model.Metric{
							"__name__": "go_goroutines",
							"labela":   "value2",
							"labelb":   "value1",
						},
						Value:     model.SampleValue(59),
						Timestamp: model.Now(),
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey("test-container"): &core.MetricValueSet{
						Timestamp: end,
						Values:    map[string]core.MetricValue{},
						CommonLabels: map[string]string{
							core.TypeLabel: core.ContainerMetricType,
						},
						LabeledValues: map[string][]core.LabeledMetricValue{
							"go_goroutines": []core.LabeledMetricValue{
								{
									MetricValue: core.MetricValue(69),
									Labels: map[string]string{
										"labela": "value1",
										"labelb": "value1",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.api == nil {
				tt.api = setupMockAPIForQuery(controller, &tt.group, tt.values)
			}
			ms := NewMetricsScraper(tt.api, tt.group)

			got := ms.Scrape(context.Background(), tt.end)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("metricsScraper.Scrape() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_metricsScraper_Scrape_Group_CommonLables(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	now := time.Now()
	start := now.Truncate(60 * time.Second)
	end := start.Add(60 * time.Second)

	tests := []struct {
		name    string
		group   core.MetricGroup
		values  []model.Value
		start   time.Time
		end     time.Time
		want    *core.DataBatch
		wantErr bool
	}{
		{
			name:  "with_common_labels_for_container",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.ContainerMetricType,
				Metrics: []core.Metric{
					{
						Name: "container_go_goroutines",
						Expr: "go_goroutines",
					},
				},
				Labels: []string{
					core.ContainerIDLabel,
					core.ImageLabel,
					core.PodNameLabel,
					core.NamespaceLabel,
				},
			},
			values: []model.Value{
				model.Vector{
					&model.Sample{
						Metric: model.Metric{
							model.MetricNameLabel: "container_go_goroutines",
							core.ContainerIDLabel: "test-container-1",
							core.ImageLabel:       "test-image-1",
							core.PodNameLabel:     "test-pod-1",
							core.NamespaceLabel:   "test-namespace-1",
						},
						Value:     model.SampleValue(69),
						Timestamp: model.Now(),
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey("test-container-1"): &core.MetricValueSet{
						Timestamp: end,
						Values: map[string]core.MetricValue{
							"container_go_goroutines": core.MetricValue(69),
						},
						CommonLabels: map[string]string{
							core.ContainerIDLabel: "test-container-1",
							core.ImageLabel:       "test-image-1",
							core.PodNameLabel:     "test-pod-1",
							core.NamespaceLabel:   "test-namespace-1",
							core.TypeLabel:        core.ContainerMetricType,
						},
						LabeledValues: map[string][]core.LabeledMetricValue{},
					},
				},
			},
		},
		{
			name:  "with_common_labels_for_node",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.NodeMetricType,
				Metrics: []core.Metric{
					{
						Name: "node_go_goroutines",
						Expr: "go_goroutines",
					},
				},
				Labels: []string{
					"hostname",
				},
			},
			values: []model.Value{
				model.Vector{
					&model.Sample{
						Metric: model.Metric{
							model.MetricNameLabel: "node_go_goroutines",
							"hostname":            "test",
						},
						Value:     model.SampleValue(69),
						Timestamp: model.Now(),
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.NodeKey(): &core.MetricValueSet{
						Timestamp: end,
						Values: map[string]core.MetricValue{
							"node_go_goroutines": core.MetricValue(69),
						},
						CommonLabels: map[string]string{
							"hostname":     "test",
							core.TypeLabel: core.NodeMetricType,
						},
						LabeledValues: map[string][]core.LabeledMetricValue{},
					},
				},
			},
		},
		{
			name:  "merge_common_labels_for_container",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.ContainerMetricType,
				Metrics: []core.Metric{
					{
						Name: "container_go_goroutines",
						Expr: "go_goroutines",
					},
				},
				Labels: []string{
					core.ContainerIDLabel,
					core.ImageLabel,
					core.PodNameLabel,
					core.NamespaceLabel,
				},
			},
			values: []model.Value{
				model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{
							model.MetricNameLabel: "container_go_goroutines",
							core.ContainerIDLabel: "test-container-1",
							core.ImageLabel:       "test-image-1",
						},
						Values: []model.SamplePair{
							{
								Value:     model.SampleValue(69),
								Timestamp: model.Now(),
							},
						},
					},
					&model.SampleStream{
						Metric: model.Metric{
							model.MetricNameLabel: "container_go_goroutines",
							core.ContainerIDLabel: "test-container-1",
							core.ImageLabel:       "test-image-1",
							core.PodNameLabel:     "test-pod-1",
							core.NamespaceLabel:   "test-namespace-1",
						},
						Values: []model.SamplePair{
							{
								Value:     model.SampleValue(69),
								Timestamp: model.Now(),
							},
						},
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey("test-container-1"): &core.MetricValueSet{
						Timestamp: end,
						Values: map[string]core.MetricValue{
							"container_go_goroutines": core.MetricValue(69),
						},
						CommonLabels: map[string]string{
							core.ContainerIDLabel: "test-container-1",
							core.ImageLabel:       "test-image-1",
							core.PodNameLabel:     "test-pod-1",
							core.NamespaceLabel:   "test-namespace-1",
							core.TypeLabel:        core.ContainerMetricType,
						},
						LabeledValues: map[string][]core.LabeledMetricValue{},
					},
				},
			},
		},
		{
			name:  "merge_common_labels_for_pod",
			start: start,
			end:   end,
			group: core.MetricGroup{
				Type: core.PodMetricType,
				Metrics: []core.Metric{
					{
						Name: "pod_go_goroutines",
						Expr: "go_goroutines",
					},
				},
				Labels: []string{
					core.ContainerIDLabel,
					core.ImageLabel,
					core.PodNameLabel,
					core.NamespaceLabel,
				},
			},
			values: []model.Value{
				model.Matrix{
					&model.SampleStream{
						Metric: model.Metric{
							model.MetricNameLabel: "pod_go_goroutines",
							core.ContainerIDLabel: "test-container-1",
							core.ImageLabel:       "test-image-1",
						},
						Values: []model.SamplePair{
							{
								Value:     model.SampleValue(69),
								Timestamp: model.Now(),
							},
						},
					},
					&model.SampleStream{
						Metric: model.Metric{
							model.MetricNameLabel: "pod_go_goroutines",
							core.ContainerIDLabel: "test-container-1",
							core.ImageLabel:       "test-image-1",
							core.PodNameLabel:     "test-pod-1",
							core.NamespaceLabel:   "test-namespace-1",
						},
						Values: []model.SamplePair{
							{
								Value:     model.SampleValue(69),
								Timestamp: model.Now(),
							},
						},
					},
				},
			},
			want: &core.DataBatch{
				Timestamp: end,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.PodKey("test-namespace-1", "test-pod-1"): &core.MetricValueSet{
						Timestamp: end,
						Values: map[string]core.MetricValue{
							"pod_go_goroutines": core.MetricValue(69),
						},
						CommonLabels: map[string]string{
							core.ContainerIDLabel: "test-container-1",
							core.ImageLabel:       "test-image-1",
							core.PodNameLabel:     "test-pod-1",
							core.NamespaceLabel:   "test-namespace-1",
							core.TypeLabel:        core.PodMetricType,
						},
						LabeledValues: map[string][]core.LabeledMetricValue{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMetricsScraper(
				setupMockAPIForQuery(controller, &tt.group, tt.values),
				tt.group)

			got := ms.Scrape(context.Background(), tt.end)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("metricsScraper.Scrape() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestWalleName(t *testing.T) {
	// just for cover Name method
	s := &metricsScraper{}
	_ = s.Name()
	t.Skip()
}
