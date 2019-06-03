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
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/topology"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/kubernetes/pkg/scheduler/cache"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
	"k8s.io/kubernetes/pkg/scheduler/util"
)

type CPUPool struct {
	nodeInfo        *cache.NodeInfo
	shareCntRef     CPUCntRef // CPU count ref by CPU ID
	exclusiveCntRef CPUCntRef // ex CPU count ref by CPU ID
	// cached the Node state
	cachedNodeCPUSet *cpuset.CPUSet
	top              *topology.CPUTopology
	overRatio        float64
}

type CPUCntRef map[int]int

func (ref CPUCntRef) CPUs() cpuset.CPUSet {
	builder := cpuset.NewBuilder()
	for cpu := range ref {
		builder.Add(cpu)
	}
	return builder.Result()
}

func (ref CPUCntRef) Clone() CPUCntRef {
	cloned := make(CPUCntRef, len(ref))
	for k, v := range ref {
		cloned[k] = v
	}
	return cloned
}
func (ref CPUCntRef) Increase(cpuId int) {
	value := ref[cpuId]
	ref[cpuId] = value + 1
}

func (ref CPUCntRef) Decrease(cpuId int) {
	value := ref[cpuId]
	if value > 0 {
		value -= 1
	}
	ref[cpuId] = value
	if value == 0 {
		delete(ref, cpuId)
	}
}

// LeastUsedCPUSet
// 拿被绑定次数最小的CPU,注意: 返回的CPUSet可能小于请求的numCpu
func (ref CPUCntRef) LeastUsedCPUSet(origin CPUCntRef, numCpu int) cpuset.CPUSet {
	if origin == nil {
		origin = ref.Clone()
	}
	if numCpu > 1 {
		ret := cpuset.NewCPUSet()
		for i := 0; i < numCpu; i++ {
			newAllocate := ref.LeastUsedCPUSet(origin, 1)
			ret = ret.Union(newAllocate)
			for _, cpu := range newAllocate.ToSlice() {
				delete(origin, cpu)
			}
		}
		return ret
	} else if numCpu == 1 {
		max := -1
		min := -1
		minUseCpu := -1

		for cpu, useCount := range origin {
			if max < 0 {
				max = useCount
				min = useCount
				minUseCpu = cpu
				continue
			}

			if useCount > max {
				max = useCount
			}

			if useCount < min {
				min = useCount
				minUseCpu = cpu
			}
		}

		if max >= min && max > -1 {
			return cpuset.NewCPUSet(minUseCpu)
		} else {
			return cpuset.NewBuilder().Result()
		}
	} else {
		return cpuset.CPUSet{}
	}
}

func NewCPUPool(nodeInfo *cache.NodeInfo) *CPUPool {
	pool := &CPUPool{
		nodeInfo: nodeInfo,
	}
	pool.Initialize()
	return pool
}

func (pool *CPUPool) Topology() *topology.CPUTopology {
	return pool.top
}

// RefreshRefCount refreshes the reference count by adding new ContainerCPUAssignments
func (pool *CPUPool) RefreshRefCount(set cpuset.CPUSet, exclusive bool) bool {
	refToAdd := pool.shareCntRef
	if exclusive {
		refToAdd = pool.exclusiveCntRef
	}
	for _, v := range set.ToSlice() {
		refToAdd.Increase(v)
	}
	return true
}

func (pool *CPUPool) LeaseUsedSharedCPUSet(numCPUs int) cpuset.CPUSet {
	return pool.shareCntRef.LeastUsedCPUSet(nil, numCPUs)
}

func (pool *CPUPool) Initialize() {
	if pool.nodeInfo == nil {
		glog.Fatal("no node info to initialize CPUPool")
	}
	top, err := pool.parseNodeCPUInfo(pool.nodeInfo.Node())
	if err != nil || top.NumCPUs <= 0 {
		glog.Error("fatal error %s, or NumCPUS=%d", err, 0)
		glog.Fatal(err) // Shit
	}
	pool.exclusiveCntRef = make(map[int]int, 0)
	pool.shareCntRef = make(map[int]int, 0)
	for _, pod := range pool.nodeInfo.Pods() {
		for _, c := range pod.Spec.Containers {
			if IsExclusiveContainer(pod, &c) {
				set, _ := getPodCPUSet(pod)
				pool.RefreshRefCount(set, true)
				continue
			}
			if IsSharedCPUSetPod(pod) {
				set, _ := getPodCPUSet(pod)
				pool.RefreshRefCount(set, false)
				break
			}
		}
	}
	glog.V(5).Infof("CPUPool for node %s: shareref: %v, exclusiveref: %v",
		pool.nodeInfo.Node().Name, pool.shareCntRef.CPUs(), pool.exclusiveCntRef.CPUs())
	pool.top = top

}

func (pool *CPUPool) GetNodeCPUSet() cpuset.CPUSet {
	if pool.cachedNodeCPUSet != nil { // speed up the computation
		return *pool.cachedNodeCPUSet
	}
	info, err := pool.parseNodeCPUInfo(pool.nodeInfo.Node())
	if err != nil {
		glog.Errorf("error getting the cpu INFO: %s", err.Error())
		return cpuset.CPUSet{}
	}
	return info.CPUDetails.CPUs()
}

func (pool *CPUPool) GetExclusiveCPUSet() cpuset.CPUSet {
	return pool.exclusiveCntRef.CPUs()
}

func (pool *CPUPool) GetSharedCPUSet() cpuset.CPUSet {
	return pool.shareCntRef.CPUs()
}

func (pool *CPUPool) GetNonExclusiveCPUSet() cpuset.CPUSet {
	return pool.GetNodeCPUSet().Difference(pool.exclusiveCntRef.CPUs())
}

// GetCPUSharePoolCPUSet return the cpuset from
// CPUShare pool so that CPUSet pod can use:
// 1. CPUShare Pool
// 2. Shared CPUSet Pool
func (pool *CPUPool) GetCPUSharePoolCPUSet() cpuset.CPUSet {
	return pool.GetNonExclusiveCPUSet().Difference(pool.GetSharedCPUSet())
}

// AvailableCPUs returns available CPU count for exclusive container
func (pool *CPUPool) AvailableCPUs() int {
	return pool.GetCPUSharePoolCPUSet().Size() - pool.CPUShareOccupiedCPUs()
}

// CPUShareOccupiedCPUs returns CPU Numbers ocupied by CPUShare pods with rounded up
// also includes the over ratio
func (pool *CPUPool) CPUShareOccupiedCPUs() int {
	overRatio := pool.NodeOverRatio()
	return int(float64(pool.GetAllocatedCPUShare()+int64(overRatio*float64(1000)-1)) / (overRatio * 1000))
}
func (pool *CPUPool) GetAllocatedCPUShare() int64 {
	pods := pool.nodeInfo.Pods()
	//cpuRatios, _ := util.CPUOverQuotaRatio(pool.nodeInfo.Node())

	allocated := int64(0)
	for _, pod := range pods {
		if IsSharedCPUSetPod(pod) {
			continue
		}
		isExclusive := false
		for _, c := range pod.Spec.Containers {
			if IsExclusiveContainer(pod, &c) {
				// TODO(yuzhi.wx) if any container is exclusive, skip the whole pod
				isExclusive = true
				break
			}
		}
		if isExclusive {
			continue
		}
		res, _, _ := schedulercache.CalculateResource(pod)
		milliCPU := res.MilliCPU
		allocated += int64(float64(milliCPU))
	}
	return allocated
}

func (pool *CPUPool) GetAllocatedSharedCPUSetReq() int64 {
	pods := pool.nodeInfo.Pods()
	//cpuRatios, _ := util.CPUOverQuotaRatio(pool.nodeInfo.Node())

	allocated := int64(0)
	for _, pod := range pods {
		if !IsSharedCPUSetPod(pod) {
			continue
		}
		//if IsExclusiveContainer(pod, nil) {
		//	// TODO(yuzhi.wx) if any container is exclusive, skip the whole pod
		//	continue
		//}
		//
		_, milliCPU, _ := schedulercache.CalculateResource(pod)
		allocated += milliCPU
	}
	return allocated
}

func (pool *CPUPool) NodeOverRatio() float64 {
	if pool.overRatio == 0 {
		value, _ := util.CPUOverQuotaRatio(pool.nodeInfo.Node())
		return value
	}
	return pool.overRatio
}

func (pool *CPUPool) parseNodeCPUInfo(node *v1.Node) (*topology.CPUTopology, error) {

	localInfo := util.LocalInfoFromNode(node)
	if localInfo != nil {
		CPUDetails := topology.CPUDetails{}

		cpuInfos := localInfo.CPUInfos
		numSockets := make(map[int]string, 0)
		numCores := make(map[int]string, 0)
		for _, info := range cpuInfos {
			CPUDetails[int(info.CPUID)] = topology.CPUInfo{
				SocketID: int(info.SocketID),
				CoreID:   int(info.CoreID),
			}
			numSockets[int(info.SocketID)] = ""
			numCores[int(info.CoreID)] = ""
		}
		return &topology.CPUTopology{
			NumCPUs:    len(cpuInfos),
			NumSockets: len(numSockets),
			NumCores:   len(numCores),
			CPUDetails: CPUDetails,
		}, nil
	}
	return nil, fmt.Errorf("no cpu info from node annotation")
}
