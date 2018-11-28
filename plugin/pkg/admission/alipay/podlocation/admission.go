package podlocation

import (
	"fmt"
	"io"

	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	alipaysigmav2 "gitlab.alipay-inc.com/sigma/apis/pkg/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
)

const (
	PluginName = "AlipayPodLocation"
)

type AlipayPodLocation struct {
	*admission.Handler

	nodeLister corelisters.NodeLister
}

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewAlipayPodLocation(), nil
	})
}

var (
	_ admission.MutationInterface                             = &AlipayPodLocation{}
	_ admission.InitializationValidator                       = &AlipayPodLocation{}
	_ kubeapiserveradmission.WantsInternalKubeInformerFactory = &AlipayPodLocation{}
)

func NewAlipayPodLocation() *AlipayPodLocation {
	return &AlipayPodLocation{Handler: admission.NewHandler(admission.Create, admission.Update)}
}

func (l *AlipayPodLocation) ValidateInitialization() error {
	if l.nodeLister == nil {
		return fmt.Errorf("missing nodeLister")
	}
	return nil
}

func (l *AlipayPodLocation) Admit(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}
	if !l.WaitForReady() {
		return admission.NewForbidden(a, fmt.Errorf("not yet ready to handle request"))
	}

	pod := a.GetObject().(*core.Pod)

	var old *core.Pod
	if a.GetOperation() == admission.Update {
		old = a.GetOldObject().(*core.Pod)
	}

	if !isPodScheduled(pod, old) {
		return nil
	}

	if err = l.HandleLocationEnv(pod); err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func (l *AlipayPodLocation) SetInternalKubeInformerFactory(f internalversion.SharedInformerFactory) {
	l.nodeLister = f.Core().InternalVersion().Nodes().Lister()
	l.SetReadyFunc(f.Core().InternalVersion().Nodes().Informer().HasSynced)
}

func isPodScheduled(new, old *core.Pod) bool {
	if old == nil {
		return len(new.Spec.NodeName) > 0
	}
	return len(old.Spec.NodeName) == 0 && len(new.Spec.NodeName) > 0
}

var topologyKeyMap = []struct {
	env          string
	label        string
	defaultValue string
}{
	{alipaysigmav2.EnvSafetyOut, alipaysigmav2.EnvSafetyOut, "0"},
	{alipaysigmav2.EnvSigmaSite, sigmak8sapi.LabelSite, ""},
	{alipaysigmav2.EnvSigmaRegion, sigmak8sapi.LabelRegion, ""},
	{alipaysigmav2.EnvSigmaNCSN, sigmak8sapi.LabelNodeSN, ""},
	{alipaysigmav2.EnvSigmaNCHostname, sigmak8sapi.LabelHostname, ""},
	{alipaysigmav2.EnvSigmaNCIP, sigmak8sapi.LabelNodeIP, ""},
	{alipaysigmav2.EnvSigmaParentServiceTag, sigmak8sapi.LabelParentServiceTag, ""},
	{alipaysigmav2.EnvSigmaRoom, sigmak8sapi.LabelRoom, ""},
	{alipaysigmav2.EnvSigmaRack, sigmak8sapi.LabelRack, ""},
	{alipaysigmav2.EnvSigmaNetArchVersion, sigmak8sapi.LabelNetArchVersion, ""},
	{alipaysigmav2.EnvSigmaUplinkHostName, alipaysigmak8sapi.LabelUplinkHostname, ""},
	{alipaysigmav2.EnvSigmaUplinkIP, alipaysigmak8sapi.LabelUplinkIP, ""},
	{alipaysigmav2.EnvSigmaUplinkSN, alipaysigmak8sapi.LabelUplinkSN, ""},
	{alipaysigmav2.EnvSigmaASW, sigmak8sapi.LabelASW, ""},
	{alipaysigmav2.EnvSigmaLogicPod, sigmak8sapi.LabelLogicPOD, ""},
	{alipaysigmav2.EnvSigmaPod, sigmak8sapi.LabelPOD, ""},
	{alipaysigmav2.EnvSigmaDSWCluster, sigmak8sapi.LabelDSWCluster, ""},
	{alipaysigmav2.EnvSigmaNetLogicSite, sigmak8sapi.LabelNetLogicSite, ""},
	{alipaysigmav2.EnvSigmaSMName, sigmak8sapi.LabelMachineModel, ""},
	{alipaysigmav2.EnvSigmaModel, alipaysigmak8sapi.LabelModel, ""},
}

func (l *AlipayPodLocation) HandleLocationEnv(pod *core.Pod) error {
	// 调度器完成调度，更新调度调度结果时刷入Location 变量
	node, err := l.nodeLister.Get(pod.Spec.NodeName)
	if err != nil {
		return err
	}

	toEnvs := make([]core.EnvVar, 0, len(topologyKeyMap))
	for _, km := range topologyKeyMap {
		v := km.defaultValue
		if x := node.Labels[km.label]; len(x) > 0 {
			v = x
		}
		toEnvs = append(toEnvs, core.EnvVar{Name: km.env, Value: v})
	}

	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, toEnvs...)
	}
	return nil
}

func shouldIgnore(a admission.Attributes) bool {
	resource := a.GetResource().GroupResource()
	if resource != core.Resource("pods") {
		return true
	}

	_, ok := a.GetObject().(*core.Pod)
	if !ok {
		glog.Errorf("expected pod but got %s", a.GetKind().Kind)
		return true
	}

	if a.GetOperation() == admission.Update {
		if _, ok := a.GetOldObject().(*core.Pod); !ok {
			glog.Errorf("expected pod but got %s", a.GetOldObject().GetObjectKind().GroupVersionKind().Kind)
			return true
		}
	}
	return false
}
