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
	"fmt"
	"os"
)

func (f *FakeDockerClient) PauseContainer(id string) error {
	f.Lock()
	defer f.Unlock()
	f.appendCalled(calledDetail{name: "pause"})
	if err := f.popError("pause"); err != nil {
		return err
	}
	f.appendContainerTrace("Paused", id)
	container, ok := f.ContainerMap[id]
	timestamp := f.Clock.Now()
	if !ok {
		container = convertFakeContainer(&FakeContainer{ID: id, Name: id, CreatedAt: timestamp})
	}
	container.State.Running = true
	container.State.Pid = os.Getpid()
	container.State.Paused = true
	container.State.Status = "paused"
	container.State.StartedAt = dockerTimestampToString(timestamp)
	r := f.RandGenerator.Uint32()
	container.NetworkSettings.IPAddress = fmt.Sprintf("10.%d.%d.%d", byte(r>>16), byte(r>>8), byte(r))
	f.ContainerMap[id] = container
	f.updateContainerStatus(id, StatusRunningPrefix)
	f.normalSleep(200, 50, 50)
	return nil
}

func (f *FakeDockerClient) UnpauseContainer(id string) error {
	f.Lock()
	defer f.Unlock()
	f.appendCalled(calledDetail{name: "start"})
	if err := f.popError("start"); err != nil {
		return err
	}
	f.appendContainerTrace("Started", id)
	container, ok := f.ContainerMap[id]
	timestamp := f.Clock.Now()
	if !ok {
		container = convertFakeContainer(&FakeContainer{ID: id, Name: id, CreatedAt: timestamp})
	}
	container.State.Running = true
	container.State.Pid = os.Getpid()
	container.State.Paused = false
	container.State.Status = "running"
	container.State.StartedAt = dockerTimestampToString(timestamp)
	r := f.RandGenerator.Uint32()
	container.NetworkSettings.IPAddress = fmt.Sprintf("10.%d.%d.%d", byte(r>>16), byte(r>>8), byte(r))
	f.ContainerMap[id] = container
	f.updateContainerStatus(id, StatusRunningPrefix)
	f.normalSleep(200, 50, 50)
	return nil
}
