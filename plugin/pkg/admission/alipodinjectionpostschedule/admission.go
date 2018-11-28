package alipodinjectionpostschedule

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	settingslisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
)

const (
	PluginName = "AliPodInjectionPostSchedule"
)

// aliPodInjectionPostSchedule is an implementation of admission.Interface.
type aliPodInjectionPostSchedule struct {
	*admission.Handler
	client          internalclientset.Interface
	configMapLister settingslisters.ConfigMapLister
	nodeLister      settingslisters.NodeLister
}

var _ admission.MutationInterface = &aliPodInjectionPostSchedule{}
var _ = kubeapiserveradmission.WantsInternalKubeInformerFactory(&aliPodInjectionPostSchedule{})
var _ = kubeapiserveradmission.WantsInternalKubeClientSet(&aliPodInjectionPostSchedule{})

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewPlugin(), nil
	})
}

// NewPlugin creates a new aliPodInjectionPostSchedule plugin.
func NewPlugin() *aliPodInjectionPostSchedule {
	return &aliPodInjectionPostSchedule{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}
}

func (plugin *aliPodInjectionPostSchedule) ValidateInitialization() error {
	if plugin.client == nil {
		return fmt.Errorf("%s requires a client", PluginName)
	}
	return nil
}

func (c *aliPodInjectionPostSchedule) SetInternalKubeClientSet(client internalclientset.Interface) {
	c.client = client
}

func (a *aliPodInjectionPostSchedule) SetInternalKubeInformerFactory(f informers.SharedInformerFactory) {
	configMapInformer := f.Core().InternalVersion().ConfigMaps()
	nodeInformer := f.Core().InternalVersion().Nodes()
	a.configMapLister = configMapInformer.Lister()
	a.nodeLister = nodeInformer.Lister()
	a.SetReadyFunc(func() bool { return configMapInformer.Informer().HasSynced() && nodeInformer.Informer().HasSynced() })
}

// Admit injects a pod with the specific fields for each pod preset it matches.
func (c *aliPodInjectionPostSchedule) Admit(a admission.Attributes) error {
	// Ignore all calls to subresources or resources other than pods.
	if len(a.GetSubresource()) != 0 || a.GetResource().GroupResource() != api.Resource("pods") {
		return nil
	}

	newPod, ok := a.GetObject().(*api.Pod)
	if !ok {
		return errors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted by aliPodInjectionPostSchedule")
	}

	// container model must be dockervm
	if newPod.Labels[sigmak8sapi.LabelPodContainerModel] != "dockervm" {
		return nil
	}

	// 1. pod created with node name
	// 2. pod just be scheduled
	if a.GetOperation() == admission.Create {
		if newPod.Spec.NodeName == "" {
			return nil
		}
	} else if a.GetOperation() == admission.Update {
		oldPod, ok := a.GetOldObject().(*api.Pod)
		if !ok {
			return errors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted by aliPodInjectionPostSchedule")
		}

		if !(newPod.Spec.NodeName != "" && oldPod.Spec.NodeName == "") {
			return nil
		}
	} else {
		return nil
	}

	key := newPod.Namespace + "/" + newPod.Name

	node, err := c.nodeLister.Get(newPod.Spec.NodeName)
	if errors.IsNotFound(err) {
		return errors.NewNotFound(api.Resource("Node"), newPod.Spec.NodeName)
	} else if err != nil {
		return errors.NewInternalError(fmt.Errorf("failed to find new pod.Spec.NodeName %s: %v", newPod.Spec.NodeName, err))
	}

	var podAllocSpec sigmak8sapi.AllocSpec
	if data, ok := newPod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]; ok {
		if err := json.Unmarshal([]byte(data), &podAllocSpec); err != nil {
			return errors.NewBadRequest(fmt.Sprintf("aliPodInjectionPostSchedule unmarshal alloc-spec for %s failed: %v", key, err))
		}
	}
	defer func() {
		if !reflect.DeepEqual(podAllocSpec, sigmak8sapi.AllocSpec{}) {
			newPod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = dumpJson(podAllocSpec)
		}
	}()

	// 1. 根据node更新HostConfig
	updateHostConfig(newPod, node, &podAllocSpec)

	return nil
}

func updateHostConfig(pod *api.Pod, node *api.Node, allocSpec *sigmak8sapi.AllocSpec) {

	mainContainer := getMainContainer(pod)
	allocSpecContainer := getAllocSpecContainer(allocSpec, mainContainer.Name)
	hostConfigInfo := &allocSpecContainer.HostConfig

	if hostConfigInfo.CPUBvtWarpNs == 0 {
		//在线任务使用1，离线任务使用-1, 目前仅调度在线任务固定使用1
		// http://baike.corp.taobao.com/index.php/Task_prempt
		// http://baike.corp.taobao.com/index.php/Cpu_Isolation_Config
		if strings.Contains(node.Status.NodeInfo.KernelVersion, "ali2010.rc1") {
			hostConfigInfo.CPUBvtWarpNs = 1
		} else {
			hostConfigInfo.CPUBvtWarpNs = 2
		}
	}

	setAllocSpecContainer(allocSpec, allocSpecContainer)
}

func getMainContainer(pod *api.Pod) *api.Container {
	var mainContainer *api.Container
	if len(pod.Spec.Containers) == 1 {
		mainContainer = &pod.Spec.Containers[0]
	} else {
		for i := 0; i < len(pod.Spec.Containers); i++ {
			if pod.Spec.Containers[i].Name == "main" {
				mainContainer = &pod.Spec.Containers[i]
				break
			}
		}
	}
	return mainContainer
}

func getAllocSpecContainer(podAllocSpec *sigmak8sapi.AllocSpec, contianerName string) *sigmak8sapi.Container {
	for i := 0; i < len(podAllocSpec.Containers); i++ {
		c := &podAllocSpec.Containers[i]
		if c.Name == contianerName {
			return c
		}
	}
	return &sigmak8sapi.Container{
		Name: contianerName,
	}
}

func setAllocSpecContainer(podAllocSpec *sigmak8sapi.AllocSpec, container *sigmak8sapi.Container) {
	for _, c := range podAllocSpec.Containers {
		if c.Name == container.Name {
			return
		}
	}
	podAllocSpec.Containers = append(podAllocSpec.Containers, *container)
}

func dumpJson(v interface{}) string {
	str, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(str)
}
