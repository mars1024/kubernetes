package cluster

const (
	SystemTopPriorityBand    PriorityBand = "Top"
	SystemHighPriorityBand   PriorityBand = "High"
	SystemMediumPriorityBand PriorityBand = "Medium"
	SystemNormalPriorityBand PriorityBand = "Normal"
	SystemLowPriorityBand    PriorityBand = "Low"

	// This is an implicit priority that cannot be set via API
	SystemLowestPriorityBand PriorityBand = "Lowest"
)

var (
	AllPriorities = []PriorityBand{
		SystemTopPriorityBand,
		SystemHighPriorityBand,
		SystemMediumPriorityBand,
		SystemNormalPriorityBand,
		SystemLowPriorityBand,
		SystemLowestPriorityBand,
	}
)

const (
	BucketBindingSubjectFieldUserName           = "user.name"
	BucketBindingSubjectFieldUserGroup          = "user.group"
	BucketBindingSubjectFieldRequestVerb        = "verb"
	BucketBindingSubjectFieldRequestNamespace   = "namespace"
	BucketBindingSubjectFieldRequestName        = "name"
	BucketBindingSubjectFieldRequestResource    = "resource"
	BucketBindingSubjectFieldRequestSubresource = "subresource"
	BucketBindingSubjectFieldRequestAPIVersion  = "version"
	BucketBindingSubjectFieldRequestAPIGroup    = "group"
	BucketBindingSubjectFieldTenantName         = "tenant.name"
	BucketBindingSubjectFieldTenantWorkspace    = "tenant.workspace"
	BucketBindingSubjectFieldTenantCluster      = "tenant.cluster"
	BucketBindingSubjectFieldRequestPath        = "path"
)

var (
	AllBucketBindingRuleSubjects = []string{
		BucketBindingSubjectFieldUserName,
		BucketBindingSubjectFieldUserGroup,
		BucketBindingSubjectFieldRequestVerb,
		BucketBindingSubjectFieldRequestNamespace,
		BucketBindingSubjectFieldRequestName,
		BucketBindingSubjectFieldRequestResource,
		BucketBindingSubjectFieldRequestSubresource,
		BucketBindingSubjectFieldRequestAPIVersion,
		BucketBindingSubjectFieldRequestAPIGroup,
		BucketBindingSubjectFieldTenantName,
		BucketBindingSubjectFieldTenantWorkspace,
		BucketBindingSubjectFieldTenantCluster,
		BucketBindingSubjectFieldRequestPath,
	}
)
