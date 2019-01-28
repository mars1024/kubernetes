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

// GenerateCustomLabelKey() generate a new label key use custom prefix and sub key.
func GenerateCustomLabelKey(key string) string {
	return CustomAlipayPrefix + "/" + key
}
