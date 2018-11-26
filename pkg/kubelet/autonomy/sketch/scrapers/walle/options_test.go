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
	"testing"
	"time"
)

func TestOptions_Validate(t *testing.T) {
	tests := []struct {
		name        string
		Address     string
		DialTimeout time.Duration
		wantErr     bool
	}{
		{name: "sucess", Address: DefaultWalleUnixSocketAddress, DialTimeout: DefaultDialTimeout},
		{name: "empty-address", DialTimeout: DefaultDialTimeout, wantErr: true},
		{name: "invalid-DialTimeout", Address: DefaultWalleUnixSocketAddress, DialTimeout: -1 * time.Second, wantErr: true},
		{name: "empty-DialTimeout", Address: DefaultWalleUnixSocketAddress, DialTimeout: 0, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				Address:     tt.Address,
				DialTimeout: tt.DialTimeout,
			}
			if err := o.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Options.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
