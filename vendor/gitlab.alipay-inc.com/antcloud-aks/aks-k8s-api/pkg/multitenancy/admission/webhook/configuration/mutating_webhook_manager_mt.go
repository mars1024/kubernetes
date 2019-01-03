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

	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	admissionregistrationlisters "k8s.io/client-go/listers/admissionregistration/v1beta1"
	"k8s.io/client-go/tools/cache"
	"sync"
	"k8s.io/client-go/informers"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/generic"
)

// MutatingWebhookConfigurationManager collects the mutating webhook objects so that they can be called.
type MutatingWebhookConfigurationManager struct {
	rwLock                *sync.RWMutex
	tenantToConfiguration map[multitenancyutil.TenantHash]*v1beta1.MutatingWebhookConfiguration
	lister                admissionregistrationlisters.MutatingWebhookConfigurationLister
	hasSynced             func() bool
}

func NewMutatingWebhookConfigurationManager(factory informers.SharedInformerFactory) generic.Source {
	informer := factory.Admissionregistration().V1beta1().MutatingWebhookConfigurations()
	manager := &MutatingWebhookConfigurationManager{
		rwLock:                &sync.RWMutex{},
		tenantToConfiguration: make(map[multitenancyutil.TenantHash]*v1beta1.MutatingWebhookConfiguration),
		lister:                informer.Lister(),
		hasSynced:             informer.Informer().HasSynced,
	}

	// On any change, rebuild the config
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			webhook := obj.(*v1beta1.MutatingWebhookConfiguration)
			tenant, err := multitenancyutil.TransformTenantInfoFromAnnotations(webhook.Annotations)
			if err != nil {
				glog.Warningf("fail to extract tenant info from validating webhook: %v", webhook.Name)
				return
			}
			manager.updateConfiguration(tenant)
		},
		UpdateFunc: func(_, obj interface{}) {
			webhook := obj.(*v1beta1.MutatingWebhookConfiguration)
			tenant, err := multitenancyutil.TransformTenantInfoFromAnnotations(webhook.Annotations)
			if err != nil {
				glog.Warningf("fail to extract tenant info from validating webhook: %v", webhook.Name)
				return
			}
			manager.updateConfiguration(tenant)
		},
		DeleteFunc: func(obj interface{}) {
			webhook := obj.(*v1beta1.MutatingWebhookConfiguration)
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

func (m *MutatingWebhookConfigurationManager) HasSynced() bool {
	return m.hasSynced()
}

// Webhooks returns the merged MutatingWebhookConfiguration.
func (m *MutatingWebhookConfigurationManager) Webhooks() []v1beta1.Webhook {
	panic("must not reach this line")
}

func (m *MutatingWebhookConfigurationManager) updateConfiguration(tenant multitenancy.TenantInfo) {
	m = m.innerShallowCopyWithTenant(tenant).(*MutatingWebhookConfigurationManager)
	configurations, err := m.lister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("error updating configuration: %v", err))
		return
	}
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	m.tenantToConfiguration[multitenancyutil.GetHashFromTenant(tenant)] = mergeMutatingWebhookConfigurations(configurations)
}

func mergeMutatingWebhookConfigurations(configurations []*v1beta1.MutatingWebhookConfiguration) *v1beta1.MutatingWebhookConfiguration {
	var ret v1beta1.MutatingWebhookConfiguration
	// The internal order of webhooks for each configuration is provided by the user
	// but configurations themselves can be in any order. As we are going to run these
	// webhooks in serial, they are sorted here to have a deterministic order.
	sort.SliceStable(configurations, MutatingWebhookConfigurationSorter(configurations).ByName)
	for _, c := range configurations {
		ret.Webhooks = append(ret.Webhooks, c.Webhooks...)
	}
	return &ret
}

type MutatingWebhookConfigurationSorter []*v1beta1.MutatingWebhookConfiguration

func (a MutatingWebhookConfigurationSorter) ByName(i, j int) bool {
	return a[i].Name < a[j].Name
}

func (m *MutatingWebhookConfigurationManager) innerShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copied := *m
	copied.lister = m.lister.(multitenancymeta.TenantWise).ShallowCopyWithTenant(tenant).(admissionregistrationlisters.MutatingWebhookConfigurationLister)
	return &copied
}

func (m *MutatingWebhookConfigurationManager) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return tenantMutatingWebhookConfigurationManager{
		MutatingWebhookConfigurationManager: m,
		tenant:                              tenant,
	}
}

type tenantMutatingWebhookConfigurationManager struct {
	*MutatingWebhookConfigurationManager
	tenant multitenancy.TenantInfo
}

func (m tenantMutatingWebhookConfigurationManager) Webhooks() []v1beta1.Webhook {
	m.MutatingWebhookConfigurationManager.rwLock.RLock()
	defer m.MutatingWebhookConfigurationManager.rwLock.RUnlock()
	if conf, ok := m.MutatingWebhookConfigurationManager.tenantToConfiguration[multitenancyutil.GetHashFromTenant(m.tenant)]; ok {
		return conf.Webhooks
	}
	return nil
}
