package podpreset

import (
	"fmt"
	"io"
	"strconv"

	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
)

const PluginName = "AlipayPodPreset"

// AlipayPodPreset is an implementation of admission.Interface.
type AlipayPodPreset struct {
	*admission.Handler

	configMapLister corelisters.ConfigMapLister
}

var (
	_ admission.ValidationInterface                           = &AlipayPodPreset{}
	_ admission.MutationInterface                             = &AlipayPodPreset{}
	_ admission.InitializationValidator                       = &AlipayPodPreset{}
	_ kubeapiserveradmission.WantsInternalKubeInformerFactory = &AlipayPodPreset{}
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewAlipayPodPreset(), nil
	})
}

// NewAlipayPodPreset create a new admission plugin
func NewAlipayPodPreset() *AlipayPodPreset {
	return &AlipayPodPreset{Handler: admission.NewHandler(admission.Create)}
}

func (p *AlipayPodPreset) SetInternalKubeInformerFactory(f internalversion.SharedInformerFactory) {
	p.configMapLister = f.Core().InternalVersion().ConfigMaps().Lister()
	p.SetReadyFunc(f.Core().InternalVersion().ConfigMaps().Informer().HasSynced)
}

func (p *AlipayPodPreset) ValidateInitialization() error {
	if p.configMapLister == nil {
		return fmt.Errorf("missing configMapLister")
	}
	return nil
}

func (p *AlipayPodPreset) Validate(a admission.Attributes) (err error) {
	if !isPodPresetConfigMap(a) {
		return nil
	}
	if !p.WaitForReady() {
		return admission.NewForbidden(a, fmt.Errorf("not yet ready to handle request"))
	}

	cm, ok := a.GetObject().(*core.ConfigMap)
	if !ok {
		return admission.NewForbidden(a, fmt.Errorf("unexpected resource"))
	}

	isDefault, err := strconv.ParseBool(cm.Labels[alipaysigmak8sapi.LabelDefaultPodPreset])
	if err != nil {
		return admission.NewForbidden(a, fmt.Errorf("label %s is invalid", alipaysigmak8sapi.LabelDefaultPodPreset))
	}
	if isDefault {
		defaultPreset, err := p.findDefaultConfigMap(cm.Namespace)
		if err != nil {
			return fmt.Errorf("findDefaultConfigMap error: %v", err)
		}
		if defaultPreset != nil {
			return admission.NewForbidden(a, fmt.Errorf("only one default AlipayPodPreset is allowed in a namespace"))
		}
	}

	if err = yaml.UnmarshalStrict([]byte(cm.Data["metadata"]), &metav1.ObjectMeta{}); err != nil {
		return admission.NewForbidden(a, err)
	}

	return nil
}

func (p *AlipayPodPreset) Admit(a admission.Attributes) (err error) {
	if !isPod(a) {
		return nil
	}
	if !p.WaitForReady() {
		return admission.NewForbidden(a, fmt.Errorf("not yet ready to handle request"))
	}

	pod, ok := a.GetObject().(*core.Pod)
	if !ok {
		return admission.NewForbidden(a, fmt.Errorf("unexpected resource"))
	}

	if err = p.HandlePodPresetConfig(pod); err != nil {
		return fmt.Errorf("HandlePodPresetConfig error: %v", err)
	}

	return nil
}

func (p *AlipayPodPreset) getPodPreset(pod *core.Pod) (*core.ConfigMap, error) {
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	presetName := pod.Labels[alipaysigmak8sapi.LabelPodPresetName]
	if len(presetName) == 0 {
		return p.findDefaultConfigMap(pod.Namespace)
	}

	preset, err := p.configMapLister.ConfigMaps(pod.Namespace).Get(presetName)
	if err != nil {
		return nil, err
	}
	if !isPodPreset(preset) {
		return nil, fmt.Errorf("ConfigMap %s is not a AlipayPodPreset type", preset.Name)
	}
	return preset, nil
}

const (
	podPresetUID             = "admission.sigma.alipay.com/podpreset-uid"
	podPresetResourceVersion = "admission.sigma.alipay.com/podpreset-resource-version"
)

func (p *AlipayPodPreset) HandlePodPresetConfig(pod *core.Pod) (err error) {
	preset, err := p.getPodPreset(pod)
	if err != nil {
		return err
	}

	if preset != nil {
		v, exists := preset.Data["metadata"]
		if !exists {
			return nil
		}

		var metadata metav1.ObjectMeta
		if err = yaml.Unmarshal([]byte(v), &metadata); err != nil {
			return err
		}
		for k, v := range metadata.Labels {
			if _, exists := pod.Labels[k]; !exists {
				pod.Labels[k] = v
			}
		}

		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string, 2)
		}
		pod.Annotations[podPresetUID] = string(preset.UID)
		pod.Annotations[podPresetResourceVersion] = preset.ResourceVersion
	}
	return nil
}

func (p *AlipayPodPreset) findDefaultConfigMap(ns string) (*core.ConfigMap, error) {
	cms, err := p.configMapLister.ConfigMaps(ns).List(
		labels.SelectorFromSet(map[string]string{alipaysigmak8sapi.LabelDefaultPodPreset: strconv.FormatBool(true)}),
	)
	if err != nil {
		return nil, err
	}

	if len(cms) == 0 {
		return nil, nil
	}
	return cms[0], nil
}

func isPod(attributes admission.Attributes) bool {
	if len(attributes.GetSubresource()) != 0 || attributes.GetResource().GroupResource() != core.Resource("pods") {
		return false
	}
	return true
}

func isPodPreset(cm *core.ConfigMap) bool {
	_, exists := cm.Labels[alipaysigmak8sapi.LabelDefaultPodPreset]
	return exists
}

func isPodPresetConfigMap(attributes admission.Attributes) bool {
	if len(attributes.GetSubresource()) != 0 || attributes.GetResource().GroupResource() != core.Resource("configmaps") {
		return false
	}

	cm, ok := attributes.GetObject().(*core.ConfigMap)
	if !ok {
		return false
	}
	return isPodPreset(cm)
}
