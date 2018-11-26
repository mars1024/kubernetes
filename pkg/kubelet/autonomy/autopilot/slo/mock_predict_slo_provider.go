package slo

import (
	"github.com/stretchr/testify/mock"
)

// MockRuntimeSLOPredictValue just mock nodeSketch.
type MockRuntimeSLOPredictValue struct {
	mock.Mock
}

//GetContainerRuntimeSLOValue return container latest cpi or qps or rt sketch value.
func (n *MockRuntimeSLOPredictValue) GetContainerRuntimeSLOValue(sketchType ContainerSLOType, podnamespace string, podname string, containerName string) (float32, error) {
	var value float32
	switch sketchType {
	case CPIViolate:
		value = CPINULL
	case RTViolate:
		value = RTNULL
	case QPSViolate:
		value = QPSNULL
	}
	return value, nil
}
