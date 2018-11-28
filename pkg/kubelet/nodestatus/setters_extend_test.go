package nodestatus


import (
	"testing"
	"encoding/json"
	"os"
	"net"
	"fmt"
	"k8s.io/api/core/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/diff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	cadvisorapi "github.com/google/cadvisor/info/v1"

)

const (
	testKubeletHostIP = "127.0.0.1"
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


func TestSetNodeAddressWithoutCloudProvider(t *testing.T) {
	hostname, err := os.Hostname()
	assert.NoError(t, err)

	var testKubeletIp string

	addrs, err := net.InterfaceAddrs()
	assert.NoError(t, err)
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if err := validateNodeIP(ip); err == nil {
			testKubeletIp = ip.String()
		}
	}

	type test struct {
		existingNode      v1.Node
		expectedAddresses []v1.NodeAddress
		testName          string
		success           bool
	}

	fmt.Println(testKubeletIp)
	tests := []test{
		{
			existingNode: v1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: testKubeletIp},
			},
			expectedAddresses: []v1.NodeAddress{
				{
					Type:    v1.NodeInternalIP,
					Address: testKubeletIp,
				},
				{
					Type:    v1.NodeHostName,
					Address: hostname,
				},
			},
			success:  true,
			testName: "node name is valid ip, so get host ip from node name",
		},
		{
			existingNode: v1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: testKubeletHostIP},
			},
			expectedAddresses: []v1.NodeAddress{
				{
					Type:    v1.NodeInternalIP,
					Address: testKubeletHostIP,
				},
				{
					Type:    v1.NodeHostName,
					Address: hostname,
				},
			},
			success: false,
			testName: "node name is invalid ip," +
				"in this condition expected ip address is invalid ip, so not equal nodename in fact",
		},
	}



	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// construct setter
			setter := NodeAddress(nil,
				validateNodeIP,
				hostname,
				false,
				false,
				nil,
				nil)

			// call setter on existing node
			err := setter(&test.existingNode)
			assert.NoError(t,err)
			if test.success {
				assert.True(t, apiequality.Semantic.DeepEqual(test.expectedAddresses, test.existingNode.Status.Addresses),
					"%s", diff.ObjectDiff(test.expectedAddresses, test.existingNode.Status.Addresses))
			} else {
				assert.False(t, apiequality.Semantic.DeepEqual(test.expectedAddresses, test.existingNode.Status.Addresses),
					"%s", diff.ObjectDiff(test.expectedAddresses, test.existingNode.Status.Addresses))
			}
		})
	}
}

// Validate given node IP belongs to the current host
func validateNodeIP(nodeIP net.IP) error {
	// Honor IP limitations set in setNodeStatus()
	if nodeIP.To4() == nil && nodeIP.To16() == nil {
		return fmt.Errorf("nodeIP must be a valid IP address")
	}
	if nodeIP.IsLoopback() {
		return fmt.Errorf("nodeIP can't be loopback address")
	}
	if nodeIP.IsMulticast() {
		return fmt.Errorf("nodeIP can't be a multicast address")
	}
	if nodeIP.IsLinkLocalUnicast() {
		return fmt.Errorf("nodeIP can't be a link-local unicast address")
	}
	if nodeIP.IsUnspecified() {
		return fmt.Errorf("nodeIP can't be an all zeros address")
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip != nil && ip.Equal(nodeIP) {
			return nil
		}
	}
	return fmt.Errorf("node IP: %q not found in the host's network interfaces", nodeIP.String())
}