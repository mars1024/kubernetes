package sidecar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/client-go/tools/cache"
	api "k8s.io/kubernetes/pkg/apis/core"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	alipaysigmak8sapi "gitlab.alipay-inc.com/sigma/apis/pkg/apis"
)

const (
	// PluginName AlipaySidecar, injector for common sidecars.
	PluginName = "AlipaySidecar"
)

const (
	// Describe supported sidecars' names, which are also ConfigMaps of injection template
	supportedSidecars = metav1.NamespaceSystem + "/sidecars"
	// Key of sidecars' names in supportedSidecars
	supportedSidecarKey = "names"
	// Key of each sidecar template in Template ConfigMap
	sidecarTemplateKey = "template"
)

const (
	// Resource convert ratio compared to biz container
	cpuConvertRatio = 4000
	memConvertScale = 128
	// Gathered all the logs of sidecars and biz container in the same path.
	defaultLogsDir = "/home/admin/logs"
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

// alipaySidecar is an implementation of admission.Interface.
type alipaySidecar struct {
	*admission.Handler
	configMapLister corelisters.ConfigMapLister
}

var (
	_ admission.ValidationInterface                           = &alipaySidecar{}
	_ admission.MutationInterface                             = &alipaySidecar{}
	_ admission.InitializationValidator                       = &alipaySidecar{}
	_ kubeapiserveradmission.WantsInternalKubeInformerFactory = &alipaySidecar{}
)

// Register registers a admission plugin.
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return newAlipaySidecarPlugin(), nil
	})
}

// newalipayMOSNSidecarPlugin create a new admission plugin.
func newAlipaySidecarPlugin() *alipaySidecar {
	return &alipaySidecar{Handler: admission.NewHandler(admission.Create, admission.Update)}
}

func (s *alipaySidecar) SetInternalKubeInformerFactory(f informers.SharedInformerFactory) {
	s.configMapLister = f.Core().InternalVersion().ConfigMaps().Lister()
	s.SetReadyFunc(f.Core().InternalVersion().ConfigMaps().Informer().HasSynced)
}

func (s *alipaySidecar) ValidateInitialization() error {
	if s.configMapLister == nil {
		return fmt.Errorf("missing configMapLister")
	}
	return nil
}

func (s *alipaySidecar) Validate(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}

	op := a.GetOperation()
	if op != admission.Create && op != admission.Update {
		glog.Infof("sidecar admission.validate only handles create and update event, this operations is: %+v", op)
		return nil
	}

	r := a.GetResource().GroupResource()
	if r == api.Resource("pods") {
		pod, ok := a.GetObject().(*api.Pod)
		if !ok {
			return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
		}

		names, err := s.getSupportSidecarNames()
		if err != nil {
			return admission.NewForbidden(a, fmt.Errorf("ConfigMap %s get failed: %v", supportedSidecars, err))
		}
		for _, name := range names {
			if err = s.validatePod(pod, name); err != nil {
				return admission.NewForbidden(a, err)
			}
		}
	}

	return nil
}

func (s *alipaySidecar) validatePod(pod *api.Pod, name string) error {
	// Check sidecar injection annotation.
	v := getPodSidecarInjectionPolicy(pod, name)

	if v != alipaysigmak8sapi.SidecarInjectionPolicyEnabled &&
		v != alipaysigmak8sapi.SidecarInjectionPolicyDisabled {
		return apierrors.NewBadRequest(
			fmt.Sprintf("Value of %s sidecar injection error, must be \"disabled\" or \"enabled\"", name),
		)
	}

	return nil
}

// Admit makes an admission decision based on the request attributes.
func (s *alipaySidecar) Admit(a admission.Attributes) (err error) {
	if shouldIgnore(a) {
		return nil
	}

	op := a.GetOperation()
	if op != admission.Create && op != admission.Update {
		glog.Infof("sidecar admission.admit only handles create and update event, this operation is: %+v", op)
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

	names, err := s.getSupportSidecarNames()
	if err != nil {
		return admission.NewForbidden(a, fmt.Errorf("ConfigMap %s get failed: %v", supportedSidecars, err))
	}

	// Ignore if this is inplace update request.
	_, ok = pod.Annotations[sigmak8sapi.AnnotationPodInplaceUpdateState]
	if ok {
		glog.Infof("pod (%s/%s) this is inplace update request, no need to do injection, return", pod.Namespace, pod.Name)
		return nil
	}

	// 因为sidecar container每次都是插入到 pod.spec.containers 的第一个，
	// 所以为了保证sidecar container按照SupportedSidecarNames的顺序注入必须要反序
	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]

		v := getPodSidecarInjectionPolicy(pod, name)
		if v != alipaysigmak8sapi.SidecarInjectionPolicyEnabled {
			glog.Infof("pod (%s/%s) %s sidecar injection not enabled, return", pod.Namespace, pod.Name, name)
			continue
		}

		if op == admission.Create {
			err = s.admitPodCreation(pod, name)
		}
		if op == admission.Update {
			err = s.admitPodUpdate(pod, name)
		}
		if err != nil {
			return admission.NewForbidden(a, err)
		}
	}

	return nil
}

func (s *alipaySidecar) admitPodCreation(pod *api.Pod, sidecarName string) error {
	sidecarSpec, err := s.constructSidecarInjectionSpecFromConfigMap(pod, getSidecarTemplateName(sidecarName))
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
		errMsg := fmt.Sprintf("failed to set disk quota model of %s container due to err: %+v", sidecarName, err)
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
		// Inject logs volume mount into sidecar container.
		for i, _ := range sidecarSpec.Containers {
			sidecarSpec.Containers[i].VolumeMounts = append(sidecarSpec.Containers[i].VolumeMounts, logsVolumeMount)
		}
	}

	// Inject sidecar container.
	pod.Spec.Containers = []api.Container{}
	pod.Spec.Containers = append(pod.Spec.Containers, sidecarSpec.Containers...)
	pod.Spec.Containers = append(pod.Spec.Containers, appContainers...)

	// Inject sidecar volumes.
	pod.Spec.Volumes = append(pod.Spec.Volumes, sidecarSpec.Volumes...)

	return nil
}

func (s *alipaySidecar) getSupportSidecarNames() ([]string, error) {
	namespace, name, _ := cache.SplitMetaNamespaceKey(supportedSidecars)
	cm, err := s.configMapLister.ConfigMaps(namespace).Get(name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	if err := yaml.Unmarshal([]byte(cm.Data[supportedSidecarKey]), &names); err != nil {
		return nil, err
	}
	return names, nil
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
			return fmt.Errorf("alipay sidecar unmarshal alloc spec string error: %+v", err)
		}
	}

	appContainer := sigmak8sapi.Container{}
	for _, c := range allocSpec.Containers {
		if c.Name != containerName {
			appContainer = c
			break
		}
	}

	sidecarContainer := sigmak8sapi.Container{
		Name:       containerName,
		HostConfig: appContainer.HostConfig,
		Resource:   appContainer.Resource,
	}
	sidecarContainer.HostConfig.DiskQuotaMode = sigmak8sapi.DiskQuotaModeRootFsOnly

	sidecarContainerFound := false
	for i, c := range allocSpec.Containers {
		if c.Name == containerName {
			sidecarContainerFound = true
			allocSpec.Containers[i] = sidecarContainer
		}
	}

	if !sidecarContainerFound {
		allocSpec.Containers = append(allocSpec.Containers, sidecarContainer)
	}

	allocSpecBytes, err := json.Marshal(allocSpec)
	if err != nil {
		return fmt.Errorf("alipay sidecar marshal alloc spec error: %+v", err)
	}

	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = string(allocSpecBytes)
	return nil
}

func (s *alipaySidecar) admitPodUpdate(pod *api.Pod, sidecarName string) error {
	if len(pod.Spec.Containers) < 2 {
		glog.Infof("no sidecar container in pod spec, return")
		return nil
	}

	sidecarSpec, err := s.constructSidecarInjectionSpecFromConfigMap(pod, getSidecarTemplateName(sidecarName))
	if err != nil {
		glog.Errorf("failed to construct sidecar injection spec due to err: %+v", err)
		return err
	}

	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == sidecarSpec.Containers[0].Name { // FIXME 默认sidecar就一个容器
			// Reset container image.
			pod.Spec.Containers[i].Image = sidecarSpec.Containers[0].Image
			return nil
		}
	}

	glog.Errorf("%s sidecar container name not matched, return", sidecarName)
	return err
}

func (s *alipaySidecar) constructSidecarInjectionSpecFromConfigMap(pod *api.Pod, cmKey string) (*sidecarInjectionSpec, error) {
	namespace, name, _ := cache.SplitMetaNamespaceKey(cmKey)
	cm, err := s.configMapLister.ConfigMaps(namespace).Get(name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("config map (%s) not found", cmKey)
		}
		return nil, err
	}
	if len(cm.Data) == 0 {
		return nil, fmt.Errorf("no data in config map (%s)", cmKey)
	}

	template, ok := cm.Data[sidecarTemplateKey]
	if !ok {
		return nil, fmt.Errorf("template not exists in data of %s sidecar config map error", cmKey)
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
		"valueOfMap":                   valueOfMap,
		"isSet":                        isSet,
		"isCPUSet":                     isCPUSet,
		"CPUSetToInt64":                CPUSetToInt64,
		"CPUShareToInt64":              CPUShareToInt64,
		"convertMemoryBasedOnCPUCount": convertMemoryBasedOnCPUCount,
		"ToUpper":                      strings.ToUpper,
		"ToLower":                      strings.ToLower,
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

func valueOfMap(m map[string]string, key string, defaultValue interface{}) string {
	value, ok := m[key]
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

func getPodSidecarInjectionPolicy(pod *api.Pod, name string) alipaysigmak8sapi.SidecarInjectionPolicy {
	v, exists := pod.Annotations[name+"."+alipaysigmak8sapi.SidecarAlipayPrefix+"/inject"]
	if !exists {
		return alipaysigmak8sapi.SidecarInjectionPolicyDisabled
	}
	return alipaysigmak8sapi.SidecarInjectionPolicy(v)
}

func getSidecarTemplateName(name string) string {
	return name + "-system/default-template"
}
