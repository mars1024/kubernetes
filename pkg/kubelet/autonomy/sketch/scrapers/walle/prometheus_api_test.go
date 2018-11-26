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
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"

	walletest "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/scrapers/walle/testing"
)

func TestNewPrometheusAPIWithUnixSocket(t *testing.T) {
	address, clean := walletest.SetupPrometheusServerWithUnixSocket(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`
			{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {
								"__name__": "go_test"
							},
							"value": [
								1533136816.73,
								"1"
							]
						}
					]
				}
			}`))
		}))
	defer clean()

	promAPI, err := NewPrometheusAPI(WithUnixSocketTransport(address, DefaultDialTimeout))
	assert.NoError(t, err)

	value, err := promAPI.Query(context.Background(), "go_test", time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, value)

	vector := value.(model.Vector)
	assert.Equal(t, 1, len(vector))
	assert.Equal(t, model.SampleValue(1), vector[0].Value)
	assert.Equal(t, model.LabelValue("go_test"), vector[0].Metric[model.MetricNameLabel])
}

func TestNewPrometheusAPIWithTCP(t *testing.T) {
	listener, clean := walletest.SetupPrometheusServer(t, "tcp4://127.0.0.1:0",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`
			{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {
								"__name__": "go_test"
							},
							"value": [
								1533136816.73,
								"1"
							]
						}
					]
				}
			}`))
		}))
	defer clean()

	promAPI, err := NewPrometheusAPI(WithHTTPURL("http://" + listener.Addr().String()))
	assert.NoError(t, err)

	value, err := promAPI.Query(context.Background(), "go_test", time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, value)

	vector := value.(model.Vector)
	assert.Equal(t, 1, len(vector))
	assert.Equal(t, model.SampleValue(1), vector[0].Value)
	assert.Equal(t, model.LabelValue("go_test"), vector[0].Metric[model.MetricNameLabel])
}

type filterTransport struct {
	filter    func(r *http.Request)
	transport http.RoundTripper
}

func (f filterTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.filter(r)
	return f.transport.RoundTrip(r)
}

func TestNewPrometheusAPIWithTransport(t *testing.T) {
	fd, err := ioutil.TempFile("", "sketch")
	assert.NoError(t, err)
	filepath := fd.Name()
	fd.Close()
	os.Remove(filepath)

	_, clean := walletest.SetupPrometheusServer(t, "unix://"+filepath,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`
			{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {
								"__name__": "go_test"
							},
							"value": [
								1533136816.73,
								"1"
							]
						}
					]
				}
			}`))
		}))
	defer clean()

	filteredTransport := false
	promAPI, err := NewPrometheusAPI(
		WithUnixSocketTransport(filepath, DefaultDialTimeout),
		func(c *api.Config) error {
			return WithTransport(filterTransport{
				filter: func(r *http.Request) {
					filteredTransport = true
				},
				transport: c.RoundTripper,
			})(c)
		})
	assert.NoError(t, err)

	value, err := promAPI.Query(context.Background(), "go_test", time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, value)

	assert.True(t, filteredTransport)

	vector := value.(model.Vector)
	assert.Equal(t, 1, len(vector))
	assert.Equal(t, model.SampleValue(1), vector[0].Value)
	assert.Equal(t, model.LabelValue("go_test"), vector[0].Metric[model.MetricNameLabel])
}

func TestNewPrometheusAPIWithOptionException(t *testing.T) {
	expectErr := errors.New("must failed")
	_, err := NewPrometheusAPI(func(*api.Config) error {
		return expectErr
	})
	assert.Equal(t, expectErr, err)
}

func TestNewPrometheusAPIWithInvalidAddr(t *testing.T) {
	_, err := NewPrometheusAPI(func(c *api.Config) error {
		c.Address = "htt p://xxx"
		return nil
	})
	assert.Error(t, err)
}

func TestWithHTTPURLWithInvalidURL(t *testing.T) {
	_, err := NewPrometheusAPI(WithHTTPURL("ht tp://xxx"))
	assert.Error(t, err)
}
