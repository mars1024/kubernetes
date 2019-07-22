package generic

import (
	"hash/fnv"
	"io"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/davecgh/go-spew/spew"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	webhooklister "k8s.io/client-go/listers/admissionregistration/v1beta1"
	"k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"github.com/golang/glog"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

// TODO(zuoxiu.jm): Remove it after rebasing onto 1.15 release
// Proposal: https://yuque.antfin-inc.com/antcloud-paas/aks/hg66u4

const (
	ObjectLabelSelectorAnnotationKey = "aks.cafe.sofastack.io/object-selectors"
)

// TODO(zuoxiu.jm): Remove it after rebasing onto 1.15 release
// GetObjectLabelSelector try gets qualified label-selector from webhook
func GetObjectLabelSelector(object runtime.Object) labels.Selector {
	// if thing unhappy happens, rollbacks to the everything
	accessor, err := meta.Accessor(object)
	if err != nil {
		// everything
		return labels.Everything()
	}

	if accessor.GetAnnotations() == nil {
		// everything
		return labels.Everything()
	}

	selectorMap := make(map[string]string)
	if nestedObjSelector, ok := accessor.GetAnnotations()[ObjectLabelSelectorAnnotationKey]; ok {
		err := json.Unmarshal([]byte(nestedObjSelector), &selectorMap)
		if err != nil {
			klog.Warningf("ignoring object selector of %v: failed deserialization from annotation content %v",
				accessor.GetName(), nestedObjSelector)
			// explicitly returning "everything" selector
			return labels.Everything()
		}
	}
	// returning an "everything" label-selector on receiving invalid map
	return labels.SelectorFromSet(selectorMap)
}

type webhookConstructor func(handler *admission.Handler, configFile io.Reader, sourceFactory sourceFactory, dispatcherFactory dispatcherFactory) (*Webhook, error)

// TODO(zuoxiu.jm): Remove it after rebasing onto 1.15 release
// NewMutatingWebhookWithObjectSelectorProxy proxies NewWebhook
func NewMutatingWebhookWithObjectSelectorProxy(deletgate webhookConstructor, handler *admission.Handler, configFile io.Reader, sourceFactory sourceFactory, dispatcherFactory dispatcherFactory) (*Webhook, error) {
	// one webhook holds one instance of object selector map supports
	objectSelectorMap := &atomic.Value{}
	objectSelectorMap.Store(make(map[string]labels.Selector))

	rwLock := &sync.RWMutex{}

	hash := func(h *v1beta1.Webhook) string {
		hasher := fnv.New32a()
		printer := spew.ConfigState{
			Indent:         " ",
			SortKeys:       true,
			DisableMethods: true,
			SpewKeys:       true,
		}
		printer.Fprintf(hasher, "%#v", h)
		return strconv.Itoa(int(hasher.Sum32()))
	}
	reload := func(lister webhooklister.MutatingWebhookConfigurationLister) {
		// reset
		newObjectSelectorMap := make(map[string]labels.Selector)
		list, err := lister.List(labels.Everything())
		if err == nil {
			for i := range list {
				webhook := list[i]
				for j := range webhook.Webhooks {
					// deepcopying for thread-safety
					subWebhook := webhook.Webhooks[j].DeepCopy()
					if feature.DefaultFeatureGate.Enabled(multitenancy.FeatureName) {
						if tenant, err := multitenancyutil.TransformTenantInfoFromAnnotations(webhook.Annotations); err == nil {
							injectTenantIntoWebhookName(tenant, subWebhook)
						}
					}
					key := hash(subWebhook)
					glog.V(5).Infof("loading webhook %v/%v, key %v", webhook.Name, subWebhook.Name, key)
					newObjectSelectorMap[key] = GetObjectLabelSelector(webhook)
				}
			}
		}
		rwLock.Lock()
		defer rwLock.Unlock()
		objectSelectorMap.Store(newObjectSelectorMap)
	}

	proxiedSourceFactory := func(f informers.SharedInformerFactory) Source {
		f.Admissionregistration().V1beta1().MutatingWebhookConfigurations().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(_ interface{}) {
				glog.V(5).Infof("triggered reloading due to add event")
				reload(f.Admissionregistration().V1beta1().MutatingWebhookConfigurations().Lister())
			},
			UpdateFunc: func(_, _ interface{}) {
				glog.V(5).Infof("triggered reloading due to update event")
				reload(f.Admissionregistration().V1beta1().MutatingWebhookConfigurations().Lister())
			},
			DeleteFunc: func(_ interface{}) {
				glog.V(5).Infof("triggered reloading due to delete event")
				reload(f.Admissionregistration().V1beta1().MutatingWebhookConfigurations().Lister())
			},
		})
		return sourceFactory(f)
	}
	w, err := deletgate(handler, configFile, proxiedSourceFactory, dispatcherFactory)
	if err != nil {
		return nil, err
	}

	w.selectorGetter = func(h *v1beta1.Webhook) labels.Selector {
		rwLock.RLock()
		defer rwLock.RUnlock()
		key := hash(h)
		selectorMap := objectSelectorMap.Load().(map[string]labels.Selector)
		if selectorMap[key] != nil {
			return selectorMap[key]
		}
		glog.V(5).Infof("does not find matching object selector for webhook %v, key %v", h.Name, key)
		return labels.Everything()
	}
	return w, nil
}

// TODO(zuoxiu.jm): Remove it after rebasing onto 1.15 release
// NewValidatingWebhookWithObjectSelectorProxy proxies NewWebhook
func NewValidatingWebhookWithObjectSelectorProxy(deletgate webhookConstructor, handler *admission.Handler, configFile io.Reader, sourceFactory sourceFactory, dispatcherFactory dispatcherFactory) (*Webhook, error) {
	// one webhook holds one instance of object selector map supports
	objectSelectorMap := &atomic.Value{}
	objectSelectorMap.Store(make(map[string]labels.Selector))

	rwLock := &sync.RWMutex{}

	hash := func(h *v1beta1.Webhook) string {
		hasher := fnv.New32a()
		printer := spew.ConfigState{
			Indent:         " ",
			SortKeys:       true,
			DisableMethods: true,
			SpewKeys:       true,
		}
		printer.Fprintf(hasher, "%#v", h)
		return strconv.Itoa(int(hasher.Sum32()))
	}
	reload := func(lister webhooklister.ValidatingWebhookConfigurationLister) {
		// reset
		newObjectSelectorMap := make(map[string]labels.Selector)
		list, err := lister.List(labels.Everything())
		if err == nil {
			for i := range list {
				webhook := list[i]
				for j := range webhook.Webhooks {
					// deepcopying for thread-safety
					subWebhook := webhook.Webhooks[j].DeepCopy()
					if feature.DefaultFeatureGate.Enabled(multitenancy.FeatureName) {
						if tenant, err := multitenancyutil.TransformTenantInfoFromAnnotations(webhook.Annotations); err == nil {
							injectTenantIntoWebhookName(tenant, subWebhook)
						}
					}
					key := hash(subWebhook)
					newObjectSelectorMap[key] = GetObjectLabelSelector(webhook)
				}
			}
		}
		rwLock.Lock()
		defer rwLock.Unlock()
		objectSelectorMap.Store(newObjectSelectorMap)
	}

	proxiedSourceFactory := func(f informers.SharedInformerFactory) Source {
		f.Admissionregistration().V1beta1().MutatingWebhookConfigurations().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(_ interface{}) {
				reload(f.Admissionregistration().V1beta1().ValidatingWebhookConfigurations().Lister())
			},
			UpdateFunc: func(_, _ interface{}) {
				reload(f.Admissionregistration().V1beta1().ValidatingWebhookConfigurations().Lister())
			},
			DeleteFunc: func(_ interface{}) {
				reload(f.Admissionregistration().V1beta1().ValidatingWebhookConfigurations().Lister())
			},
		})
		return sourceFactory(f)
	}
	w, err := deletgate(handler, configFile, proxiedSourceFactory, dispatcherFactory)
	if err != nil {
		return nil, err
	}

	w.selectorGetter = func(h *v1beta1.Webhook) labels.Selector {
		rwLock.RLock()
		defer rwLock.RUnlock()
		key := hash(h)
		selectorMap := objectSelectorMap.Load().(map[string]labels.Selector)
		if selectorMap[key] != nil {
			return selectorMap[key]
		}
		return labels.Everything()
	}
	return w, nil
}
