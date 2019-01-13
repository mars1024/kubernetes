// +build multitenancy

/*
Copyright 2016 The Kubernetes Authors.

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

package clusterrole

import (
	"k8s.io/apiserver/pkg/registry/rest"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
)

func (s *storage) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *s
	copied.Getter = s.Getter.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(rest.Getter)
	return &copied
}

func (a AuthorizerAdapter) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return AuthorizerAdapter{
		a.Registry.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(Registry),
	}
}
