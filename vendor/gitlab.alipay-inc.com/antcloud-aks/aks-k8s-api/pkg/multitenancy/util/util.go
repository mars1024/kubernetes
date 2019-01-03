package util

import (
	"fmt"
	"strings"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
)

type UserInfo interface {
	GetName() string
	GetUID() string
	GetGroups() []string
	GetExtra() map[string][]string
}

type ErrMissingOrMalformedTenantInfo []string

func (e ErrMissingOrMalformedTenantInfo) Error() string {
	return fmt.Sprintf("missing or malformed tenant info: %v", []string(e))
}

// TransformTenantInfoFromAnnotations takes all the annotations from an object and returns
// tenant info on success.
func TransformTenantInfoFromAnnotations(annotations map[string]string) (multitenancy.TenantInfo, error) {
	missingFields := []string{}
	tenantID, ok := annotations[multitenancy.MultiTenancyAnnotationKeyTenantID]
	if !ok {
		missingFields = append(missingFields, "tenantID")
	}
	workspaceID, ok := annotations[multitenancy.MultiTenancyAnnotationKeyWorkspaceID]
	if !ok {
		missingFields = append(missingFields, "workspaceID")
	}
	clusterID, ok := annotations[multitenancy.MultiTenancyAnnotationKeyClusterID]
	if !ok {
		missingFields = append(missingFields, "clusterID")
	}
	if len(missingFields) > 0 {
		return nil, ErrMissingOrMalformedTenantInfo(missingFields)
	}
	return multitenancy.NewTenantInfo(tenantID, workspaceID, clusterID), nil
}

// TransformTenantInfoToAnnotations takes tenant info then returns a map w/ tenancy.
func TransformTenantInfoToAnnotations(tenant multitenancy.TenantInfo) map[string]string {
	annotations := make(map[string]string)
	if len(tenant.GetTenantID()) > 0 {
		annotations[multitenancy.MultiTenancyAnnotationKeyTenantID] = tenant.GetTenantID()
	}
	if len(tenant.GetWorkspaceID()) > 0 {
		annotations[multitenancy.MultiTenancyAnnotationKeyWorkspaceID] = tenant.GetWorkspaceID()
	}
	if len(tenant.GetClusterID()) > 0 {
		annotations[multitenancy.MultiTenancyAnnotationKeyClusterID] = tenant.GetClusterID()
	}
	return annotations
}

// TransformTenantInfoToAnnotations takes tenant info then returns a map w/ tenancy.
func TransformTenantInfoToAnnotationsIncremental(tenant multitenancy.TenantInfo, annotations *map[string]string) {
	if annotations == nil {
		return
	}
	if *annotations == nil {
		*annotations = make(map[string]string)
	}
	if len(tenant.GetTenantID()) > 0 {
		(*annotations)[multitenancy.MultiTenancyAnnotationKeyTenantID] = tenant.GetTenantID()
	}
	if len(tenant.GetWorkspaceID()) > 0 {
		(*annotations)[multitenancy.MultiTenancyAnnotationKeyWorkspaceID] = tenant.GetWorkspaceID()
	}
	if len(tenant.GetClusterID()) > 0 {
		(*annotations)[multitenancy.MultiTenancyAnnotationKeyClusterID] = tenant.GetClusterID()
	}
	return
}

// TransformTenantInfoToAnnotations joins tenant info w/ a delimiter as a string.
func TransformTenantInfoToJointString(tenant multitenancy.TenantInfo, delimiter string) string {
	arrayToPrint := []string{
		tenant.GetTenantID(),
		tenant.GetWorkspaceID(),
		tenant.GetClusterID(),
	}
	return strings.Join(arrayToPrint, delimiter)
}

// TransformTenantInfoFromJointString parses tenant info from a formatted string.
func TransformTenantInfoFromJointString(str string, delimiter string) (multitenancy.TenantInfo, error) {
	splitted := strings.SplitN(str, delimiter, 3)
	if len(splitted) != 3 {
		return nil, fmt.Errorf("fail to transform string %s into tenant info with separator %s", str, delimiter)
	}
	return multitenancy.NewTenantInfo(splitted[0], splitted[1], splitted[2]), nil
}

// TransformTenantInfoToUser put tenant info into user's extra info field
func TransformTenantInfoToUser(tenant multitenancy.TenantInfo, user UserInfo) error {
	if user.GetExtra() == nil {
		return fmt.Errorf("fail to inject tenant info into user %s: nil extra info map", user.GetName())
	}
	if len(tenant.GetTenantID()) > 0 {
		user.GetExtra()[multitenancy.UserExtraInfoTenantID] = []string{tenant.GetTenantID()}
	}
	if len(tenant.GetWorkspaceID()) > 0 {
		user.GetExtra()[multitenancy.UserExtraInfoWorkspaceID] = []string{tenant.GetWorkspaceID()}
	}
	if len(tenant.GetClusterID()) > 0 {
		user.GetExtra()[multitenancy.UserExtraInfoClusterID] = []string{tenant.GetClusterID()}
	}
	return nil
}

// TransformTenantInfoFromUser parses tenant info from user's extra info field
func TransformTenantInfoFromUser(user UserInfo) (multitenancy.TenantInfo, error) {
	extra := user.GetExtra()
	if extra == nil {
		return nil, fmt.Errorf("fail to extract tenant info from user meta: nil extra info")
	}
	missingFields := []string{}
	tenantID, ok := extra[multitenancy.UserExtraInfoTenantID]
	if !ok || len(tenantID) != 1 {
		missingFields = append(missingFields, "tenantID")
	}
	workspaceID, ok := extra[multitenancy.UserExtraInfoWorkspaceID]
	if !ok || len(tenantID) != 1 {
		missingFields = append(missingFields, "workspaceID")
	}
	clusterID, ok := extra[multitenancy.UserExtraInfoClusterID]
	if !ok || len(tenantID) != 1 {
		missingFields = append(missingFields, "clusterID")
	}
	if len(missingFields) > 0 {
		return nil, ErrMissingOrMalformedTenantInfo(missingFields)
	}
	return multitenancy.NewTenantInfo(tenantID[0], workspaceID[0], clusterID[0]), nil
}

// IsMultiTenancyWiseAdmin judges if a user is privileged as a global admin
func IsMultiTenancyWiseAdmin(username string) bool {
	switch username {
	case "system:admin", "kubeapiserver":
		return true
	case "system:kube-scheduler", "system:kube-controller-manager", "system:apiserver":
		return true
	default:
		return false
	}
}

// IsMultiTenancyWiseTenant judges if a user is privileged as a global admin
func IsMultiTenancyWiseTenant(tenant multitenancy.TenantInfo) bool {
	return strings.HasPrefix(tenant.GetTenantID(), multitenancy.GlobalAdminTenantNamePrefix)
}