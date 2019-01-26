package setdefault

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strconv"
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
	defaultCGroupName = flag.String("default-cgroup-parent", defaultCGroupParent, "default cgroup parent for each pods")
)

const (
	PluginName = "AlipaySetDefault"

	customCGroupParentNamespace = "kube-system"
	customCGroupParentName      = "custom-cgroup-parents"
	customCGroupParentDataKey   = "custom-cgroup-parents"
)

const (
	bestEffortCGroupName = "/sigma-stream"
	defaultCGroupParent  = "/sigma"

	// 网络优先级的分配： 保留:0-2， 在线业务: 3-7, 离线业务： 8-15
	// Network QoS http://docs.alibaba-inc.com/pages/viewpage.action?pageId=479572415
	netPriorityUnknown          = 0
	netPriorityLatencySensitive = 5
	netPriorityBatchJobs        = 7

	// 在线任务使用1，离线任务使用-1, 目前仅调度在线任务固定使用2，ali2010rc1内核需要使用1，但蚂蚁线上没有此内核版本的机器
	// http://baike.corp.taobao.com/index.php/Task_prempt
	// http://baike.corp.taobao.com/index.php/Cpu_Isolation_Config
	cpuBvtWarpUnknown           = 0
	cpuBvtWarpNsLatencySensitve = 2
	cpuBvtWarpNsBatchJobs       = -1

	// 每个容器都要需要设置 SN 环境变量
	containerSNEnvName = "SN"
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

	if err = setDefaultHostConfig(pod); err != nil {
		return admission.NewForbidden(a, err)
	}
	return nil
}

func (c *AlipaySetDefault) cgroupParents() ([]string, error) {
	cm, err := c.configMapLister.ConfigMaps(customCGroupParentNamespace).Get(customCGroupParentName)
	if err != nil {
		return nil, err
	}
	return strings.Split(cm.Data[customCGroupParentDataKey], ";"), nil
}

func addEnvSNToContainer(pod *core.Pod) error {
	sn := pod.Labels[sigmak8sapi.LabelPodSn]
	if len(sn) == 0 {
		return fmt.Errorf("%s is missing", sigmak8sapi.LabelPodSn)
	}

next:
	for i, c := range pod.Spec.Containers {
		for _, env := range c.Env {
			if env.Name == containerSNEnvName {
				continue next
			}
		}
		pod.Spec.Containers[i].Env = append(c.Env, core.EnvVar{Name: containerSNEnvName, Value: sn})
	}
	return nil
}

func setDefaultHostConfig(pod *core.Pod) error {
	allocSpec, err := podAllocSpec(pod)
	if err != nil {
		return err
	}

	if allocSpec == nil {
		allocSpec = &sigmak8sapi.AllocSpec{}
	}
	if allocSpec.Containers == nil {
		allocSpec.Containers = make([]sigmak8sapi.Container, 0, len(pod.Spec.Containers))
	}

next:
	for _, c := range pod.Spec.Containers {
		for _, ac := range allocSpec.Containers {
			if c.Name == ac.Name {
				continue next
			}
		}
		allocSpec.Containers = append(allocSpec.Containers, newAllocSpecContainer(c.Name))
	}

	// 设置默认的 cgroup 根节点名，这里比较 tricky 的是部分参数是根据设置的 cgroup parent 来决定使用值.
	// 期望能有更好的方式解决这类配置问题
	netPriority := podNetPriority(pod)
	for i, c := range allocSpec.Containers {
		if len(c.HostConfig.CgroupParent) == 0 {
			c.HostConfig.CgroupParent = *defaultCGroupName
		}
		allocSpec.Containers[i].HostConfig.CgroupParent = addSlashFrontIfNotExists(c.HostConfig.CgroupParent)

		switch allocSpec.Containers[i].HostConfig.CgroupParent {
		case bestEffortCGroupName:
			allocSpec.Containers[i].HostConfig.CPUBvtWarpNs = cpuBvtWarpNsBatchJobs
			netPriority = netPriorityBatchJobs
		default:
			if c.HostConfig.CPUBvtWarpNs == cpuBvtWarpUnknown {
				allocSpec.Containers[i].HostConfig.CPUBvtWarpNs = cpuBvtWarpNsLatencySensitve
			}
			if netPriority == netPriorityUnknown {
				netPriority = netPriorityLatencySensitive
			}
		}
	}

	data, err := json.Marshal(allocSpec)
	if err != nil {
		return err
	}
	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(data)
	pod.Annotations[sigmak8sapi.AnnotationNetPriority] = strconv.FormatInt(netPriority, 10)
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

func newAllocSpecContainer(name string) sigmak8sapi.Container {
	return sigmak8sapi.Container{
		Name: name,
		Resource: sigmak8sapi.ResourceRequirements{
			// GPU.ShareMode is validated in admission controller 'sigmascheduling'
			GPU: sigmak8sapi.GPUSpec{ShareMode: sigmak8sapi.GPUShareModeExclusive},
		},
	}
}

func podNetPriority(pod *core.Pod) int64 {
	if v, exists := pod.Annotations[sigmak8sapi.AnnotationNetPriority]; exists {
		i, _ := strconv.ParseInt(v, 10, 64)
		return i
	}
	return 0
}

func shouldIgnore(a admission.Attributes) bool {
	// Ignore all calls to subresources or resources other than pods.
	if len(a.GetSubresource()) != 0 || a.GetResource().GroupResource() != core.Resource("pods") {
		return true
	}

	return false
}
