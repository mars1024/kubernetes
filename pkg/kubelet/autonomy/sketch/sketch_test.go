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
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	buildertest "k8s.io/kubernetes/pkg/kubelet/autonomy/sketch/builders/testing"
)


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
