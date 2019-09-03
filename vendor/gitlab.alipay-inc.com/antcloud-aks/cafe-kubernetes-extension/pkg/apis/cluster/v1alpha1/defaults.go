package v1alpha1

import "net"

const (
	CafeCoreDNSDefaultImage     = "reg-cnhz.cloud.alipay.com/library/coredns:1.2.5"
	CafeCoreDNSDefaultClusterIP = "10.0.0.2"
)

// DefaultServiceNodePortRange is the default port range for NodePort services.
var DefaultServiceNodePortRange = PortRange{Base: 30000, Size: 2768}

// DefaultServiceIPCIDR is a CIDR notation of IP range from which to allocate service cluster IPs
var DefaultServiceIPCIDR = net.IPNet{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(24, 32)}

func SetDefaults_MinionCluster(obj *MinionCluster) {
	SetDefaults_MinionClusterSpec(&obj.Spec)
}

func SetDefaults_MinionClusterSpec(obj *MinionClusterSpec) {
	if obj.Networking != nil {
		if len(obj.Networking.NetworkType) == 0 {
			obj.Networking.NetworkType = ClassicNetwork
		}
		if len(obj.Networking.ServiceClusterIPRange) == 0 {
			obj.Networking.ServiceClusterIPRange = DefaultServiceIPCIDR.String()
		}
		if obj.Networking.ServiceNodePortRange.Size == 0 {
			obj.Networking.ServiceNodePortRange = DefaultServiceNodePortRange
		}
		if obj.Networking.DNS == nil {
			obj.Networking.DNS = &DNS{
				Local: &LocalDNS{
					Image:     CafeCoreDNSDefaultImage,
					ClusterIP: CafeCoreDNSDefaultClusterIP,
				},
			}
		}
	}
}
