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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sketchapi "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/api/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
)

func Test_buildMemorySketch(t *testing.T) {
	timestamp := time.Now()

	tests := []struct {
		name     string
		valueSet *core.MetricValueSet
		metrics  map[string]core.Metric
		expect   *sketchapi.MemorySketch
		want     bool
	}{
		{
			name: "miss-workingset-bytes",
			valueSet: &core.MetricValueSet{
				Timestamp: timestamp,
				Values: map[string]core.MetricValue{
					core.NodeMemoryAvailableBytes.Name: 10 * 1024 * 1024 * 1024,
					core.NodeMemoryUsedBytes.Name:      100 * 1024,
				},
			},
			metrics: map[string]core.Metric{
				availableBytes: core.NodeMemoryAvailableBytes,
				usageBytes:     core.NodeMemoryUsedBytes,
			},
			want: true,
			expect: &sketchapi.MemorySketch{
				Time:           metav1.NewTime(timestamp),
				AvailableBytes: 10 * 1024 * 1024 * 1024,
				UsageBytes:     100 * 1024,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sketch := &sketchapi.MemorySketch{}
			if got := buildMemorySketch(sketch, tt.valueSet, tt.metrics); got != tt.want && !reflect.DeepEqual(tt.expect, sketch) {
				t.Errorf("buildMemorySketch() = %v, want %v", got, tt.want)
			}
		})
	}
}
