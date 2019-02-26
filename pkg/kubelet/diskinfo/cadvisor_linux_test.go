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
	"bytes"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/util/slice"
)

type dfInfo struct {
	Filesystem  string
	Type        string
	Block       int64
	Used        int64
	Available   int64
	UsedPercent int64
	Mounted     string
}

var (
	ErrDfNotResult    = errors.New("Command df not output result")
	ErrDfNotOkResult  = errors.New("Command df not right result")
	ErrIostatNoResult = errors.New("Command iostat not output result")
)

/*

Filesystem      1K-blocks      Used Available Use% Mounted on
/dev/sda2        51475068   5446652  43390592  12% /
devtmpfs        131918316         0 131918316   0% /dev

*/

// Copy from sigma-agent code
func parseDf() ([]*dfInfo, error) {
	out, err := exec.Command("/bin/sh", "-c", "df -T -B 1 -P 2>/dev/null").Output()
	if err != nil {
		if len(out) == 0 {
			glog.Error("df error %v", err)
			return nil, err
		}
	}

	lines := bytes.Split(out, []byte("\n"))

	if len(lines) == 0 || len(lines) == 1 {
		return nil, ErrDfNotResult
	}

	i := 0
	infos := make([]*dfInfo, len(lines))

	// skip the first title line.
	for _, l := range lines[1:] {
		fields := strings.Fields(string(l))
		if len(fields) != 7 {
			continue
		}

		df := &dfInfo{}

		df.Filesystem = fields[0]

		df.Type = fields[1]

		if block, err := strconv.ParseInt(fields[2], 10, 64); err != nil {
			glog.Error("%v", err)
			continue
		} else {
			df.Block = block
		}

		if used, err := strconv.ParseInt(fields[3], 10, 64); err != nil {
			glog.Error("%v", err)
			continue
		} else {
			df.Used = used
		}

		if avail, err := strconv.ParseInt(fields[4], 10, 64); err != nil {
			glog.Error("%v", err)
			continue
		} else {
			df.Available = avail
		}

		if usedPercent, err := strconv.ParseInt(strings.Trim(fields[5], "%"), 10, 64); err != nil {
			glog.Error("%v", err)
			continue
		} else {
			df.UsedPercent = usedPercent
		}

		df.Mounted = fields[6]

		infos[i] = df
		i++
	}

	if i == 0 {
		return nil, ErrDfNotOkResult
	}
	return infos[:i], nil
}

func TestGetDiskInfo(t *testing.T) {
	infos, err := getDiskInfo("/")
	if nil != err {
		t.Errorf("invoke getDiskInfo err: %v", err)
	}

	dfInfos, err := parseDf()
	if nil != err {
		t.Errorf("invoke df err: %v", err)
	}

	type dfDisk struct {
		Device string
		Size   int64

		MountPoints []string
	}

	for _, disk := range infos {
		var foundDfDisk *dfDisk

		for _, df := range dfInfos {
			if disk.Device == df.Filesystem {
				if nil == foundDfDisk {
					foundDfDisk = &dfDisk{
						Device:      df.Filesystem,
						Size:        df.Block,
						MountPoints: []string{},
					}
				}

				foundDfDisk.MountPoints = append(foundDfDisk.MountPoints, df.Mounted)
			}
		}

		if nil == foundDfDisk {
			t.Errorf("can not found df info for partiion %s", disk.Device)
		}

		if disk.Size != foundDfDisk.Size {
			t.Errorf("disk %s size from cadvisor and df are not equal. cadvisor got %d, df got %d", disk.Device, disk.Size, foundDfDisk.Size)
		}

		if !slice.ContainsString(foundDfDisk.MountPoints, disk.MountPoint, nil) {
			t.Errorf("disk %s could not find the mount point from df, cadvisor got %s, df mount points: %v", disk.Device, disk.MountPoint, foundDfDisk.MountPoints)
		}
	}
}
