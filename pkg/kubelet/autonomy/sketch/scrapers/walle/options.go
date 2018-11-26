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
	"errors"
	"time"
)

// Definition of the default options that be used by Walle client
const (
	DefaultWalleUnixSocketAddress = "unix:///var/run/walle/walle.sock"
	DefaultWalleURLAddress        = "http://127.0.0.1:9199/"
	DefaultDialTimeout            = 1 * time.Second
)

// Options configs walle scraper
type Options struct {
	Address     string
	DialTimeout time.Duration
}

// Validate verifies Options
func (o *Options) Validate() error {
	if o.Address == "" {
		return errors.New("walle: invalid Address in Options")
	}
	if o.DialTimeout < 0 {
		return errors.New("walle: invalid DialTimeout in Options")
	}
	return nil
}
