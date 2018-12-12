package kuberuntime

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	alipaysigmav2 "gitlab.alipay-inc.com/sigma/apis/pkg/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func TestGetAnonymousVolumesMount(t *testing.T) {
	testCase := []struct {
		name            string
		annotationValue *sigmak8sapi.RebuildContainerInfo
		expectValue     []*runtimeapi.Mount
	}{
		{
			name:        "annotation is nil, so mount is nil",
			expectValue: nil,
		},
		{
			name:        "volume is nil, so mount is nil",
			expectValue: nil,
			annotationValue: &sigmak8sapi.RebuildContainerInfo{
				ContainerID: "123-test",
			},
		},
		{
			name: "everything is ok",
			annotationValue: &sigmak8sapi.RebuildContainerInfo{
				ContainerID: "123-test",
				AnonymousVolumesMounts: []sigmak8sapi.MountPoint{
					{
						Source:      "s1",
						Destination: "d1",
						RW:          true,
					},
					{
						Source:      "s2",
						Destination: "d2",
					},
				},
			},
			expectValue: []*runtimeapi.Mount{
				{
					ContainerPath: "d1",
					HostPath:      "s1",
					Readonly:      false,
				},
				{
					ContainerPath: "d2",
					HostPath:      "s2",
					Readonly:      true,
				},
			},
		},
	}

	for _, cs := range testCase {
		t.Run(cs.name, func(t *testing.T) {
			pod := &v1.Pod{}
			if cs.annotationValue != nil {
				annotationValue, err := json.Marshal(cs.annotationValue)
				assert.NoError(t, err)

				pod.Annotations = map[string]string{
					sigmak8sapi.AnnotationRebuildContainerInfo: string(annotationValue),
				}
			}
			mount := getAnonymousVolumesMount(pod)
			assert.Equal(t, mount, cs.expectValue)
		})
	}
}

func TestGetDiskQuotaID(t *testing.T) {
	testCase := []struct {
		name            string
		annotationValue *sigmak8sapi.RebuildContainerInfo
		expectValue     string
	}{
		{
			name: "annotation is nil, so quotaID is empty",
		},
		{
			name: "disk quota is nil, so quotaID is empty",
			annotationValue: &sigmak8sapi.RebuildContainerInfo{
				ContainerID: "123-test",
			},
		},
		{
			name: "everything is ok",
			annotationValue: &sigmak8sapi.RebuildContainerInfo{
				ContainerID: "123-test",
				DiskQuotaID: "1234",
			},
			expectValue: "1234",
		},
	}

	for _, cs := range testCase {
		t.Run(cs.name, func(t *testing.T) {
			pod := &v1.Pod{}
			if cs.annotationValue != nil {
				annotationValue, err := json.Marshal(cs.annotationValue)
				assert.NoError(t, err)

				pod.Annotations = map[string]string{
					sigmak8sapi.AnnotationRebuildContainerInfo: string(annotationValue),
				}
			}
			quotaID := GetDiskQuotaID(pod)
			assert.Equal(t, quotaID, cs.expectValue)
		})
	}
}

func TestGetCidrIpMask(t *testing.T) {
	for desc, test := range map[string]struct {
		maskLen       int
		expectMaskStr string
	}{
		"0.0.0.0": {
			maskLen:       0,
			expectMaskStr: "0.0.0.0",
		},
		"224.0.0.0": {
			maskLen:       3,
			expectMaskStr: "224.0.0.0",
		},
		"255.0.0.0": {
			maskLen:       8,
			expectMaskStr: "255.0.0.0",
		},
		"255.252.0.0": {
			maskLen:       14,
			expectMaskStr: "255.252.0.0",
		},
		"255.255.0.0": {
			maskLen:       16,
			expectMaskStr: "255.255.0.0",
		},
		"255.255.248.0": {
			maskLen:       21,
			expectMaskStr: "255.255.248.0",
		},
		"255.255.255.0": {
			maskLen:       24,
			expectMaskStr: "255.255.255.0",
		},
		"255.255.255.128": {
			maskLen:       25,
			expectMaskStr: "255.255.255.128",
		},
		"255.255.255.248": {
			maskLen:       29,
			expectMaskStr: "255.255.255.248",
		},
		"255.255.255.255": {
			maskLen:       32,
			expectMaskStr: "255.255.255.255",
		},
	} {
		actualMaskStr := getCidrIPMask(test.maskLen)
		if actualMaskStr != test.expectMaskStr {
			t.Errorf("TestCase %s: expect %s, but got %s", desc, test.expectMaskStr, actualMaskStr)
		}
	}
}

func TestGetEnvsFromNetworkStatus(t *testing.T) {
	for desc, test := range map[string]struct {
		networkStatus  *sigmak8sapi.NetworkStatus
		expectedEnvMap map[string]string
	}{
		"network status": {
			networkStatus: &sigmak8sapi.NetworkStatus{
				VlanID:              "700",
				NetworkPrefixLength: 22,
				Gateway:             "100.81.187.247",
				Ip:                  "100.81.187.21",
			},
			expectedEnvMap: map[string]string{
				envDefaultMask:  "255.255.252.0",
				envRequestIP:    "100.81.187.21",
				envDefaultRoute: "100.81.187.247",
				envDefaultNic:   "bond0.700",
			},
		},
		"network status with ecs network": {
			networkStatus: &sigmak8sapi.NetworkStatus{
				VlanID:              "700",
				NetworkPrefixLength: 22,
				Gateway:             "100.81.187.247",
				Ip:                  "100.81.187.21",
				NetType:             "ecs",
			},
			expectedEnvMap: map[string]string{
				envDefaultMask:  "255.255.252.0",
				envRequestIP:    "100.81.187.21",
				envDefaultRoute: "100.81.187.247",
				envDefaultNic:   "docker0",
				envVpcECS:       "true",
			},
		},
	} {
		envs := getEnvsFromNetworkStatus(test.networkStatus)
		for _, env := range envs {
			value, exists := test.expectedEnvMap[env.Key]
			if !exists {
				t.Errorf("TestCase %s: got unexpect env %v", desc, env)
			}
			if value != env.Value {
				t.Errorf("TestCase %s: key %s expect value %s, but got %s", desc, env.Key, value, env.Value)
			}
		}
	}
}

func TestGenerateTopologyEnvs(t *testing.T) {
	for desc, test := range map[string]struct {
		node           *v1.Node
		expectedEnvMap map[string]string
	}{
		"node has labels": {
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "host1",
					Labels: map[string]string{
						sigmak8sapi.LabelSite:                 "site-test",
						sigmak8sapi.LabelRegion:               "region-test",
						sigmak8sapi.LabelNodeSN:               "nodesn-test",
						sigmak8sapi.LabelHostname:             "hostname-test",
						sigmak8sapi.LabelNodeIP:               "ip-test",
						sigmak8sapi.LabelRoom:                 "room-test",
						sigmak8sapi.LabelRack:                 "rack-test",
						sigmak8sapi.LabelParentServiceTag:     "servicetag-test",
						sigmak8sapi.LabelNetArchVersion:       "archversion-test",
						alipaysigmak8sapi.LabelUplinkHostname: "uplinghostname-test",
						alipaysigmak8sapi.LabelUplinkIP:       "uplinkip-test",
						alipaysigmak8sapi.LabelUplinkSN:       "uplinksn-test",
						sigmak8sapi.LabelASW:                  "asw-test",
						sigmak8sapi.LabelLogicPOD:             "logicpod-test",
						sigmak8sapi.LabelPOD:                  "pod-test",
						sigmak8sapi.LabelDSWCluster:           "dswcluster-test",
						sigmak8sapi.LabelNetLogicSite:         "netlogicsite-test",
						sigmak8sapi.LabelMachineModel:         "machinemodel-test",
						alipaysigmak8sapi.LabelModel:          "model-test",
					},
				},
			},
			expectedEnvMap: map[string]string{
				alipaysigmav2.EnvSafetyOut:             "0",
				alipaysigmav2.EnvSigmaSite:             "site-test",
				alipaysigmav2.EnvSigmaRegion:           "region-test",
				alipaysigmav2.EnvSigmaNCSN:             "nodesn-test",
				alipaysigmav2.EnvSigmaNCHostname:       "hostname-test",
				alipaysigmav2.EnvSigmaNCIP:             "ip-test",
				alipaysigmav2.EnvSigmaParentServiceTag: "servicetag-test",
				alipaysigmav2.EnvSigmaRoom:             "room-test",
				alipaysigmav2.EnvSigmaRack:             "rack-test",
				alipaysigmav2.EnvSigmaNetArchVersion:   "archversion-test",
				alipaysigmav2.EnvSigmaUplinkHostName:   "uplinghostname-test",
				alipaysigmav2.EnvSigmaUplinkIP:         "uplinkip-test",
				alipaysigmav2.EnvSigmaUplinkSN:         "uplinksn-test",
				alipaysigmav2.EnvSigmaASW:              "asw-test",
				alipaysigmav2.EnvSigmaLogicPod:         "logicpod-test",
				alipaysigmav2.EnvSigmaPod:              "pod-test",
				alipaysigmav2.EnvSigmaDSWCluster:       "dswcluster-test",
				alipaysigmav2.EnvSigmaNetLogicSite:     "netlogicsite-test",
				alipaysigmav2.EnvSigmaSMName:           "machinemodel-test",
				alipaysigmav2.EnvSigmaModel:            "model-test",
			},
		},
		"node has no label": {
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "host1",
					Labels: map[string]string{},
				},
			},
			expectedEnvMap: map[string]string{
				alipaysigmav2.EnvSafetyOut:             "0",
				alipaysigmav2.EnvSigmaSite:             "",
				alipaysigmav2.EnvSigmaRegion:           "",
				alipaysigmav2.EnvSigmaNCSN:             "",
				alipaysigmav2.EnvSigmaNCHostname:       "",
				alipaysigmav2.EnvSigmaNCIP:             "",
				alipaysigmav2.EnvSigmaParentServiceTag: "",
				alipaysigmav2.EnvSigmaRoom:             "",
				alipaysigmav2.EnvSigmaRack:             "",
				alipaysigmav2.EnvSigmaNetArchVersion:   "",
				alipaysigmav2.EnvSigmaUplinkHostName:   "",
				alipaysigmav2.EnvSigmaUplinkIP:         "",
				alipaysigmav2.EnvSigmaUplinkSN:         "",
				alipaysigmav2.EnvSigmaASW:              "",
				alipaysigmav2.EnvSigmaLogicPod:         "",
				alipaysigmav2.EnvSigmaPod:              "",
				alipaysigmav2.EnvSigmaDSWCluster:       "",
				alipaysigmav2.EnvSigmaNetLogicSite:     "",
				alipaysigmav2.EnvSigmaSMName:           "",
				alipaysigmav2.EnvSigmaModel:            "",
			},
		},
	} {
		envs := generateTopologyEnvs(test.node)
		for _, env := range envs {
			value, exists := test.expectedEnvMap[env.Key]
			if !exists {
				t.Errorf("TestCase %s: got unexpect env %v", desc, env)
			}
			if value != env.Value {
				t.Errorf("TestCase %s: key %s expect value %s, but got %s", desc, env.Key, value, env.Value)
			}
		}
	}
}
