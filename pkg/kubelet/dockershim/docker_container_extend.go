package dockershim

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/golang/glog"
	sigmautil "k8s.io/kubernetes/pkg/kubelet/sigma"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

const (
	// labelAutoQuotaId is always be true. So aliDocker can generate a QuotaId automatically.
	labelAutoQuotaId = "AutoQuotaId"
	// labelQuotaId is used to set container's quotaid.
	labelQuotaId = "QuotaId"
	// labelDiskQuota is used to set container's disk quota.
	labelDiskQuota = "DiskQuota"
	// if labelHostDNS is true, AliDocker will get independent DNS related files such as resolv.conf.
	labelHostDNS = "ali.host.dns"
	// diskQuotaiLimitAllKey ".*" means the limitation of rootfs and volumes.
	diskQuotaLimitAllKey = ".*"
)

// getQuotaIdFromContainer get QuotaId from container's label
func getQuotaIdFromContainer(r *dockertypes.ContainerJSON) (string, bool) {
	if r == nil || r.Config == nil || r.Config.Labels == nil {
		return "", false
	}
	if quotaId, exists := r.Config.Labels[labelQuotaId]; exists {
		return quotaId, true
	}
	return "", false
}

// getDiskQuotaFromContainer get DiskQuota from container's label
func getDiskQuotaFromContainer(r *dockertypes.ContainerJSON) (map[string]string, bool) {
	if r == nil || r.Config == nil || r.Config.Labels == nil {
		return map[string]string{}, false
	}
	if diskQuotaStr, exists := r.Config.Labels[labelDiskQuota]; exists {
		diskQuotaMap := sigmautil.ParseDiskQuota(diskQuotaStr)
		return diskQuotaMap, true
	}
	return map[string]string{}, false
}

// updateContainerHostInfo can copy resolv.conf, hostname, hosts file into container.
// Only used in AliDocker when ali.host.dns label is true.
func (ds *dockerService) updateContainerHostInfo(podSandboxID, podContainerID string) error {
	// Get sandbox status.
	sandboxContainerInfo, err := ds.client.InspectContainer(podSandboxID)
	if err != nil {
		return fmt.Errorf("failed to inspect sandbox container %s: %v", podSandboxID, err)
	}
	if sandboxContainerInfo.ContainerJSONBase == nil || sandboxContainerInfo.Config == nil {
		return fmt.Errorf("get invalid sandbox status of sandbox %s, ContainerJSONBase or Config is nil: %v",
			podSandboxID, sandboxContainerInfo)
	}

	// Get container status.
	containerInfo, err := ds.client.InspectContainer(podContainerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container %s: %v", podContainerID, err)
	}
	if containerInfo.ContainerJSONBase == nil {
		return fmt.Errorf("get invalid container status of container %s, ContainerJSONBase is nil: %v",
			podContainerID, containerInfo)
	}

	// Update /etc/resolv.conf file
	if err := updateContainerResolv(podContainerID, sandboxContainerInfo); err != nil {
		return fmt.Errorf("failed to update resolv file of container: %s, error: %v", podContainerID, err)
	}

	// Update /etc/hostname file
	if err := updateContainerHostname(podContainerID, sandboxContainerInfo); err != nil {
		return fmt.Errorf("failed to update hostname file of container: %s, error: %v", podContainerID, err)
	}

	// Update /etc/hosts file
	if err := updateContainerHosts(podContainerID, containerInfo); err != nil {
		return fmt.Errorf("failed to update hosts file of container: %s, error: %v", podContainerID, err)
	}

	return nil
}

// updateContainerResolv copy sandbox's ResolvConfPath file into container.
func updateContainerResolv(podContainerID string, sandboxContainerInfo *dockertypes.ContainerJSON) error {
	resolvPath := sandboxContainerInfo.ContainerJSONBase.ResolvConfPath
	destPath := "/etc/"

	err := copyFileToContainer(resolvPath, destPath, podContainerID)
	if err != nil {
		return err
	}
	return nil
}

// updateContainerHostname copy sandbox's HostnamePath file into container.
func updateContainerHostname(podContainerID string, sandboxContainerInfo *dockertypes.ContainerJSON) error {
	// Deal with /etc/hostname
	// Get container hostname path from config
	// hostnamePath: /home/t4/docker/containers/264b539d505e2b7e6e53bf1ffc0c4f60a673bd16f6cbaf128b318588e5c35755/hostname
	hostnamePath := sandboxContainerInfo.ContainerJSONBase.HostnamePath
	destPath := "/etc/"

	err := copyFileToContainer(hostnamePath, destPath, podContainerID)
	if err != nil {
		return err
	}
	return nil
}

// copyFileToContainer can copy srcFile from disk to destPath in container.
func copyFileToContainer(srcFile, destPath, containerID string) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	// Copy hostname file into container with "docker cp" command.
	cmd := exec.CommandContext(ctx, "docker", "cp", srcFile, string(containerID)+":"+destPath)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to exec %v, out: %q, err: %v", cmd, string(out), err)
	}
	return nil
}

// UpdateContainerExtraResources will update container's resources which not supported by docker official client.
func UpdateContainerExtraResources(resources *runtimeapi.LinuxContainerResources, id string) error {
	// Get DiskQuota.
	DiskQuota := resources.DiskQuota

	if len(DiskQuota) > 0 {
		// Set timeout value as 10 second.
		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()

		// Update DiskQuota with "docker update".
		cmd := exec.CommandContext(ctx, "docker", "update", "--disk", sigmautil.ParseDiskQuotaToLabel(DiskQuota), id)
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to exec %v, out: %q, err: %v", cmd, string(out), err)
		}
	}

	return nil
}

// updateContainerHosts copy etc-host which is prepared by kubelet  into container.
func updateContainerHosts(podContainerID string, containerInfo *dockertypes.ContainerJSON) error {
	hostsPath := ""
	for _, mount := range containerInfo.Mounts {
		if mount.Destination == "/etc/hosts" {
			hostsPath = mount.Source
		}
	}

	if len(hostsPath) == 0 {
		return fmt.Errorf("Invalid hostsPath")
	}

	// Deal with /etc/hosts
	destPath := "/etc/hosts"

	err := copyFileToContainer(hostsPath, destPath, podContainerID)
	if err != nil {
		return err
	}
	return nil
}

// getHostHosts can get hosts file from disk.
func getHostHosts(hostHostsPath string) (string, error) {
	file, err := ioutil.ReadFile(hostHostsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read host's hosts file: %s", hostHostsPath)
	}
	return string(file), nil
}

// getPodValidHosts filter out the validate host items.
// Valid hosts is the host item after(include) hostname line.
//# Kubernetes-managed hosts file.
//127.0.0.1	localhost
//::1	localhost ip6-localhost ip6-loopback
//fe00::0	ip6-localnet
//fe00::0	ip6-mcastprefix
//fe00::1	ip6-allnodes
//fe00::2	ip6-allrouters
//11.166.4.112	sigma-slave110

//# Entries added by HostAliases.
//127.0.0.1	localhost.localdomain1
//127.0.0.1	localhost.localdomain2
func getPodValidHosts(podHostsPath, hostname string) (string, error) {
	var buffer bytes.Buffer
	file, err := ioutil.ReadFile(podHostsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read hosts file: %s", podHostsPath)
	}
	segs := strings.Split(string(file), "\n")

	// Append all lines after(include) hostname line
	isAppended := false
	for _, seg := range segs {
		if isAppended {
			buffer.WriteString("\n" + seg)
			continue
		}
		if strings.Contains(seg, hostname) {
			isAppended = true
			buffer.WriteString("\n" + seg)
		}
	}
	return buffer.String(), nil
}

// updateCreateConfigExtend can update docker container config with extended fields of CRI config.
func updateCreateConfigExtend(config *dockertypes.ContainerCreateConfig, runtimeConfig *runtimeapi.ContainerConfig) {
	if runtimeConfig == nil {
		glog.Warning("Ignore to update extend hostconfig field because runtimeConfig is nil")
		return
	}

	// Set NetPriority field.
	config.Config.NetPriority = int(runtimeConfig.NetPriority)

	if runtimeConfig.Linux == nil || runtimeConfig.Linux.Resources == nil {
		glog.Warningf("Ignore to update extend hostconfig field because of invalid ContainerConfig: %v", runtimeConfig)
		return
	}

	// Set Swappiness field.
	if runtimeConfig.Linux.Resources.MemorySwappiness != nil {
		config.HostConfig.Resources.MemorySwappiness = &runtimeConfig.Linux.Resources.MemorySwappiness.Value
	}
	// Set MemorySwap field.
	config.HostConfig.Resources.MemorySwap = runtimeConfig.Linux.Resources.MemorySwap
	// Set CPUBvtWarpNs field.
	config.HostConfig.Resources.CPUBvtWarpNs = runtimeConfig.Linux.Resources.CpuBvtWarpNs
	// Set PidsLimit field.
	config.HostConfig.Resources.PidsLimit = runtimeConfig.Linux.Resources.PidsLimit
}

// PauseContainer pauses the container.
func (ds *dockerService) PauseContainer(_ context.Context, r *runtimeapi.PauseContainerRequest) (*runtimeapi.PauseContainerResponse, error) {
	err := ds.client.PauseContainer(r.ContainerId)
	if err != nil {
		return nil, err
	}

	return &runtimeapi.PauseContainerResponse{}, nil
}

// UnpauseContainer unpauses the container.
func (ds *dockerService) UnpauseContainer(_ context.Context, r *runtimeapi.UnpauseContainerRequest) (*runtimeapi.UnpauseContainerResponse, error) {
	err := ds.client.UnpauseContainer(r.ContainerId)
	if err != nil {
		return nil, err
	}

	return &runtimeapi.UnpauseContainerResponse{}, nil
}
