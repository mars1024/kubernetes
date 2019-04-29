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

package servicenetallocator

import (
	"net"
	"sync"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apiserver/pkg/server/storage"
	api "k8s.io/kubernetes/pkg/apis/core"
	coreclient "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/internalversion"
	"k8s.io/kubernetes/pkg/registry/core/rangeallocation"
	"k8s.io/kubernetes/pkg/registry/core/service/allocator"
	serviceallocator "k8s.io/kubernetes/pkg/registry/core/service/allocator/storage"
	"k8s.io/kubernetes/pkg/registry/core/service/ipallocator"
	"k8s.io/kubernetes/pkg/registry/core/service/portallocator"
)

type AllocatorFactory struct {
	mu             sync.Mutex // protects ipAllocators and portAllocators
	storageFactory storage.StorageFactory
	ipAllocators   map[string]ipallocator.Interface
	portAllocators map[string]portallocator.Interface
}

func (af *AllocatorFactory) IPAllocatorForCluster(cluster *v1alpha1.MinionCluster, serviceGetter coreclient.ServicesGetter, eventGetter coreclient.EventsGetter) (ipallocator.Interface, error) {
	af.mu.Lock()
	defer af.mu.Unlock()

	memIPAllocator, ok := af.ipAllocators[cluster.Name]
	if ok {
		return memIPAllocator, nil
	}

	storageConfig, err := af.storageFactory.NewConfig(schema.GroupResource{
		Group:    "",
		Resource: "serviceipallocations",
	})
	if err != nil {
		return nil, err
	}
	var serviceClusterIPRegistry rangeallocation.RangeRegistry
	_, cidr, _ := net.ParseCIDR(cluster.Spec.Networking.ServiceClusterIPRange)
	serviceClusterIPAllocator := ipallocator.NewAllocatorCIDRRange(cidr, func(max int, rangeSpec string) allocator.Interface {
		mem := allocator.NewAllocationMap(max, rangeSpec)
		// TODO etcdallocator package to return a storage interface via the storageFactory
		etcd := serviceallocator.NewEtcd(mem,
			"/ranges/"+cluster.Name+"/serviceips",
			api.Resource("serviceipallocations"), storageConfig)
		serviceClusterIPRegistry = etcd
		return etcd
	})

	// DIRTY-HACK(zuoxiu.jm): disabling repairing
	/*
		repairClusterIPs := ipallocatorcontroller.NewRepair(0, serviceGetter, eventGetter, cidr, serviceClusterIPRegistry)
		if err := repairClusterIPs.RunOnce(); err != nil {
			return nil, err
		}
	*/
	af.ipAllocators[cluster.Name] = serviceClusterIPAllocator

	return serviceClusterIPAllocator, nil
}

func (af *AllocatorFactory) NodePortAllocatorForCluster(cluster *v1alpha1.MinionCluster, serviceGetter coreclient.ServicesGetter, eventGetter coreclient.EventsGetter) (portallocator.Interface, error) {
	af.mu.Lock()
	defer af.mu.Unlock()

	nodePortAllocator, ok := af.portAllocators[cluster.Name]
	if ok {
		return nodePortAllocator, nil
	}
	storageConfig, err := af.storageFactory.NewConfig(schema.GroupResource{
		Group:    "",
		Resource: "servicenodeportallocations",
	})
	if err != nil {
		return nil, err
	}

	portRange := utilnet.PortRange{
		Base: int(cluster.Spec.Networking.ServiceNodePortRange.Base),
		Size: int(cluster.Spec.Networking.ServiceNodePortRange.Size),
	}
	var serviceNodePortRegistry rangeallocation.RangeRegistry
	nodePortAllocator = portallocator.NewPortAllocatorCustom(portRange, func(max int, rangeSpec string) allocator.Interface {
		mem := allocator.NewAllocationMap(max, rangeSpec)
		// TODO etcdallocator package to return a storage interface via the storageFactory
		etcd := serviceallocator.NewEtcd(mem, "/ranges/"+cluster.Name+"/servicenodeports", api.Resource("servicenodeportallocations"), storageConfig)
		serviceNodePortRegistry = etcd
		return etcd
	})

	// DIRTY-HACK(zuoxiu.jm): disabling repairing
	/*
	repairNodePorts := portallocatorcontroller.NewRepair(0, serviceGetter, eventGetter, portRange, serviceNodePortRegistry)
	if err := repairNodePorts.RunOnce(); err != nil {
		return nil, err
	}
	*/

	nodePortAllocator = portallocator.NewPortAllocator(portRange)
	af.portAllocators[cluster.Name] = nodePortAllocator
	return nodePortAllocator, nil
}
