package node

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"fmt"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/types"
)

// GetNodeName returns the node name.
func GetNodeName(nodeNameOverride string) types.NodeName {
	nodeName := nodeNameOverride
	if nodeName == "" {
		nodeName = nodeSN()
	}
	return types.NodeName(nodeName)
}

// nodeSN returns the node name get from armoryinfo cmd; otherwise, get from dmidecode cmd;
func nodeSN() string {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	out, err := exec.CommandContext(ctx, "/bin/sh", "-c",
		"/usr/alisys/dragoon/libexec/armory/bin/armoryinfo  servicetag").Output()
	if err == nil {
		return strings.ToLower(strings.Trim(string(out), "\n"))
	}
	glog.Errorf("Get hostname by amroy cmd  failed: %v,out:%v,context err:%v", err, out, ctx.Err())

	out, err = exec.CommandContext(ctx, "/bin/sh", "-c", "dmidecode -s system-serial-number").Output()
	if err == nil {
		return strings.ToLower(strings.Trim(string(out), "\n"))
	}
	panic(fmt.Sprintf("Get hostname by dmidecode cmd failed: %v,out:%v,context err:%v", err, out, ctx.Err()))
}
