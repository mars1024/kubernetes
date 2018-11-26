package testing

import (
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
)

func (c *Fake) ContainerSpec(containerName string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerSpec, error) {
	return map[string]cadvisorapiv2.ContainerSpec{}, nil
}
