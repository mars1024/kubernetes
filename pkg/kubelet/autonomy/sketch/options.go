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
	"errors"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/scrapers/walle"
)

// Options configs sketch
type Options struct {
	Resolution    time.Duration
	ScrapeOffset  time.Duration
	ScrapeTimeout time.Duration
	Walle         walle.Options
}

// Definition of default options
const (
	DefaultResolution    = 10 * time.Second
	DefaultScrapeOffset  = 5 * time.Second
	DefaultScrapeTimeout = 10 * time.Second
)

var defaultOptions = Options{
	Resolution:    DefaultResolution,
	ScrapeOffset:  DefaultScrapeOffset,
	ScrapeTimeout: DefaultScrapeTimeout,
	Walle: walle.Options{
		Address:     walle.DefaultWalleUnixSocketAddress,
		DialTimeout: walle.DefaultDialTimeout,
	},
}

// AddFlagSet adds sketch flag into pflag.FlagSet
func AddFlagSet(fs *pflag.FlagSet) {
	fs.DurationVar(&defaultOptions.Resolution, "sketch-resolution", defaultOptions.Resolution, "scrape resolution duration")
	fs.DurationVar(&defaultOptions.ScrapeOffset, "sketch-scrape-offset", defaultOptions.ScrapeOffset, "scrape duration offset")
	fs.DurationVar(&defaultOptions.ScrapeTimeout, "sketch-scrape-timeout", defaultOptions.ScrapeTimeout, "scrape timeout")
	fs.StringVar(&defaultOptions.Walle.Address, "walle-address", defaultOptions.Walle.Address, "walle agent server address")
	fs.DurationVar(&defaultOptions.Walle.DialTimeout, "walle-dial-timeout", defaultOptions.Walle.DialTimeout, "dial walle timeout")
}

// NewOptions contructs Options
func NewOptions() Options {
	return defaultOptions
}

// Validate verifies Options
func (o *Options) Validate() error {
	if o.Resolution <= 0 {
		return errors.New("sketch: invalid Resolution in Options")
	}
	if o.ScrapeTimeout < 0 {
		return errors.New("sketch: invalid ScrapeTimeout in Options")
	}

	return o.Walle.Validate()
}
