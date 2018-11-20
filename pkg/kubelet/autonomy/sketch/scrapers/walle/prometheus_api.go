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
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	apiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// NewPrometheusAPI constructs prometheus v1.API interface
func NewPrometheusAPI(opts ...func(*api.Config) error) (apiv1.API, error) {
	var config api.Config
	config.RoundTripper = api.DefaultRoundTripper
	config.Address = DefaultWalleURLAddress

	for _, opt := range opts {
		if err := opt(&config); err != nil {
			return nil, err
		}
	}

	c, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	return apiv1.NewAPI(c), nil
}

// WithUnixSocketTransport configs prometheus api config with unix socket
func WithUnixSocketTransport(address string, dialTimeout time.Duration) func(config *api.Config) error {
	address = strings.Replace(address, "unix://", "", -1)
	dialer := &net.Dialer{
		Timeout: dialTimeout,
	}
	return WithTransport(&http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", address)
		},
	})
}

// WithHTTPURL configs prometheus api config with speical Restful url
func WithHTTPURL(address string) func(config *api.Config) error {
	return func(config *api.Config) error {
		_, err := url.Parse(address)
		if err != nil {
			return err
		}
		config.Address = address
		return nil
	}
}

// WithTransport config prometheus api config with speical http.RoundTripper
func WithTransport(transport http.RoundTripper) func(config *api.Config) error {
	return func(config *api.Config) error {
		config.RoundTripper = transport
		return nil
	}
}
