/*
Copyright 2019 The Alipay.com Inc Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cache

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
)

const (
	TenantNamespaceIndex = "tenant_namespace"

	// Deprecated
	MultiTenancyAnnotationKeyTenantID = "alpha.cloud.alipay.com/tenant-id"
	// Deprecated
	MultiTenancyAnnotationKeyWorkspaceID = "alpha.cloud.alipay.com/workspace-id"
	// Deprecated
	MultiTenancyAnnotationKeyClusterID = "alpha.cloud.alipay.com/cluster-id"
)

// Tenant is an interface for accessing antcloud-aks multitenancy meta information
type TenantInfo interface {
	// GetTenantID returns tenant id
	GetTenantID() string
	// GetWorkspaceID returns workspace id
	GetWorkspaceID() string
	// GetClusterID returns cluster id
	GetClusterID() string
}

type ErrMissingOrMalformedTenantInfo []string

func (e ErrMissingOrMalformedTenantInfo) Error() string {
	return fmt.Sprintf("missing or malformed tenant info: %v", []string(e))
}

type defaultTenant struct {
	tenantID    string
	workspaceID string
	clusterID   string
}

var _ TenantInfo = &defaultTenant{}

func (t *defaultTenant) GetTenantID() string {
	return t.tenantID
}

func (t *defaultTenant) GetWorkspaceID() string {
	return t.workspaceID
}

func (t *defaultTenant) GetClusterID() string {
	return t.clusterID
}

func NewTenantInfo(tenantID, workspaceID, clusterID string) TenantInfo {
	return &defaultTenant{tenantID, workspaceID, clusterID}
}

func MetaTenantNamespaceIndexFunc(obj interface{}) ([]string, error) {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return []string{""}, fmt.Errorf("object has no meta: %v", err)
	}
	tenantWrappedKeyFunc := MultiTenancyKeyFuncWrapper(func(obj interface{}) (string, error) {
		return metadata.GetNamespace(), nil
	})
	namespaceWithTenant, err := tenantWrappedKeyFunc(obj)
	if err != nil {
		return []string{""}, err
	}
	return []string{namespaceWithTenant}, nil
}

func MultiTenancyKeyFuncWrapper(keyFunc cache.KeyFunc) cache.KeyFunc {
	return func(obj interface{}) (string, error) {
		if key, ok := obj.(cache.ExplicitKey); ok {
			return string(key), nil
		}
		key, err := keyFunc(obj)
		if err != nil {
			return key, err
		}
		if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
			obj = d.Obj
		}
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return "", fmt.Errorf("fail to extract tenant info from %#v: %s", obj, err.Error())
		}
		tenantInfo, err := TransformTenantInfoFromAnnotations(accessor.GetAnnotations())
		if err == nil {
			return TransformTenantInfoToJointString(tenantInfo, "/") + "/" + key, nil
		}
		return key, nil
	}
}

// TransformTenantInfoFromAnnotations takes all the annotations from an object and returns
// tenant info on success.
func TransformTenantInfoFromAnnotations(annotations map[string]string) (TenantInfo, error) {
	var missingFields []string
	tenantID, ok := annotations[MultiTenancyAnnotationKeyTenantID]
	if !ok {
		missingFields = append(missingFields, "tenantID")
	}
	workspaceID, ok := annotations[MultiTenancyAnnotationKeyWorkspaceID]
	if !ok {
		missingFields = append(missingFields, "workspaceID")
	}
	clusterID, ok := annotations[MultiTenancyAnnotationKeyClusterID]
	if !ok {
		missingFields = append(missingFields, "clusterID")
	}
	if len(missingFields) > 0 {
		return nil, ErrMissingOrMalformedTenantInfo(missingFields)
	}
	return NewTenantInfo(tenantID, workspaceID, clusterID), nil
}

// TransformTenantInfoToAnnotations joins tenant info w/ a delimiter as a string.
func TransformTenantInfoToJointString(tenant TenantInfo, delimiter string) string {
	arrayToPrint := []string{
		tenant.GetTenantID(),
		tenant.GetWorkspaceID(),
		tenant.GetClusterID(),
	}
	return strings.Join(arrayToPrint, delimiter)
}
