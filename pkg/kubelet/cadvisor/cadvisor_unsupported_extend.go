// +build !linux,!windows linux,!cgo

package cadvisor

import (
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
)

func (cu *cadvisorUnsupported) ContainerSpec(containerName string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerSpec, error) {
	return nil, unsupportedErr
}
