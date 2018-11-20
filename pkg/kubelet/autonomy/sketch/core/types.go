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
	"context"
	"fmt"
	"time"
)

// MetricGroup defines generality metrics, scaper and interval
type MetricGroup struct {
	Name     string
	Help     string
	Type     string
	Scraper  string
	Interval time.Duration
	Labels   []string
	Metrics  []Metric
}

// Metric represents a specified metric
type Metric struct {
	Name   string
	Help   string
	Expr   string
	Labels []string
}

// MetricValue represents a specific value for Metric
type MetricValue float64

// LabeledMetricValue is a special MetricValue but have labels
type LabeledMetricValue struct {
	MetricValue
	Labels map[string]string
}

// MetricValueSet represents a batch of MetricValue
type MetricValueSet struct {
	Timestamp     time.Time
	CommonLabels  map[string]string
	Values        map[string]MetricValue
	LabeledValues map[string][]LabeledMetricValue
}

// HasLabel determine Metric has labels
func (m *Metric) HasLabel() bool { return len(m.Labels) > 0 }

// HasLabel returns true if have labels
func (g *MetricGroup) HasLabel() bool { return len(g.Labels) > 0 }

// NewMetricValueSet constructs MetricValueSet instance
func NewMetricValueSet() *MetricValueSet {
	return &MetricValueSet{
		CommonLabels:  make(map[string]string),
		Values:        make(map[string]MetricValue),
		LabeledValues: make(map[string][]LabeledMetricValue),
	}
}

// Merge merges from other MetricValueSet into current MetricValueSet
func (s *MetricValueSet) Merge(ss *MetricValueSet) {
	for k, v := range ss.CommonLabels {
		s.CommonLabels[k] = v
	}
	for k, v := range ss.LabeledValues {
		s.LabeledValues[k] = append(s.LabeledValues[k], v...)
	}
	for k, v := range ss.Values {
		s.Values[k] = v
	}
}

// MetricsScraper is a metric scaper interface that is used to scrape metrics
type MetricsScraper interface {
	Name() string
	Scrape(ctx context.Context, timestamp time.Time) *DataBatch
}

// MetricsScraperProvider provides a list of scrapers
type MetricsScraperProvider interface {
	GetMetricsScrapers() []MetricsScraper
}

// DataProcessor is a interface that aggregates/filter the DataBatch
type DataProcessor interface {
	Name() string
	Process(*DataBatch) (*DataBatch, error)
}

// DataBatch represents a set of MetricValueSet
type DataBatch struct {
	Timestamp       time.Time
	MetricValueSets map[string]*MetricValueSet
}

// ContainerKey returns the key that is used to index MetricValueSet
func ContainerKey(id string) string { return "container:" + id }

// PodKey returns the key that is used to index MetricValueSet
func PodKey(namespace, name string) string {
	return fmt.Sprintf("namespace:%s/pod:%s", namespace, name)
}

// NodeKey returns the key that is used to index MetricValueSet
func NodeKey() string { return "node" }
