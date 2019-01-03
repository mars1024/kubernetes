// +build multitenancy

/*
Copyright 2015 The Kubernetes Authors.

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

package discovery

import (
	restclient "k8s.io/client-go/rest"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
)

func (c *DiscoveryClient) SetRESTClient(restClient restclient.Interface) {
	c.restClient = restClient
}

func (c *DiscoveryClient) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *c
	copied.restClient = c.restClient.(meta.TenantWise).ShallowCopyWithTenant(tenant).(restclient.Interface)
	return &copied
}
