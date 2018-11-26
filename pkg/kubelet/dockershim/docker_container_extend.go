package dockershim

import (
	"bytes"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
)

const (
	// labelAutoQuotaId is always be true. So aliDocker can generate a QuotaId automatically.
	labelAutoQuotaId = "AutoQuotaId"
	// labelQuotaId is used to set container's quotaid.
	labelQuotaId = "QuotaId"
	// labelDiskQuota is used to set container's disk quota.
	labelDiskQuota = "DiskQuota"
	// diskQuotaiLimitAllKey ".*" means the limitation of rootfs and volumes.
	diskQuotaLimitAllKey = ".*"
)

// parseDiskQuotaToLabel can convert diskQuota map into a label string.
// 1 DiskQuota=/=60g;/data1=50
// 2 DiskQutoa=60g
func parseDiskQuotaToLabel(diskQuota map[string]string) string {
	var buffer bytes.Buffer
	for k, v := range diskQuota {
		// ".*" means the limitation of rootfs and volume.
		// https://github.com/alibaba/pouch/blob/master/docs/features/pouch_with_diskquota.md#parameter-details
		if k == diskQuotaLimitAllKey {
			// Convert ".*" to the type such as "10g".
			buffer.WriteString(v)
			buffer.WriteString(";")
			continue
		}
		buffer.WriteString(k)
		buffer.WriteString("=")
		buffer.WriteString(v)
		buffer.WriteString(";")
	}
	diskQuotaLabel := buffer.String()
	return strings.Trim(diskQuotaLabel, ";")
}

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

// parseDiskQuota support to analysis:
// 1 DiskQuota=/=60g;/data1=50
// 2 DiskQutoa=60g
func parseDiskQuota(diskQuota string) map[string]string {
	diskQuotaMap := map[string]string{}
	if diskQuota == "" {
		return diskQuotaMap
	}
	diskQuotas := strings.Split(diskQuota, ";")
	for _, pair := range diskQuotas {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			diskQuotaMap[parts[0]] = parts[1]
		} else if len(parts) == 1 {
			diskQuotaMap[diskQuotaLimitAllKey] = parts[0]
		}
	}
	return diskQuotaMap
}

// getDiskQuotaFromContainer get DiskQuota from container's label
func getDiskQuotaFromContainer(r *dockertypes.ContainerJSON) (map[string]string, bool) {
	if r == nil || r.Config == nil || r.Config.Labels == nil {
		return map[string]string{}, false
	}
	if diskQuotaStr, exists := r.Config.Labels[labelDiskQuota]; exists {
		diskQuotaMap := parseDiskQuota(diskQuotaStr)
		return diskQuotaMap, true
	}
	return map[string]string{}, false
}
