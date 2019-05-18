/*
Copyright 2019 The Alipay Authors. All Rights Reserved.
*/

// Package apis for annotations contain all annotations used in alipay.com.
package apis

import (
	"time"
)

const (
	// ServiceAlipayPrefix is special sub-domain for Service
	ServiceAlipayPrefix = "service." + AlipayGroupName
	// ServiceProvisioner is the provisioner of Service
	ServiceProvisioner = ServiceAlipayPrefix + "/provisioner"
	// ServiceTopology is the name of IDC/zone
	ServiceTopology = ServiceAlipayPrefix + "/topology"
	// ServiceEnv is the environment name
	ServiceEnv = ServiceAlipayPrefix + "/env"
	// ServiceStation is the site name
	ServiceStation = ServiceAlipayPrefix + "/station"
	// ServiceAntVIPType is the type of AntVIP domain
	ServiceAntVIPType = ServiceAlipayPrefix + "/antvip-type"
	// ServiceAntVIPDomainName is the domain name configured by AntVIP
	ServiceAntVIPDomainName = ServiceAlipayPrefix + "/antvip-domain-name"

	// ServiceDNSRRZone is the domain suffix of DNS resord
	ServiceDNSRRZone = ServiceAlipayPrefix + "/dnsrr-zone"

	AnnotationZappinfo = MetaAlipayPrefix + "/pod-zappinfo"

	AnnotationXvipApplyUser       = ServiceAlipayPrefix + "/xvip-apply-user"
	AnnotationXvipAppGroup        = ServiceAlipayPrefix + "/xvip-app-group"
	AnnotationXvipAppId           = ServiceAlipayPrefix + "/xvip-app-id"
	AnnotationXvipBuType          = ServiceAlipayPrefix + "/xvip-bu-type"
	AnnotationXvipHealthcheckType = ServiceAlipayPrefix + "/xvip-healthcheck-type"
	AnnotationXvipHealthcheckPath = ServiceAlipayPrefix + "/xvip-healthcheck-path"
	AnnotationXvipReqAvgSize      = ServiceAlipayPrefix + "/xvip-req-avg-size"
	AnnotationXvipQpsLimit        = ServiceAlipayPrefix + "/xvip-qps-limit"
	AnnotationXvipOrderId         = ServiceAlipayPrefix + "/xvip-order-id"
	AnnotationXvipLbName          = ServiceAlipayPrefix + "/xvip-lb-name"
	AnnotationXvipAllocatedVip    = ServiceAlipayPrefix + "/xvip-allocated-vip"

	// AnnotationPodVolumeMountPath indicates volume source on host for multi disk.
	AnnotationPodVolumeMountPath = CustomAlipayPrefix + "/volume-host-path"

	// AnnotationBindCompleted annotation applies to PVCs. It indicates that the lifecycle
	// of the PVC has passed through the initial setup. This information changes how
	// we interpret some observations of the state of the objects. Value of this
	// annotation does not matter.
	AnnotationBindCompleted = "pv.kubernetes.io/bind-completed"

	// This annotation is added to a PVC that has been triggered by scheduler to
	// be dynamically provisioned. Its value is the name of the selected node.
	AnnotationSelectedNode = "volume.kubernetes.io/selected-node"

	// This annotation is added to a PVC that has been triggered by scheduler to
	// be dynamically provisioned. Its values is the mount point of the selected node.
	AnnotationSelectedDisk = "volume.kubernetes.io/selected-disk"
)

// ServiceProvisionerType is the set of provisioners can be used for ServiceProvisioner
type ServiceProvisionerType string

const (
	// ServiceProvisionerAntVIP is the AntVIP provisioner
	ServiceProvisionerAntVIP ServiceProvisionerType = "antvip"
	// ServiceProvisionerXVIP is the XVIP provisioner
	ServiceProvisionerXVIP ServiceProvisionerType = "xvip"
)

// ServiceEnvType is the set of environments can be used for ServiceEnv
type ServiceEnvType string

const (
	// ServiceEnvProd is production environment
	ServiceEnvProd ServiceEnvType = "PROD"
	// ServiceEnvPre is pre-production environment
	ServiceEnvPre ServiceEnvType = "PRE"
)

// ServiceStationType is the set of site names can be used for ServiceStation
type ServiceStationType string

const (
	// ServiceStationMainSite is the default site name of ServiceStation
	ServiceStationMainSite ServiceStationType = "MAIN_SITE"
)

// ServiceAntVIPTypeEnum is the set of the AntVIP types that can be used for ServiceAntVIPType
type ServiceAntVIPTypeEnum string

const (
	// ServiceAntVIPTypeStandard is the default type of AntVIP Service
	ServiceAntVIPTypeStandard ServiceAntVIPTypeEnum = "standard"
)

const (
	// SidecarAlipayPrefix is special sub-domain for sidecar.
	SidecarAlipayPrefix         = "sidecar." + AlipayGroupName
	MOSNSidecarAlipayPrefix     = "mosn." + SidecarAlipayPrefix
	MOSNSidecarInject           = MOSNSidecarAlipayPrefix + "/inject"
	MOSNSidecarImage            = MOSNSidecarAlipayPrefix + "/image"
	MOSNSidecarPostStartCommand = MOSNSidecarAlipayPrefix + "/post-start-command"
	MOSNSidecarCPU              = MOSNSidecarAlipayPrefix + "/cpu"
	MOSNSidecarMemory           = MOSNSidecarAlipayPrefix + "/memory"
	MOSNSidecarEphemeralStorage = MOSNSidecarAlipayPrefix + "/ephemeral-storage"
	MOSNSidecarIngressPort      = MOSNSidecarAlipayPrefix + "/ingress-port"
	MOSNSidecarEgressPort       = MOSNSidecarAlipayPrefix + "/egress-port"
	MOSNSidecarRegistryPort     = MOSNSidecarAlipayPrefix + "/registry-port"
	MOSNSidecarSmoothUpgrade    = MOSNSidecarAlipayPrefix + "/smooth-upgrade"
	MOSNSidecarUpgradeStatus    = MOSNSidecarAlipayPrefix + "/upgrade-status"
)

type SidecarStatus string

const (
	// 表示 sidecar容器正处于升级过程中，未来也可能有其他的 Sidecar
	SidecarStatusUpgrading SidecarStatus = "upgrading"
	// 表示 sidecar容器处于稳定运行过程中
	SidecarStatusRunning SidecarStatus = "running"
	// 标识平滑升级失败的状态
	SidecarStatusFailed SidecarStatus = "failed"
)

// SidecarInjectionPolicy determines the policy for injecting the
// sidecar container into the pod.
type SidecarInjectionPolicy string

const (
	// SidecarInjectionPolicyDisabled specifies that the sidecar injector
	// in admission control will not inject the sidecar container into the spec of pod.
	// Pod can enable injection using the "mosn.sidecar.k8s.alipay.com/inject"
	// annotation with value of "enabled".
	SidecarInjectionPolicyDisabled SidecarInjectionPolicy = "disabled"

	// SidecarInjectionPolicyEnabled specifies that the sidecar injector
	// in admission control will inject the sidecar container into the spec of pod.
	// Pod can disable injection using the "mosn.sidecar.k8s.alipay.com/inject"
	// annotation with value of "disabled".
	// This annotation must be submitted with "mosn.sidecar.k8s.alipay.com/image" together.
	SidecarInjectionPolicyEnabled SidecarInjectionPolicy = "enabled"
)

const (
	// CSIAlipayPrefix is special sub-domain for csi
	CSIAlipayPrefix = "csi." + AlipayGroupName
	CSIDataRecycle  = CSIAlipayPrefix + "/data-recycle"
)

const (
	// PodAlipayPrefix is special sub-domain for pod
	PodAlipayPrefix = "pod." + AlipayGroupName

	PodIPReuseTTL = PodAlipayPrefix + "/ip-reuse-ttl"

	// SpecifiedPodIP is the specified ip for pod.
	SpecifiedPodIP = PodAlipayPrefix + "/specified-container-ip"
)

const (
	// Skip the AlipayResourceAdmission validation
	SkipResourceAdmission = MetaAlipayPrefix + "/skip-resource-validation"
)

// ==========================
// pod traffic annotation definition

// PodTrafficType indicates pod traffic on or off
type PodTrafficType string

const (
	// PodTrafficOn means pod traffic is on, and should receive requests
	PodTrafficOn PodTrafficType = "on"
	// PodTrafficOff means pod traffic is off, and should NOT receive requests
	PodTrafficOff PodTrafficType = "off"
)

const (
	// AnnotationPodTraffic is the annotation to store pod traffic info
	// the value is traffic.k8s.alipay.com/status
	AnnotationPodTraffic = "traffic." + AlipayGroupName + "/status"
)

// PodTraffic 是上述 annotation 中保存的 pod 流量终态和当前状态
type PodTraffic struct {
	Spec   PodTrafficSpec   `json:"spec"`   // pod 流量终态配置
	Status PodTrafficStatus `json:"status"` // pod 流量当前状态
}

// PodTrafficSpec stores pod traffic expectation
type PodTrafficSpec struct {
	PodTraffic           PodTrafficType `json:"podTraffic"`           // on 或者 off，代表 spanner 和 antvip 流量的开关
	PodTrafficPercentage float64        `json:"podTrafficPercentage"` // 流量百分比， 到 mosn 那边会转换为0-100 之间的数字，分时调度只关心这个字段
}

// PodTrafficStatus records current pod traffic setting
type PodTrafficStatus struct {
	LastUpdateTime              *time.Time
	CurrentPodTraffic           PodTrafficType
	CurrentPodTrafficPercentage float64
}
