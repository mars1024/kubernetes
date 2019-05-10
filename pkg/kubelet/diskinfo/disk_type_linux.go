// +build cgo,linux

/*
Copyright 2019 Alipay.

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

package diskinfo

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/utils/exec"
)

// getDeviceTypes get device types of all block device
func getDeviceTypes() (map[string]api.DiskType, error) {
	cmdExecutor := exec.New()

	devicesCmd := cmdExecutor.Command("ls", "/sys/block/")
	devices, err := devicesCmd.CombinedOutput()
	if nil != err {
		glog.Errorf("failed to get diskType during run `ls /sys/block/`, err: %v", err)
		return nil, err
	}

	ret := make(map[string]api.DiskType, len(devices))
	for _, device := range devices {
		rotationPath := fmt.Sprintf("/sys/block/%s/queue/rotational", string(device))
		cmd := cmdExecutor.Command("cat", rotationPath)
		out, err := cmd.CombinedOutput()
		if nil != err {
			glog.Errorf("failed to get diskType during run `cat %s`, err: %v", rotationPath, err)
			return nil, err
		}
		outString := strings.TrimSpace(string(out))

		deviceType := diskTypeUnknown
		if outString == "0" {
			deviceType = api.DiskTypeSSD
		} else if "1" == outString {
			deviceType = api.DiskTypeHDD
		}

		ret[string(device)] = deviceType
	}
	return ret, nil
}

// isNvme check if partition is nvme device
func isNvme(partition string) (bool, error) {
	if !strings.HasPrefix(partition, "/dev/nvme") {
		return false, nil
	}

	if _, err := os.Stat(partition); nil != err {
		cmdExecutor := exec.New()

		cmd := cmdExecutor.Command("ls", fmt.Sprintf("-al %s | awk -F, '{print $1}' | awk '{print $5}'", partition))
		out, err := cmd.CombinedOutput()

		if nil != err {
			glog.Errorf("check partition `%s` is nvme failed, err: %v", partition, err)
			return false, err
		}

		if strings.TrimSpace(string(out)) == "259" {
			return true, nil
		}
	}

	return false, nil
}

// getPartitionDiskType get the diskType of given partition from the given deviceTypes
func getPartitionDiskType(partition string, deviceTypes map[string]api.DiskType) api.DiskType {
	// we treat nvme as SSD
	if nvme, err := isNvme(partition); nil == err && nvme {
		return api.DiskTypeSSD
	}

	for device, deviceType := range deviceTypes {
		if strings.Contains(partition, device) {
			return deviceType
		}
	}

	return diskTypeUnknown
}
