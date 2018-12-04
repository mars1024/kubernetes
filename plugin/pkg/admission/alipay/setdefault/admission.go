package setdefault

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
	"k8s.io/kubernetes/pkg/util/slice"
)

var (
	defaultCGroupName = flag.String("default-cgroup-parent", "/sigma", "default cgroup parent for each pods")
)

const (
	PluginName = "AlipaySetDefault"

	customCgroupParentNamespace = "kube-system"
	customCgroupParentName      = "custom-cgroup-parents"
	customCgroupParentDataKey   = "custom-cgroup-parents"
)

type AlipaySetDefault struct {
	*admission.Handler

	configMapLister corelisters.ConfigMapLister
}

var (
	_ admission.ValidationInterface                           = &AlipaySetDefault{}
	_ admission.MutationInterface                             = &AlipaySetDefault{}
	_ kubeapiserveradmission.WantsInternalKubeInformerFactory = &AlipaySetDefault{}
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewAlipaySetDefault(), nil
	})
}

func NewAlipaySetDefault() *AlipaySetDefault {
	return &AlipaySetDefault{Handler: admission.NewHandler(admission.Create)}
}

func (c *AlipaySetDefault) SetInternalKubeInformerFactory(f internalversion.SharedInformerFactory) {
	c.configMapLister = f.Core().InternalVersion().ConfigMaps().Lister()
	c.SetReadyFunc(f.Core().InternalVersion().ConfigMaps().Informer().HasSynced)
}

func (c *AlipaySetDefault) ValidateInitialization() error {
	if c.configMapLister == nil {
		return fmt.Errorf("missing configMapLister")
	}
	return nil
}

func (c *AlipaySetDefault) Validate(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}
	if !c.WaitForReady() {
		return admission.NewForbidden(a, fmt.Errorf("not yet ready to handle request"))
	}

	pod, ok := a.GetObject().(*core.Pod)
	if !ok {
		return admission.NewForbidden(a, fmt.Errorf("unexpected resource"))
	}

	if err = validateCgroupName(pod, c.cgroupParents); err != nil {
		return admission.NewForbidden(a, err)
	}
	return nil
}

func (c *AlipaySetDefault) Admit(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}

	pod, ok := a.GetObject().(*core.Pod)
	if !ok {
		return admission.NewForbidden(a, fmt.Errorf("unexpected resource"))
	}

	if err = addEnvSNToContainer(pod); err != nil {
		return admission.NewForbidden(a, err)
	}

	if err = setDefaultCgroupParent(pod); err != nil {
		return admission.NewForbidden(a, err)
	}
	return nil
}

func (c *AlipaySetDefault) cgroupParents() ([]string, error) {
	cm, err := c.configMapLister.ConfigMaps(customCgroupParentNamespace).Get(customCgroupParentName)
	if err != nil {
		return nil, err
	}
	return strings.Split(cm.Data[customCgroupParentDataKey], ";"), nil
}

func addEnvSNToContainer(pod *core.Pod) error {
	sn := pod.Labels[sigmak8sapi.LabelPodSn]
	if len(sn) == 0 {
		return fmt.Errorf("%s is missing", sigmak8sapi.LabelPodSn)
	}

next:
	for i, c := range pod.Spec.Containers {
		for _, env := range c.Env {
			if env.Name == "SN" {
				continue next
			}
		}
		pod.Spec.Containers[i].Env = append(c.Env, core.EnvVar{Name: "SN", Value: sn})
	}
	return nil
}

func setDefaultCgroupParent(pod *core.Pod) error {
	allocSpec, err := podAllocSpec(pod)
	if err != nil {
		return err
	}

	if allocSpec == nil {
		allocSpec = &sigmak8sapi.AllocSpec{}
	}
	if allocSpec.Containers == nil {
		allocSpec.Containers = make([]sigmak8sapi.Container, len(pod.Spec.Containers))
	}

next:
	for _, c := range pod.Spec.Containers {
		for _, ac := range allocSpec.Containers {
			if c.Name == ac.Name {
				continue next
			}
		}
		allocSpec.Containers = append(allocSpec.Containers, sigmak8sapi.Container{})
	}

	for i, c := range allocSpec.Containers {
		if len(c.HostConfig.CgroupParent) == 0 {
			c.HostConfig.CgroupParent = *defaultCGroupName
		}
		allocSpec.Containers[i].HostConfig.CgroupParent = addSlashFrontIfNotExists(c.HostConfig.CgroupParent)
	}

	data, err := json.Marshal(allocSpec)
	if err != nil {
		return err
	}
	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)
	return nil
}

func addSlashFrontIfNotExists(s string) string {
	if s[0] != '/' {
		return "/" + s
	}
	return s
}

func validateCgroupName(pod *core.Pod, listCgroupParent func() ([]string, error)) error {
	allocSpec, err := podAllocSpec(pod)
	if err != nil {
		return err
	}

	choices, err := listCgroupParent()
	if err != nil {
		return err
	}

	for _, c := range allocSpec.Containers {
		if !slice.ContainsString(choices, c.HostConfig.CgroupParent, nil) {
			return fmt.Errorf("%s container %s cgroup parent invalid, choices: %v",
				sigmak8sapi.AnnotationPodAllocSpec, c.Name, choices)
		}
	}
	return nil
}

func podAllocSpec(pod *core.Pod) (*sigmak8sapi.AllocSpec, error) {
	if v, exists := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]; exists {
		var allocSpec *sigmak8sapi.AllocSpec
		if err := json.Unmarshal([]byte(v), &allocSpec); err != nil {
			return nil, err
		}
		return allocSpec, nil
	}
	return nil, nil
}

func shouldIgnore(a admission.Attributes) bool {
	// Ignore all calls to subresources or resources other than pods.
	if len(a.GetSubresource()) != 0 || a.GetResource().GroupResource() != core.Resource("pods") {
		return true
	}

	return false
}
