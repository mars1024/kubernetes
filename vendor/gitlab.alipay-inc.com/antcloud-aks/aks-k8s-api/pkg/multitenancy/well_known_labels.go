package multitenancy

const (
	// The value of the key is in format of {tenant}:{workspace}:{cluster}
	// Deprecated
	MultiTenancyRequestHeaderKey = "X-Request-Multitenance"

	LabelCellName = "cafe.sofastack.io/cell"

	// The target cell id of pod
	LabelPodTargetCell = "pod.cloud.alipay.com/cell-id"

	LabelCellFailureDomain = "failure-domain.beta.kubernetes.io/cell"
)