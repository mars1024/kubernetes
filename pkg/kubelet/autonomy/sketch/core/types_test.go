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

package core

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetricValueSet_Merge(t *testing.T) {
	timestamp := time.Now()

	tables := map[string]struct {
		s      *MetricValueSet
		ss     *MetricValueSet
		expect *MetricValueSet
	}{
		"merge-empty": {
			s: &MetricValueSet{
				Timestamp: timestamp,
				CommonLabels: map[string]string{
					"xxx": "value1",
				},
				LabeledValues: map[string][]LabeledMetricValue{
					"test-label-1": []LabeledMetricValue{
						{
							MetricValue: 1.0,
							Labels: map[string]string{
								"l1": "v1",
							},
						},
					},
				},
				Values: map[string]MetricValue{
					"test-value-1": 123,
				},
			},
			ss: NewMetricValueSet(),
			expect: &MetricValueSet{
				Timestamp: timestamp,
				CommonLabels: map[string]string{
					"xxx": "value1",
				},
				LabeledValues: map[string][]LabeledMetricValue{
					"test-label-1": []LabeledMetricValue{
						{
							MetricValue: 1.0,
							Labels: map[string]string{
								"l1": "v1",
							},
						},
					},
				},
				Values: map[string]MetricValue{
					"test-value-1": 123,
				},
			},
		},
		"merge": {
			s: &MetricValueSet{
				Timestamp: timestamp,
				CommonLabels: map[string]string{
					"xxx": "value1",
				},
				LabeledValues: map[string][]LabeledMetricValue{
					"test-label-1": []LabeledMetricValue{
						{
							MetricValue: 1.0,
							Labels: map[string]string{
								"l1": "v1",
							},
						},
					},
				},
				Values: map[string]MetricValue{
					"test-value-1": 123,
				},
			},
			ss: &MetricValueSet{
				CommonLabels: map[string]string{
					"yyy": "value1",
				},
				LabeledValues: map[string][]LabeledMetricValue{
					"test-label-2": []LabeledMetricValue{
						{
							MetricValue: 2.0,
							Labels: map[string]string{
								"l2": "v2",
							},
						},
					},
				},
				Values: map[string]MetricValue{
					"test-value-2": 456,
				},
			},
			expect: &MetricValueSet{
				Timestamp: timestamp,
				CommonLabels: map[string]string{
					"xxx": "value1",
					"yyy": "value1",
				},
				LabeledValues: map[string][]LabeledMetricValue{
					"test-label-1": []LabeledMetricValue{
						{
							MetricValue: 1.0,
							Labels: map[string]string{
								"l1": "v1",
							},
						},
					},
					"test-label-2": []LabeledMetricValue{
						{
							MetricValue: 2.0,
							Labels: map[string]string{
								"l2": "v2",
							},
						},
					},
				},
				Values: map[string]MetricValue{
					"test-value-1": 123,
					"test-value-2": 456,
				},
			},
		},
		"merge-exist-labels": {
			s: &MetricValueSet{
				Timestamp: timestamp,
				CommonLabels: map[string]string{
					"xxx": "value1",
				},
			},
			ss: &MetricValueSet{
				CommonLabels: map[string]string{
					"xxx": "value2",
				},
			},
			expect: &MetricValueSet{
				Timestamp: timestamp,
				CommonLabels: map[string]string{
					"xxx": "value2",
				},
			},
		},
		"merge-exist-values": {
			s: &MetricValueSet{
				Timestamp: timestamp,
				Values: map[string]MetricValue{
					"test-value-1": 123,
				},
			},
			ss: &MetricValueSet{
				Values: map[string]MetricValue{
					"test-value-1": 456,
				},
			},
			expect: &MetricValueSet{
				Timestamp: timestamp,
				Values: map[string]MetricValue{
					"test-value-1": 456,
				},
			},
		},
		"merge-exist-label-value": {
			s: &MetricValueSet{
				Timestamp: timestamp,
				LabeledValues: map[string][]LabeledMetricValue{
					"test-label-1": []LabeledMetricValue{
						{
							MetricValue: 1.0,
							Labels: map[string]string{
								"l1": "v1",
							},
						},
					},
				},
			},
			ss: &MetricValueSet{
				LabeledValues: map[string][]LabeledMetricValue{
					"test-label-1": []LabeledMetricValue{
						{
							MetricValue: 2.0,
							Labels: map[string]string{
								"l1": "v2",
							},
						},
					},
				},
			},
			expect: &MetricValueSet{
				Timestamp: timestamp,
				LabeledValues: map[string][]LabeledMetricValue{
					"test-label-1": []LabeledMetricValue{
						{
							MetricValue: 1.0,
							Labels: map[string]string{
								"l1": "v1",
							},
						},
						{
							MetricValue: 2.0,
							Labels: map[string]string{
								"l1": "v2",
							},
						},
					},
				},
			},
		},
	}

	for name, tt := range tables {
		t.Run(name, func(t *testing.T) {
			tt.s.Merge(tt.ss)
			if !reflect.DeepEqual(tt.s, tt.expect) {
				t.Errorf("merge = %#v, want %#v", tt.s, tt.expect)
			}
		})
	}
}

func TestMetric_HasLabel(t *testing.T) {
	m := Metric{
		Labels: []string{"1", "2"},
	}
	assert.True(t, m.HasLabel())

	m = Metric{}
	assert.False(t, m.HasLabel())
}

func TestMetricGroup_HasLabel(t *testing.T) {
	g := MetricGroup{
		Labels: []string{"id"},
	}
	assert.True(t, g.HasLabel())

	g = MetricGroup{}
	assert.False(t, g.HasLabel())
}

func TestContainerKey(t *testing.T) {
	// just for cover
	assert.Equal(t, "container:123", ContainerKey("123"))
}

func TestPodKey(t *testing.T) {
	// just for cover
	assert.Equal(t, "namespace:n/pod:p", PodKey("n", "p"))
}

func TestNodeKey(t *testing.T) {
	// just for cover
	assert.Equal(t, "node", NodeKey())
}
