// +build windows

package cadvisor

import (
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
)

func (cc *cadvisorClient) ContainerSpec(containerName string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerSpec, error) {
	return nil, nil
}
