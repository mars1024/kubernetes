package framework

import (
	"fmt"
	"time"
	"strings"
	"path"
	"os"
	"os/exec"
	"bytes"
)

func runCmd(binaryPath string, env []string, args ...string) error {
	Logf("cmd: %s, args: %v", binaryPath, args)
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = env
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run cmd [%v], err %s", cmd, err.Error())
	}
	return nil
}

func getPProfAddressForComponent(componentName string) (string, error) {
	switch componentName {
	case "kube-apiserver":
		return TestContext.SigmaApiServerPProfAddress, nil
	case "kube-scheduler":
		return TestContext.SigmaSchedulerPProfAddress, nil
	case "kube-controller-manager":
		return TestContext.SigmaControllerPProfAddress, nil
	}
	return "", fmt.Errorf("pprof address for component %v unknown", componentName)
}

// Gathers sigma profiles from a master component. E.g usages:
//   - gatherProfile("kube-apiserver", "someTest", "heap")
//   - gatherProfile("kube-scheduler", "someTest", "profile")
//   - gatherProfile("kube-controller-manager", "someTest", "profile?seconds=20")
//
// We don't export this method but wrappers around it (see below).
func gatherSigmaProfile(componentName, profileBaseName, profileKind string) error {
	Logf("gather sigma component[%s] profile[%s]", componentName, profileKind)
	if err := checkProfileGatheringPrerequisites(); err != nil {
		return fmt.Errorf("Profile gathering pre-requisite failed: %v", err)
	}
	profileAddress, err := getPProfAddressForComponent(componentName)
	if err != nil {
		return fmt.Errorf("profile gathering failed finding component address: %v", err)
	}
	if profileBaseName == "" {
		profileBaseName = time.Now().Format(time.RFC3339)
	}

	// Get the profile data
	profilePrefix := componentName
	switch {
	case profileKind == "heap":
		profilePrefix += "_MemoryProfile_"
	case strings.HasPrefix(profileKind, "profile"):
		profilePrefix += "_CPUProfile_"
	default:
		return fmt.Errorf("unknown profile kind provided: %s", profileKind)
	}
	rawProfilePath := path.Join(getProfilesDirectoryPath(), profilePrefix+profileBaseName+".pprof")

	args := []string{"-o", rawProfilePath, "-s"}
	args = append(args, fmt.Sprintf("%v/debug/pprof/%s", profileAddress, profileKind))
	var env []string
	Logf("begin to run cmd: %v", args)
	err = runCmd("curl", env, args...)
	if err != nil {
		return fmt.Errorf("failed to execute curl command on master : %v", err)
	}

	// Create a graph from the data and write it to a svg file.
	var cmd *exec.Cmd
	switch {
	// TODO: Support other profile kinds if needed (e.g inuse_space, alloc_objects, mutex, etc)
	case profileKind == "heap":
		cmd = exec.Command("go", "tool", "pprof", "-svg", "-symbolize=none", "--alloc_space", rawProfilePath)
	case strings.HasPrefix(profileKind, "profile"):
		cmd = exec.Command("go", "tool", "pprof", "-svg", "-symbolize=none", rawProfilePath)
	default:
		return fmt.Errorf("Unknown profile kind provided: %s", profileKind)
	}
	outfilePath := path.Join(getProfilesDirectoryPath(), profilePrefix+profileBaseName+".svg")
	outfile, err := os.Create(outfilePath)
	if err != nil {
		return fmt.Errorf("failed to create file for the profile graph: %v", err)
	}
	defer outfile.Close()
	cmd.Stdout = outfile
	stderr := bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	if err := cmd.Run(); nil != err {
		return fmt.Errorf("failed to run 'go tool pprof': %v, stderr: %#v", err, stderr.String())
	}
	return nil
}
