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

package sketch

import (
	"context"
	"sync/atomic"
	"time"

	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/builders"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/scrapers"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/scrapers/walle"
)

var _ Provider = &providerImpl{}

type providerImpl struct {
	Options
	scraper core.MetricsScraper
	builder builders.SketchBuilder
	summary atomic.Value
	done    chan struct{}
}

// New constructs Interface instance
func New(options Options,  statsProvider builders.StatsProvider) (Provider, error) {

	err := options.Validate()
	if err != nil {
		return nil, err
	}

	provider, err := walle.NewProvider(core.AllMetricGroups, options.Walle)
	if err != nil {
		return nil, err
	}

	return &providerImpl{
		Options: options,
		scraper: scrapers.NewManager(provider, options.ScrapeTimeout),
		builder: builders.New(statsProvider),
		done:    make(chan struct{}, 1),
	}, nil
}

func (p *providerImpl) Start() error {
	go p.houseKeep()
	return nil
}

func (p *providerImpl) Stop() {
	close(p.done)
}

func (p *providerImpl) GetSketch() Snapshoter {
	return newSnapshotImpl(p.summary.Load())
}

func (p *providerImpl) houseKeep() {
	for {
		// align the immediate timestamp by m.resolution,
		// and calc the next time to sync that is offset by end.
		// The m.scrapeOffset delays the sync time that can help warming-up the metrics.
		now := time.Now()
		start := now.Truncate(p.Resolution)
		end := start.Add(p.Resolution)
		nextSyncTime := end.Add(p.ScrapeOffset).Sub(now)

		select {
		case <-time.After(nextSyncTime):
			batch := p.scraper.Scrape(context.Background(), end)
			if batch != nil {
				summary := p.builder.Build(batch)
				if summary != nil {
					p.summary.Store(summary)
				}
			}

		case <-p.done:
			return
		}
	}
}
