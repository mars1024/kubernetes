package api

import (
	"testing"
	"reflect"
	"encoding/json"
	"github.com/stretchr/testify/assert"
)

func TestGetContainerDesiredStateFromAnnotation(t *testing.T) {
	containerNameFirst := "foo1"
	containerNameSecond := "foo2"
	containerNameThird := "foo3"
	containerStateSpec :=  ContainerStateSpec{
		States: map[ ContainerInfo] ContainerState{
			 ContainerInfo{Name: containerNameFirst}:   ContainerStateExited,
			 ContainerInfo{Name: containerNameSecond}:  ContainerStateRunning,
			 ContainerInfo{Name: containerNameThird}:   ContainerStateUnknown,
		},
	}
	containerStateSpecByte, err := json.Marshal(containerStateSpec)
	assert.NoError(t, err)

	var containerStateSpecFromJson  ContainerStateSpec
	err = json.Unmarshal(containerStateSpecByte,&containerStateSpecFromJson)
	assert.NoError(t, err)

	assert.True(t,reflect.DeepEqual(containerStateSpec,containerStateSpecFromJson))
}
