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
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/mount"
	"github.com/golang/glog"
	"github.com/google/cadvisor/utils"
	zfs "github.com/mistifyio/go-zfs"
	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
)

const (
	diskTypeUnknown api.DiskType = "unknown"
)

// getDiskInfo can collect disk information just as "df".
func getDiskInfo(rootDir string) ([]api.DiskInfo, error) {
	mounts, err := mount.GetMounts(nil)
	if nil != err {
		return nil, err
	}

	// get detail information of each mount.
	partitions := processMounts(mounts, nil)

	rootPartition, err := getDirPartition(rootDir, partitions)
	if nil != err {
		return nil, err
	}

	var infos []api.DiskInfo
	for _, p := range partitions {
		infos = append(infos, api.DiskInfo{
			Device:         p.name,
			FileSystemType: p.fsType,
			Size:           int64(p.capacity),
			MountPoint:     p.mountpoint,
			//TODO (chenjun.cj): fixme
			DiskType:    diskTypeUnknown,
			IsGraphDisk: p.name == rootPartition.name,
		})
	}

	return infos, nil
}

func major(devNumber uint64) uint {
	return uint((devNumber >> 8) & 0xfff)
}

func minor(devNumber uint64) uint {
	return uint((devNumber & 0xff) | ((devNumber >> 12) & 0xfff00))
}

// getDirPartition can get partition based on mount dir.
func getDirPartition(dir string, partitions map[string]partition) (*partition, error) {
	buf := new(syscall.Stat_t)
	err := syscall.Stat(dir, buf)
	if err != nil {
		return nil, fmt.Errorf("stat failed on %s with error: %s", dir, err)
	}

	major := major(buf.Dev)
	minor := minor(buf.Dev)

	for _, p := range partitions {
		if p.major == uint64(major) && p.minor == uint64(minor) {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("can not find the patition for dir %q", dir)
}

// partition describes the detail information of a mount.
type partition struct {
	// Name of partition, such as /dev/sda1
	name string

	// MountPoint of partition, such as /home
	mountpoint string

	// FsType, such as ext4, ext3,
	fsType string

	// Capacity of partition, in bytes
	capacity uint64

	// Major indicates one half of the device ID which identifies the device class.
	major uint64

	// Minor indicates one half of the device ID which identifies a specific
	// instance of device.
	minor uint64
}

// processMounts will return partitions of mounts.
func processMounts(mounts []*mount.Info, excludedMountpointPrefixes []string) map[string]partition {
	partitions := make(map[string]partition, 0)

	supportedFsType := map[string]bool{
		// all ext systems are checked through prefix.
		"btrfs": true,
		"tmpfs": true,
		"xfs":   true,
		"zfs":   true,
	}

	for _, mount := range mounts {
		if !strings.HasPrefix(mount.Fstype, "ext") && !supportedFsType[mount.Fstype] {
			continue
		}
		// Avoid bind mounts.
		if _, ok := partitions[mount.Source]; ok {
			continue
		}

		hasPrefix := false
		for _, prefix := range excludedMountpointPrefixes {
			if strings.HasPrefix(mount.Mountpoint, prefix) {
				hasPrefix = true
				break
			}
		}
		if hasPrefix {
			continue
		}

		// btrfs fix: following workaround fixes wrong btrfs Major and Minor Ids reported in /proc/self/mountinfo.
		// instead of using values from /proc/self/mountinfo we use stat to get Ids from btrfs mount point
		if mount.Fstype == "btrfs" && mount.Major == 0 && strings.HasPrefix(mount.Source, "/dev/") {
			major, minor, err := getBtrfsMajorMinorIds(mount)
			if err != nil {
				glog.Warningf("%s", err)
			} else {
				mount.Major = major
				mount.Minor = minor
			}
		}

		capacity, err := getPartitionCapacity(mount.Fstype, mount.Mountpoint)
		if nil != err {
			glog.Errorf("get mountpoint %s capacity failed: %v", mount.Mountpoint, err)
			continue
		}

		p := partition{
			name:       mount.Source,
			fsType:     mount.Fstype,
			mountpoint: mount.Mountpoint,
			major:      uint64(mount.Major),
			minor:      uint64(mount.Minor),
			capacity:   capacity,
		}

		partitions[mount.Source] = p
	}

	return partitions
}

// getPartitionCapacity can get the size from partition.
func getPartitionCapacity(fsType string, partition string) (uint64, error) {
	switch fsType {
	case "zfs":
		capacity, _, _, err := getZfstats(partition)
		return capacity, err
	default:
		if utils.FileExists(partition) {
			capacity, _, _, _, _, err := getVfsStats(partition)
			return capacity, err
		} else {
			return 0, fmt.Errorf("unable to determine file system type, partition mountpoint does not exist: %v", partition)
		}
	}
}

func getVfsStats(path string) (total uint64, free uint64, avail uint64, inodes uint64, inodesFree uint64, err error) {
	var s syscall.Statfs_t
	if err = syscall.Statfs(path, &s); err != nil {
		return 0, 0, 0, 0, 0, err
	}
	total = uint64(s.Frsize) * s.Blocks
	free = uint64(s.Frsize) * s.Bfree
	avail = uint64(s.Frsize) * s.Bavail
	inodes = uint64(s.Files)
	inodesFree = uint64(s.Ffree)
	return total, free, avail, inodes, inodesFree, nil
}

// getZfstats returns ZFS mount stats using zfsutils
func getZfstats(poolName string) (uint64, uint64, uint64, error) {
	dataset, err := zfs.GetDataset(poolName)
	if err != nil {
		return 0, 0, 0, err
	}

	total := dataset.Used + dataset.Avail + dataset.Usedbydataset

	return total, dataset.Avail, dataset.Avail, nil
}

// getBtrfsMajorMinorIds can get major and minor Ids for a mount point using btrfs as filesystem.
func getBtrfsMajorMinorIds(mount *mount.Info) (int, int, error) {
	// btrfs fix: following workaround fixes wrong btrfs Major and Minor Ids reported in /proc/self/mountinfo.
	// instead of using values from /proc/self/mountinfo we use stat to get Ids from btrfs mount point

	buf := new(syscall.Stat_t)
	err := syscall.Stat(mount.Source, buf)
	if err != nil {
		err = fmt.Errorf("stat failed on %s with error: %s", mount.Source, err)
		return 0, 0, err
	}

	glog.V(4).Infof("btrfs mount %#v", mount)
	if buf.Mode&syscall.S_IFMT == syscall.S_IFBLK {
		err := syscall.Stat(mount.Mountpoint, buf)
		if err != nil {
			err = fmt.Errorf("stat failed on %s with error: %s", mount.Mountpoint, err)
			return 0, 0, err
		}

		glog.V(4).Infof("btrfs dev major:minor %d:%d\n", int(major(buf.Dev)), int(minor(buf.Dev)))
		glog.V(4).Infof("btrfs rdev major:minor %d:%d\n", int(major(buf.Rdev)), int(minor(buf.Rdev)))

		return int(major(buf.Dev)), int(minor(buf.Dev)), nil
	} else {
		return 0, 0, fmt.Errorf("%s is not a block device", mount.Source)
	}
}
