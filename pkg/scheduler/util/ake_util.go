package util

import (
	"encoding/json"
	"fmt"
	"strconv"

	log "github.com/golang/glog"
	sigmak8s "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func AllocSpecFromPod(pod *v1.Pod) *sigmak8s.AllocSpec {
	spec := &sigmak8s.AllocSpec{}
	if pod == nil {
		return nil
	}

	specData, ok := pod.Annotations[sigmak8s.AnnotationPodAllocSpec]
	if !ok || len(specData) == 0 {
		return nil
	}

	if err := json.Unmarshal([]byte(specData), spec); err != nil {
		log.Errorf("unmarshal allocspec from pod[%s] failed: %v", pod.Name, err)
		return nil
	}
	return spec
}

func LocalInfoFromNode(node *v1.Node) *sigmak8s.LocalInfo {
	if node == nil {
		return nil
	}

	localData, ok := node.Annotations[sigmak8s.AnnotationLocalInfo]
	if !ok {
		return nil
	}

	local := &sigmak8s.LocalInfo{}
	if err := json.Unmarshal([]byte(localData), local); err != nil {
		log.Errorf("unmarshal localinfo from node[%s] failed: %v", node.Name, err)
		return nil
	}

	return local
}

func CreatePodPatch(curPod, modPod *v1.Pod) ([]byte, error) {
	curPodJSON, err := json.Marshal(curPod)
	if err != nil {
		return nil, fmt.Errorf("failed json marshal patch current pod: %v", err)
	}
	modPodJSON, err := json.Marshal(modPod)
	if err != nil {
		return nil, fmt.Errorf("failed json marshal patch modify pod: %v", err)
	}
	patch, err := strategicpatch.CreateTwoWayMergePatch(curPodJSON, modPodJSON, v1.Pod{})
	if err != nil {
		return nil, fmt.Errorf("create patch failed: %v", err)
	}
	return patch, nil
}

// TODO(yuzhi.wx) duplicated code of predicates.go
func CPUOverQuotaRatio(node *v1.Node) (float64, bool) {
	if v, exists := node.Labels[sigmak8sapi.LabelCPUOverQuota]; exists {
		if ratio, err := strconv.ParseFloat(v, 64); err == nil {
			return ratio, true
		}
	}
	return 1.0, false
}



func MemoryOverQuotaRatio(node *v1.Node) (float64, bool) {
	if v, exists := node.Labels[sigmak8sapi.LabelMemOverQuota]; exists {
		if ratio, err := strconv.ParseFloat(v, 64); err == nil {
			return ratio, true
		}
	}
	return 1.0, false
}

func EphemeralStorageOverQuotaRatio(node *v1.Node) (float64, bool) {
	if v, exists := node.Labels[sigmak8sapi.LabelDiskOverQuota]; exists {
		if ratio, err := strconv.ParseFloat(v, 64); err == nil {
			return ratio, true
		}
	}
	return 1.0, false
}