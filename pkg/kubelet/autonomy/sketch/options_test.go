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

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestAddFlagSet(t *testing.T) {
	expect := NewOptions()
	expect.Resolution = 1 * time.Second
	expect.ScrapeOffset = 2 * time.Second
	expect.ScrapeTimeout = 3 * time.Second
	expect.Walle.DialTimeout = 4 * time.Second
	expect.Walle.Address = "unix:///var/test.sock"

	fs := pflag.NewFlagSet("", pflag.ExitOnError)
	AddFlagSet(fs)

	err := fs.Parse([]string{
		"--sketch-resolution", expect.Resolution.String(),
		"--sketch-scrape-offset", expect.ScrapeOffset.String(),
		"--sketch-scrape-timeout", expect.ScrapeTimeout.String(),
		"--walle-dial-timeout", expect.Walle.DialTimeout.String(),
		"--walle-address", "unix:///var/test.sock"})

	assert.NoError(t, err)
	assert.Equal(t, expect, NewOptions())
}

func TestOptions_Validate(t *testing.T) {
	tests := []struct {
		name string
		Options
		wantErr bool
	}{
		{
			name:    "normal",
			Options: defaultOptions,
		},
		{
			name: "invalid-resolution-with-empty",
			Options: Options{
				Resolution:    0,
				ScrapeOffset:  defaultOptions.ScrapeOffset,
				ScrapeTimeout: defaultOptions.ScrapeTimeout,
				Walle:         defaultOptions.Walle,
			},
			wantErr: true,
		},
		{
			name: "invalid-resolution-with-negative",
			Options: Options{
				Resolution:    -1 * time.Second,
				ScrapeOffset:  defaultOptions.ScrapeOffset,
				ScrapeTimeout: defaultOptions.ScrapeTimeout,
				Walle:         defaultOptions.Walle,
			},
			wantErr: true,
		},
		{
			name: "invalid-scrape-timeout-with-empty",
			Options: Options{
				Resolution:   defaultOptions.Resolution,
				ScrapeOffset: defaultOptions.ScrapeOffset,
				Walle:        defaultOptions.Walle,
			},
		},
		{
			name: "invalid-scrape-timeout-with-negative",
			Options: Options{
				Resolution:    defaultOptions.Resolution,
				ScrapeOffset:  defaultOptions.ScrapeOffset,
				ScrapeTimeout: -1 * time.Second,
				Walle:         defaultOptions.Walle,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				Resolution:    tt.Resolution,
				ScrapeOffset:  tt.ScrapeOffset,
				ScrapeTimeout: tt.ScrapeTimeout,
				Walle:         tt.Walle,
			}
			if err := o.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Options.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
