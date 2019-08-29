package podpostschedule

import (
	"fmt"
	"io"

	"github.com/golang/glog"
	cafev1alpha1 "gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/apps/v1alpha1"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	settingslisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
	kubeletapis "k8s.io/kubernetes/pkg/kubelet/apis"

	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	multitenancymeta "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/meta"
	multitenancyutil "gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy/util"
)

const (
	PluginName = "PodPostSchedule"
	// PodAvailabilityZoneEnvKey is the env variable in pod to carry availability zone name
	PodAvailabilityZoneEnvKey = "SOFA_CAFE_AVAILABILITY_ZONE"
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

	if _, hasAppSvc := pod.Labels[cafev1alpha1.AppServiceNameLabel]; hasAppSvc {
		glog.Infof("pod %s/%s needs availability zone info", pod.Namespace, pod.Name)
		err := plugin.attachAvailabilityZoneInfo(pod)
		if err != nil {
			glog.Warningf("fail to attach availability zone info to pod %s/%s: %s", pod.Namespace, pod.Name, err)
		}
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

func (plugin *podPostSchedulePlugin) attachAvailabilityZoneInfo(pod *api.Pod) error {
	if len(pod.Spec.NodeName) == 0 {
		return nil
	}

	zone, err := plugin.getScheduledNodeAZ(pod.Spec.NodeName)
	if err != nil {
		glog.Warningf("fail to get availability zone info from node %s: %s", pod.Spec.NodeName, err)
		return err
	}

	// Add AZ label and env to pod.
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[kubeletapis.LabelZoneFailureDomain] = zone
	plugin.injectAvailabilityZoneInfo(&pod.Spec.InitContainers, zone)
	plugin.injectAvailabilityZoneInfo(&pod.Spec.Containers, zone)

	return nil
}

func (i *podPostSchedulePlugin) getScheduledNodeAZ(name string) (string, error) {
	node, err := i.nodeLister.Get(name)
	if err != nil {
		return "", fmt.Errorf("fail to find the pod scheduled node %s: %s", name, err)
	}

	if node.Labels == nil {
		glog.Warningf("node %s has no labels", name)
		return "", nil
	}

	zone, exist := node.Labels[kubeletapis.LabelZoneFailureDomain]
	if !exist {
		glog.Warningf("node %s has no availability zone label %s", name, kubeletapis.LabelZoneFailureDomain)
		return "", nil
	}

	return zone, nil
}

func (i *podPostSchedulePlugin) injectAvailabilityZoneInfo(containers *[]api.Container, zone string) {
	for i, c := range *containers {
		found := false
		for j, env := range c.Env {
			if env.Name == PodAvailabilityZoneEnvKey {
				(*containers)[i].Env[j].Value = zone
				found = true

				break
			}
		}

		if !found {
			(*containers)[i].Env = append((*containers)[i].Env, api.EnvVar{
				Name:  PodAvailabilityZoneEnvKey,
				Value: zone,
			})
		}
	}
}

func (plugin *podPostSchedulePlugin) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	copyPodPostSchedulePlugin := *plugin
	tenantNodeLister, ok := plugin.nodeLister.(multitenancymeta.TenantWise)
	if ok {
		copyPodPostSchedulePlugin.nodeLister = tenantNodeLister.ShallowCopyWithTenant(tenant).(settingslisters.NodeLister)
	}
	return &copyPodPostSchedulePlugin
}
