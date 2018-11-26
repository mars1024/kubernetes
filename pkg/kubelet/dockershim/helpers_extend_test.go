package dockershim

import (
	"testing"

	"github.com/stretchr/testify/assert"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func TestParseEnvList(t *testing.T) {
	envs := []string{
		"KUBERNETES_SERVICE_PORT_HTTPS=443",
		"KUBERNETES_PORT_443_TCP_PORT=443",
		"KUBERNETES_SERVICE_PORT=443",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"DefaultMask=24=23",
		"DefaultRoute",
	}
	expect := []*runtimeapi.KeyValue{
		{
			Key:   "KUBERNETES_SERVICE_PORT_HTTPS",
			Value: "443",
		},
		{
			Key:   "KUBERNETES_PORT_443_TCP_PORT",
			Value: "443",
		},
		{
			Key:   "KUBERNETES_SERVICE_PORT",
			Value: "443",
		},
		{
			Key:   "PATH",
			Value: "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
	}
	result := parseEnvList(envs)
	assert.Equal(t, expect, result)
}
