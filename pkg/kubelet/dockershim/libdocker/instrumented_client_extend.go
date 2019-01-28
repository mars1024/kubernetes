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

package libdocker

import (
	"time"
)

func (in instrumentedInterface) PauseContainer(id string) error {
	const operation = "pause_container"
	defer recordOperation(operation, time.Now())

	err := in.client.PauseContainer(id)
	recordError(operation, err)
	return err
}

func (in instrumentedInterface) UnpauseContainer(id string) error {
	const operation = "unpause_container"
	defer recordOperation(operation, time.Now())

	err := in.client.UnpauseContainer(id)
	recordError(operation, err)
	return err
}
