/*
Copyright 2018 The Alipay Authors. All Rights Reserved.
*/

// This files contains all annotations used in alipay.com.
package apis

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
