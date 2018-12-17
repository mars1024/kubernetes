package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
)

// ContainerdType use to identify containerd type , docker or pouch
type ContainerdType string

const (
	ContainerdTypeDocker ContainerdType = "docker"
	ContainerdTypePouch  ContainerdType = "pouch"
	ContainerdUnknown    ContainerdType = "unknown"
)

// GetDockerPsOutput execute 'docker ps' and get output of specified container
func GetDockerPsOutput(hostIP, containerName string) string {
	hostSn := GetHostSnFromHostIp(hostIP)
	cmd := fmt.Sprintf("cmd://docker(ps -a | grep %s | grep -v pause)", containerName)
	resp, err := ResponseFromStarAgentTask(cmd, hostIP, hostSn)
	if err != nil {
		glog.Error(err)
		return ""
	}
	return resp
}

// GetPouchPsOutput execute 'pouch ps' and get output of specified container
func GetPouchPsOutput(hostIP, containerName string) string {
	hostSn := GetHostSnFromHostIp(hostIP)
	cmd := fmt.Sprintf("cmd://pouch(ps -a | grep %s | grep -v pause)", containerName)
	resp, err := ResponseFromStarAgentTask(cmd, hostIP, hostSn)
	if err != nil {
		glog.Error(err)
		return ""
	}
	return resp
}

// GetContainerDType execute 'pouch info' and 'docker info' to get containerd type
func GetContainerDType(hostIP string) (ContainerdType, error) {
	glog.Infof("get container runtime of host: %s", hostIP)
	hostSn := GetHostSnFromHostIp(hostIP)
	cmd := "cmd://pouch info"
	resp, err := ResponseFromStarAgentTask(cmd, hostIP, hostSn)
	if err == nil {
		if strings.Contains(resp, "Containers:") {
			glog.Infof("runtime of host %s is pouch", hostIP)
			return ContainerdTypePouch, nil
		}
	}
	cmd = "cmd://docker info"
	resp, err = ResponseFromStarAgentTask(cmd, hostIP, hostSn)
	if err == nil {
		if strings.Contains(resp, "Containers:") {
			glog.Infof("runtime of host %s is docker", hostIP)
			return ContainerdTypeDocker, nil
		}
	}
	return ContainerdUnknown, err
}

// GetContainerInspectField get container's field regardless of pouch or alidocker.
func GetContainerInspectField(hostIP, containerID, format string) (string, error) {
	hostSn := GetHostSnFromHostIp(hostIP)
	runtimeType, err := GetContainerDType(hostIP)
	if err != nil {
		return "", fmt.Errorf("Failed to get runtime type of node: %s, error: %s", hostIP, err.Error())
	}
	cmd := ""
	switch runtimeType {
	case ContainerdTypePouch:
		cmd = fmt.Sprintf("cmd://pouch(inspect -f %s %s)", format, containerID)
	case ContainerdTypeDocker:
		cmd = fmt.Sprintf("cmd://docker(inspect -f %s %s)", format, containerID)
	case ContainerdUnknown:
		return "", fmt.Errorf("Can't find Pouch or Docker runtime on node: %s", hostIP)
	}
	resp, err := ResponseFromStarAgentTask(cmd, hostIP, hostSn)
	if err != nil {
		return "", fmt.Errorf("Failed to exec command %s on node %s: %v", cmd, hostIP, err)
	}
	return resp, nil
}

// ContainerStop stop container by imageID or containerName
func ContainerStop(hostIP, containerIdentify string) (bool, error) {
	hostSN := GetHostSnFromHostIp(hostIP)
	containerType, err := GetContainerDType(hostIP)
	if err != nil {
		return false, err
	}
	cmd := ""
	switch containerType {
	case ContainerdTypeDocker:
		cmd = fmt.Sprintf("cmd://docker stop %s", containerIdentify)
	case ContainerdTypePouch:
		cmd = fmt.Sprintf("cmd://pouch stop %s", containerIdentify)
	}
	if cmd == "" {
		return false, fmt.Errorf("cmd is empty")
	}
	resp, err := ResponseFromStarAgentTask(cmd, hostIP, hostSN)
	if err == nil {
		if len(strings.Fields(resp)) == len(strings.Fields(containerIdentify)) {
			return true, nil
		}
	}
	return false, err
}

// ListContainers get container list by 'docker ps' or 'pouch ps', if allContainer is true,
// list by 'docker ps -a' or 'pouch ps -a'
func ListContainers(hostIP string, allContainer bool) ([]string, error) {
	hostSN := GetHostSnFromHostIp(hostIP)
	containerType, err := GetContainerDType(hostIP)
	if err != nil {
		return nil, err
	}
	cmd := ""
	switch containerType {
	case ContainerdTypeDocker:
		cmd = "cmd://docker ps "
	case ContainerdTypePouch:
		cmd = "cmd://pouch ps"
	}
	if cmd == "" {
		return nil, fmt.Errorf("cmd is empty")
	}
	if allContainer {
		cmd = fmt.Sprintf("%s -a", cmd)
	}
	resp, err := ResponseFromStarAgentTask(cmd, hostIP, hostSN)
	if err == nil {
		return strings.Split(resp, "\n"), nil
	}
	return nil, err
}

// GetContainerPsOutPut execute 'docker ps or pouch ps' and get output of specified container
func GetContainerPsOutPut(hostIP, containerName string) string {
	runOutput := GetDockerPsOutput(hostIP, containerName)
	if runOutput == "" {
		runOutput = GetPouchPsOutput(hostIP, containerName)
	}
	glog.Infof(runOutput)
	return runOutput
}

// CheckContainerNotExistInHost check whether container not exists in host
func CheckContainerNotExistInHost(hostIP, containerID string, timeout time.Duration) error {
	t := time.Now()
	for {
		runOutput := GetDockerPsOutput(hostIP, containerID)
		if runOutput == "" {
			runOutput = GetPouchPsOutput(hostIP, containerID)
		}
		if !strings.Contains(runOutput, containerID) {
			return nil
		}
		if time.Since(t) >= timeout {
			glog.Errorf("timeout for check container[%s] not exists", containerID)
			break
		}
		glog.Infof(runOutput)
		time.Sleep(15 * time.Second)
	}
	return fmt.Errorf("timeout for check container[%s] not exists", containerID)
}

// ExecCmdInContainer exec a command in container
func ExecCmdInContainer(hostIP, containerID, cmd string) string {
	hostSn := GetHostSnFromHostIp(hostIP)
	dockerCmd := fmt.Sprintf("cmd://docker(exec %s %s)", containerID, cmd)
	resp, err := ResponseFromStarAgentTask(dockerCmd, hostIP, hostSn)
	if err != nil {
		glog.Error(err)
		//return ""
		pouchCmd := fmt.Sprintf("cmd://pouch(exec %s %s)", containerID, cmd)
		resp, err = ResponseFromStarAgentTask(pouchCmd, hostIP, hostSn)
		if err != nil {
			glog.Error(err)
			return ""
		}
		return resp
	}
	return resp
}

// GetContainerQuotaID get quota ID of container
func GetContainerQuotaID(hostIP, containerID string) string {
	hostSn := GetHostSnFromHostIp(hostIP)
	pouchCmd := fmt.Sprintf("cmd://pouch(inspect %s | grep -w QuotaId | cut -d':' -f2)", containerID)
	resp, err := ResponseFromStarAgentTask(pouchCmd, hostIP, hostSn)
	if err != nil {
		glog.Error(err)
		dockerCmd := fmt.Sprintf("cmd://docker(inspect %s | grep -w QuotaId | cut -d':' -f2)", containerID)
		resp, err = ResponseFromStarAgentTask(dockerCmd, hostIP, hostSn)
		if err != nil {
			glog.Error(err)
			return ""
		}
		return resp

	}
	return resp
}

// GetContainerAdminUID get ali_admin_uid of container
func GetContainerAdminUID(hostIP, containerID string) string {
	hostSn := GetHostSnFromHostIp(hostIP)
	pouchCmd := fmt.Sprintf("cmd://pouch(inspect %s | grep -w ali_admin_uid | cut -d',' -f1)", containerID)
	resp, err := ResponseFromStarAgentTask(pouchCmd, hostIP, hostSn)
	if err != nil {
		glog.Error(err)
		dockerCmd := fmt.Sprintf("cmd://docker(inspect %s | grep -w ali_admin_uid | cut -d',' -f1)", containerID)
		resp, err = ResponseFromStarAgentTask(dockerCmd, hostIP, hostSn)
		if err != nil {
			glog.Error(err)
			return ""
		}
		return strings.Replace(resp, " ", "", -1)

	}
	return strings.Replace(resp, " ", "", -1)
}

// GetContainerCpusets get cpu set of container
func GetContainerCpusets(hostIP, containerID string) string {
	hostSn := GetHostSnFromHostIp(hostIP)
	pouchCmd := fmt.Sprintf("cmd://pouch(inspect %s | grep -w CpusetCpus | cut -d',' -f1)", containerID)
	resp, err := ResponseFromStarAgentTask(pouchCmd, hostIP, hostSn)
	if err != nil {
		glog.Error(err)
		dockerCmd := fmt.Sprintf("cmd://docker(inspect %s | grep -w CpusetCpus | cut -d',' -f1)", containerID)
		resp, err = ResponseFromStarAgentTask(dockerCmd, hostIP, hostSn)
		if err != nil {
			glog.Error(err)
			return ""
		}
		return strings.Replace(resp, " ", "", -1)

	}
	return strings.Replace(resp, " ", "", -1)
}