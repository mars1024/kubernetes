package antitamper

// Supports leading / trailing "*"
var immutableLabels = []string{
	"cafe.sofastack.io/sub-cluster",
	"system.sas.cafe.sofastack.io/*",
	"system.ase.cafe.sofastack.io/*",
}

// Supports leading / trailing "*"
var immutableAnnotations = []string{
	"system.sas.cafe.sofastack.io/*",
	"system.ase.cafe.sofastack.io/*",
	AnnotationCafeSystemReservedNamespace,
}

// See `ResourceIdentifier` below
var protectedResources = []ResourceIdentifier{
	makeResourceIdentifier("ase-managed-cluster", "default", "", "v1", "ConfigMap"),
	makeResourceIdentifier("*", "*", "apps.cafe.cloud.alipay.com", "v1alpha1", "CafeDeployment"),
	makeResourceIdentifier("*", "*", "apps.cafe.cloud.alipay.com", "v1alpha1", "CafeInPlaceDeployment"),
	makeResourceIdentifier("*", "*", "apps.cafe.cloud.alipay.com", "v1alpha1", "InPlaceSet"),
	// makeResourceIdentifier("my-resource-*", any, any, any, any),
}

/*
 * 如果将这个Annotation打在namespace上，AdmissionController会禁止非上帝证书的用户修改namespace里任何资源
 */
var AnnotationCafeSystemReservedNamespace = "cafe.sofastack.io/system-reserved-namespace"

/*
 * AdmissionController会禁止非上帝证书的用户创建以下namespace
 */
var cafeSystemReservedNamespaceNames = []string{"sofastack-system"}

type ResourceIdentifier struct {
	name      *string // Supports value "*" to mean any, supports leading / trailing "*"
	namespace *string // Supports value "*" to mean any, supports leading / trailing "*"
	group     *string // e.g. "networking.istio.io", Supports value "*" to mean any, supports leading / trailing "*"
	version   *string // e.g. "v1", Supports value "*" to mean any, supports leading / trailing "*"
	kind      *string // e.g. "ConfigMap", Supports value "*" to mean any, supports leading / trailing "*"
}
