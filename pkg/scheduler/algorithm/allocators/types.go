package allocators

import "fmt"

type GPUSetMode string

var (
	// TODO(yuzhi.wx) set this pod label
	//ExclusiveCPU = "sigma.ali/exclusive-cpu"
	ExclusiveCPU = "com.alipay.acs.container.cpu.exclusive"
)

// PredicateFailureError describes a failure error of predicate.
type AllocatorFailureError struct {
	AllocatorName string
	FailureDesc   string
}

func newAllocatorFailureError(name, reason string) *AllocatorFailureError {
	return &AllocatorFailureError{
		name,
		reason,
	}
}
func (a AllocatorFailureError) Error() string {
	return fmt.Sprintf("name: %s, reason: %s", a.AllocatorName, a.FailureDesc)
}

var (
	ErrAllocatorFailure = newAllocatorFailureError
)
//
//
//func (s SpreadStrategy) FromString(source string) SpreadStrategy {
//	source = strings.ToLower(source)
//	var result SpreadStrategy
//	switch source {
//	case "spread":
//		result = SpreadStrategySpread
//		break
//	case "samecorefirst":
//		result = SpreadStrategySameCoreFirst
//		break
//	default:
//		result = SpreadStrategySameCoreFirst
//	}
//	return result
//}
//
//func (m CPUSetMode) FromString(source string) CPUSetMode {
//	source = strings.ToLower(source)
//	var result CPUSetMode
//	switch source {
//	case "exclusive":
//		result = CPUSetMode_Exclusive
//		break
//	case "mutex":
//		result = CpuSetModeMutex
//		break
//	default:
//		result = CPUSetMode_Cpushare
//	}
//	return result
//}
