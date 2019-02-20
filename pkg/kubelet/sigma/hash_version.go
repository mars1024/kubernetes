package sigma

import (
	"regexp"

	"github.com/blang/semver"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/kubernetes/pkg/features"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
)

const (
	VERSION_CURRENT = "1.12.2"
	VERSION_1_10    = "1.10.0"

	VERSION_LOWEST = "0.0.0"
)

// -1 == versionStr1 is less than versionStr2
// 0 == versionStr1 is equal to versionStr2
// 1 == versionStr1 is greater than versionStr2
// HashVersion "" is the default version, we just reguard as "0.0.0".
// That means all sigmalet can compute container's hash directly and correctly.
func CompareHashVersion(versionStr1, versionStr2 string) (int, error) {
	if versionStr1 == "" {
		versionStr1 = VERSION_LOWEST
	}

	if versionStr2 == "" {
		versionStr2 = VERSION_LOWEST
	}

	version1, err := semver.New(versionStr1)
	if err != nil {
		return 0, err
	}

	version2, err := semver.New(versionStr2)
	if err != nil {
		return 0, err
	}

	return version1.Compare(*version2), nil
}

// GetHashDecorateFunc returns a DecorateFunc to modify the container spec string.
// Important: Always check up this func for sigmalet with new version.
func GetHashDecorateFunc(hashVersion string) hashutil.DecorateFunc {
	switch hashVersion {
	case VERSION_CURRENT:
		return hackGoStringDefault
	case VERSION_1_10:
		return hackGoString112to110
	case "":
		if utilfeature.DefaultFeatureGate.Enabled(features.DefaultHashVersionTo110) {
			return hackGoString112to110
		}
		return hackGoStringDefault
	default:
		// Invalid hashVersion
		return hackGoStringDefault
	}
}

// hackGoStringDefault just return string directly.
func hackGoStringDefault(str string) string {
	return str
}

// hackGoString112to110 can convert container spec from 1.12 to 1.10.
func hackGoString112to110(str string) string {
	re := regexp.MustCompile(`\s*ProcMount:(.*?)([\s}])`)
	hackedString := re.ReplaceAllString(str, `${2}`)
	return hackedString
}
