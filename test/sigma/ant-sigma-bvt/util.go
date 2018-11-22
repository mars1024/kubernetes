package ant_sigma_bvt

import (
	"io"
	"io/ioutil"
	"strconv"
	"strings"
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

// parseResolveConf reads a resolv.conf file from the given reader, and parses
// it into nameservers, searches and options, possibly returning an error.
func parseResolvConf(reader io.Reader) (nameservers []string, searches []string, options []string, err error) {
	file, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, nil, nil, err
	}

	// Lines of the form "nameserver 1.2.3.4" accumulate.
	nameservers = []string{}

	// Lines of the form "search example.com" overrule - last one wins.
	searches = []string{}

	// Lines of the form "option ndots:5 attempts:2" overrule - last one wins.
	// Each option is recorded as an element in the array.
	options = []string{}

	lines := strings.Split(string(file), "\n")
	for l := range lines {
		trimmed := strings.TrimSpace(lines[l])
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "nameserver" && len(fields) >= 2 {
			nameservers = append(nameservers, fields[1])
		}
		if fields[0] == "search" {
			searches = fields[1:]
		}
		if fields[0] == "options" {
			options = fields[1:]
		}
	}
	return nameservers, searches, options, nil
}

func IsSubslice(parent, sub []string) bool {
	parentMap := map[string]string{}
	for _, item := range parent {
		parentMap[item] = item
	}
	for _, item := range sub {
		_, ok := parentMap[item]
		if !ok {
			return false
		}
	}
	return true
}

func IsEqualSlice(parent, sub []string) bool {
	if len(parent) == len(sub) && IsSubslice(parent, sub) {
		return true
	}
	return false
}
