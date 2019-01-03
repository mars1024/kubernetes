package filter

import (
	"fmt"
	"net/http"

	cluster "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/apis/cluster/v1alpha1"
	"k8s.io/apiserver/pkg/endpoints/request"
	clusterlisters "gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/pkg/client/listers_generated/cluster/v1alpha1"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
)

type subject struct {
	username   string
	usergroups []string

	namespace   string
	name        string
	verb        string
	apigroup    string
	apiversion  string
	resource    string
	subresource string
	path        string

	tenant multitenancy.TenantInfo
}

func getSubject(req *http.Request) (*subject, error) {
	user, userOk := request.UserFrom(req.Context())
	if !userOk {
		return nil, fmt.Errorf("missing user info")
	}
	requestInfo, requestInfoOk := request.RequestInfoFrom(req.Context())
	if !requestInfoOk {
		return nil, fmt.Errorf("missing request info")
	}
	tenant, err := util.TransformTenantInfoFromUser(user)
	if err != nil {
		return nil, err
	}
	return &subject{
		username:   user.GetName(),
		usergroups: user.GetGroups(),

		namespace:   requestInfo.Namespace,
		verb:        requestInfo.Verb,
		apigroup:    requestInfo.APIGroup,
		apiversion:  requestInfo.APIVersion,
		resource:    requestInfo.Resource,
		subresource: requestInfo.Subresource,
		tenant:      tenant,
		path:        requestInfo.Path,
	}, nil
}

func matchBucketBindings(req *http.Request, bktLister clusterlisters.BucketLister, bindings []*cluster.BucketBinding) (*cluster.Bucket, error) {
	sub, err := getSubject(req)
	if err != nil {
		return nil, err
	}
	for _, binding := range bindings {
		if match(sub, binding) {
			if bkt, err := bktLister.Get(binding.Spec.BucketRef.Name); err == nil {
				return bkt, nil
			}
		}
	}

	// return extra bkt
	return extraBucket, nil
}

func match(sub *subject, binding *cluster.BucketBinding) bool {
	for _, rule := range binding.Spec.Rules {
		ruleMatched := false
		for _, v := range rule.Values {
			switch rule.Field {
			case cluster.BucketBindingSubjectFieldUserName:
				if v == sub.username {
					ruleMatched = true
				}
			case cluster.BucketBindingSubjectFieldUserGroup:
				matched := false
				for _, g := range sub.usergroups {
					if v == g {
						matched = true
					}
				}
				if matched {
					ruleMatched = matched
				}
			case cluster.BucketBindingSubjectFieldRequestName:
				if v == sub.name {
					ruleMatched = true
				}
			case cluster.BucketBindingSubjectFieldRequestNamespace:
				if v == sub.namespace {
					ruleMatched = true
				}
			case cluster.BucketBindingSubjectFieldRequestVerb:
				if v == sub.verb {
					ruleMatched = true
				}
			case cluster.BucketBindingSubjectFieldRequestAPIGroup:
				if v == sub.apigroup {
					ruleMatched = true
				}
			case cluster.BucketBindingSubjectFieldRequestAPIVersion:
				if v == sub.apiversion {
					ruleMatched = true
				}
			case cluster.BucketBindingSubjectFieldRequestResource:
				if v == sub.resource {
					ruleMatched = true
				}
			case cluster.BucketBindingSubjectFieldRequestSubresource:
				if v == sub.subresource {
					ruleMatched = true
				}
			case cluster.BucketBindingSubjectFieldTenantName:
				if v == sub.tenant.GetTenantID() {
					ruleMatched = true
				}
			case cluster.BucketBindingSubjectFieldTenantWorkspace:
				if v == sub.tenant.GetWorkspaceID() {
					ruleMatched = true
				}
			case cluster.BucketBindingSubjectFieldTenantCluster:
				if v == sub.tenant.GetClusterID() {
					ruleMatched = true
				}
			case cluster.BucketBindingSubjectFieldRequestPath:
				if v == sub.path {
					ruleMatched = true
				}
			}
		}
		if !ruleMatched {
			return false
		}
	}
	return true
}
