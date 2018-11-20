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
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	buildertest "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/builders/testing"
	walletest "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/scrapers/walle/testing"
)

func TestSketchProvider(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	provider := buildertest.NewMockStatsProvider(controller)
	provider.EXPECT().GetNode().AnyTimes().Return(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}, error(nil))

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
							"__name__": "node_cpu_usage"
						},
						"value": [
							1533136816.73,
							"1.2"
						]
					}
				]
			}
		}`))
		}))
	defer clean()

	options := NewOptions()
	options.Resolution = 50 * time.Millisecond
	options.ScrapeOffset = 0
	options.Walle.Address = "http://" + listener.Addr().String()

	sketchProvider, err := New(options, provider)
	assert.NoError(t, err)
	assert.NotNil(t, sketchProvider)

	assert.NoError(t, sketchProvider.Start())
	defer func() {
		sketchProvider.Stop()
		// waiting for sketch provider stopped
		time.Sleep(100 * time.Millisecond)
	}()

	time.Sleep(110 * time.Millisecond)
	nodeSketch, err := sketchProvider.GetSketch().GetNodeSketch()
	assert.NoError(t, err)
	assert.NotNil(t, nodeSketch)
	assert.Equal(t, "test", nodeSketch.Name)
	assert.Equal(t, float64(1.2), nodeSketch.CPU.Usage.Latest)
}

func TestSketchProviderWithOptions(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	provider := buildertest.NewMockStatsProvider(controller)

	options := NewOptions()
	options.Resolution = 100 * time.Millisecond

	sketchProvider, err := New(options, provider)
	assert.NoError(t, err)
	assert.NotNil(t, sketchProvider)
	p := sketchProvider.(*providerImpl)
	assert.Equal(t, options.Resolution, p.Options.Resolution)
}

func TestSketchProviderWithInvalidOptions(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	provider := buildertest.NewMockStatsProvider(controller)

	options := NewOptions()
	options.Resolution = 0

	sketchProvider, err := New(options, provider)
	assert.Error(t, err)
	assert.Nil(t, sketchProvider)
}

func TestSketchProviderWithInvalidWalleAddress(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	provider := buildertest.NewMockStatsProvider(controller)

	options := NewOptions()
	options.Walle.Address = "ht tp://xxxx"

	sketchProvider, err := New(options, provider)
	assert.Error(t, err)
	assert.Nil(t, sketchProvider)
}
