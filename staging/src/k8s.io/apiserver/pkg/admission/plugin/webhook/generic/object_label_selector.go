package generic

import (
	"hash/fnv"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
	"k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// TODO(zuoxiu.jm): Remove it after rebasing onto 1.15 release
// Proposal: https://yuque.antfin-inc.com/antcloud-paas/aks/hg66u4

const (
	ObjectLabelSelectorAnnotationQualifiedPrefix = "object-selectors.aks.cafe.sofastack.io/"
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

	selectorMap := make(map[string]string)
	for k, v := range accessor.GetAnnotations() {
		if strings.HasPrefix(k, ObjectLabelSelectorAnnotationQualifiedPrefix) {
			labelKey := strings.TrimPrefix(k, ObjectLabelSelectorAnnotationQualifiedPrefix)
			if len(labelKey) > 0 {
				selectorMap[labelKey] = v
			}
		}
	}
	return labels.SelectorFromValidatedSet(selectorMap)
}

type webhookConstructor func(handler *admission.Handler, configFile io.Reader, sourceFactory sourceFactory, dispatcherFactory dispatcherFactory) (*Webhook, error)

// TODO(zuoxiu.jm): Remove it after rebasing onto 1.15 release
// NewWebhookWithObjectSelector proxies NewWebhook
func NewWebhookWithObjectSelectorProxy(deletgate webhookConstructor, handler *admission.Handler, configFile io.Reader, sourceFactory sourceFactory, dispatcherFactory dispatcherFactory) (*Webhook, error) {
	// one webhook holds one instance of object selector map supports
	objectSelectorMap := make(map[string]labels.Selector)
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
	reload := func(f informers.SharedInformerFactory) {
		rwLock.Lock()
		defer rwLock.Unlock()
		// reset
		objectSelectorMap = make(map[string]labels.Selector)
		list, err := f.Admissionregistration().V1beta1().MutatingWebhookConfigurations().Lister().List(labels.Everything())
		if err == nil {
			for i := range list {
				for j := range list[i].Webhooks {
					key := hash(&list[i].Webhooks[j])
					if feature.DefaultFeatureGate.Enabled(multitenancy.FeatureName) {
						if tenant, err := multitenancyutil.TransformTenantInfoFromAnnotations(list[i].Annotations); err != nil {
							key = strings.Join([]string{tenant.GetTenantID(), tenant.GetWorkspaceID(), tenant.GetWorkspaceID(), key}, "/")
						}
					}
					objectSelectorMap[key] = GetObjectLabelSelector(list[i])
				}
			}
		}
	}

	proxiedSourceFactory := func(f informers.SharedInformerFactory) Source {
		f.Admissionregistration().V1beta1().MutatingWebhookConfigurations().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(_ interface{}) {
				reload(f)
			},
			UpdateFunc: func(_, _ interface{}) {
				reload(f)
			},
			DeleteFunc: func(_ interface{}) {
				reload(f)
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
		if objectSelectorMap[hash(h)] != nil {
			return objectSelectorMap[hash(h)]
		}
		return labels.Everything()
	}
	return w, nil
}
