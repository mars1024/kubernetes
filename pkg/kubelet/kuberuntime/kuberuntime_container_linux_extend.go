// +build linux

package kuberuntime

import (
	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	sigmautil "k8s.io/kubernetes/pkg/kubelet/sigma"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
)

// AdjustResourcesByAnnotation adjusts container resource requirement(currently only adjust cpu period) according
// to pod annotations.
func AdjustResourcesByAnnotation(pod *v1.Pod, containerName string, resources *runtimeapi.LinuxContainerResources, milliCPU int64) {
	currentCpuPeriod := resources.CpuPeriod
	if currentCpuPeriod == 0 {
		return
	}
	newCpuPeriod := sigmautil.GetCpuPeriodFromAnnotation(pod, containerName)
	if newCpuPeriod < quotaPeriod {
		return
	}
	newCpuQuota := (milliCPU * newCpuPeriod) / milliCPUToCPU
	if newCpuQuota < minQuotaPeriod {
		newCpuQuota = minQuotaPeriod
	}
	resources.CpuPeriod = newCpuPeriod
	resources.CpuQuota = newCpuQuota
}

// applyExtendContainerConfig can merge extended feilds of hostconfig into container config of CRI.
func applyExtendContainerConfig(pod *v1.Pod, container *v1.Container, config *runtimeapi.ContainerConfig) {
	// Set NetPriority field
	netpriority := int64(sigmautil.GetNetPriorityFromAnnotation(pod))
	config.NetPriority = netpriority

	// Ensure QuotaId exists in container when DiskQuota is set.
	if len(config.Linux.Resources.DiskQuota) != 0 && config.QuotaId == "" {
		// Set QuotaId as -1 to generate a new quotaid.
		config.QuotaId = "-1"
	}
}

// applyDiskQuota can set diskQuota in containerConfig.
// Resources field of containerConfig should not be nil.
func applyDiskQuota(pod *v1.Pod, container *v1.Container, imageStatus *runtimeapi.Image, lc *runtimeapi.LinuxContainerConfig) error {
	// Set "/" quota as the size of ephemeral storage in limits.
	limitEphemeralStorage, limitESExists := container.Resources.Limits[v1.ResourceEphemeralStorage]
	if !limitESExists || limitEphemeralStorage.IsZero() {
		glog.V(4).Infof("limitEphemeralStorage is not defined in pod: %q, ignore to setup diskquota", format.Pod(pod))
		return nil
	}

	// Default diskQuotaMode is "DiskQuotaModeRootFsAndVolume"
	diskQuotaMode := sigmak8sapi.DiskQuotaModeRootFsAndVolume
	// Change diskQuotaMode if needed.
	containerHostConfig := sigmautil.GetHostConfigFromAnnotation(pod, container.Name)
	if containerHostConfig != nil && containerHostConfig.DiskQuotaMode == sigmak8sapi.DiskQuotaModeRootFsOnly {
		diskQuotaMode = sigmak8sapi.DiskQuotaModeRootFsOnly
	}
	glog.V(4).Infof("Set RootFs DiskQuotaMode as %s for container %s in pod %s",
		string(diskQuotaMode), container.Name, format.Pod(pod))

	lc.Resources.DiskQuota = map[string]string{}

	// We should set diskquota in the format such as "/&/home/admin=10g" if DiskQuotaMode is ".*".
	// The volumes include normal volumeMount and image volume.
	// VolumeMount from PVC is ignored because PVC has its own DiskQuota.
	if diskQuotaMode == sigmak8sapi.DiskQuotaModeRootFsAndVolume {
		diskQuotaKey := sigmautil.DiskQuotaLimitRootFsOnly

		// Get volumes from PVC.
		volumeFromPVC := sets.NewString()
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				volumeFromPVC.Insert(volume.Name)
			}
		}

		// volumePathFromPVC records the mount path from pvc.
		// volumePath records the mount path we should set DiskQuota.
		volumePathFromPVC := sets.NewString()
		volumePath := sets.NewString()
		for _, volumeMount := range container.VolumeMounts {
			if volumeFromPVC.Has(volumeMount.Name) {
				volumePathFromPVC.Insert(volumeMount.MountPath)
				continue
			}
			volumePath.Insert(volumeMount.MountPath)
		}

		// Add image volume path to volumePath.
		if imageStatus != nil {
			for containerPath := range imageStatus.Volumes {
				if !volumePathFromPVC.Has(containerPath) && !volumePath.Has(containerPath) {
					volumePath.Insert(containerPath)
				}
			}
		}

		// Generate DiskQuotaKey.
		for _, path := range volumePath.List() {
			diskQuotaKey = diskQuotaKey + "&" + path
		}

		lc.Resources.DiskQuota[diskQuotaKey] = getDiskSize(limitEphemeralStorage.String())
	} else {
		// Set DiskQuota in the format such as "/=10g" if DiskQuotaMode is "/".
		lc.Resources.DiskQuota[string(diskQuotaMode)] = getDiskSize(limitEphemeralStorage.String())
	}

	glog.V(0).Infof("Set ResourceQuota with %v for container %s in pod %s", lc.Resources.DiskQuota, container.Name, format.Pod(pod))

	return nil
}

// applyExtendContainerResource can merge extended resource feilds into container config of CRI.
// Deprecated: Use annotations instead of extending CRI in future.
func applyExtendContainerResource(pod *v1.Pod, container *v1.Container, imageStatus *runtimeapi.Image,
	lc *runtimeapi.LinuxContainerConfig, enforceCPULimits bool) {
	// Set ulimits if possible.
	ulimits := sigmautil.GetUlimitsFromAnnotation(container, pod)
	if len(ulimits) != 0 {
		for _, ulimit := range ulimits {
			lc.Resources.Ulimits = append(lc.Resources.Ulimits, &runtimeapi.Ulimit{Name: ulimit.Name,
				Soft: ulimit.Soft, Hard: ulimit.Hard})
		}
	}

	// Set diskquota.
	applyDiskQuota(pod, container, imageStatus, lc)

	// Set other fields defined in hostconfig.
	hostConfig := sigmautil.GetHostConfigFromAnnotation(pod, container.Name)
	if hostConfig != nil {
		// Change cpushares to DefaultCPUShares if possible.
		if lc.Resources.CpuShares == minShares {
			if hostConfig.DefaultCpuShares != nil && *hostConfig.DefaultCpuShares > minShares {
				glog.V(0).Infof("Set cpushares with default value %d for container %s in pod %s",
					*hostConfig.DefaultCpuShares, container.Name, format.Pod(pod))
				lc.Resources.CpuShares = *hostConfig.DefaultCpuShares
			}
		}

		// Set extra resources.
		lc.Resources.MemorySwappiness = &runtimeapi.Int64Value{hostConfig.MemorySwappiness}
		lc.Resources.MemorySwap = hostConfig.MemorySwap
		lc.Resources.PidsLimit = int64(hostConfig.PidsLimit)

		// NOTE(tongkai.ytk): DELETE ME IF NOT NECESSARY
		//   At this point, when container is CPUSET with Cpu resource set, CpuShares = Cpu.Request * 1024,
		//                         CpuPeriod = default 100000, CpuQuota = -1.
		//                  when container is not CPUSET, CpuShares = Cpu.Request * 1024, CpuPeriod = default 100000 or
		//                         annotation.Container.HostConfig.CpuPeriod(must be larger than 100000),
		//                         CpuQuota = Cpu.Limit * CpuPeriod.
		//                  when container is CPUSET with Cpu Resource not set or set 0, CpuShares = 2(minShares),
		//                         CpuPeriod = default 100000, CpuQuota = -1.
		//                         if hostConfig.DefaultCpuShares != nil, CpuShares will be DefaultCpuShares.
		//                  when container is not CPUSET with Cpu Resource not set or set 0, CpuShares = 2(minShares),
		//                         CpuPeriod = default 100000 or annotation.Container.HostConfig.CpuPeriod(must be
		//                         larger than 100000), CpuQuota = 0(equal to -1).
		//                         if hostConfig.DefaultCpuShares != nil, CpuShares will be DefaultCpuShares.
		//   When m.cpuCFSQuota is turn off, CpuPeriod = 0, CpuQuota = 0 (the struct initialized value)
		// reset CPU resources: CpuShares/CpuQuota/CpuPeriod/CpuBvtWarpNs with hostConfig
		if hostConfig.CpuShares >= minShares {
			lc.Resources.CpuShares = hostConfig.CpuShares
			glog.V(0).Infof("Set cpushares with value %d for container %s in pod %s",
				hostConfig.CpuShares, container.Name, format.Pod(pod))
		}
		if enforceCPULimits {
			// only when cpu CFS quota is turn on, reset CpuQuota and CpuPeriod
			if hostConfig.CpuPeriod >= minQuotaPeriod {
				lc.Resources.CpuPeriod = hostConfig.CpuPeriod
				glog.V(0).Infof("Set CpuPeriod with hostConfig value %d for container %s in pod %s",
					hostConfig.CpuPeriod, container.Name, format.Pod(pod))
				if container.Resources.Limits.Cpu().MilliValue() != 0 {
					lc.Resources.CpuQuota = (container.Resources.Limits.Cpu().MilliValue() *
						lc.Resources.CpuPeriod) / milliCPUToCPU
					glog.V(0).Infof("Set CpuQuota with value %d by cpu.limits*period for container %s in pod %s",
						lc.Resources.CpuQuota, container.Name, format.Pod(pod))
				}
			}
			if hostConfig.CpuQuota >= minQuotaPeriod {
				lc.Resources.CpuQuota = hostConfig.CpuQuota
				glog.V(0).Infof("Set CpuQuota with hostConfig value %d for container %s in pod %s",
					hostConfig.CpuQuota, container.Name, format.Pod(pod))
			}
		}
		if hostConfig.CPUBvtWarpNs != 0 {
			lc.Resources.CpuBvtWarpNs = int64(hostConfig.CPUBvtWarpNs)
		}

		// reset Memory resource: OomScoreAdj with hostConfig
		if hostConfig.OomScoreAdj != 0 {
			lc.Resources.OomScoreAdj = hostConfig.OomScoreAdj
			glog.V(0).Infof("Set OomScoreAdj with hostConfig value %d for container %s in pod %s",
				hostConfig.OomScoreAdj, container.Name, format.Pod(pod))
		}

		// reset MemoryWmarkRatio defined in hostconfig
		if hostConfig.MemoryWmarkRatio != 0 {
			lc.Resources.MemoryWmarkRatio = float32(hostConfig.MemoryWmarkRatio)
			glog.V(0).Infof("Set MemoryWmarkRatio with hostConfig value %v for container %s in pod %s",
				hostConfig.MemoryWmarkRatio, container.Name, format.Pod(pod))
		}

		// reset IntelRdtMba if defined in hostconfig
		if hostConfig.IntelRdtMba != "" {
			if lc.IntelRdt == nil {
				lc.IntelRdt = &runtimeapi.IntelRdt{}
			}
			lc.IntelRdt.IntelRdtMba = hostConfig.IntelRdtMba
			glog.V(0).Infof("Set IntelRdtMba with hostConfig value %v for container %s in pod %s",
				hostConfig.IntelRdtMba, container.Name, format.Pod(pod))
		}

		// reuse general cgroup parent
		if hostConfig.IntelRdtGroup == "" {
			hostConfig.IntelRdtGroup = hostConfig.CgroupParent
		}

		// reset IntelRdtGroup if defined in hostconfig
		if hostConfig.IntelRdtGroup != "" {
			if lc.IntelRdt == nil {
				lc.IntelRdt = &runtimeapi.IntelRdt{}
			}
			lc.IntelRdt.IntelRdtGroup = hostConfig.IntelRdtGroup
			glog.V(0).Infof("Set IntelRdtGroup with hostConfig value %v for container %s in pod %s",
				hostConfig.IntelRdtGroup, container.Name, format.Pod(pod))
		}
	}
}
