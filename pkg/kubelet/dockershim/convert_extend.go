package dockershim

import (
	dockerblkiodev "github.com/docker/docker/api/types/blkiodev"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerunits "github.com/docker/go-units"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

// toDockerResources() can convert runtimeapi.LinuxContainerResources in CRI to dockercontainer.Resources in Docker side.
func toDockerResources(resources *runtimeapi.LinuxContainerResources) *dockercontainer.Resources {
	if resources == nil {
		return nil
	}
	dockerResources := dockercontainer.Resources{
		CPUPeriod:         resources.CpuPeriod,
		CPUQuota:          resources.CpuQuota,
		CPUShares:         resources.CpuShares,
		Memory:            resources.MemoryLimitInBytes,
		CpusetCpus:        resources.CpusetCpus,
		CpusetMems:        resources.CpusetMems,
		BlkioWeight:       uint16(resources.BlkioWeight),
		KernelMemory:      resources.KernelMemory,
		MemoryReservation: resources.MemoryReservation,
	}
	if resources.MemorySwappiness != nil {
		dockerResources.MemorySwappiness = &resources.MemorySwappiness.Value
	}
	// Convert BlkioWeightDevice to Docker definition.
	if resources.BlkioWeightDevice != nil {
		targetWeightDevices := []*dockerblkiodev.WeightDevice{}
		for _, weightDevice := range resources.BlkioWeightDevice {
			target := &dockerblkiodev.WeightDevice{
				Path:   weightDevice.Path,
				Weight: uint16(weightDevice.Weight),
			}
			targetWeightDevices = append(targetWeightDevices, target)
		}
		if len(targetWeightDevices) > 0 {
			dockerResources.BlkioWeightDevice = targetWeightDevices
		}
	}
	// Convert BlkioDeviceReadBps to Docker definition.
	if resources.BlkioDeviceReadBps != nil {
		targetDeviceReadBps := []*dockerblkiodev.ThrottleDevice{}
		for _, deviceReadBps := range resources.BlkioDeviceReadBps {
			target := &dockerblkiodev.ThrottleDevice{
				Path: deviceReadBps.Path,
				Rate: deviceReadBps.Rate,
			}
			targetDeviceReadBps = append(targetDeviceReadBps, target)
		}
		if len(targetDeviceReadBps) > 0 {
			dockerResources.BlkioDeviceReadBps = targetDeviceReadBps
		}
	}
	// Convert BlkioDeviceWriteBps to Docker definition.
	if resources.BlkioDeviceWriteBps != nil {
		targetDeviceWriteBps := []*dockerblkiodev.ThrottleDevice{}
		for _, deviceWriteBps := range resources.BlkioDeviceWriteBps {
			target := &dockerblkiodev.ThrottleDevice{
				Path: deviceWriteBps.Path,
				Rate: deviceWriteBps.Rate,
			}
			targetDeviceWriteBps = append(targetDeviceWriteBps, target)
		}
		if len(targetDeviceWriteBps) > 0 {
			dockerResources.BlkioDeviceWriteBps = targetDeviceWriteBps
		}
	}
	// Convert BlkioDeviceReadIOps to Docker definition.
	if resources.BlkioDeviceRead_IOps != nil {
		targetDeviceReadIOps := []*dockerblkiodev.ThrottleDevice{}
		for _, deviceReadIOps := range resources.BlkioDeviceRead_IOps {
			target := &dockerblkiodev.ThrottleDevice{
				Path: deviceReadIOps.Path,
				Rate: deviceReadIOps.Rate,
			}
			targetDeviceReadIOps = append(targetDeviceReadIOps, target)
		}
		if len(targetDeviceReadIOps) > 0 {
			dockerResources.BlkioDeviceReadIOps = targetDeviceReadIOps
		}
	}
	// Convert BlkioDeviceWriteIOps to Docker definition.
	if resources.BlkioDeviceWrite_IOps != nil {
		targetDeviceWriteIOps := []*dockerblkiodev.ThrottleDevice{}
		for _, deviceWriteIOps := range resources.BlkioDeviceWrite_IOps {
			target := &dockerblkiodev.ThrottleDevice{
				Path: deviceWriteIOps.Path,
				Rate: deviceWriteIOps.Rate,
			}
			targetDeviceWriteIOps = append(targetDeviceWriteIOps, target)
		}
		if len(targetDeviceWriteIOps) > 0 {
			dockerResources.BlkioDeviceWriteIOps = targetDeviceWriteIOps
		}
	}
	// Convert Ulimits to Docker definition.
	if resources.Ulimits != nil {
		targetUlimits := []*dockerunits.Ulimit{}
		for _, Ulimit := range resources.Ulimits {
			target := &dockerunits.Ulimit{
				Name: Ulimit.Name,
				Hard: Ulimit.Hard,
				Soft: Ulimit.Soft,
			}
			targetUlimits = append(targetUlimits, target)
		}
		if len(targetUlimits) > 0 {
			dockerResources.Ulimits = targetUlimits
		}
	}
	return &dockerResources
}

// toRuntimeAPIResources() can convert dockercontainer.Resources  to runtimeapi.LinuxContainerResources in CRI.
func toRuntimeAPIResources(resources *dockercontainer.Resources) *runtimeapi.LinuxContainerResources {
	if resources == nil {
		return nil
	}
	CRIResources := runtimeapi.LinuxContainerResources{
		CpuPeriod:          resources.CPUPeriod,
		CpuQuota:           resources.CPUQuota,
		CpuShares:          resources.CPUShares,
		MemoryLimitInBytes: resources.Memory,
		CpusetCpus:         resources.CpusetCpus,
		CpusetMems:         resources.CpusetMems,
		// resources.BlkioWeight's type is uint16 in container side definition.
		BlkioWeight:       uint32(resources.BlkioWeight),
		KernelMemory:      resources.KernelMemory,
		MemoryReservation: resources.MemoryReservation,
	}
	if resources.MemorySwappiness != nil {
		CRIResources.MemorySwappiness = &runtimeapi.Int64Value{Value: *resources.MemorySwappiness}
	}
	// Convert BlkioWeightDevice to CRI definition.
	if resources.BlkioWeightDevice != nil {
		targetWeightDevices := []*runtimeapi.WeightDevice{}
		for _, weightDevice := range resources.BlkioWeightDevice {
			target := &runtimeapi.WeightDevice{
				Path: weightDevice.Path,
				// weightDevice.Weight's type is uint16 in container side definition.
				Weight: uint32(weightDevice.Weight),
			}
			targetWeightDevices = append(targetWeightDevices, target)
		}
		if len(targetWeightDevices) > 0 {
			CRIResources.BlkioWeightDevice = targetWeightDevices
		}
	}
	// Convert BlkioDeviceReadBps to CRI definition.
	if resources.BlkioDeviceReadBps != nil {
		targetDeviceReadBps := []*runtimeapi.ThrottleDevice{}
		for _, deviceReadBps := range resources.BlkioDeviceReadBps {
			target := &runtimeapi.ThrottleDevice{
				Path: deviceReadBps.Path,
				Rate: deviceReadBps.Rate,
			}
			targetDeviceReadBps = append(targetDeviceReadBps, target)
		}
		if len(targetDeviceReadBps) > 0 {
			CRIResources.BlkioDeviceReadBps = targetDeviceReadBps
		}
	}
	// Convert BlkioDeviceWriteBps to CRI definition.
	if resources.BlkioDeviceWriteBps != nil {
		targetDeviceWriteBps := []*runtimeapi.ThrottleDevice{}
		for _, deviceWriteBps := range resources.BlkioDeviceWriteBps {
			target := &runtimeapi.ThrottleDevice{
				Path: deviceWriteBps.Path,
				Rate: deviceWriteBps.Rate,
			}
			targetDeviceWriteBps = append(targetDeviceWriteBps, target)
		}
		if len(targetDeviceWriteBps) > 0 {
			CRIResources.BlkioDeviceWriteBps = targetDeviceWriteBps
		}
	}
	// Convert BlkioDeviceReadIOps to CRI definition.
	if resources.BlkioDeviceReadIOps != nil {
		targetDeviceReadIOps := []*runtimeapi.ThrottleDevice{}
		for _, deviceReadIOps := range resources.BlkioDeviceReadIOps {
			target := &runtimeapi.ThrottleDevice{
				Path: deviceReadIOps.Path,
				Rate: deviceReadIOps.Rate,
			}
			targetDeviceReadIOps = append(targetDeviceReadIOps, target)
		}
		if len(targetDeviceReadIOps) > 0 {
			CRIResources.BlkioDeviceRead_IOps = targetDeviceReadIOps
		}
	}
	// Convert BlkioDeviceWriteIOps to CRI definition.
	if resources.BlkioDeviceWriteIOps != nil {
		targetDeviceWriteIOps := []*runtimeapi.ThrottleDevice{}
		for _, deviceWriteIOps := range resources.BlkioDeviceWriteIOps {
			target := &runtimeapi.ThrottleDevice{
				Path: deviceWriteIOps.Path,
				Rate: deviceWriteIOps.Rate,
			}
			targetDeviceWriteIOps = append(targetDeviceWriteIOps, target)
		}
		if len(targetDeviceWriteIOps) > 0 {
			CRIResources.BlkioDeviceWrite_IOps = targetDeviceWriteIOps
		}
	}
	// Convert Ulimits to CRI definition.
	if resources.Ulimits != nil {
		targetUlimits := []*runtimeapi.Ulimit{}
		for _, Ulimit := range resources.Ulimits {
			target := &runtimeapi.Ulimit{
				Name: Ulimit.Name,
				Hard: Ulimit.Hard,
				Soft: Ulimit.Soft,
			}
			targetUlimits = append(targetUlimits, target)
		}
		if len(targetUlimits) > 0 {
			CRIResources.Ulimits = targetUlimits
		}
	}
	return &CRIResources
}
