package nodestatus

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/golang/glog"
	cadvisorapiv1 "github.com/google/cadvisor/info/v1"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
)

// getCPUTopology get cpu topology info from cadvisor machine info.
func getCPUTopology(machineInfoFunc func() (*cadvisorapiv1.MachineInfo, error)) ([]sigmak8sapi.CPUInfo, error) {
	cpuInfoArray := make([]sigmak8sapi.CPUInfo, 0)

	machineInfo, err := machineInfoFunc()
	if err != nil {
		glog.Errorf("get machine info from cadvisor api err :%v", err)
		return cpuInfoArray, err
	}

	for _, socket := range machineInfo.Topology {
		for _, core := range socket.Cores {
			for _, cpu := range core.Threads {
				cpuInfoArray = append(cpuInfoArray, sigmak8sapi.CPUInfo{
					CPUID:    int32(cpu),
					CoreID:   int32(core.Id),
					SocketID: int32(socket.Id),
				})
			}
		}
	}
	sort.Slice(cpuInfoArray, func(i, j int) bool {
		return cpuInfoArray[i].CPUID < cpuInfoArray[j].CPUID
	})

	return cpuInfoArray, nil
}

// LocalInfo update node localInfo include cpu info and disk info
// it executing following steps:
// 1.get cpuInfo
// 2.get disk info
// 3.update node annotation
func LocalInfo(machineInfoFunc func() (*cadvisorapiv1.MachineInfo, error), // typically Kubelet.GetCachedMachineInfo
) Setter {
	return func(node *v1.Node) error {
		// step 1: get cpu info
		cpuTopologyInfo, err := getCPUTopology(machineInfoFunc)
		if err != nil {
			glog.Errorf("get cpu topology err: %v", err)
		}
		localInfo := sigmak8sapi.LocalInfo{
			CPUInfos: cpuTopologyInfo,
		}

		// step 2: get disk info
		//TODO

		// step 3: update node annotation.
		if node.Annotations == nil {
			node.Annotations = make(map[string]string)
		}

		localInfoJSON, err := json.Marshal(localInfo)
		if err != nil {
			msg := fmt.Sprintf("json marshal: %+v err, error is %s", localInfo, err.Error())
			glog.Error(msg)
			node.Annotations[sigmak8sapi.AnnotationLocalInfo] = msg
		} else {
			node.Annotations[sigmak8sapi.AnnotationLocalInfo] = string(localInfoJSON)
		}
		return nil
	}
}
