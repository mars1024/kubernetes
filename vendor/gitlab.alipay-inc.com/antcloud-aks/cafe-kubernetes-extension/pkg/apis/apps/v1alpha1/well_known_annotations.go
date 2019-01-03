package v1alpha1

const (
	// Annotations for operating CafeDeployments
	AlphaCafeDeploymentAnnotationReleaseConfirmed   = "cafe.sofastack.io/upgrade-confirmed"
	AlphaCafeDeploymentAnnotationRollbackToRevision = "cafe.sofastack.io/rollback-to-revision"

	// Indicate the updated pod spec hash which will be used in pod annotation to tell the current revision
	InPlaceSetAnnotationPodSpecHash = "pod.beta1.sigma.ali/pod-spec-hash"

	PodAnnotationUpdateStatus = "pod.beta1.sigma.ali/update-status"
)
