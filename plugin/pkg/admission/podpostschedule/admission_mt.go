package podpostschedule

import (
	"fmt"
	"io"

	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	settingslisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
)

const (
	PluginName = "PodPostSchedule"
)

type podPostSchedulePlugin struct {
	*admission.Handler
	client     internalclientset.Interface
	nodeLister settingslisters.NodeLister
}

var _ admission.MutationInterface = &podPostSchedulePlugin{}
var _ = kubeapiserveradmission.WantsInternalKubeInformerFactory(&podPostSchedulePlugin{})
var _ = kubeapiserveradmission.WantsInternalKubeClientSet(&podPostSchedulePlugin{})

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewPlugin(), nil
	})
}

// NewPlugin creates a new pod post schedule admission plugin.
func NewPlugin() *podPostSchedulePlugin {
	return &podPostSchedulePlugin{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}
}

func (plugin *podPostSchedulePlugin) ValidateInitialization() error {
	if plugin.client == nil {
		return fmt.Errorf("%s requires a client", PluginName)
	}
	return nil
}

func (plugin *podPostSchedulePlugin) SetInternalKubeClientSet(client internalclientset.Interface) {
	plugin.client = client
}

func (plugin *podPostSchedulePlugin) SetInternalKubeInformerFactory(f informers.SharedInformerFactory) {
	nodeInformer := f.Core().InternalVersion().Nodes()
	plugin.nodeLister = nodeInformer.Lister()
	plugin.SetReadyFunc(func() bool { return nodeInformer.Informer().HasSynced() })
}

// Admit updates a pod cell info for each pod post schedule it matches.
func (plugin *podPostSchedulePlugin) Admit(a admission.Attributes) error {
	// Ignore all calls to subresources or resources other than pods so that the following type
	// assertion doesnt panic.
	if a.GetResource().GroupResource() != api.Resource("pods") || len(a.GetSubresource()) > 0 {
		return nil
	}

	pod := a.GetObject().(*api.Pod)

	tenant, _ := multitenancyutil.TransformTenantInfoFromAnnotations(pod.Annotations)
	plugin = plugin.ShallowCopyWithTenant(tenant).(*podPostSchedulePlugin)

	if !pod.DeletionTimestamp.IsZero() {
		return nil
	}

	if _, exist := pod.Annotations[multitenancy.LabelCellName]; !exist && len(pod.Spec.NodeName) > 0 {
		node, err := plugin.nodeLister.Get(pod.Spec.NodeName)
		if err != nil {
			return err
		}
		nodeCellName, ok := node.Labels[multitenancy.LabelCellName]
		if ok && len(nodeCellName) > 0 {
			pod.Annotations[multitenancy.LabelCellName] = node.Labels[multitenancy.LabelCellName]
		}
	}

	return nil
}

func (plugin *podPostSchedulePlugin) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copyPodPostSchedulePlugin := *plugin
	tenantNodeLister := plugin.nodeLister.(multitenancymeta.TenantWise)
	copyPodPostSchedulePlugin.nodeLister = tenantNodeLister.ShallowCopyWithTenant(tenant).(settingslisters.NodeLister)
	return &copyPodPostSchedulePlugin
}
