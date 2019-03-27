/*
Copyright 2019 The Alipay Authors. All Rights Reserved.
*/

package apis

const (
	// FQDN is full qualified domain name of Pods
	FQDN = MetaAlipayPrefix + "/fqdn"

	// pod ip
	LabelPodIp = MetaAlipayPrefix + "/pod-ip"

	// pod container id
	LabelPodContainerId = MetaAlipayPrefix + "/container-id"

	// pod container name
	LabelPodContainerName = MetaAlipayPrefix + "/container-name"

	// pod hostname
	LabelPodContainerHostName = MetaAlipayPrefix + "/hostname"

	// application AppDeployUnit
	LabelAppDeployUnit = MetaAlipayPrefix + "/app-deploy-unit"

	// Label Pod Preset
	LabelPodPresetName = "pod." + AlipayGroupName + "/preset"

	// Label default PodPreset
	LabelDefaultPodPreset = "podpreset." + AlipayGroupName + "/default"

	// Label default MOSN sidecar config
	LabelDefaultMOSNSidecar = MOSNSidecarAlipayPrefix + "/default"

	// Label Zone
	LabelZone = MetaAlipayPrefix + "/zone"

	// LabelPodAppEnv is the application environment for pod
	LabelPodAppEnv = MetaAlipayPrefix + "/app-env"
)

// PodPromotionType defines a new type for podPromotion type value
type PodPromotionType string

// promotion related label and values
const (
	// LabelPodPromotionType is the promotion type current pod is in,
	// supported values include: taobao, antmember, share, and empty string.
	LabelPodPromotionType = "promotion.pod." + AlipayGroupName + "/type"

	// PodPromotionTypeAntMember means the pod is ready for antmember related traffic only
	PodPromotionTypeAntMember PodPromotionType = "antmember"
	// PodPromotionTypeTaobao means the pod is ready for taobao related traffic only
	PodPromotionTypeTaobao PodPromotionType = "taobao"
	// PodPromotionTypeShare means the pod is ready for both antmember and taobao traffic
	PodPromotionTypeShare PodPromotionType = "share"
	// PodPromotionTypeNone means the pod is not in any promotion
	PodPromotionTypeNone PodPromotionType = ""
)

// Validate returns if a given PodPromotionType value is valid.
func (p PodPromotionType) Validate() bool {
	if p == PodPromotionTypeShare ||
		p == PodPromotionTypeTaobao ||
		p == PodPromotionTypeAntMember ||
		p == PodPromotionTypeNone {
		return true
	}
	return false
}

// String convert PodPromotionType to string.
func (p PodPromotionType) String() string {
	return string(p)
}

// CanShareResourceWith check if one PodPromotionType can share resource with another.
func (p PodPromotionType) CanShareResourceWith(o PodPromotionType) bool {
	if (p == PodPromotionTypeTaobao && o == PodPromotionTypeAntMember) ||
		(p == PodPromotionTypeAntMember && o == PodPromotionTypeTaobao) {
		return true
	}
	return false
}

// GetAnotherShareType get another role that can share resource with.
func (p PodPromotionType) GetAnotherShareType() (PodPromotionType, bool) {
	if p == PodPromotionTypeTaobao {
		return PodPromotionTypeAntMember, true
	} else if p == PodPromotionTypeAntMember {
		return PodPromotionTypeTaobao, true
	}
	return PodPromotionTypeNone, false
}

// GetPromotionType determine the pod which PodPromotionType belongs to.
func GetPromotionType(labels map[string]string) (PodPromotionType, bool) {
	if value, ok := labels[LabelPodPromotionType]; ok {
		switch value {
		case PodPromotionTypeAntMember.String():
			return PodPromotionTypeAntMember, true
		case PodPromotionTypeTaobao.String():
			return PodPromotionTypeTaobao, true
		case PodPromotionTypeShare.String():
			return PodPromotionTypeShare, true
		case PodPromotionTypeNone.String():
			fallthrough
		default:
			return PodPromotionTypeNone, false
		}
	}
	return PodPromotionTypeNone, false
}

// GenerateCustomLabelKey generates a new label key use custom prefix and sub key.
func GenerateCustomLabelKey(key string) string {
	return CustomAlipayPrefix + "/" + key
}
