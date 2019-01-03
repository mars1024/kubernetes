package v1alpha1

const (
	// Lables on InPlaceSets and Pods
	UnitNameCellLabel = "cafe.sofastack.io/cell"
	UnitNameZoneLabel = "cafe.sofastack.io/zone"

	// Labels on CafeDeployment, ControllerRevision, InPlaceSet and Pods
	AppNameLabel           = "app.kubernetes.io/name"
	AppServiceNameLabel    = "app.kubernetes.io/instance"
	AppServiceVersionLabel = "app.kubernetes.io/version"

	// Labels on nodes in order to work with node affinity
	PodNodeAffinityNodeCellKey = "cafe.sofastack.io/cell"
	PodNodeAffinityNodeZoneKey = "failure-domain.beta.kubernetes.io/zone"
)
