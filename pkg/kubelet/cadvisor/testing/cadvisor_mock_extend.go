package testing

import (
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
)

func (c *Mock) ContainerSpec(containerName string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerSpec, error) {
	args := c.Called(containerName,options)
	return args.Get(0).(map[string]cadvisorapiv2.ContainerSpec), args.Error(1)
}
