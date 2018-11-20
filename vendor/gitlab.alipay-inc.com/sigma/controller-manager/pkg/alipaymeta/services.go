package alipaymeta

import (
	"k8s.io/api/core/v1"

	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"gitlab.alipay-inc.com/sigma/apis/pkg/apis"
)

const (
	AnnotationServiceProvisioner       = apis.ServiceProvisioner
	AnnotationServiceProvisionerAntVIP = apis.ServiceProvisionerAntVIP
	AnnotationServiceProvisionerXVIP   = apis.ServiceProvisionerXVIP
	AnnotationServiceAntVIPDomain      = apis.ServiceAntVIPDomainName
	AnnotationServiceZone              = apis.ServiceDNSRRZone
)

func GetServiceZone(sv *v1.Service) string {
	if nil == sv.Annotations {
		return ""
	}

	return sv.Annotations[AnnotationServiceZone]
}

func GetServiceProvisioner(sv *v1.Service) string {
	if nil == sv.Annotations {
		return ""
	}

	return sv.Annotations[AnnotationServiceProvisioner]
}

func GetServiceAppName(sv *v1.Service) string {
	if sv.Annotations == nil {
		return ""
	}
	return sv.Annotations[api.LabelAppName]
}

func GetServiceTopology(sv *v1.Service) string {
	if sv.Annotations == nil {
		return ""
	}
	return sv.Annotations[apis.ServiceTopology]
}

func GetServiceEnv(sv *v1.Service) string {
	if sv.Annotations == nil {
		return ""
	}
	return sv.Annotations[apis.ServiceEnv]
}

func GetServiceStation(sv *v1.Service) string {
	if sv.Annotations == nil {
		return ""
	}
	return sv.Annotations[apis.ServiceStation]
}

func GetServiceHealthCheckPort(sv *v1.Service) int32 {
	if len(sv.Spec.Ports) == 0 {
		return 0
	}
	return sv.Spec.Ports[0].Port
}

func GetAntVipServiceDomain(svc *v1.Service) string {
	if nil == svc.Annotations {
		return ""
	}

	return svc.Annotations[apis.ServiceAntVIPDomainName]
}
