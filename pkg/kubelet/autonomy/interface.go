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

package autonomy

// Autonomist interface with fake features for autonomy modules.
// TODO: Add actual functions.
type Autonomist interface {
	// Start autonomy services.
	Start()
	// Enable autonomy services, default is true.
	Enable()
	// Disable autonomy services, default is false.
	Disable()
	// Statistics show services' metrics and status.
	Statistics()
	// Reload services with custom configer, need paramrters.
	Reload()
	// Stop services.
	Stop()
}
