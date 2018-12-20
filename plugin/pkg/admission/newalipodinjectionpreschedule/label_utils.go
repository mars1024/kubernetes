package newalipodinjectionpreschedule

import (
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	api "k8s.io/kubernetes/pkg/apis/core"
)

var (
	defaultLabelMapping = map[string]string{
		"ASW":                sigmak8sapi.LabelASW,
		"DswCluster":         sigmak8sapi.LabelDSWCluster,
		"Pod":                sigmak8sapi.LabelPOD,
		"LogicPod":           sigmak8sapi.LabelLogicPOD,
		"Rack":               sigmak8sapi.LabelRack,
		"Region":             sigmak8sapi.LabelRegion,
		"Room":               sigmak8sapi.LabelRoom,
		"NetLogicSite":       sigmak8sapi.LabelNetLogicSite,
		"Site":               sigmak8sapi.LabelSite,
		"NcSn":               sigmak8sapi.LabelNodeSN,
		"NcHostname":         "kubernetes.io/hostname",
		"NcIp":               sigmak8sapi.LabelNodeIP,
		"NetArchVersion":     sigmak8sapi.LabelNetArchVersion,
		"NetCardType":        "sigma.ali/net-card-type",
		"ParentServiceTag":   sigmak8sapi.LabelParentServiceTag,
		"PhyIpRange":         sigmak8sapi.LabelPhyIPRange,
		"EcsInstanceId":      sigmak8sapi.LabelECSInstanceID,
		"EcsDeploymentSetId": sigmak8sapi.LabelECSDeploymentSetID,
		"EcsRegionId":        sigmak8sapi.LabelECSRegionID,
		"EcsVpcId":           "sigma.ali/ecs-vpc-id",

		"IsEcs":           sigmak8sapi.LabelIsECS,
		"IsMixrun":        sigmak8sapi.LabelIsMixRun,
		"EnableOverQuota": sigmak8sapi.LabelEnableOverQuota,
		"cpushare":        "sigma.ali/is-cpu-share",
		"OverlayNetwork":  sigmak8sapi.LabelIsOverlayNetwork,
		"OverlaySriov":    sigmak8sapi.LabelIsOverlaySriov,
		"IsPubNetServer":  sigmak8sapi.LabelIsPubNetServer,
		"IsEni":           sigmak8sapi.LabelIsENI,
		"ResourcePool":    sigmak8sapi.LabelResourcePool,
		"Kernel":          "sigma.ali/kernel",
		"OS":              sigmak8sapi.LabelOS,
		"CpuSpec":         "sigma.ali/cpu-spec",
		"DiskType":        sigmak8sapi.LabelEphemeralDiskType,
		"SmName":          sigmak8sapi.LabelMachineModel,
		"CpuOverQuota":    sigmak8sapi.LabelCPUOverQuota,
		"DiskOverQuota":   sigmak8sapi.LabelDiskOverQuota,
		"MemoryOverQuota": sigmak8sapi.LabelMemOverQuota,
		"ali.AppUnit":     labelAppUnit,
		"ali.AppStage":    labelAppStage,
	}
)

func sigma2ToSigma3Label(cm *api.ConfigMap, sigma2label string) string {
	if cm != nil {
		if v, ok := cm.Data[sigma2label]; ok {
			return v
		}
	}

	if v, ok := defaultLabelMapping[sigma2label]; ok {
		return v
	}
	return sigma2label
}
