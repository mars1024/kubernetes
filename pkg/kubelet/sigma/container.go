package sigma

import (
	"bytes"
	"strings"
)

const (
	// DiskQuotaiLimitAllKey ".*" means the limitation of rootfs and volumes.
	DiskQuotaLimitAllKey = ".*"
	// DiskQuotaLimitRootFsOnly "/" means the limitation of rootfs only.
	DiskQuotaLimitRootFsOnly = "/"
)

// ParseDiskQuota support to analysis:
// 1 DiskQuota=/=60g;/data1=50
// 2 DiskQutoa=60g
// Used for Alidocker
func ParseDiskQuota(diskQuota string) map[string]string {
	diskQuotaMap := map[string]string{}
	if diskQuota == "" {
		return diskQuotaMap
	}

	// Record the order.
	keys := []string{}

	diskQuotas := strings.Split(diskQuota, ";")
	for _, pair := range diskQuotas {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			diskQuotaMap[parts[0]] = parts[1]
			keys = append(keys, parts[0])
		} else if len(parts) == 1 {
			diskQuotaMap[DiskQuotaLimitAllKey] = parts[0]
			keys = append(keys, DiskQuotaLimitAllKey)
		}
	}

	// Just return if there is only one diskquota.
	if len(diskQuotas) < 2 {
		return diskQuotaMap
	}

	// Convert DiskQuota to the format such as "/&/home/test1&/home/test2=10g" if needed.
	shouldConverted := true
	targetValue := ""
	// If all values are the same and DiskQuotaMap doesn't contains ".*", we should convert.
	for key, value := range diskQuotaMap {
		// Use the first value as the comparing object.
		if targetValue == "" {
			targetValue = value
		}
		if value != targetValue || key == DiskQuotaLimitAllKey {
			shouldConverted = false
		}
	}

	if shouldConverted {
		diskQuotaKey := DiskQuotaLimitRootFsOnly
		for _, key := range keys {
			// Ignore DiskQuotaLimitRootFsOnly and DiskQuotaLimitAllKey.
			if key == DiskQuotaLimitRootFsOnly {
				continue
			}
			diskQuotaKey = diskQuotaKey + "&" + key
		}
		return map[string]string{diskQuotaKey: targetValue}
	}

	return diskQuotaMap
}

// ParseDiskQuotaToLabel can convert diskQuota map into a label string.
// 1 DiskQuota=/=60g;/data1=50
// 2 DiskQutoa=60g
// Used for Alidocker
func ParseDiskQuotaToLabel(diskQuota map[string]string) string {
	var buffer bytes.Buffer

	for k, v := range diskQuota {
		// ".*" means the limitation of rootfs and volume.
		// https://github.com/alibaba/pouch/blob/master/docs/features/pouch_with_diskquota.md#parameter-details
		if k == DiskQuotaLimitAllKey {
			// Convert ".*" to the type such as "10g".
			buffer.WriteString(v)
			buffer.WriteString(";")
			continue
		}

		// Convert "/&/home/test1&/home/test2=10g" to "/home/test1=10g;/home/test2=10g;/=10g"
		shouldSetRootFsOnlyDiskQuota := false
		subKeys := strings.Split(k, "&")
		for _, subKey := range subKeys {
			if subKey == DiskQuotaLimitRootFsOnly {
				shouldSetRootFsOnlyDiskQuota = true
				continue
			}
			buffer.WriteString(subKey)
			buffer.WriteString("=")
			buffer.WriteString(v)
			buffer.WriteString(";")
		}
		if shouldSetRootFsOnlyDiskQuota {
			buffer.WriteString(DiskQuotaLimitRootFsOnly)
			buffer.WriteString("=")
			buffer.WriteString(v)
			buffer.WriteString(";")
		}
	}
	diskQuotaLabel := buffer.String()
	return strings.Trim(diskQuotaLabel, ";")
}
