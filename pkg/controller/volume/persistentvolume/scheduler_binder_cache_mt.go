/*
Copyright 2017 The Kubernetes Authors.

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

package persistentvolume

import (
	"sync"

	"k8s.io/api/core/v1"
)

type clusterAwarePodBindingCache struct {
	// synchronizes bindingDecisions
	rwMutex sync.RWMutex

	// Key = pod name
	// Value = nodeDecisions
	bindingDecisions map[string]nodeDecisions
}

func NewClusterAwarePodBindingCache() PodBindingCache {
	return &clusterAwarePodBindingCache{bindingDecisions: map[string]nodeDecisions{}}
}

func (c *clusterAwarePodBindingCache) DeleteBindings(pod *v1.Pod) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()

	podFullName := getPodNameWithCluster(pod)
	delete(c.bindingDecisions, podFullName)
}

func (c *clusterAwarePodBindingCache) UpdateBindings(pod *v1.Pod, node string, bindings []*bindingInfo) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()

	podFullName := getPodNameWithCluster(pod)
	decisions, ok := c.bindingDecisions[podFullName]
	if !ok {
		decisions = nodeDecisions{}
		c.bindingDecisions[podFullName] = decisions
	}
	decision, ok := decisions[node]
	if !ok {
		decision = nodeDecision{
			bindings: bindings,
		}
	} else {
		decision.bindings = bindings
	}
	decisions[node] = decision
}

func (c *clusterAwarePodBindingCache) GetBindings(pod *v1.Pod, node string) []*bindingInfo {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()

	podFullName := getPodNameWithCluster(pod)
	decisions, ok := c.bindingDecisions[podFullName]
	if !ok {
		return nil
	}
	decision, ok := decisions[node]
	if !ok {
		return nil
	}
	return decision.bindings
}

func (c *clusterAwarePodBindingCache) UpdateProvisionedPVCs(pod *v1.Pod, node string, pvcs []*v1.PersistentVolumeClaim) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()

	podFullName := getPodNameWithCluster(pod)
	decisions, ok := c.bindingDecisions[podFullName]
	if !ok {
		decisions = nodeDecisions{}
		c.bindingDecisions[podFullName] = decisions
	}
	decision, ok := decisions[node]
	if !ok {
		decision = nodeDecision{
			provisionings: pvcs,
		}
	} else {
		decision.provisionings = pvcs
	}
	decisions[node] = decision
}

func (c *clusterAwarePodBindingCache) GetProvisionedPVCs(pod *v1.Pod, node string) []*v1.PersistentVolumeClaim {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()

	podFullName := getPodNameWithCluster(pod)
	decisions, ok := c.bindingDecisions[podFullName]
	if !ok {
		return nil
	}
	decision, ok := decisions[node]
	if !ok {
		return nil
	}
	return decision.provisionings
}
