/*
Copyright 2019 The Kubernetes Authors.

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

package allocators

import (
	"fmt"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
	"k8s.io/kubernetes/pkg/scheduler/util"
)

// ContainerCPUAssignments type used in cpu allocator
type ContainerCPUAssignments map[string]cpuset.CPUSet

// Clone returns a copy of ContainerCPUAssignments
func (as ContainerCPUAssignments) Clone() ContainerCPUAssignments {
	ret := make(ContainerCPUAssignments)
	for key, val := range as {
		ret[key] = val
	}
	return ret
}

func (as ContainerCPUAssignments) Add(key string, set cpuset.CPUSet) {
	if key == "" {
		glog.Warningf("[CPUAllocator]cannot add cpuset %s to ContainerCPUAssignments,"+
			" key cannot be empty", set.String())
		return
	}
	as[key] = set
}

// Default CPU allocator

type AllocatorInterface interface {
	Name() string
	Allocate(pod *v1.Pod) ContainerCPUAssignments
	Reallocate(newPod *v1.Pod) ContainerCPUAssignments
}

type CPUAllocator struct {
	nodeInfo *schedulercache.NodeInfo
	pool     *CPUPool
}

func NewCPUAllocator(nodeInfo *schedulercache.NodeInfo) AllocatorInterface {
	cloned := nodeInfo.Clone()
	return &CPUAllocator{
		nodeInfo: cloned,
		pool:     NewCPUPool(cloned)}
}

func (allocator *CPUAllocator) Name() string {
	return "CPUAllocator"
}
func (allocator *CPUAllocator) Allocate(pod *v1.Pod) ContainerCPUAssignments {
	allocator.reload(pod)
	result := make(ContainerCPUAssignments, 0)
	alloc := util.AllocSpecFromPod(pod)
	if alloc == nil {
		// Native pod goes native way
		return ContainerCPUAssignments{}
	}

	// InitContainer allocate CPUSet but not take it into account in pool
	for _, c := range pod.Spec.InitContainers {
		glog.V(5).Infof("[CPUAllocator]container %s/%s/%s will use CPUSetMode=cpuset", pod.Namespace, pod.Name, c.Name)

		allocContainer := getAllocContainer(alloc.Containers, c.Name)
		if allocContainer != nil && allocContainer.Resource.CPU.CPUSet != nil {
			allocated, err := allocator.allocateExclusiveCPUSet(pod, &c)
			if err != nil {
				glog.Error(err)
			}
			// allocate cpuset but don't add to refcount
			glog.V(5).Infof("[CPUAllocator]initconatiner %s/%s doesn't refresh the CPU reference count, cpuset: %s", pod.Name, c.Name, allocated)
		} else {
			// does not allocate cpuset, default to the cpumanager behavior
			glog.V(5).Infof("[CPUAllocator]initconatiner %s/%s will use share pool", pod.Name, c.Name)
		}
	}
	for _, c := range pod.Spec.Containers {
		allocContainer := getAllocContainer(alloc.Containers, c.Name)
		if allocContainer != nil && allocContainer.Resource.CPU.CPUSet != nil {
			// for CPU-affinity type container
			glog.V(5).Infof("[CPUAllocator]container %s will use CPUSetMode=cpuset", ContainerName(pod, &c))
			spreadStrategy := allocContainer.Resource.CPU.CPUSet.SpreadStrategy
			glog.V(5).Infof("[CPUAllocator]spread strategy is %q", spreadStrategy)
			// Exclusive container
			if IsExclusiveContainer(pod, &c) {
				allocated, err := allocator.allocateExclusiveCPUSet(pod, &c)
				if err != nil {
					glog.Error(err)
				}
				allocator.pool.RefreshRefCount(allocated, true)
				result.Add(c.Name, allocated)
				glog.V(5).Infof("[CPUAllocator]container %s allocated exclusive CPUSet=%v", ContainerName(pod, &c), allocated)
			} else {
				// Non-exclusive container
				allocated, err := allocator.allocateShareCPUSet(pod, &c)
				if err != nil {
					glog.Error(err)
				}
				allocator.pool.RefreshRefCount(allocated, false)
				result.Add(c.Name, allocated)
				glog.V(5).Infof("[CPUAllocator]container %s allocated shared CPUSet=%s", ContainerName(pod, &c), allocated)
			}
		} else {
			// does not allocate cpuset, default to the cpumanager behavior
			glog.V(5).Infof("[CPUAllocator]container %s will default to the cpumanager policy", ContainerName(pod, &c))
		}
	}
	return result
}

func (allocator *CPUAllocator) Reallocate(newPod *v1.Pod) ContainerCPUAssignments {

	// Remove new pod from nodeInfo
	allocator.reload(newPod)
	return allocator.Allocate(newPod)
}

func (allocator *CPUAllocator) reload(pod *v1.Pod) {
	// Remove current pod from node cache for computing
	oldSh := allocator.pool.shareCntRef
	oldEx := allocator.pool.exclusiveCntRef
	glog.V(5).Infof("[CPUAllocator]oldSh:%v, oldEx:%v", oldSh, oldEx)
	_ = allocator.nodeInfo.RemovePod(pod)
	newSh := allocator.pool.shareCntRef
	newEx := allocator.pool.exclusiveCntRef
	glog.V(5).Infof("[CPUAllocator]newSh:%v, newEx:%v", newSh, newEx)
	allocator.pool.Initialize()
}

func (allocator *CPUAllocator) allocateShareCPUSet(pod *v1.Pod, container *v1.Container) (cpuset.CPUSet, error) {
	overRatio, _ := util.CPUOverQuotaRatio(allocator.pool.nodeInfo.Node())
	req := container.Resources.Requests[v1.ResourceCPU]
	numCPUs := ContainerCPUCount(container) // it is already round up, ex 1700m => 2
	reqMilli := req.MilliValue()
	newRequested := int64(float64(reqMilli+allocator.pool.GetAllocatedSharedCPUSetReq()) / overRatio)
	currentCPUSetMilli := int64(float64(allocator.pool.GetSharedCPUSet().Size()) * 1000 * overRatio)
	if newRequested <= currentCPUSetMilli && numCPUs <= allocator.pool.GetSharedCPUSet().Size() { // current cpuset is enough for the container
		// Step 1: calculate current cpuset, if available, get the least used cpuset
		glog.V(5).Infof("taking %d CPUs from existing shared CPUSet pool")
		existing := allocator.pool.LeaseUsedSharedCPUSet(numCPUs)
		return existing, nil
	} else {
		// Step 2: if not enough

		// Step 2.1: if request not enough
		neededMilli := newRequested - currentCPUSetMilli
		neededNumCPUsForReq := int((float64(neededMilli) + (1000*overRatio - 1)) / 1000) // New from pool

		neededNumCPUs := neededNumCPUsForReq
		if neededForLimit := numCPUs - (allocator.pool.GetSharedCPUSet().Size() + neededNumCPUsForReq); neededForLimit > 0 {
			// Container limit is larger than existing Shared CPUSet pool
			if neededForLimit > neededNumCPUs {
				glog.V(5).Infof("taking %d new CPUs for CPU limit", neededForLimit)
				neededNumCPUs = neededForLimit
			}

		}
		ret, err := takeByTopology(allocator.pool.Topology(), allocator.pool.GetCPUSharePoolCPUSet(), neededNumCPUs)
		if err != nil {
			err = fmt.Errorf("[CPUAllocator]failed to allocate %d CPUs from CPUShare pool: %s", neededNumCPUs, err.Error())
			return cpuset.CPUSet{}, err
		}
		glog.V(3).Infof("allocated %s (%d) CPUs for pod %s", ret.String(), ret.Size(), ContainerName(pod, container))
		numCPUsFromExisting := numCPUs - neededNumCPUs
		// then take least used cpu from current SharedCPUSet pool

		existing := allocator.pool.LeaseUsedSharedCPUSet(numCPUsFromExisting)
		if existing.Size() != numCPUsFromExisting {
			err = fmt.Errorf("[CPUAllocator]failed to take least used CPUSet from SharedCPUSet pool, expected: %d CPUs, actual: %s", numCPUsFromExisting, existing.String())
			return cpuset.CPUSet{}, err
		}
		return existing.Union(ret), nil
	}
	// Step 2.2: if limit not enough

	//return cpuset.CPUSet{}, nil

}

func (allocator *CPUAllocator) allocateExclusiveCPUSet(pod *v1.Pod, container *v1.Container) (cpuset.CPUSet, error) {
	// Step 1: Take directly from CPUShare pool
	numCPUs := ContainerCPUCount(container)
	glog.V(3).Infof("[CPUAllocator]allocating %d cpu for %s", ContainerName(pod, container))
	if allocator.pool.AvailableCPUs() >= int(numCPUs) {
		glog.V(5).Infof("[CPUAllocator]available CPUs from exclusive container", allocator.pool.AvailableCPUs()-int(numCPUs))
		result, err := takeByTopology(allocator.pool.Topology(), allocator.pool.GetCPUSharePoolCPUSet(), int(numCPUs))
		if err != nil {
			return cpuset.CPUSet{}, err
		}
		glog.V(5).Infof("[CPUAllocator]allocated cpuset %s for %s", result.String(), ContainerName(pod, container))
		return result, nil
	} else {
		glog.Error("[CPUAllocator]not enough cpuset from CPUShare pool, Available CPU count is %s, allocated millicpu: %d",
			allocator.pool.AvailableCPUs(), allocator.pool.GetAllocatedCPUShare())
	}
	// Step 2: if not enough fail immediately
	// TODO(yuzhi.wx) need to enhance later for shrink the cpuset
	// Step 2.1: shrink the CPUSet share pool, and then patch relevant pods
	// Step 2.2: allocate for the current pod
	return cpuset.CPUSet{}, fmt.Errorf("failed to allocated %d CPUs for container %s", numCPUs, ContainerName(pod, container))

}

// allocateAllCPUSet supports BindingAll cpu strategy
// allocate all cpu for container but does not increase reference count
// TODO(yuzhi.wx) need to add when needed
func (allocator *CPUAllocator) allocateAllCPUSet(pod *v1.Pod, container *v1.Container) ContainerCPUAssignments {
	return ContainerCPUAssignments{}
}

// NodeCPUSharePool returns the latest CPUShare pool after pod cpu assignment/reclaim
func (allocator *CPUAllocator) NodeCPUSharePool() cpuset.CPUSet {
	return allocator.pool.GetCPUSharePoolCPUSet()
}

// resetPool reset the underliying pool status with new nodeInfo
// NOTE: only use for testing purpose
func (allocator *CPUAllocator) resetPool(nodeInfo *schedulercache.NodeInfo) {
	allocator.nodeInfo = nodeInfo
	allocator.pool.nodeInfo = nodeInfo
	allocator.pool.Initialize()
}
