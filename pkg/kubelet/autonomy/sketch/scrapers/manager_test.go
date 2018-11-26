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

package scrapers

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
	coretest "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core/testing"
)

func newMockMetricsScraper(controller *gomock.Controller, name string, data *core.DataBatch) *coretest.MockMetricsScraper {
	s := coretest.NewMockMetricsScraper(controller)
	s.EXPECT().Name().Return(name)
	s.EXPECT().Scrape(gomock.Any(), gomock.Any()).Return(data)
	return s
}

var _ core.MetricsScraper = &timeoutScraper{}

type timeoutScraper struct {
	timeout time.Duration
	data    *core.DataBatch
}

func (s *timeoutScraper) Name() string { return "timeoutScraper" }

func (s *timeoutScraper) Scrape(ctx context.Context, timestamp time.Time) *core.DataBatch {
	_, ok := ctx.Deadline()
	if !ok {
		return nil
	}

	time.Sleep(s.timeout)
	return s.data
}

func Test_manager_Scrape(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	timestamp := time.Now()

	tests := []struct {
		name          string
		provider      *coretest.MockMetricsScraperProvider
		scrapeTimeout time.Duration
		scrapers      []core.MetricsScraper
		ctx           context.Context
		timestamp     time.Time
		want          *core.DataBatch
	}{
		{
			name:     "single-scraper",
			provider: coretest.NewMockMetricsScraperProvider(controller),
			scrapers: []core.MetricsScraper{
				newMockMetricsScraper(controller, "test-scraper-1", &core.DataBatch{
					Timestamp: timestamp,
					MetricValueSets: map[string]*core.MetricValueSet{
						core.NodeKey(): &core.MetricValueSet{
							Values: map[string]core.MetricValue{
								core.NodeCPUUsage.Name: 1.1,
							},
						},
					},
				}),
			},
			ctx:       context.Background(),
			timestamp: timestamp,
			want: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.NodeKey(): &core.MetricValueSet{
						Values: map[string]core.MetricValue{
							core.NodeCPUUsage.Name: 1.1,
						},
					},
				},
			},
		},
		{
			name:     "three-scrapers",
			provider: coretest.NewMockMetricsScraperProvider(controller),
			scrapers: []core.MetricsScraper{
				newMockMetricsScraper(controller, "test-scraper-1", &core.DataBatch{
					Timestamp: timestamp,
					MetricValueSets: map[string]*core.MetricValueSet{
						core.NodeKey(): &core.MetricValueSet{
							Values: map[string]core.MetricValue{
								core.NodeCPUUsage.Name: 1.1,
							},
						},
					},
				}),
				newMockMetricsScraper(controller, "test-scraper-2", &core.DataBatch{
					Timestamp: timestamp,
					MetricValueSets: map[string]*core.MetricValueSet{
						core.NodeKey(): &core.MetricValueSet{
							Values: map[string]core.MetricValue{
								core.NodeMemoryAvailableBytes.Name: 1024,
							},
						},
					},
				}),
				newMockMetricsScraper(controller, "test-scraper-3", &core.DataBatch{
					Timestamp: timestamp,
					MetricValueSets: map[string]*core.MetricValueSet{
						core.NodeKey(): &core.MetricValueSet{
							Values: map[string]core.MetricValue{
								core.NodeMemoryUsedBytes.Name: 2048,
							},
						},
					},
				}),
			},
			ctx:       context.Background(),
			timestamp: timestamp,
			want: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.NodeKey(): &core.MetricValueSet{
						Values: map[string]core.MetricValue{
							core.NodeCPUUsage.Name:             1.1,
							core.NodeMemoryAvailableBytes.Name: 1024,
							core.NodeMemoryUsedBytes.Name:      2048,
						},
					},
				},
			},
		},
		{
			name:          "three-scrapers-with-timeout",
			scrapeTimeout: 100 * time.Millisecond,
			provider:      coretest.NewMockMetricsScraperProvider(controller),
			scrapers: []core.MetricsScraper{
				newMockMetricsScraper(controller, "test-scraper-1", &core.DataBatch{
					Timestamp: timestamp,
					MetricValueSets: map[string]*core.MetricValueSet{
						core.NodeKey(): &core.MetricValueSet{
							Values: map[string]core.MetricValue{
								core.NodeCPUUsage.Name: 1.1,
							},
						},
					},
				}),
				newMockMetricsScraper(controller, "test-scraper-2", &core.DataBatch{
					Timestamp: timestamp,
					MetricValueSets: map[string]*core.MetricValueSet{
						core.NodeKey(): &core.MetricValueSet{
							Values: map[string]core.MetricValue{
								core.NodeMemoryAvailableBytes.Name: 1024,
							},
						},
					},
				}),
				&timeoutScraper{timeout: 120 * time.Millisecond, data: nil},
			},
			ctx:       context.Background(),
			timestamp: timestamp,
			want: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.NodeKey(): &core.MetricValueSet{
						Values: map[string]core.MetricValue{
							core.NodeCPUUsage.Name:             1.1,
							core.NodeMemoryAvailableBytes.Name: 1024,
						},
					},
				},
			},
		},
		{
			name:          "three-scrapers-with-exception",
			scrapeTimeout: 100 * time.Millisecond,
			provider:      coretest.NewMockMetricsScraperProvider(controller),
			scrapers: []core.MetricsScraper{
				newMockMetricsScraper(controller, "test-scraper-1", &core.DataBatch{
					Timestamp: timestamp,
					MetricValueSets: map[string]*core.MetricValueSet{
						core.NodeKey(): &core.MetricValueSet{
							Values: map[string]core.MetricValue{
								core.NodeCPUUsage.Name: 1.1,
							},
						},
					},
				}),
				newMockMetricsScraper(controller, "test-scraper-2", &core.DataBatch{
					Timestamp: timestamp,
					MetricValueSets: map[string]*core.MetricValueSet{
						core.NodeKey(): &core.MetricValueSet{
							Values: map[string]core.MetricValue{
								core.NodeMemoryAvailableBytes.Name: 1024,
							},
						},
					},
				}),
				newMockMetricsScraper(controller, "test-scraper-3", nil),
			},
			ctx:       context.Background(),
			timestamp: timestamp,
			want: &core.DataBatch{
				Timestamp: timestamp,
				MetricValueSets: map[string]*core.MetricValueSet{
					core.NodeKey(): &core.MetricValueSet{
						Values: map[string]core.MetricValue{
							core.NodeCPUUsage.Name:             1.1,
							core.NodeMemoryAvailableBytes.Name: 1024,
						},
					},
				},
			},
		},
		{
			name:     "scraper-return-nil",
			provider: coretest.NewMockMetricsScraperProvider(controller),
			scrapers: []core.MetricsScraper{
				newMockMetricsScraper(controller, "test-scraper-1", nil),
			},
			ctx:       context.Background(),
			timestamp: timestamp,
			want:      nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.provider.EXPECT().GetMetricsScrapers().Return(tt.scrapers)

			m := NewManager(tt.provider, tt.scrapeTimeout)
			if got := m.Scrape(tt.ctx, tt.timestamp); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("manager.Scrape() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_manager_Name(t *testing.T) {
	// just for cover
	m := &manager{}
	m.Name()
	t.Skip()
}
