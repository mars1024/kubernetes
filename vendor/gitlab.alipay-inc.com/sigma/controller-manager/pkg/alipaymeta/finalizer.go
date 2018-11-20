package alipaymeta

const (
	// AntVipFinalizer for AntVip controller to release domains
	AntVipFinalizer = "finalizer.k8s.alipay.com/antvip"
	// DNSResourceRecord finalizer for DNSRR controller to release DNS resource records
	DNSRRFinalizer = "finalizer.k8s.alipay.com/dnsrr"
	// PodPostSet finalizer for podpostset controller to release pod
	PodPostSetFinalizer = "finalizer.k8s.alipay.com/podpostset"
	// ZappinfoFinalizer for Zappinfo controller to release zappinfo records
	ZappinfoFinalizer = "finalizer.k8s.alipay.com/zappinfo"
	// XvipFinalizer for Xvip controller to release domains
	XvipFinalizer = "finalizer.k8s.alipay.com/xvip"

	// CMDBFinalizer for CMDB controller to release cmdb records
	CMDBFinalizer = "finalizer.k8s.alipay.com/cmdb"
)
