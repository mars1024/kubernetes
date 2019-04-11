package sigma

import (
	"bytes"
	"strings"
)

const (
	// labelQuotaId is used to set container's quotaid.
	labelQuotaId = "QuotaId"
	// labelDiskQuota is used to set container's disk quota.
	labelDiskQuota = "DiskQuota"
	// diskQuotaiLimitAllKey ".*" means the limitation of rootfs and volumes.
	diskQuotaLimitAllKey = ".*"
)

// ParseDiskQuota support to analysis:
// 1 DiskQuota=/=60g;/data1=50
// 2 DiskQutoa=60g
func ParseDiskQuota(diskQuota string) map[string]string {
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

// ParseDiskQuotaToLabel can convert diskQuota map into a label string.
// 1 DiskQuota=/=60g;/data1=50
// 2 DiskQutoa=60g
func ParseDiskQuotaToLabel(diskQuota map[string]string) string {
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
