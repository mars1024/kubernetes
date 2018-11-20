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
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
)

var (
	lastScrapeTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "sketch",
			Subsystem: "scrapers",
			Name:      "last_time_seconds",
			Help:      "Last time Sketch performed a scrape since unix epoch in seconds.",
		},
		[]string{"scraper"},
	)

	scraperDuration = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "sketch",
			Subsystem: "scrapers",
			Name:      "duration_microseconds",
			Help:      "Time spent scraping sources in microseconds.",
		},
		[]string{"scraper"},
	)
)

func init() {
	prometheus.MustRegister(lastScrapeTimestamp)
	prometheus.MustRegister(scraperDuration)
}

var _ core.MetricsScraper = &manager{}

type manager struct {
	provider      core.MetricsScraperProvider
	scrapeTimeout time.Duration
}

// NewManager constructs manager to dispatch all scraper to scrape metrics at the same time
// and merged all core.DataBatch into result
func NewManager(provider core.MetricsScraperProvider, scrapeTimeout time.Duration) core.MetricsScraper {
	return &manager{
		provider:      provider,
		scrapeTimeout: scrapeTimeout,
	}
}

func (m *manager) Name() string { return "scrapers_manager" }

func (m *manager) Scrape(ctx context.Context, timestamp time.Time) *core.DataBatch {
	if m.scrapeTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.scrapeTimeout)
		defer cancel()
	}

	scrapers := m.provider.GetMetricsScrapers()

	if len(scrapers) == 1 {
		return invokeScrape(ctx, scrapers[0], timestamp)
	}

	return m.parallelScrape(ctx, timestamp, scrapers)
}

func (m *manager) parallelScrape(ctx context.Context, timestamp time.Time, scrapers []core.MetricsScraper) *core.DataBatch {
	responseChannel := make(chan *core.DataBatch, len(scrapers))
	for _, scraper := range scrapers {
		go func(scraper core.MetricsScraper) {
			batch := invokeScrape(ctx, scraper, timestamp)
			select {
			case responseChannel <- batch:
			case <-ctx.Done():
			}
		}(scraper)
	}

	mergedBatch := &core.DataBatch{
		Timestamp:       timestamp,
		MetricValueSets: make(map[string]*core.MetricValueSet),
	}

receiveLoop:
	for i := range scrapers {
		select {
		case batch := <-responseChannel:
			if batch == nil {
				continue
			}

			for k, v := range batch.MetricValueSets {
				set := mergedBatch.MetricValueSets[k]
				if set == nil {
					mergedBatch.MetricValueSets[k] = v
					continue
				}
				set.Merge(v)
			}
		case <-ctx.Done():
			glog.Warningf("failed to receive all response in time (got %d/%d)", i, len(scrapers))
			break receiveLoop
		}
	}

	return mergedBatch
}

func invokeScrape(
	ctx context.Context,
	scraper core.MetricsScraper,
	timestamp time.Time) *core.DataBatch {

	scraperName := scraper.Name()
	startTime := time.Now()
	defer func() {
		lastScrapeTimestamp.
			WithLabelValues(scraperName).
			Set(float64(time.Now().Unix()))
		scraperDuration.
			WithLabelValues(scraperName).
			Observe(float64(time.Since(startTime)) / float64(time.Microsecond))
	}()

	return scraper.Scrape(ctx, timestamp)
}
