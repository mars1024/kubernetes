package slo

import "strings"

// ResourceViolateType desc which resource violate.such as cpi,rt,qps.
type ContainerSLOType int

const (
	//CPIViolate cpi violate.
	CPIViolate  ContainerSLOType = iota //value --> 0
	//RTViolate rt violate.
	RTViolate
	//QPSViolate qps violate.
	QPSViolate
)
const (
	//CPINULL without get container's cpi value.
	CPINULL = -1
	//RTNULL without get container's rt value.
	RTNULL = -1
	//QPSNULL without get container's qps
	QPSNULL = -1
)

func (resType ContainerSLOType) String() string {
	switch resType {
	case CPIViolate:
		return "CPIViloate"
	case RTViolate:
		return "RTViolate"
	case QPSViolate:
		return "QPSViolate"
	default:
		return "Unknow"
	}
}

// ArrayToString transform array to string with ,
func ArrayToString(vs []ContainerSLOType) string {
	var strs []string
	for _, item := range vs {
		strs = append(strs, item.String())
	}
	return strings.Join(strs[:], ",")
}

// RuntimeSLOValuePredict from which get the container SLO value to violate check.
type RuntimeSLOPredictValue interface {
	//GetContainerRuntimeSLOValue return container latest cpi or rt or qps value,according sketchType param.
	GetContainerRuntimeSLOValue(sketchType ContainerSLOType, podnamespace string, podname string, containerName string) (float32, error)
}
