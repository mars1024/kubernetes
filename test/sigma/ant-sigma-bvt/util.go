package ant_sigma_bvt

import (
	"strconv"
)

func Atoi64(str string, def int64) int64 {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return def
	}
	return i
}

func Quota2Byte(quota string) int64 {
	return Quota2ByteWithDefault(quota, 10240*1024*1024)
}

func Quota2ByteWithDefault(quota string, defaultValue int64) int64 {
	if quota == "" {
		return defaultValue
	}
	res := int64(128)
	lastIndex := len(quota) - 1
	switch unit := quota[lastIndex]; unit {
	case 'g', 'G':
		res = Atofloat64(quota[:lastIndex], -1) * 1024 * 1024 * 1024
	case 'm', 'M':
		res = Atofloat64(quota[:lastIndex], -1) * 1024 * 1024
	case 'k', 'K':
		res = Atofloat64(quota[:lastIndex], -1) * 1024
	default:
		res = Atofloat64(quota, -1)
	}
	if res < 1 {
		return defaultValue //保护模式，0的时候zeus业务调度器会报错
	} else {
		return res
	}
}

func Atofloat64(str string, def int64) int64 {
	i, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return def
	}
	return int64(i)
}
