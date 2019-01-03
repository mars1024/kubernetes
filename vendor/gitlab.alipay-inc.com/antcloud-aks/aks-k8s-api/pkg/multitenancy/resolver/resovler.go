package resolver

import (
	"encoding/json"
	"fmt"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"net"
	"net/url"
	"k8s.io/api/core/v1"
)

const (
	AntCloadLoadBalancerStatusAnnotationKey = "lb.service.beta.cloud.alipay.com"
)

type AntCloudLoadBalancerStatus struct {
	LoadBalancerID string `json:"loadBalancerId"`
	VirtualIP      string `json:"vip"`
	Status         string `json:"status"`
}

func NewLoadBalancerServiceResolver(services listersv1.ServiceLister) *aggregatorLoadBalancerRouting {
	return &aggregatorLoadBalancerRouting{
		services: services,
	}
}

type aggregatorLoadBalancerRouting struct {
	services listersv1.ServiceLister
}

func (r *aggregatorLoadBalancerRouting) ResolveEndpoint(namespace, name string) (*url.URL, error) {
	svc, err := r.services.Services(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	if svc.Spec.Type != v1.ServiceTypeLoadBalancer {
		return nil, fmt.Errorf("[AKS] failed to connect service: %v/%v should be loadbalancer type", namespace, name)
	}
	lbStatusRaw, exists := svc.Annotations[AntCloadLoadBalancerStatusAnnotationKey]
	if !exists {
		return nil, fmt.Errorf("no load balancer status found on service %v/%v", svc.Namespace, svc.Name)
	}

	lbStatus := AntCloudLoadBalancerStatus{}
	if err := json.Unmarshal([]byte(lbStatusRaw), &lbStatus); err != nil {
		return nil, err
	}
	return &url.URL{
		Scheme: "https",
		Host:   net.JoinHostPort(lbStatus.VirtualIP, "443"),
	}, nil
}

func (r *aggregatorLoadBalancerRouting) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *r
	copied.services = r.services.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(listersv1.ServiceLister)
	return &copied
}
