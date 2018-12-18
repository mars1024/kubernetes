/*
Copyright 2018 The Alipay Authors. All Rights Reserved.
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
