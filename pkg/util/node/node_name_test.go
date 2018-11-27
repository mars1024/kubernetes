package node

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNodeName(t *testing.T) {
	hostname, err := GetHostname("")
	assert.NoError(t, err)

	if runtime.GOOS == "darwin" {
		assert.Panics(t, func() { GetNodeName("") }, "In darwin env, nodeName cannot be SN and panic")
	} else if runtime.GOOS == "linux" {
		nodeName := GetNodeName("")
		assert.True(t, string(nodeName) != hostname, "In linux env, nodeName is sn, "+
			"should diff from hostname, but nodeName is :%s, hostname is :%s", nodeName, hostname)
	}
}
