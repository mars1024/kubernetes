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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/core"
	walletest "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/scrapers/walle/testing"
)

func TestNewProviderWithUnixSocket(t *testing.T) {
	address, clean := walletest.SetupPrometheusServerWithUnixSocket(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer clean()

	opts := Options{
		Address:     address,
		DialTimeout: DefaultDialTimeout,
	}

	provider, err := NewProvider([]core.MetricGroup{
		{
			Scraper: "walle",
		}}, opts)

	assert.NoError(t, err)
	assert.NotNil(t, provider)
	scrapers := provider.GetMetricsScrapers()
	assert.NotNil(t, scrapers)
	assert.Equal(t, 1, len(scrapers))
	assert.Equal(t, "walle", scrapers[0].Name())
}

func TestNewProviderWithHTTPURL(t *testing.T) {
	listener, clean := walletest.SetupPrometheusServer(t, "tcp4://127.0.0.1:0",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer clean()

	opts := Options{
		Address:     "http://" + listener.Addr().String(),
		DialTimeout: DefaultDialTimeout,
	}

	provider, err := NewProvider([]core.MetricGroup{
		{
			Scraper: "walle",
		}}, opts)

	assert.NoError(t, err)
	assert.NotNil(t, provider)
	scrapers := provider.GetMetricsScrapers()
	assert.NotNil(t, scrapers)
	assert.Equal(t, 1, len(scrapers))
	assert.Equal(t, "walle", scrapers[0].Name())
}

func TestNewProviderWithUnsupportScheme(t *testing.T) {
	opts := Options{
		Address:     "xxx://xxxx",
		DialTimeout: DefaultDialTimeout,
	}
	provider, err := NewProvider(nil, opts)
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestNewProviderWithInvalidURL(t *testing.T) {
	opts := Options{
		Address:     "ht tp://xxx",
		DialTimeout: DefaultDialTimeout,
	}
	provider, err := NewProvider(nil, opts)
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestNewProviderWithInvalidOptions(t *testing.T) {
	opts := Options{
		DialTimeout: DefaultDialTimeout,
	}
	provider, err := NewProvider(nil, opts)
	assert.Error(t, err)
	assert.Nil(t, provider)
}
