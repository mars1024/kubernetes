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

func (d *kubeDockerClient) PauseContainer(id string) error {
	ctx, cancel := d.getTimeoutContext()
	defer cancel()
	err := d.client.ContainerPause(ctx, id)
	if ctxErr := contextError(ctx); ctxErr != nil {
		return ctxErr
	}
	return err
}

func (d *kubeDockerClient) UnpauseContainer(id string) error {
	ctx, cancel := d.getTimeoutContext()
	defer cancel()
	err := d.client.ContainerUnpause(ctx, id)
	if ctxErr := contextError(ctx); ctxErr != nil {
		return ctxErr
	}
	return err
}
