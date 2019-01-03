// +build multitenancy

/*
Copyright 2017 The Kubernetes Authors.

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

package configuration

import (
	"fmt"
	"sort"
	"sync"

	"k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"

	"github.com/golang/glog"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/generic"
	"k8s.io/client-go/informers"
	admissionregistrationlisters "k8s.io/client-go/listers/admissionregistration/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// ValidatingWebhookConfigurationManager collects the validating webhook objects so that they can be called.
type ValidatingWebhookConfigurationManager struct {
	rwLock                *sync.RWMutex
	tenantToConfiguration map[multitenancyutil.TenantHash]*v1beta1.ValidatingWebhookConfiguration
	lister                admissionregistrationlisters.ValidatingWebhookConfigurationLister
	hasSynced             func() bool
}

func NewValidatingWebhookConfigurationManager(factory informers.SharedInformerFactory) generic.Source {
	informer := factory.Admissionregistration().V1beta1().ValidatingWebhookConfigurations()
	manager := &ValidatingWebhookConfigurationManager{
		rwLock:                &sync.RWMutex{},
		tenantToConfiguration: make(map[multitenancyutil.TenantHash]*v1beta1.ValidatingWebhookConfiguration),
		lister:                informer.Lister(),
		hasSynced:             informer.Informer().HasSynced,
	}

	// On any change, rebuild the config
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			webhook := obj.(*v1beta1.ValidatingWebhookConfiguration)
			tenant, err := multitenancyutil.TransformTenantInfoFromAnnotations(webhook.Annotations)
			if err != nil {
				glog.Warningf("fail to extract tenant info from validating webhook: %v", webhook.Name)
				return
			}
			manager.updateConfiguration(tenant)
		},
		UpdateFunc: func(_, obj interface{}) {
			webhook := obj.(*v1beta1.ValidatingWebhookConfiguration)
			tenant, err := multitenancyutil.TransformTenantInfoFromAnnotations(webhook.Annotations)
			if err != nil {
				glog.Warningf("fail to extract tenant info from validating webhook: %v", webhook.Name)
				return
			}
			manager.updateConfiguration(tenant)
		},
		DeleteFunc: func(obj interface{}) {
			webhook := obj.(*v1beta1.ValidatingWebhookConfiguration)
			tenant, err := multitenancyutil.TransformTenantInfoFromAnnotations(webhook.Annotations)
			if err != nil {
				glog.Warningf("fail to extract tenant info from validating webhook: %v", webhook.Name)
				return
			}
			manager.updateConfiguration(tenant)
		},
	})

	return manager
}

// Webhooks returns the merged ValidatingWebhookConfiguration.
func (v *ValidatingWebhookConfigurationManager) Webhooks() []v1beta1.Webhook {
	panic("must not reach this line")
}

func (m *ValidatingWebhookConfigurationManager) updateConfiguration(tenant multitenancy.TenantInfo) {
	m = m.innerShallowCopyWithTenant(tenant).(*ValidatingWebhookConfigurationManager)
	configurations, err := m.lister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("error updating configuration: %v", err))
		return
	}
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	m.tenantToConfiguration[multitenancyutil.GetHashFromTenant(tenant)] = mergeValidatingWebhookConfigurations(configurations)
}

func mergeValidatingWebhookConfigurations(
	configurations []*v1beta1.ValidatingWebhookConfiguration,
) *v1beta1.ValidatingWebhookConfiguration {
	sort.SliceStable(configurations, ValidatingWebhookConfigurationSorter(configurations).ByName)
	var ret v1beta1.ValidatingWebhookConfiguration
	for _, c := range configurations {
		ret.Webhooks = append(ret.Webhooks, c.Webhooks...)
	}
	return &ret
}

type ValidatingWebhookConfigurationSorter []*v1beta1.ValidatingWebhookConfiguration

func (a ValidatingWebhookConfigurationSorter) ByName(i, j int) bool {
	return a[i].Name < a[j].Name
}

func (m *ValidatingWebhookConfigurationManager) innerShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *m
	copied.lister = m.lister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(admissionregistrationlisters.ValidatingWebhookConfigurationLister)
	return &copied
}

func (m *ValidatingWebhookConfigurationManager) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return tenantValidatingWebhookConfigurationManager{
		ValidatingWebhookConfigurationManager: m,
		tenant:                                tenant,
	}
}

func (m *ValidatingWebhookConfigurationManager) HasSynced() bool {
	return m.hasSynced()
}

type tenantValidatingWebhookConfigurationManager struct {
	*ValidatingWebhookConfigurationManager
	tenant multitenancy.TenantInfo
}

func (m tenantValidatingWebhookConfigurationManager) Webhooks() []v1beta1.Webhook {
	m.ValidatingWebhookConfigurationManager.rwLock.RLock()
	defer m.ValidatingWebhookConfigurationManager.rwLock.RUnlock()
	if conf, ok := m.ValidatingWebhookConfigurationManager.tenantToConfiguration[multitenancyutil.GetHashFromTenant(m.tenant)]; ok {
		return conf.Webhooks
	}
	return nil
}
