package dockershim

import (
	"strings"

	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

// parseEnvList converts a list of strings to KeyValue list, from the form of
// '<key>=<value>', which can be understood by docker.
func parseEnvList(envs []string) (result []*runtimeapi.KeyValue) {
	for _, env := range envs {
		kv := strings.Split(env, "=")
		if len(kv) != 2 {
			continue
		}
		result = append(result, &runtimeapi.KeyValue{
			Key:   kv[0],
			Value: kv[1],
		})
	}
	return
}
