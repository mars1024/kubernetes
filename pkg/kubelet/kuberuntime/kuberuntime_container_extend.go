package kuberuntime

import (
	"fmt"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/util/format"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	alipaysigmav2 "gitlab.alipay-inc.com/sigma/apis/pkg/v2"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	kubepod "k8s.io/kubernetes/pkg/kubelet/pod"
	sigmautil "k8s.io/kubernetes/pkg/kubelet/sigma"
)

var (
	// envRequestIP is the env key of "RequestedIP"
	envRequestIP = "RequestedIP"
	// envDefaultMask is the env key of "DefaultMask"
	envDefaultMask = "DefaultMask"
	// envDefaultNic is the env key of "DefaultNic"
	envDefaultNic = "DefaultNic"
	// envDefaultRoute is the env key of "DefaultRoute"
	envDefaultRoute = "DefaultRoute"
	// envVpcECS is the env key of "VpcECS"
	envVpcECS = "VpcECS"
)

// getAnonymousVolumesMount get anonymous volume from pod  annotation.
func getAnonymousVolumesMount(pod *v1.Pod) []*runtimeapi.Mount {
	rebuildContainerInfo, err := sigmautil.GetContainerRebuildInfoFromAnnotation(pod)
	if err != nil {
		glog.V(4).Info(err.Error())
		return nil
	}
	if rebuildContainerInfo == nil || len(rebuildContainerInfo.AnonymousVolumesMounts) == 0 {
		glog.V(4).Infof("rebuild container info is :%v", rebuildContainerInfo)
		return nil
	}
	mounts := make([]*runtimeapi.Mount, len(rebuildContainerInfo.AnonymousVolumesMounts))
	for index, volumeMount := range rebuildContainerInfo.AnonymousVolumesMounts {
		mounts[index] = &runtimeapi.Mount{
			ContainerPath: volumeMount.Destination,
			HostPath:      volumeMount.Source,
			Readonly:      !volumeMount.RW,
		}
	}
	return mounts
}

// GetDiskQuotaID get disk quota ID which get from sigma2.0 container, if not exist, return ""
func GetDiskQuotaID(pod *v1.Pod) string {
	rebuildContainerInfo, err := sigmautil.GetContainerRebuildInfoFromAnnotation(pod)
	if err != nil {
		glog.V(4).Info(err.Error())
		return ""
	}
	if rebuildContainerInfo == nil {
		glog.V(4).Infof("pod %q rebuild container info is nil", format.Pod(pod))
		return ""
	}
	return rebuildContainerInfo.DiskQuotaID
}

// getCidrIPMask converts mask lenth to the format such as 255.255.0.0
func getCidrIPMask(maskLen int) string {
	// get mask in the format of 2-system
	cidrMask := ^uint32(0) << uint(32-maskLen)
	// uint8() can get number with low-8 bits
	cidrMaskSeg1 := uint8(cidrMask >> 24)
	cidrMaskSeg2 := uint8(cidrMask >> 16)
	cidrMaskSeg3 := uint8(cidrMask >> 8)
	cidrMaskSeg4 := uint8(cidrMask & uint32(255))

	return fmt.Sprint(cidrMaskSeg1) + "." + fmt.Sprint(cidrMaskSeg2) + "." + fmt.Sprint(cidrMaskSeg3) + "." + fmt.Sprint(cidrMaskSeg4)
}

// getEnvsFromNetworkStatus generate envs from network status.
func getEnvsFromNetworkStatus(networkStatus *sigmak8sapi.NetworkStatus) []*runtimeapi.KeyValue {
	envs := []*runtimeapi.KeyValue{}
	// add network mask to env
	envs = append(envs, &runtimeapi.KeyValue{
		Key:   envDefaultMask,
		Value: getCidrIPMask(int(networkStatus.NetworkPrefixLength)),
	})
	// add network ip to env
	envs = append(envs, &runtimeapi.KeyValue{
		Key:   envRequestIP,
		Value: networkStatus.Ip,
	})
	// add network gateway to env
	envs = append(envs, &runtimeapi.KeyValue{
		Key:   envDefaultRoute,
		Value: networkStatus.Gateway,
	})
	// add network nic to env
	// DefaultNic logic is in simga2 apiserver: /cluster/sigma/create.go
	defaultNic := "docker0"
	if networkStatus.NetType == "ecs" {
		envs = append(envs, &runtimeapi.KeyValue{
			Key:   envVpcECS,
			Value: "true",
		})
	} else {
		if len(networkStatus.VlanID) > 0 {
			defaultNic = "bond0." + networkStatus.VlanID
		}
	}

	envs = append(envs, &runtimeapi.KeyValue{
		Key:   envDefaultNic,
		Value: defaultNic,
	})

	return envs
}

// generateNetworkEnvs gets latest pod from podManager and get netwrok envs from network status.
func generateNetworkEnvs(pod *v1.Pod, podManager kubepod.Manager) []*runtimeapi.KeyValue {
	newPod, exists := podManager.GetPodByUID(pod.UID)
	if !exists {
		return []*runtimeapi.KeyValue{}
	}
	networkStatus := sigmautil.GetNetworkStatusFromAnnotation(newPod)
	if networkStatus == nil {
		return []*runtimeapi.KeyValue{}
	}
	return getEnvsFromNetworkStatus(networkStatus)
}

var topologyKeyMap = []struct {
	env          string
	label        string
	defaultValue string
}{
	{alipaysigmav2.EnvSafetyOut, alipaysigmav2.EnvSafetyOut, "0"},
	{alipaysigmav2.EnvSigmaSite, sigmak8sapi.LabelSite, ""},
	{alipaysigmav2.EnvSigmaRegion, sigmak8sapi.LabelRegion, ""},
	{alipaysigmav2.EnvSigmaNCSN, sigmak8sapi.LabelNodeSN, ""},
	{alipaysigmav2.EnvSigmaNCHostname, sigmak8sapi.LabelHostname, ""},
	{alipaysigmav2.EnvSigmaNCIP, sigmak8sapi.LabelNodeIP, ""},
	{alipaysigmav2.EnvSigmaParentServiceTag, sigmak8sapi.LabelParentServiceTag, ""},
	{alipaysigmav2.EnvSigmaRoom, sigmak8sapi.LabelRoom, ""},
	{alipaysigmav2.EnvSigmaRack, sigmak8sapi.LabelRack, ""},
	{alipaysigmav2.EnvSigmaNetArchVersion, sigmak8sapi.LabelNetArchVersion, ""},
	{alipaysigmav2.EnvSigmaUplinkHostName, alipaysigmak8sapi.LabelUplinkHostname, ""},
	{alipaysigmav2.EnvSigmaUplinkIP, alipaysigmak8sapi.LabelUplinkIP, ""},
	{alipaysigmav2.EnvSigmaUplinkSN, alipaysigmak8sapi.LabelUplinkSN, ""},
	{alipaysigmav2.EnvSigmaASW, sigmak8sapi.LabelASW, ""},
	{alipaysigmav2.EnvSigmaLogicPod, sigmak8sapi.LabelLogicPOD, ""},
	{alipaysigmav2.EnvSigmaPod, sigmak8sapi.LabelPOD, ""},
	{alipaysigmav2.EnvSigmaDSWCluster, sigmak8sapi.LabelDSWCluster, ""},
	{alipaysigmav2.EnvSigmaNetLogicSite, sigmak8sapi.LabelNetLogicSite, ""},
	{alipaysigmav2.EnvSigmaSMName, sigmak8sapi.LabelMachineModel, ""},
	{alipaysigmav2.EnvSigmaModel, alipaysigmak8sapi.LabelModel, ""},
}

// generateTopologyEnvs generate topology envs from node's labels.
// The map relation is defined in topologyKeyMap.
func generateTopologyEnvs(node *v1.Node) []*runtimeapi.KeyValue {
	envs := make([]*runtimeapi.KeyValue, 0, len(topologyKeyMap))
	for _, km := range topologyKeyMap {
		v := km.defaultValue
		if x := node.Labels[km.label]; len(x) > 0 {
			v = x
		}
		envs = append(envs, &runtimeapi.KeyValue{Key: km.env, Value: v})
	}

	return envs
}
