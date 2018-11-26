package nodestatus


import (
	"testing"

	cadvisorapi "github.com/google/cadvisor/info/v1"
	"github.com/stretchr/testify/assert"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"

	"encoding/json"
)

func TestSetNodeLocalInfo(t *testing.T) {
	newMachineInfo := func(numCores int, topology []cadvisorapi.Node) *cadvisorapi.MachineInfo {
		return &cadvisorapi.MachineInfo{
			NumCores: numCores,
			Topology: topology,
		}
	}

	cases := []struct {
		name            string
		machineInfo     *cadvisorapi.MachineInfo
		expectLocalInfo sigmak8sapi.LocalInfo
	}{
		{
			name: "success case",
			machineInfo: newMachineInfo(8, []cadvisorapi.Node{
				{Id: 0,
					Cores: []cadvisorapi.Core{
						{Id: 0, Threads: []int{0, 2}},
						{Id: 1, Threads: []int{1, 3}},
					},
				},
				{Id: 1,
					Cores: []cadvisorapi.Core{
						{Id: 2, Threads: []int{4, 6}},
						{Id: 3, Threads: []int{5, 7}},
					},
				},
			}),
			expectLocalInfo: sigmak8sapi.LocalInfo{
				CPUInfos: []sigmak8sapi.CPUInfo{
					{
						CPUID:    0,
						CoreID:   0,
						SocketID: 0,
					},
					{
						CPUID:    1,
						CoreID:   1,
						SocketID: 0,
					},
					{
						CPUID:    2,
						CoreID:   0,
						SocketID: 0,
					},
					{
						CPUID:    3,
						CoreID:   1,
						SocketID: 0,
					},
					{
						CPUID:    4,
						CoreID:   2,
						SocketID: 1,
					},
					{
						CPUID:    5,
						CoreID:   3,
						SocketID: 1,
					},
					{
						CPUID:    6,
						CoreID:   2,
						SocketID: 1,
					},
					{
						CPUID:    7,
						CoreID:   3,
						SocketID: 1,
					},
				},
			},
		},
		{
			name:            "machine info about  node is zero, so cpu info is empty",
			machineInfo:     newMachineInfo(0, []cadvisorapi.Node{}),
			expectLocalInfo: sigmak8sapi.LocalInfo{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			machineInfoFunc := func() (*cadvisorapi.MachineInfo, error) {
				return testCase.machineInfo, nil
			}
			actualNode := &v1.Node{}

			// construct setter
			setter := LocalInfo(machineInfoFunc)
			err := setter(actualNode)

			assert.True(t, len(actualNode.Annotations) > 0, "annotation should have 1 element at least")

			localInfoAnnotationJSON, ok := actualNode.GetAnnotations()[sigmak8sapi.AnnotationLocalInfo]
			assert.True(t, ok, "annotation not exist")

			localInfo := &sigmak8sapi.LocalInfo{}
			err = json.Unmarshal([]byte(localInfoAnnotationJSON), localInfo)
			assert.NoError(t, err, "unmarshal err")

			assert.Equal(t, testCase.expectLocalInfo.CPUInfos, localInfo.CPUInfos)
		})
	}
}

