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

	"github.com/golang/mock/gomock"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	sketchapi "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/api/v1alpha1"
	builderstest "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/builders/testing"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	kubecontainertest "k8s.io/kubernetes/pkg/kubelet/container/testing"
	kubepod "k8s.io/kubernetes/pkg/kubelet/pod"
	kubepodtest "k8s.io/kubernetes/pkg/kubelet/pod/testing"
)

func Test_builderImpl_BuildNodeCPUSketch(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	statsProvider := builderstest.NewMockStatsProvider(controller)
	statsProvider.EXPECT().GetNode().Return(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}, error(nil))

	timestamp := time.Now()

	tests := []struct {
		name          string
		statsProvider StatsProvider
		batch         *core.DataBatch
		want          *sketchapi.SketchSummary
	}{
		{
			name:          "build-node-cpu-sketch",
			statsProvider: statsProvider,
			batch: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.NodeKey(): &core.MetricValueSet{
						Timestamp: timestamp,
						Values: map[string]core.MetricValue{
							core.NodeCPUUsage.Name:             1.1,
							core.NodeCPUUsageAvgOver1min.Name:  1.2,
							core.NodeCPUUsageMaxOver1min.Name:  1.9,
							core.NodeCPUUsageMinOver1min.Name:  1.1,
							core.NodeCPUUsageP99Over1min.Name:  1.5,
							core.NodeCPUUsagePredict1min.Name:  1.6,
							core.NodeCPUUsageAvgOver5min.Name:  2.2,
							core.NodeCPUUsageMaxOver5min.Name:  2.9,
							core.NodeCPUUsageMinOver5min.Name:  2.1,
							core.NodeCPUUsageP99Over5min.Name:  2.5,
							core.NodeCPUUsagePredict5min.Name:  2.6,
							core.NodeCPUUsageAvgOver15min.Name: 3.2,
							core.NodeCPUUsageMaxOver15min.Name: 3.9,
							core.NodeCPUUsageMinOver15min.Name: 3.1,
							core.NodeCPUUsageP99Over15min.Name: 3.5,
							core.NodeCPUUsagePredict15min.Name: 3.6,
						},
					},
				},
			},
			want: &sketchapi.SketchSummary{
				Node: sketchapi.NodeSketch{
					Name: "test",
					CPU: &sketchapi.NodeCPUSketch{
						Time: metav1.NewTime(timestamp),
						Usage: &sketchapi.SketchData{
							Latest: 1.1,
							Min1: sketchapi.SketchCumulation{
								Avg:     1.2,
								Max:     1.9,
								Min:     1.1,
								P99:     1.5,
								Predict: 1.6,
							},
							Min5: sketchapi.SketchCumulation{
								Avg:     2.2,
								Max:     2.9,
								Min:     2.1,
								P99:     2.5,
								Predict: 2.6,
							},
							Min15: sketchapi.SketchCumulation{
								Avg:     3.2,
								Max:     3.9,
								Min:     3.1,
								P99:     3.5,
								Predict: 3.6,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New(tt.statsProvider)
			if got := builder.Build(tt.batch); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("builderImpl.Build() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_builderImpl_BuildNodeLoadSketch(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	statsProvider := builderstest.NewMockStatsProvider(controller)
	statsProvider.EXPECT().GetNode().Return(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}, error(nil))

	timestamp := time.Now()

	tests := []struct {
		name          string
		statsProvider StatsProvider
		batch         *core.DataBatch
		want          *sketchapi.SketchSummary
	}{
		{
			name:          "build-node-load-sketch",
			statsProvider: statsProvider,
			batch: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.NodeKey(): &core.MetricValueSet{
						Timestamp: timestamp,
						Values: map[string]core.MetricValue{
							core.NodeLoad1m.Name:              1.1,
							core.NodeLoad1mAvgOver1min.Name:   1.2,
							core.NodeLoad1mMaxOver1min.Name:   1.9,
							core.NodeLoad1mMinOver1min.Name:   1.1,
							core.NodeLoad1mP99Over1min.Name:   1.5,
							core.NodeLoad1mPredict1min.Name:   1.6,
							core.NodeLoad1mAvgOver5min.Name:   2.2,
							core.NodeLoad1mMaxOver5min.Name:   2.9,
							core.NodeLoad1mMinOver5min.Name:   2.1,
							core.NodeLoad1mP99Over5min.Name:   2.5,
							core.NodeLoad1mPredict5min.Name:   2.6,
							core.NodeLoad1mAvgOver15min.Name:  3.2,
							core.NodeLoad1mMaxOver15min.Name:  3.9,
							core.NodeLoad1mMinOver15min.Name:  3.1,
							core.NodeLoad1mP99Over15min.Name:  3.5,
							core.NodeLoad1mPredict15min.Name:  3.6,
							core.NodeLoad5m.Name:              51.1,
							core.NodeLoad5mAvgOver1min.Name:   51.2,
							core.NodeLoad5mMaxOver1min.Name:   51.9,
							core.NodeLoad5mMinOver1min.Name:   51.1,
							core.NodeLoad5mP99Over1min.Name:   51.5,
							core.NodeLoad5mPredict1min.Name:   51.6,
							core.NodeLoad5mAvgOver5min.Name:   52.2,
							core.NodeLoad5mMaxOver5min.Name:   52.9,
							core.NodeLoad5mMinOver5min.Name:   52.1,
							core.NodeLoad5mP99Over5min.Name:   52.5,
							core.NodeLoad5mPredict5min.Name:   52.6,
							core.NodeLoad5mAvgOver15min.Name:  53.2,
							core.NodeLoad5mMaxOver15min.Name:  53.9,
							core.NodeLoad5mMinOver15min.Name:  53.1,
							core.NodeLoad5mP99Over15min.Name:  53.5,
							core.NodeLoad5mPredict15min.Name:  53.6,
							core.NodeLoad15m.Name:             15.1,
							core.NodeLoad15mAvgOver1min.Name:  15.2,
							core.NodeLoad15mMaxOver1min.Name:  15.9,
							core.NodeLoad15mMinOver1min.Name:  15.1,
							core.NodeLoad15mP99Over1min.Name:  15.5,
							core.NodeLoad15mPredict1min.Name:  15.6,
							core.NodeLoad15mAvgOver5min.Name:  25.2,
							core.NodeLoad15mMaxOver5min.Name:  25.9,
							core.NodeLoad15mMinOver5min.Name:  25.1,
							core.NodeLoad15mP99Over5min.Name:  25.5,
							core.NodeLoad15mPredict5min.Name:  25.6,
							core.NodeLoad15mAvgOver15min.Name: 35.2,
							core.NodeLoad15mMaxOver15min.Name: 35.9,
							core.NodeLoad15mMinOver15min.Name: 35.1,
							core.NodeLoad15mP99Over15min.Name: 35.5,
							core.NodeLoad15mPredict15min.Name: 35.6,
						},
					},
				},
			},
			want: &sketchapi.SketchSummary{
				Node: sketchapi.NodeSketch{
					Name: "test",
					Load: &sketchapi.NodeSystemLoadSketch{
						Time: metav1.NewTime(timestamp),
						Min1: &sketchapi.SketchData{
							Latest: 1.1,
							Min1: sketchapi.SketchCumulation{
								Avg:     1.2,
								Max:     1.9,
								Min:     1.1,
								P99:     1.5,
								Predict: 1.6,
							},
							Min5: sketchapi.SketchCumulation{
								Avg:     2.2,
								Max:     2.9,
								Min:     2.1,
								P99:     2.5,
								Predict: 2.6,
							},
							Min15: sketchapi.SketchCumulation{
								Avg:     3.2,
								Max:     3.9,
								Min:     3.1,
								P99:     3.5,
								Predict: 3.6,
							},
						},
						Min5: &sketchapi.SketchData{
							Latest: 51.1,
							Min1: sketchapi.SketchCumulation{
								Avg:     51.2,
								Max:     51.9,
								Min:     51.1,
								P99:     51.5,
								Predict: 51.6,
							},
							Min5: sketchapi.SketchCumulation{
								Avg:     52.2,
								Max:     52.9,
								Min:     52.1,
								P99:     52.5,
								Predict: 52.6,
							},
							Min15: sketchapi.SketchCumulation{
								Avg:     53.2,
								Max:     53.9,
								Min:     53.1,
								P99:     53.5,
								Predict: 53.6,
							},
						},
						Min15: &sketchapi.SketchData{
							Latest: 15.1,
							Min1: sketchapi.SketchCumulation{
								Avg:     15.2,
								Max:     15.9,
								Min:     15.1,
								P99:     15.5,
								Predict: 15.6,
							},
							Min5: sketchapi.SketchCumulation{
								Avg:     25.2,
								Max:     25.9,
								Min:     25.1,
								P99:     25.5,
								Predict: 25.6,
							},
							Min15: sketchapi.SketchCumulation{
								Avg:     35.2,
								Max:     35.9,
								Min:     35.1,
								P99:     35.5,
								Predict: 35.6,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New(tt.statsProvider)
			if got := builder.Build(tt.batch); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("builderImpl.Build() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_builderImpl_BuildNodeMemorySketch(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	podManager := &kubepodtest.MockManager{}
	runtimeCache := &kubecontainertest.MockRuntimeCache{}
	statsProvider := builderstest.NewMockStatsProvider(controller)

	statsProvider.EXPECT().GetNode().AnyTimes().Return(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}, error(nil))

	timestamp := time.Now()

	tests := []struct {
		name          string
		statsProvider StatsProvider
		runtimeCache  kubecontainer.RuntimeCache
		podManager    kubepod.Manager
		batch         *core.DataBatch
		want          *sketchapi.SketchSummary
	}{
		{
			name:          "build-node-memory-sketch",
			statsProvider: statsProvider,
			runtimeCache:  runtimeCache,
			podManager:    podManager,
			batch: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.NodeKey(): &core.MetricValueSet{
						Timestamp: timestamp,
						Values: map[string]core.MetricValue{
							core.NodeMemoryAvailableBytes.Name:  10 * 1024 * 1024 * 1024,
							core.NodeMemoryUsedBytes.Name:       100 * 1024,
							core.NodeMemoryWorkingsetBytes.Name: 200 * 1024,
						},
					},
				},
			},
			want: &sketchapi.SketchSummary{
				Node: sketchapi.NodeSketch{
					Name: "test",
					Memory: &sketchapi.NodeMemorySketch{
						MemorySketch: sketchapi.MemorySketch{
							Time:            metav1.NewTime(timestamp),
							AvailableBytes:  10 * 1024 * 1024 * 1024,
							UsageBytes:      100 * 1024,
							WorkingSetBytes: 200 * 1024,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New(tt.statsProvider)
			if got := builder.Build(tt.batch); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("builderImpl.Build() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_builderImpl_BuildPodSketchWithContainerCPUSketch(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	namespace := "test-namespace"
	podName := "test-pod"
	podUID := "123456"
	containerID := "container-1"
	containerName := "test-container"

	statsProvider := builderstest.NewMockStatsProvider(controller)
	statsProvider.EXPECT().GetPodByName(namespace, podName).AnyTimes().Return(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			UID:       types.UID(podUID),
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:        containerName,
					ContainerID: "docker://" + containerID,
				},
			},
		},
	}, true)

	timestamp := time.Now()

	tests := []struct {
		name          string
		statsProvider StatsProvider
		batch         *core.DataBatch
		want          *sketchapi.SketchSummary
	}{
		{
			name:          "build-container-cpu-sketch",
			statsProvider: statsProvider,
			batch: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey(containerID): &core.MetricValueSet{
						Timestamp: timestamp,
						CommonLabels: map[string]string{
							core.TypeLabel:        core.ContainerMetricType,
							core.NamespaceLabel:   namespace,
							core.PodNameLabel:     podName,
							core.ContainerIDLabel: containerID,
						},
						Values: map[string]core.MetricValue{
							core.ContainerCPUUsageLimit.Name:                 1.1,
							core.ContainerCPUUsageLimitAvgOver1min.Name:      1.2,
							core.ContainerCPUUsageLimitMaxOver1min.Name:      1.9,
							core.ContainerCPUUsageLimitMinOver1min.Name:      1.1,
							core.ContainerCPUUsageLimitP99Over1min.Name:      1.5,
							core.ContainerCPUUsageLimitPredict1min.Name:      1.6,
							core.ContainerCPUUsageLimitAvgOver5min.Name:      2.2,
							core.ContainerCPUUsageLimitMaxOver5min.Name:      2.9,
							core.ContainerCPUUsageLimitMinOver5min.Name:      2.1,
							core.ContainerCPUUsageLimitP99Over5min.Name:      2.5,
							core.ContainerCPUUsageLimitPredict5min.Name:      2.6,
							core.ContainerCPUUsageLimitAvgOver15min.Name:     3.2,
							core.ContainerCPUUsageLimitMaxOver15min.Name:     3.9,
							core.ContainerCPUUsageLimitMinOver15min.Name:     3.1,
							core.ContainerCPUUsageLimitP99Over15min.Name:     3.5,
							core.ContainerCPUUsageLimitPredict15min.Name:     3.6,
							core.ContainerCPUUsageRequest.Name:               51.1,
							core.ContainerCPUUsageRequestAvgOver1min.Name:    51.2,
							core.ContainerCPUUsageRequestMaxOver1min.Name:    51.9,
							core.ContainerCPUUsageRequestMinOver1min.Name:    51.1,
							core.ContainerCPUUsageRequestP99Over1min.Name:    51.5,
							core.ContainerCPUUsageRequestPredict1min.Name:    51.6,
							core.ContainerCPUUsageRequestAvgOver5min.Name:    52.2,
							core.ContainerCPUUsageRequestMaxOver5min.Name:    52.9,
							core.ContainerCPUUsageRequestMinOver5min.Name:    52.1,
							core.ContainerCPUUsageRequestP99Over5min.Name:    52.5,
							core.ContainerCPUUsageRequestPredict5min.Name:    52.6,
							core.ContainerCPUUsageRequestAvgOver15min.Name:   53.2,
							core.ContainerCPUUsageRequestMaxOver15min.Name:   53.9,
							core.ContainerCPUUsageRequestMinOver15min.Name:   53.1,
							core.ContainerCPUUsageRequestP99Over15min.Name:   53.5,
							core.ContainerCPUUsageRequestPredict15min.Name:   53.6,
							core.ContainerCPULoadAverage10s.Name:             15.1,
							core.ContainerCPULoadAverage10sAvgOver1min.Name:  15.2,
							core.ContainerCPULoadAverage10sMaxOver1min.Name:  15.9,
							core.ContainerCPULoadAverage10sMinOver1min.Name:  15.1,
							core.ContainerCPULoadAverage10sP99Over1min.Name:  15.5,
							core.ContainerCPULoadAverage10sPredict1min.Name:  15.6,
							core.ContainerCPULoadAverage10sAvgOver5min.Name:  25.2,
							core.ContainerCPULoadAverage10sMaxOver5min.Name:  25.9,
							core.ContainerCPULoadAverage10sMinOver5min.Name:  25.1,
							core.ContainerCPULoadAverage10sP99Over5min.Name:  25.5,
							core.ContainerCPULoadAverage10sPredict5min.Name:  25.6,
							core.ContainerCPULoadAverage10sAvgOver15min.Name: 35.2,
							core.ContainerCPULoadAverage10sMaxOver15min.Name: 35.9,
							core.ContainerCPULoadAverage10sMinOver15min.Name: 35.1,
							core.ContainerCPULoadAverage10sP99Over15min.Name: 35.5,
							core.ContainerCPULoadAverage10sPredict15min.Name: 35.6,
						},
					},
				},
			},
			want: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Name:      podName,
							Namespace: namespace,
							UID:       podUID,
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: containerName,
								ID:   containerID,
								CPU: &sketchapi.ContainerCPUSketch{
									Time: metav1.NewTime(timestamp),
									UsageInLimit: &sketchapi.SketchData{
										Latest: 1.1,
										Min1: sketchapi.SketchCumulation{
											Avg:     1.2,
											Max:     1.9,
											Min:     1.1,
											P99:     1.5,
											Predict: 1.6,
										},
										Min5: sketchapi.SketchCumulation{
											Avg:     2.2,
											Max:     2.9,
											Min:     2.1,
											P99:     2.5,
											Predict: 2.6,
										},
										Min15: sketchapi.SketchCumulation{
											Avg:     3.2,
											Max:     3.9,
											Min:     3.1,
											P99:     3.5,
											Predict: 3.6,
										},
									},
									UsageInRequest: &sketchapi.SketchData{
										Latest: 51.1,
										Min1: sketchapi.SketchCumulation{
											Avg:     51.2,
											Max:     51.9,
											Min:     51.1,
											P99:     51.5,
											Predict: 51.6,
										},
										Min5: sketchapi.SketchCumulation{
											Avg:     52.2,
											Max:     52.9,
											Min:     52.1,
											P99:     52.5,
											Predict: 52.6,
										},
										Min15: sketchapi.SketchCumulation{
											Avg:     53.2,
											Max:     53.9,
											Min:     53.1,
											P99:     53.5,
											Predict: 53.6,
										},
									},
									LoadAverage: &sketchapi.SketchData{
										Latest: 15.1,
										Min1: sketchapi.SketchCumulation{
											Avg:     15.2,
											Max:     15.9,
											Min:     15.1,
											P99:     15.5,
											Predict: 15.6,
										},
										Min5: sketchapi.SketchCumulation{
											Avg:     25.2,
											Max:     25.9,
											Min:     25.1,
											P99:     25.5,
											Predict: 25.6,
										},
										Min15: sketchapi.SketchCumulation{
											Avg:     35.2,
											Max:     35.9,
											Min:     35.1,
											P99:     35.5,
											Predict: 35.6,
										},
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
			builder := New(tt.statsProvider)
			if got := builder.Build(tt.batch); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("builderImpl.Build() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_builderImpl_BuildPodWithContainerMemorySketch(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	namespace := "test-namespace"
	podName := "test-pod"
	podUID := "123456"
	containerID := "container-1"
	containerName := "test-container"

	statsProvider := builderstest.NewMockStatsProvider(controller)
	statsProvider.EXPECT().GetPodByName(namespace, podName).AnyTimes().Return(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			UID:       types.UID(podUID),
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:        containerName,
					ContainerID: "docker://" + containerID,
				},
			},
		},
	}, true)

	timestamp := time.Now()

	tests := []struct {
		name          string
		statsProvider StatsProvider
		batch         *core.DataBatch
		want          *sketchapi.SketchSummary
	}{
		{
			name:          "build-container-memory-sketch",
			statsProvider: statsProvider,
			batch: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey(containerID): &core.MetricValueSet{
						Timestamp: timestamp,
						CommonLabels: map[string]string{
							core.TypeLabel:        core.ContainerMetricType,
							core.NamespaceLabel:   namespace,
							core.PodNameLabel:     podName,
							core.ContainerIDLabel: containerID,
						},
						Values: map[string]core.MetricValue{
							core.ContainerMemoryAvailableBytes.Name:  10 * 1024 * 1024 * 1024,
							core.ContainerMemoryUsageBytes.Name:      100 * 1024,
							core.ContainerMemoryWorkingSetBytes.Name: 200 * 1024,
						},
					},
				},
			},
			want: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Name:      podName,
							Namespace: namespace,
							UID:       podUID,
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								Name: containerName,
								ID:   containerID,
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										Time:            metav1.NewTime(timestamp),
										AvailableBytes:  10 * 1024 * 1024 * 1024,
										UsageBytes:      100 * 1024,
										WorkingSetBytes: 200 * 1024,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:          "missing-container-id-label",
			statsProvider: statsProvider,
			batch: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey(containerID): &core.MetricValueSet{
						Timestamp: timestamp,
						CommonLabels: map[string]string{
							core.TypeLabel:      core.ContainerMetricType,
							core.NamespaceLabel: namespace,
							core.PodNameLabel:   podName,
						},
						Values: map[string]core.MetricValue{
							core.ContainerMemoryAvailableBytes.Name: 10 * 1024 * 1024 * 1024,
						},
					},
				},
			},
			want: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Name:      podName,
							Namespace: namespace,
							UID:       podUID,
						},
					},
				},
			},
		},
		{
			name:          "missing-namespace",
			statsProvider: statsProvider,
			batch: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey(containerID): &core.MetricValueSet{
						Timestamp: timestamp,
						CommonLabels: map[string]string{
							core.TypeLabel:    core.ContainerMetricType,
							core.PodNameLabel: podName,
						},
						Values: map[string]core.MetricValue{
							core.ContainerMemoryAvailableBytes.Name: 10 * 1024 * 1024 * 1024,
						},
					},
				},
			},
			want: &sketchapi.SketchSummary{},
		},
		{
			name:          "missing-pod-name",
			statsProvider: statsProvider,
			batch: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey(containerID): &core.MetricValueSet{
						Timestamp: timestamp,
						CommonLabels: map[string]string{
							core.TypeLabel: core.ContainerMetricType,
						},
						Values: map[string]core.MetricValue{
							core.ContainerMemoryAvailableBytes.Name: 10 * 1024 * 1024 * 1024,
						},
					},
				},
			},
			want: &sketchapi.SketchSummary{},
		},
		{
			name: "missing-stats-provider",
			batch: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey(containerID): &core.MetricValueSet{
						Timestamp: timestamp,
						CommonLabels: map[string]string{
							core.TypeLabel:        core.ContainerMetricType,
							core.NamespaceLabel:   namespace,
							core.PodNameLabel:     podName,
							core.ContainerIDLabel: containerID,
						},
						Values: map[string]core.MetricValue{
							core.ContainerMemoryAvailableBytes.Name:  10 * 1024 * 1024 * 1024,
							core.ContainerMemoryUsageBytes.Name:      100 * 1024,
							core.ContainerMemoryWorkingSetBytes.Name: 200 * 1024,
						},
					},
				},
			},
			want: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Name:      podName,
							Namespace: namespace,
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								ID: containerID,
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										Time:            metav1.NewTime(timestamp),
										AvailableBytes:  10 * 1024 * 1024 * 1024,
										UsageBytes:      100 * 1024,
										WorkingSetBytes: 200 * 1024,
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
			builder := New(tt.statsProvider)
			if got := builder.Build(tt.batch); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("builderImpl.Build() = %#v, want %#v", got.Pods[0].Containers[0], tt.want.Pods[0].Containers[0])
			}
		})
	}
}

func Test_builderImpl_InvalidContainerID(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	namespace := "test-namespace"
	podName := "test-pod"
	podUID := "123456"
	containerID := "container-1"
	containerName := "test-container"

	statsProvider := builderstest.NewMockStatsProvider(controller)
	statsProvider.EXPECT().GetPodByName(namespace, podName).AnyTimes().Return(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			UID:       types.UID(podUID),
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name: containerName,
					// ContainerID must be format like docker://<container id>
					ContainerID: containerID,
				},
			},
		},
	}, true)

	timestamp := time.Now()

	tests := []struct {
		name          string
		statsProvider StatsProvider
		batch         *core.DataBatch
		want          *sketchapi.SketchSummary
	}{
		{
			name:          "invalid-container-id",
			statsProvider: statsProvider,
			batch: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.ContainerKey(containerID): &core.MetricValueSet{
						Timestamp: timestamp,
						CommonLabels: map[string]string{
							core.TypeLabel:        core.ContainerMetricType,
							core.NamespaceLabel:   namespace,
							core.PodNameLabel:     podName,
							core.ContainerIDLabel: containerID,
						},
						Values: map[string]core.MetricValue{
							core.ContainerMemoryAvailableBytes.Name:  10 * 1024 * 1024 * 1024,
							core.ContainerMemoryUsageBytes.Name:      100 * 1024,
							core.ContainerMemoryWorkingSetBytes.Name: 200 * 1024,
						},
					},
				},
			},
			want: &sketchapi.SketchSummary{
				Pods: []sketchapi.PodSketch{
					{
						PodRef: sketchapi.PodReference{
							Name:      podName,
							Namespace: namespace,
							UID:       podUID,
						},
						Containers: []*sketchapi.ContainerSketch{
							&sketchapi.ContainerSketch{
								ID: containerID,
								Memory: &sketchapi.ContainerMemorySketch{
									MemorySketch: sketchapi.MemorySketch{
										Time:            metav1.NewTime(timestamp),
										AvailableBytes:  10 * 1024 * 1024 * 1024,
										UsageBytes:      100 * 1024,
										WorkingSetBytes: 200 * 1024,
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
			builder := New(tt.statsProvider)
			if got := builder.Build(tt.batch); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("builderImpl.Build() = %#v, want %#v", got.Pods[0].Containers[0], tt.want.Pods[0].Containers[0])
			}
		})
	}
}
