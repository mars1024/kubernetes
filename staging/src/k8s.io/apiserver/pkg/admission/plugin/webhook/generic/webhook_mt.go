/*
Copyright 2018 The Kubernetes Authors.

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

package generic

import (
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	"k8s.io/apiserver/pkg/util/webhook"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/namespace"
	"k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/kubernetes"
)

// ShouldCallHook makes a decision on whether to call the webhook or not by the attribute.
func (a *Webhook) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) (interface{}) {
	copied := *a
	copied.clientManager = a.clientManager.ShallowCopyWithTenant(tenant).(*webhook.ClientManager)
	copied.hookSource = a.hookSource.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(Source)
	copied.namespaceMatcher = &namespace.Matcher{
		NamespaceLister: a.namespaceMatcher.NamespaceLister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(v1.NamespaceLister),
		Client:          a.namespaceMatcher.Client.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(kubernetes.Interface),
	}
	return &copied
}
