package test

import (
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/informers/admissionregistration"
	"k8s.io/client-go/informers/apps"
	"k8s.io/client-go/informers/autoscaling"
	"k8s.io/client-go/informers/batch"
	"k8s.io/client-go/informers/certificates"
	"k8s.io/client-go/informers/coordination"
	"k8s.io/client-go/informers/core"
	"k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/informers/events"
	"k8s.io/client-go/informers/extensions"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/informers/networking"
	"k8s.io/client-go/informers/policy"
	"k8s.io/client-go/informers/rbac"
	"k8s.io/client-go/informers/scheduling"
	"k8s.io/client-go/informers/settings"
	"k8s.io/client-go/informers/storage"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"reflect"
	"time"
)

type FakeOptions struct {
	UseConfigMapLister listerscorev1.ConfigMapLister
}

type FakeSharedInformerFactory struct {
	FakeOptions
}

func (fsif FakeSharedInformerFactory) Coordination() coordination.Interface {
	panic("implement me")
}

func (FakeSharedInformerFactory) Start(stopCh <-chan struct{}) {
	panic("implement me")
}

func (FakeSharedInformerFactory) InformerFor(obj runtime.Object, newFunc internalinterfaces.NewInformerFunc) cache.SharedIndexInformer {
	panic("implement me")
}

func (FakeSharedInformerFactory) ForResource(resource schema.GroupVersionResource) (informers.GenericInformer, error) {
	panic("implement me")
}

func (FakeSharedInformerFactory) WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool {
	panic("implement me")
}

func (FakeSharedInformerFactory) Admissionregistration() admissionregistration.Interface {
	panic("implement me")
}

func (FakeSharedInformerFactory) Apps() apps.Interface {
	panic("implement me")
}

func (FakeSharedInformerFactory) Autoscaling() autoscaling.Interface {
	panic("implement me")
}

func (FakeSharedInformerFactory) Batch() batch.Interface {
	panic("implement me")
}

func (FakeSharedInformerFactory) Certificates() certificates.Interface {
	panic("implement me")
}

func (fsif FakeSharedInformerFactory) Core() core.Interface {
	return &FakeCore{
		fsif.FakeOptions,
	}
}

func (FakeSharedInformerFactory) Events() events.Interface {
	panic("implement me")
}

func (FakeSharedInformerFactory) Extensions() extensions.Interface {
	panic("implement me")
}

func (FakeSharedInformerFactory) Networking() networking.Interface {
	panic("implement me")
}

func (FakeSharedInformerFactory) Policy() policy.Interface {
	panic("implement me")
}

func (FakeSharedInformerFactory) Rbac() rbac.Interface {
	panic("implement me")
}

func (FakeSharedInformerFactory) Scheduling() scheduling.Interface {
	panic("implement me")
}

func (FakeSharedInformerFactory) Settings() settings.Interface {
	panic("implement me")
}

func (FakeSharedInformerFactory) Storage() storage.Interface {
	panic("implement me")
}

// ---------------------------------------------------------------------------------------------------

type FakeCore struct {
	FakeOptions
}

func (fc FakeCore) V1() v1.Interface {
	return &FakeCoreV1{
		fc.FakeOptions,
	}
}

// ---------------------------------------------------------------------------------------------------

type FakeCoreV1 struct {
	FakeOptions
}

func (FakeCoreV1) ComponentStatuses() v1.ComponentStatusInformer {
	panic("implement me")
}

func (fc FakeCoreV1) ConfigMaps() v1.ConfigMapInformer {
	return &FakeConfigMapInformer{
		fc.FakeOptions,
	}
}

func (FakeCoreV1) Endpoints() v1.EndpointsInformer {
	panic("implement me")
}

func (FakeCoreV1) Events() v1.EventInformer {
	panic("implement me")
}

func (FakeCoreV1) LimitRanges() v1.LimitRangeInformer {
	panic("implement me")
}

func (FakeCoreV1) Namespaces() v1.NamespaceInformer {
	panic("implement me")
}

func (FakeCoreV1) Nodes() v1.NodeInformer {
	panic("implement me")
}

func (FakeCoreV1) PersistentVolumes() v1.PersistentVolumeInformer {
	panic("implement me")
}

func (FakeCoreV1) PersistentVolumeClaims() v1.PersistentVolumeClaimInformer {
	panic("implement me")
}

func (FakeCoreV1) Pods() v1.PodInformer {
	panic("implement me")
}

func (FakeCoreV1) PodTemplates() v1.PodTemplateInformer {
	panic("implement me")
}

func (FakeCoreV1) ReplicationControllers() v1.ReplicationControllerInformer {
	panic("implement me")
}

func (FakeCoreV1) ResourceQuotas() v1.ResourceQuotaInformer {
	panic("implement me")
}

func (FakeCoreV1) Secrets() v1.SecretInformer {
	panic("implement me")
}

func (FakeCoreV1) Services() v1.ServiceInformer {
	panic("implement me")
}

func (FakeCoreV1) ServiceAccounts() v1.ServiceAccountInformer {
	panic("implement me")
}

// ---------------------------------------------------------------------------------------------------

type FakeConfigMapInformer struct {
	FakeOptions
}

func (FakeConfigMapInformer) Informer() cache.SharedIndexInformer {
	return &FakeSharedIndexInformer{}
}

func (fcmi FakeConfigMapInformer) Lister() listerscorev1.ConfigMapLister {
	if fcmi.UseConfigMapLister != nil {
		return fcmi.UseConfigMapLister
	}
	return &FakeConfigMapLister{}
}

// ---------------------------------------------------------------------------------------------------

type FakeSharedIndexInformer struct {
}

func (FakeSharedIndexInformer) AddEventHandler(handler cache.ResourceEventHandler) {
	panic("implement me")
}

func (FakeSharedIndexInformer) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) {
	panic("implement me")
}

func (FakeSharedIndexInformer) GetStore() cache.Store {
	panic("implement me")
}

func (FakeSharedIndexInformer) GetController() cache.Controller {
	panic("implement me")
}

func (FakeSharedIndexInformer) Run(stopCh <-chan struct{}) {
	panic("implement me")
}

func (FakeSharedIndexInformer) HasSynced() bool {
	return true
}

func (FakeSharedIndexInformer) LastSyncResourceVersion() string {
	panic("implement me")
}

func (FakeSharedIndexInformer) AddIndexers(indexers cache.Indexers) error {
	panic("implement me")
}

func (FakeSharedIndexInformer) GetIndexer() cache.Indexer {
	panic("implement me")
}

// ---------------------------------------------------------------------------------------------------

type FakeConfigMapLister struct {

}

func (FakeConfigMapLister) List(selector labels.Selector) (ret []*v12.ConfigMap, err error) {
	panic("implement me")
}

func (FakeConfigMapLister) ConfigMaps(namespace string) listerscorev1.ConfigMapNamespaceLister {
	panic("implement me")
}

