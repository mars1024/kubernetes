package dockershim

import (
	"testing"

	dockerblkiodev "github.com/docker/docker/api/types/blkiodev"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerunits "github.com/docker/go-units"
	"github.com/stretchr/testify/assert"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func TestConvertToDockerResources(t *testing.T) {
	memorySwappiness := int64(1000)
	dockerMemorySwappiness := int64(1000)
	blkioWeightDevice := &runtimeapi.WeightDevice{
		Path:   "/dev/sda",
		Weight: 100,
	}
	dockerBlkioWeightDevice := &dockerblkiodev.WeightDevice{
		Path:   "/dev/sda",
		Weight: 100,
	}
	blkioDeviceReadBps := &runtimeapi.ThrottleDevice{
		Path: "/dev/sda",
		Rate: 1 * 1024 * 1024,
	}
	dockerBlkioDeviceReadBps := &dockerblkiodev.ThrottleDevice{
		Path: "/dev/sda",
		Rate: 1 * 1024 * 1024,
	}
	blkioDeviceWriteBps := &runtimeapi.ThrottleDevice{
		Path: "/dev/sdb",
		Rate: 1 * 1024 * 1024,
	}
	dockerBlkioDeviceWriteBps := &dockerblkiodev.ThrottleDevice{
		Path: "/dev/sdb",
		Rate: 1 * 1024 * 1024,
	}
	blkioDeviceReadIOps := &runtimeapi.ThrottleDevice{
		Path: "/dev/sdc",
		Rate: 10,
	}
	dockerBlkioDeviceReadIOps := &dockerblkiodev.ThrottleDevice{
		Path: "/dev/sdc",
		Rate: 10,
	}
	blkioDeviceWriteIOps := &runtimeapi.ThrottleDevice{
		Path: "/dev/sdd",
		Rate: 10,
	}
	dockerBlkioDeviceWriteIOps := &dockerblkiodev.ThrottleDevice{
		Path: "/dev/sdd",
		Rate: 10,
	}
	ulimit1 := &runtimeapi.Ulimit{
		Name: "nofile",
		Hard: 20480,
		Soft: 40960,
	}
	ulimit2 := &runtimeapi.Ulimit{
		Name: "nproc",
		Hard: 1024,
		Soft: 2048,
	}
	dockerUlimit1 := &dockerunits.Ulimit{
		Name: "nofile",
		Hard: 20480,
		Soft: 40960,
	}
	dockerUlimit2 := &dockerunits.Ulimit{
		Name: "nproc",
		Hard: 1024,
		Soft: 2048,
	}
	CRIResources := runtimeapi.LinuxContainerResources{
		CpuPeriod:             1000,
		CpuQuota:              100,
		CpuShares:             100,
		MemoryLimitInBytes:    1024 * 1024 * 1024,
		BlkioWeight:           100,
		MemorySwappiness:      &runtimeapi.Int64Value{Value: memorySwappiness},
		BlkioWeightDevice:     []*runtimeapi.WeightDevice{blkioWeightDevice},
		BlkioDeviceReadBps:    []*runtimeapi.ThrottleDevice{blkioDeviceReadBps},
		BlkioDeviceWriteBps:   []*runtimeapi.ThrottleDevice{blkioDeviceWriteBps},
		BlkioDeviceRead_IOps:  []*runtimeapi.ThrottleDevice{blkioDeviceReadIOps},
		BlkioDeviceWrite_IOps: []*runtimeapi.ThrottleDevice{blkioDeviceWriteIOps},
		Ulimits:               []*runtimeapi.Ulimit{ulimit1, ulimit2},
	}

	dockerResources := *toDockerResources(&CRIResources)

	expectDockerResources := dockercontainer.Resources{
		CPUPeriod:            1000,
		CPUQuota:             100,
		CPUShares:            100,
		Memory:               1024 * 1024 * 1024,
		BlkioWeight:          100,
		MemorySwappiness:     &dockerMemorySwappiness,
		BlkioWeightDevice:    []*dockerblkiodev.WeightDevice{dockerBlkioWeightDevice},
		BlkioDeviceReadBps:   []*dockerblkiodev.ThrottleDevice{dockerBlkioDeviceReadBps},
		BlkioDeviceWriteBps:  []*dockerblkiodev.ThrottleDevice{dockerBlkioDeviceWriteBps},
		BlkioDeviceReadIOps:  []*dockerblkiodev.ThrottleDevice{dockerBlkioDeviceReadIOps},
		BlkioDeviceWriteIOps: []*dockerblkiodev.ThrottleDevice{dockerBlkioDeviceWriteIOps},
		Ulimits:              []*dockerunits.Ulimit{dockerUlimit1, dockerUlimit2},
	}
	assert.Equal(t, expectDockerResources, dockerResources)
}

func TestConvertToRuntimeAPIResources(t *testing.T) {
	memorySwappiness := int64(1000)
	criMemorySwappiness := int64(1000)
	blkioWeightDevice := &dockerblkiodev.WeightDevice{
		Path:   "/dev/sda",
		Weight: 100,
	}
	criBlkioWeightDevice := &runtimeapi.WeightDevice{
		Path:   "/dev/sda",
		Weight: 100,
	}
	blkioDeviceReadBps := &dockerblkiodev.ThrottleDevice{
		Path: "/dev/sda",
		Rate: 1 * 1024 * 1024,
	}
	criBlkioDeviceReadBps := &runtimeapi.ThrottleDevice{
		Path: "/dev/sda",
		Rate: 1 * 1024 * 1024,
	}
	blkioDeviceWriteBps := &dockerblkiodev.ThrottleDevice{
		Path: "/dev/sdb",
		Rate: 1 * 1024 * 1024,
	}
	criBlkioDeviceWriteBps := &runtimeapi.ThrottleDevice{
		Path: "/dev/sdb",
		Rate: 1 * 1024 * 1024,
	}
	blkioDeviceReadIOps := &dockerblkiodev.ThrottleDevice{
		Path: "/dev/sdc",
		Rate: 10,
	}
	criBlkioDeviceReadIOps := &runtimeapi.ThrottleDevice{
		Path: "/dev/sdc",
		Rate: 10,
	}
	blkioDeviceWriteIOps := &dockerblkiodev.ThrottleDevice{
		Path: "/dev/sdd",
		Rate: 10,
	}
	criBlkioDeviceWriteIOps := &runtimeapi.ThrottleDevice{
		Path: "/dev/sdd",
		Rate: 10,
	}
	ulimit1 := &dockerunits.Ulimit{
		Name: "nofile",
		Hard: 20480,
		Soft: 40960,
	}
	ulimit2 := &dockerunits.Ulimit{
		Name: "nproc",
		Hard: 1024,
		Soft: 2048,
	}
	criUlimit1 := &runtimeapi.Ulimit{
		Name: "nofile",
		Hard: 20480,
		Soft: 40960,
	}
	criUlimit2 := &runtimeapi.Ulimit{
		Name: "nproc",
		Hard: 1024,
		Soft: 2048,
	}
	DockerResources := dockercontainer.Resources{
		CPUPeriod:            1000,
		CPUQuota:             100,
		CPUShares:            100,
		Memory:               1024 * 1024 * 1024,
		BlkioWeight:          100,
		MemorySwappiness:     &memorySwappiness,
		BlkioWeightDevice:    []*dockerblkiodev.WeightDevice{blkioWeightDevice},
		BlkioDeviceReadBps:   []*dockerblkiodev.ThrottleDevice{blkioDeviceReadBps},
		BlkioDeviceWriteBps:  []*dockerblkiodev.ThrottleDevice{blkioDeviceWriteBps},
		BlkioDeviceReadIOps:  []*dockerblkiodev.ThrottleDevice{blkioDeviceReadIOps},
		BlkioDeviceWriteIOps: []*dockerblkiodev.ThrottleDevice{blkioDeviceWriteIOps},
		Ulimits:              []*dockerunits.Ulimit{ulimit1, ulimit2},
	}

	CRIResources := *toRuntimeAPIResources(&DockerResources)
	expectCRIResources := runtimeapi.LinuxContainerResources{
		CpuPeriod:             1000,
		CpuQuota:              100,
		CpuShares:             100,
		MemoryLimitInBytes:    1024 * 1024 * 1024,
		BlkioWeight:           100,
		MemorySwappiness:      &runtimeapi.Int64Value{Value: criMemorySwappiness},
		BlkioWeightDevice:     []*runtimeapi.WeightDevice{criBlkioWeightDevice},
		BlkioDeviceReadBps:    []*runtimeapi.ThrottleDevice{criBlkioDeviceReadBps},
		BlkioDeviceWriteBps:   []*runtimeapi.ThrottleDevice{criBlkioDeviceWriteBps},
		BlkioDeviceRead_IOps:  []*runtimeapi.ThrottleDevice{criBlkioDeviceReadIOps},
		BlkioDeviceWrite_IOps: []*runtimeapi.ThrottleDevice{criBlkioDeviceWriteIOps},
		Ulimits:               []*runtimeapi.Ulimit{criUlimit1, criUlimit2},
	}

	assert.Equal(t, expectCRIResources, CRIResources)
}
