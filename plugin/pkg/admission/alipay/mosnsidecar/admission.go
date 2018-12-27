package sidecar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/template"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/client-go/tools/cache"
	api "k8s.io/kubernetes/pkg/apis/core"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
)

const (
	PluginName                             = "AlipayMOSNSidecar"
	defaultMOSNSidecarTemplateConfigMapKey = "mosn-system/default-template"
	MOSNSidecarTemplateKey                 = "template"
	cpuConvertRatio                        = 4000
	memConvertScale                        = 128
	defaultIngressPort                     = "12200"
	defaultEgressPort                      = "12220"
	defaultRegistryPort                    = "13330"
	defaultLogsDir                         = "/home/admin/logs"
)

// sidecarInjectionSpec collects all container infos and volumes for
// sidecar container injection.
type sidecarInjectionSpec struct {
	Containers []api.Container `yaml:"containers"`
	Volumes    []api.Volume    `yaml:"volumes"`
	AppEnvs    []api.EnvVar    `yaml:"appEnvs"`
}

// sidecarTemplateData is the data object to which the templated
// version of `sidecarInjectionSpec` is applied.
type sidecarTemplateData struct {
	ObjectMeta *metav1.ObjectMeta
	PodSpec    *api.PodSpec
}

// alipayMOSNSidecar is an implementation of admission.Interface.
type alipayMOSNSidecar struct {
	*admission.Handler
	configMapLister corelisters.ConfigMapLister
}

var (
	_ admission.ValidationInterface                           = &alipayMOSNSidecar{}
	_ admission.MutationInterface                             = &alipayMOSNSidecar{}
	_ admission.InitializationValidator                       = &alipayMOSNSidecar{}
	_ kubeapiserveradmission.WantsInternalKubeInformerFactory = &alipayMOSNSidecar{}
)

// Register registers a admission plugin.
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return newAlipayMOSNSidecarPlugin(), nil
	})
}

// newalipayMOSNSidecarPlugin create a new admission plugin.
func newAlipayMOSNSidecarPlugin() *alipayMOSNSidecar {
	return &alipayMOSNSidecar{Handler: admission.NewHandler(admission.Create)}
}

func (s *alipayMOSNSidecar) SetInternalKubeInformerFactory(f informers.SharedInformerFactory) {
	s.configMapLister = f.Core().InternalVersion().ConfigMaps().Lister()
	s.SetReadyFunc(f.Core().InternalVersion().ConfigMaps().Informer().HasSynced)
}

func (s *alipayMOSNSidecar) ValidateInitialization() error {
	if s.configMapLister == nil {
		return fmt.Errorf("missing configMapLister")
	}
	return nil
}

func (s *alipayMOSNSidecar) Validate(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}

	op := a.GetOperation()
	if op != admission.Create && op != admission.Update {
		glog.Infof("mosn sidecar admission.validate only handles create and update event, this operations is: %+v", op)
		return nil
	}

	r := a.GetResource().GroupResource()
	if r == api.Resource("pods") {
		pod, ok := a.GetObject().(*api.Pod)
		if !ok {
			return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
		}

		return s.validatePod(pod)
	}

	return nil
}

func (s *alipayMOSNSidecar) validatePod(pod *api.Pod) error {
	// Check mosn sidecar injection annotation.
	v, ok := pod.Annotations[alipaysigmak8sapi.MOSNSidecarInject]
	if !ok {
		glog.Infof("no need to do injection, return")
		return nil
	}

	if v != string(alipaysigmak8sapi.SidecarInjectionPolicyEnabled) &&
		v != string(alipaysigmak8sapi.SidecarInjectionPolicyDisabled) {
		return apierrors.NewBadRequest("Value of mosn sidecar injection error, must be \"disabled\" or \"enabled\"")
	}

	return nil
}

// Admit makes an admission decision based on the request attributes.
func (s *alipayMOSNSidecar) Admit(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}

	op := a.GetOperation()
	if op != admission.Create && op != admission.Update {
		glog.Infof("mosn sidecar admission.admit only handles create and update event, this operation is: %+v", op)
		return nil
	}

	r := a.GetResource().GroupResource()
	if r != api.Resource("pods") {
		return nil
	}

	pod, ok := a.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	v, ok := pod.Annotations[alipaysigmak8sapi.MOSNSidecarInject]
	if !ok {
		glog.Infof("pod (%s/%s) no need to do injection, return", pod.Namespace, pod.Name)
		return nil
	}

	if v != string(alipaysigmak8sapi.SidecarInjectionPolicyEnabled) {
		glog.Infof("pod (%s/%s) mosn sidecar injection not enabled, return", pod.Namespace, pod.Name)
		return nil
	}

	// Ignore if this is inplace update request.
	_, ok = pod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState]
	if ok {
		glog.Infof("pod (%s/%s) this is inplace update request, no need to do injection, return", pod.Namespace, pod.Name)
		return nil
	}

	if op == admission.Create {
		return s.admitPodCreation(pod)
	}

	if op == admission.Update {
		return s.admitPodUpdate(pod)
	}

	return nil
}

func (s *alipayMOSNSidecar) admitPodCreation(pod *api.Pod) error {
	sidecarSpec, err := s.constructSidecarInjectionSpecFromConfigMap(pod)
	if err != nil {
		errMsg := fmt.Sprintf("failed to construct sidecar injection spec due to err: %+v", err)
		glog.Errorf(errMsg)
		return fmt.Errorf(errMsg)
	}

	if len(sidecarSpec.Containers) == 0 {
		errMsg := fmt.Sprintf("failed to construct sidecar injection spec, length of containers is 0")
		glog.Errorf(errMsg)
		return fmt.Errorf(errMsg)
	}

	err = setDiskQuotaModeIfNeeded(pod, sidecarSpec.Containers[0].Name)
	if err != nil {
		errMsg := fmt.Sprintf("failed to set disk quota model of mosn container due to err: %+v", err)
		glog.Errorf(errMsg)
		return fmt.Errorf(errMsg)
	}

	appContainers := pod.Spec.Containers

	// Append envs into app containers if needed.
	if len(sidecarSpec.AppEnvs) > 0 {
		for i, _ := range appContainers {
			appContainers[i].Env = append(appContainers[i].Env, sidecarSpec.AppEnvs...)
		}
	}

	if logsVolumeMount, found := findLogsVolumeFromAppContainer(appContainers); found {
		// Inject logs volume mount into mosn container.
		for i, _ := range sidecarSpec.Containers {
			sidecarSpec.Containers[i].VolumeMounts = append(sidecarSpec.Containers[i].VolumeMounts, logsVolumeMount)
		}
	}

	// Inject mosn container.
	pod.Spec.Containers = []api.Container{}
	pod.Spec.Containers = append(pod.Spec.Containers, sidecarSpec.Containers...)
	pod.Spec.Containers = append(pod.Spec.Containers, appContainers...)

	// Inject mosn volumes.
	pod.Spec.Volumes = append(pod.Spec.Volumes, sidecarSpec.Volumes...)

	return nil
}

func findLogsVolumeFromAppContainer(containers []api.Container) (api.VolumeMount, bool) {
	for _, c := range containers {
		for _, vm := range c.VolumeMounts {
			if vm.MountPath == defaultLogsDir {
				return vm, true
			}
		}
	}

	return api.VolumeMount{}, false
}

func setDiskQuotaModeIfNeeded(pod *api.Pod, containerName string) error {
	var allocSpec sigmak8sapi.AllocSpec
	if allocSpecString, ok := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]; ok {
		if err := json.Unmarshal([]byte(allocSpecString), &allocSpec); err != nil {
			return fmt.Errorf("alipay mosn sidecar unmarshal alloc spec string error: %+v", err)
		}
	}

	appContainer := sigmak8sapi.Container{}
	for _, c := range allocSpec.Containers {
		if c.Name != containerName {
			appContainer = c
			break
		}
	}

	mosnContainer := sigmak8sapi.Container{
		Name:       containerName,
		HostConfig: appContainer.HostConfig,
		Resource:   appContainer.Resource,
	}
	mosnContainer.HostConfig.DiskQuotaMode = sigmak8sapi.DiskQuotaModeRootFsOnly

	mosnContainerFound := false
	for i, c := range allocSpec.Containers {
		if c.Name == containerName {
			mosnContainerFound = true
			allocSpec.Containers[i] = mosnContainer
		}
	}

	if !mosnContainerFound {
		allocSpec.Containers = append(allocSpec.Containers, mosnContainer)
	}

	allocSpecBytes, err := json.Marshal(allocSpec)
	if err != nil {
		return fmt.Errorf("alipay mosn sidecar marshal alloc spec error: %+v", err)
	}

	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(allocSpecBytes)
	return nil
}

func (s *alipayMOSNSidecar) admitPodUpdate(pod *api.Pod) error {
	image, ok := pod.Annotations[alipaysigmak8sapi.MOSNSidecarImage]
	if !ok {
		glog.Infof("no mosn sidecar image specified on update event, return")
		return nil
	}

	if len(pod.Spec.Containers) < 2 {
		glog.Infof("no mosn sidecar container in pod spec, return")
		return nil
	}

	sidecarSpec, err := s.constructSidecarInjectionSpecFromConfigMap(pod)
	if err != nil {
		glog.Errorf("failed to construct sidecar injection spec due to err: %+v", err)
		return err
	}

	if pod.Spec.Containers[0].Name != sidecarSpec.Containers[0].Name {
		glog.Errorf("mosn sidecar container name not matched, return")
		return err
	}

	// Reset container image.
	pod.Spec.Containers[0].Image = image

	return nil
}

func (s *alipayMOSNSidecar) constructSidecarInjectionSpecFromConfigMap(pod *api.Pod) (*sidecarInjectionSpec, error) {
	namespace, name, _ := cache.SplitMetaNamespaceKey(defaultMOSNSidecarTemplateConfigMapKey)
	cm, err := s.configMapLister.ConfigMaps(namespace).Get(name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("config map (%s) not found", defaultMOSNSidecarTemplateConfigMapKey)
		}
		return nil, err
	}
	if len(cm.Data) == 0 {
		return nil, fmt.Errorf("no data in config map (%s)", defaultMOSNSidecarTemplateConfigMapKey)
	}

	template, ok := cm.Data[MOSNSidecarTemplateKey]
	if !ok {
		return nil, fmt.Errorf("template not exists in data of mosn sidecar config map error")
	}

	sidecarTemplateData := &sidecarTemplateData{
		ObjectMeta: &pod.ObjectMeta,
		PodSpec:    &pod.Spec,
	}
	templatedStr, err := executeTemplateToString(template, sidecarTemplateData)

	var sidecarSpec sidecarInjectionSpec
	if err := yaml.Unmarshal([]byte(templatedStr), &sidecarSpec); err != nil {
		errMsg := fmt.Sprintf("failed to unmarshall side car template: %s, get err: %+v", templatedStr, err)
		glog.Infof(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	if len(sidecarSpec.Containers) == 0 {
		return nil, fmt.Errorf("failed to get container from side car template")
	}

	return &sidecarSpec, nil
}

func shouldIgnore(a admission.Attributes) bool {
	resource := a.GetResource().GroupResource()
	if resource != api.Resource("pods") {
		return true
	}
	if a.GetSubresource() != "" {
		// only run the checks below on pods proper and not subresources
		return true
	}

	_, ok := a.GetObject().(*api.Pod)
	if !ok {
		glog.Errorf("expected pod but got %s", a.GetKind().Kind)
		return true
	}

	return false
}

// executeTemplate executes templateText with data and output written to w.
func executeTemplate(w io.Writer, templateText string, data interface{}) error {
	t := template.New("sidecar-injection")
	t.Funcs(template.FuncMap{
		"annotation":                   annotation,
		"isSet":                        isSet,
		"isCPUSet":                     isCPUSet,
		"CPUSetToInt64":                CPUSetToInt64,
		"CPUShareToInt64":              CPUShareToInt64,
		"convertMemoryBasedOnCPUCount": convertMemoryBasedOnCPUCount,
	})
	template.Must(t.Parse(templateText))
	return t.Execute(w, data)
}

// executeTemplateToString executes templateText with data and output written to string.
func executeTemplateToString(templateText string, data interface{}) (string, error) {
	b := bytes.Buffer{}
	err := executeTemplate(&b, templateText, data)
	return b.String(), err
}

func annotation(meta metav1.ObjectMeta, name string, defaultValue interface{}) string {
	value, ok := meta.Annotations[name]
	if !ok {
		value = fmt.Sprint(defaultValue)
	}
	return value
}

func isSet(m map[string]string, key string) bool {
	_, ok := m[key]
	return ok
}

func isCPUSet(meta metav1.ObjectMeta) bool {
	if meta.Annotations == nil {
		return false
	}

	allocSpecString, ok := meta.Annotations[sigmak8sapi.AnnotationPodAllocSpec]
	if !ok {
		// annotation not found
		return false
	}

	var allocSpec sigmak8sapi.AllocSpec
	if err := json.Unmarshal([]byte(allocSpecString), &allocSpec); err != nil {
		return false
	}
	for _, c := range allocSpec.Containers {
		if c.Resource.CPU.CPUSet != nil {
			return true
		}
	}

	return false
}

func CPUSetToInt64(podSpec *api.PodSpec, defaultValue interface{}) string {
	resourceMap := podSpec.Containers[0].Resources.Limits
	cpuValue, ok := resourceMap[api.ResourceCPU]
	if !ok {
		glog.Infof("limits.cpu of resource map not exists, use default value")
		return fmt.Sprint(defaultValue)
	}

	return strconv.FormatInt(cpuValue.MilliValue(), 10) + "m"
}

func CPUShareToInt64(podSpec *api.PodSpec, defaultValue interface{}) string {
	resourceMap := podSpec.Containers[0].Resources.Limits
	cpuValue, ok := resourceMap[api.ResourceCPU]
	if !ok {
		glog.Infof("limits.cpu of resource map not exists, use default value")
		return fmt.Sprint(defaultValue)
	}

	sidecarCPULimit := cpuValue.MilliValue() / cpuConvertRatio

	if sidecarCPULimit == 0 {
		sidecarCPULimit = 1
	}

	return strconv.FormatInt(sidecarCPULimit*1000, 10) + "m"
}

func convertMemoryBasedOnCPUCount(podSpec *api.PodSpec, defaultValue interface{}) string {
	resourceMap := podSpec.Containers[0].Resources.Limits
	_, ok := resourceMap[api.ResourceMemory]
	if !ok {
		glog.Infof("limits.mem of resource map not exists, use default value")
		return fmt.Sprint(defaultValue)
	}

	cpuValue, ok := resourceMap[api.ResourceCPU]
	if !ok {
		glog.Infof("limits.cpu of resource map not exists, use default value")
		return fmt.Sprint(defaultValue)
	}

	sidecarCPULimit := cpuValue.MilliValue() / cpuConvertRatio

	if sidecarCPULimit == 0 {
		sidecarCPULimit = 1
	}
	return strconv.FormatInt(sidecarCPULimit*memConvertScale, 10) + "Mi"
}
