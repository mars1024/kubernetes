package ase

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiMachineryLabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/admission"
	genericadmissioninitializer "k8s.io/apiserver/pkg/admission/initializer"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/apis/core"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const PluginName = "Ase"

// AdmitFuncs和ValidateFuncs前会先经过ShouldIgnore，读Annotation，判断SubClusterName是否存在，不存在就跳过
// 写Func时可假设SubClusterName这个Annotation存在，也可假设alpha.cloud.alipay.com/cluster-id存在
// 写Func时要取ClusterId，需要先调用ase.getManagedClusterInfo，并使用返回值中的信息

var CommonAdmitFuncs = []func(ase *Ase, a admission.Attributes) error{
	AdmitFuncAddAseAnnotationsAndLabels,
}

var CommonValidateFuncs []func(ase *Ase, a admission.Attributes) error

var AdmitFuncs = map[string][]func(ase *Ase, a admission.Attributes) error{
	"Pod": {
		AdmitFuncPodOverSchedule,
		makeLogAgentContainer,
		generateSignedUrl,
		injectFasLogPath,
	},
}

var ValidateFuncs = map[string][]func(ase *Ase, a admission.Attributes) error{
	"Node": {
		ValidateFuncCheckNodeAnnotations,
	},
	"Pod": {
		ValidateFuncCheckImagePermission,
	},
}

var setLogAgentResourceRequest = func(logContainer *core.Container, logConfig *corev1.ConfigMap) {

	var podLogConfig PodLogConfig
	err := json.Unmarshal([]byte(logConfig.Data["config"]), &podLogConfig)
	if err != nil {
		return
	}

	var resourceQuota LogAgentResourceQuota
	resourceQuota = podLogConfig.LogAgentRequestedResource

	if len(resourceQuota.Cpu) == 0 || len(resourceQuota.Memory) == 0 || len(resourceQuota.Storage) == 0 {
		return
	}

	logContainer.Resources = core.ResourceRequirements{
		Requests: core.ResourceList{
			core.ResourceCPU:     resource.MustParse(resourceQuota.Cpu),
			core.ResourceMemory:  resource.MustParse(resourceQuota.Memory),
			core.ResourceStorage: resource.MustParse(resourceQuota.Storage),
		},
	}
}

func getSubClusterName(resource KubeResource) (string, bool) {
	resourceLabels := resource.GetLabels()
	if resourceLabels != nil && len(resourceLabels[LabelSubCluster]) > 0 {
		return resourceLabels[LabelSubCluster], true
	}
	return "", false
}

var makeLogAgentContainer = func(ase *Ase, a admission.Attributes) error {
	pod := a.GetObject().(*core.Pod)

	if pod.Annotations == nil {
		return nil
	}

	subClusterName, _ := getSubClusterName(pod)
	clusterName, _ := pod.Labels[LabelCluster]

	glog.V(4).Infof("ase admission controller make log agent container for %s", getNameOrGenerateNameFromPod(pod))

	// 判断容器是否存在
	for _, container := range pod.Spec.Containers {
		if container.Name == AseLogAgentContainerName {
			glog.V(4).Infof("ase admission controller log agent container already exists for %s", getNameOrGenerateNameFromPod(pod))
			return nil
		}
	}

	logConfigConfigMap := ase.GetConfigMapByName(clusterName, "ase-sub-cluster-pod-log-config-"+subClusterName)

	if logConfigConfigMap == nil {
		glog.V(4).Infof("ase admission controller no log config found")
		return nil
	}

	var logConfig PodLogConfig
	err := json.Unmarshal([]byte(logConfigConfigMap.Data["config"]), &logConfig)
	if err != nil {
		glog.V(4).Infof("ase admission controller cannot unmarshal LogConfig")
		return nil
	}

	var logUserId = logConfig.DefaultLogUserId
	var tenantId = logConfig.DefaultTenantId
	var projectName = logConfig.DefaultLogProject
	var storeName = logConfig.DefaultLogStore
	var projectRegionId = logConfig.DefaultProjectRegionId
	var config = logConfig.DefaultLogConfig
	var image = logConfig.DefaultImage
	var volumeMountConfigs = logConfig.DefaultVolumeMountConfigs
	var userDefinedId string

	// 判断是否有 logAgentContext
	logAgentContextStr, ok := pod.Annotations[AnnotationLogAgentContext]

	if ok {
		var logContext LogAgentContext
		err = json.Unmarshal([]byte(logAgentContextStr), &logContext)

		if err == nil {
			if logContext.UserId != "" {
				logUserId = logContext.UserId
			}

			if logContext.LogProjectName != "" {
				projectName = logContext.LogProjectName
			}

			if logContext.LogStoreName != "" {
				storeName = logContext.LogStoreName
			}

			if logContext.ProjectRegionId != "" {
				projectRegionId = logContext.ProjectRegionId
			}

			if logContext.UserDefinedId != "" {
				userDefinedId = logContext.UserDefinedId
			}

			if logContext.Config != "" {
				config = logContext.Config
			}

			if logContext.TenantId != "" {
				tenantId = logContext.TenantId
			}

			if logContext.Image != "" {
				image = logContext.Image
			}

			if logContext.VolumeMountConfigs != nil {
				volumeMountConfigs = logContext.VolumeMountConfigs
			}
		} else {
			glog.V(4).Infof("ase admission controller unable to unmarshal annotation %s: %v", AnnotationLogAgentContext, err)
		}
	}

	if image == "" {
		glog.V(4).Infof("empty image url for logtail container, skipping")
		return nil
	}

	if userDefinedId == "" {
		userDefinedId = projectName + "/" + storeName + "/" + tenantId + "/" + logUserId
	}

	envVars := []core.EnvVar{
		{
			Name:  AliyunLogtailUserId,
			Value: logUserId,
		},
		{
			Name:  AliyunLogtailUserDefinedId,
			Value: userDefinedId,
		},
		{
			Name:  AliyunLogtailConfig,
			Value: config,
		},
	}

	var livenessProbe = &core.Probe{
		Handler: core.Handler{
			Exec: &core.ExecAction{
				Command: []string{"/etc/init.d/ilogtaild", "status"},
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       30,
		SuccessThreshold:    1,
	}

	// 添加容器
	logTailContainer := core.Container{
		Name:                     AseLogAgentContainerName,
		Image:                    image,
		LivenessProbe:            livenessProbe,
		Env:                      envVars,
		TerminationMessagePolicy: "FallbackToLogsOnError",
		ImagePullPolicy:          core.PullIfNotPresent,
		VolumeMounts:             makeVolumeMountsForLogAgentContainer(volumeMountConfigs),
	}

	setLogAgentResourceRequest(&logTailContainer, logConfigConfigMap)

	var newLogContext = LogAgentContext{}
	newLogContext.UserId = logUserId
	newLogContext.LogProjectName = projectName
	newLogContext.LogStoreName = storeName
	newLogContext.UserDefinedId = userDefinedId
	newLogContext.Config = config
	newLogContext.TenantId = tenantId
	newLogContext.ProjectRegionId = projectRegionId

	bytes, err := json.Marshal(newLogContext)

	if err != nil {
		return admission.NewForbidden(a, errors.New("Failed to Marshal context "))
	}

	containers := append(pod.Spec.Containers, logTailContainer)

	a.GetObject().(*core.Pod).Annotations[AnnotationLogAgentContext] = string(bytes[:])
	a.GetObject().(*core.Pod).Spec.Containers = containers

	return nil
}

var generateSignedUrl = func(ase *Ase, a admission.Attributes) error {
	pod := a.GetObject().(*core.Pod)

	if pod.Annotations == nil {
		return nil
	}

	pathToSign, ok := pod.Annotations[AnnotationGenerateSignedUrl]

	if !ok {
		return nil
	}

	tenant, ok := pod.Labels[LabelTenant]

	if !ok || len(tenant) == 0 {
		glog.V(4).Infof("unable to generateSignedUrl: no tenant info for pod %s", getNameOrGenerateNameFromPod(pod))
		return nil
	}

	signedUrl, err := getSignedUrl(tenant, pathToSign)

	if err != nil {
		glog.V(4).Infof("error getting signed url for pod %s: %v", getNameOrGenerateNameFromPod(pod), err)
		return nil
	}

	containers := pod.Spec.Containers
	for i := range containers {
		if containers[i].Env == nil {
			containers[i].Env = []core.EnvVar{}
		}

		containers[i].Env = append(containers[i].Env, core.EnvVar{
			Name:  EnvVarSignedUrl,
			Value: signedUrl,
		})
	}

	return nil
}

var injectFasLogPath = func(ase *Ase, a admission.Attributes) error {
	pod := a.GetObject().(*core.Pod)

	if len(pod.GetGenerateName()) > 0 && len(pod.GetName()) == 0 {
		pod.SetName(names.SimpleNameGenerator.GenerateName(pod.GetGenerateName()))
	}

	if pod.Annotations == nil {
		return nil
	}

	result, ok := pod.Annotations[AnnotationFasInteropInjectLogPath]

	if !ok || result != "true" {
		return nil
	}

	containers := pod.Spec.Containers
	for i := range containers {
		if containers[i].Env == nil {
			containers[i].Env = []core.EnvVar{}
		}

		containers[i].Env = append(containers[i].Env, core.EnvVar{
			Name:  EnvPodName,
			Value: getNameOrGenerateNameFromPod(pod),
		})
	}

	volumes := pod.Spec.Volumes
	for i := range volumes {
		if volumes[i].HostPath == nil {
			continue
		}
		if strings.HasPrefix(volumes[i].HostPath.Path, InteropFasLogsRootPath) {
			if strings.Contains(volumes[i].Name, InteropFasLogsOnFasAgent) {
				volumes[i].HostPath.Path = normalizePath(volumes[i].HostPath.Path) + pod.Labels[LabelAppVersion] + "/"
			} else {
				volumes[i].HostPath.Path = normalizePath(volumes[i].HostPath.Path) + getNameOrGenerateNameFromPod(pod) + "/"
			}
		}
	}

	return nil
}

func normalizePath(path string) string {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

type getSignedUrlResponse struct {
	Data    string `json:"data,omitempty"`
	Success bool   `json:"success,omitempty"`
}

func getSignedUrl(tenant string, pathToSign string) (string, error) {
	aseUrl := getAseUrl()
	privateApiUrl, err := url.Parse(aseUrl + getSignedDownloadUrlPath)
	if err != nil {
		return "", err
	}

	parameters := url.Values{}
	parameters.Add("tenantName", tenant)
	parameters.Add("filePath", pathToSign)
	privateApiUrl.RawQuery = parameters.Encode()

	requestUrl := privateApiUrl.String()
	req, err := http.NewRequest("GET", requestUrl, bytes.NewBuffer([]byte{}))

	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if strings.Index(resp.Status, "200") != 0 {
		return "", errors.New("bad status code: " + resp.Status)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response getSignedUrlResponse

	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return "", err
	}

	if !response.Success {
		return "", errors.New("invalid response: " + string(bodyBytes))
	}

	return response.Data, nil
}

func getAseUrl() string {
	aseSystemUrl := os.Getenv(EnvAseSystemUrl)
	if len(aseSystemUrl) > 0 {
		if strings.HasSuffix(aseSystemUrl, "/") {
			aseSystemUrl = aseSystemUrl[:len(aseSystemUrl)-1]
		}
		return aseSystemUrl
	}
	return "http://10.252.1.107:8341"
}

func makeVolumeMountsForLogAgentContainer(items *[]VolumeMountConfigItem) []core.VolumeMount {
	if items == nil {
		return []core.VolumeMount{}
	} else {
		result := []core.VolumeMount{}
		for _, item := range *items {
			result = append(result, core.VolumeMount{
				Name:      item.VolumeName,
				MountPath: getMountPathForLogAgentContainer(item.MountAs),
				ReadOnly:  true,
			})
		}
		return result
	}
}

func getMountPathForLogAgentContainer(mountAs string) string {
	return "/home/admin/logs/" + sanitizeFileName(mountAs)
}

func sanitizeFileName(fileName string) string {
	return strings.Replace(strings.Replace(fileName, "/", "_", -1), ".", "_", -1)
}

func Factory(config io.Reader) (admission.Interface, error) {
	return NewAseAdmissionController(), nil
}

func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, Factory)
}

type Ase struct {
	*admission.Handler
	client          kubernetes.Interface
	configMapLister v1.ConfigMapLister
}

var _ admission.MutationInterface = &Ase{}
var _ admission.ValidationInterface = &Ase{}
var _ = genericadmissioninitializer.WantsExternalKubeInformerFactory(&Ase{})
var _ = genericadmissioninitializer.WantsExternalKubeClientSet(&Ase{})

func NewAseAdmissionController() *Ase {
	return &Ase{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}
}

func (ase *Ase) ValidateInitialization() error {
	if ase.client == nil {
		return fmt.Errorf("%s requires a client", PluginName)
	}
	if ase.configMapLister == nil {
		return fmt.Errorf("%s requires a configMapLister", PluginName)
	}
	return nil
}

func (ase *Ase) GetConfigMapByName(clusterName string, name string) *corev1.ConfigMap {

	glog.V(4).Infof("get config map %s %s", clusterName, name)

	list, err := ase.configMapLister.List(apiMachineryLabels.Everything())

	if err != nil {
		glog.V(4).Infof("ase.configMapLister err %v", err)
		return nil
	}

	for _, item := range list {

		itemClusterName, ok := item.Labels[LabelCluster]

		if !ok || clusterName != itemClusterName {
			continue
		}

		if item.Name == name {
			return item
		}
	}

	glog.V(4).Infof("config map not found %s %s", clusterName, name)
	return nil

}

func (ase *Ase) SetExternalKubeClientSet(client kubernetes.Interface) {
	ase.client = client
}

func (ase *Ase) SetExternalKubeInformerFactory(f informers.SharedInformerFactory) {
	ase.configMapLister = f.Core().V1().ConfigMaps().Lister()
	ase.SetReadyFunc(f.Core().V1().ConfigMaps().Informer().HasSynced)
}

func (ase *Ase) Validate(attributes admission.Attributes) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = admission.NewForbidden(attributes, errors.New(fmt.Sprintf("fatal error: %v", r)))
		}
	}()

	VerboseLogIfNeeded(attributes)
	if ase.ShouldIgnore(attributes, "Validate") {
		return nil
	}
	for _, f := range CommonValidateFuncs {
		err := f(ase, attributes)
		if err != nil {
			return err
		}
	}

	validateFuncs, ok := ValidateFuncs[attributes.GetKind().Kind]
	if ok {
		for _, f := range validateFuncs {
			err := f(ase, attributes)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (ase *Ase) Admit(attributes admission.Attributes) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = admission.NewForbidden(attributes, errors.New(fmt.Sprintf("fatal error: %v", r)))
		}
	}()

	VerboseLogIfNeeded(attributes)
	if ase.ShouldIgnore(attributes, "Admit") {
		return nil
	}

	if alreadyProcessedByAse(attributes) {
		return nil
	} else {
		setProcessedByAse(attributes)
	}

	for _, f := range CommonAdmitFuncs {
		err := f(ase, attributes)
		if err != nil {
			return err
		}
	}

	admitFuncs, ok := AdmitFuncs[attributes.GetKind().Kind]
	if ok {
		for _, f := range admitFuncs {
			err := f(ase, attributes)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func VerboseLogEnabled(attributes admission.Attributes) bool {
	kubeResource, ok := attributes.GetObject().(KubeResource)
	if !ok || kubeResource == nil {
		return false
	}
	annotations := kubeResource.GetAnnotations()
	if annotations == nil {
		return false
	}
	enabled, ok := annotations[AnnotationEnableVerboseAdmissionControllerLog]
	if !ok || enabled != "true" {
		return false
	}
	return true
}

func VerboseLogIfNeeded(attributes admission.Attributes) {
	if !VerboseLogEnabled(attributes) {
		return
	}
	attributesJson, err := json.Marshal(attributes.GetObject())
	glog.V(4).Infof("ase admission controller verbose log %v, %s, %s, %s, %s, %v, %v, %s \n",
		err, attributes.GetName(), attributes.GetKind(), attributes.GetNamespace(), attributes.GetOperation(),
		attributes.IsDryRun(), attributes.GetUserInfo(), attributesJson)
	glog.V(4).Info(getStackTrace())
}

func (ase *Ase) ShouldIgnore(attributes admission.Attributes, callerName string) bool {
	if attributes == nil {
		return true
	}

	// Ignore all calls to subresources
	if len(attributes.GetSubresource()) != 0 {
		return true
	}

	// 如果没有Object，放行
	kubeResource, ok := attributes.GetObject().(KubeResource)
	if !ok || kubeResource == nil {
		return true
	}

	// 如果没有Label，也放行
	labels := kubeResource.GetLabels()
	if labels == nil {
		return true
	}

	// 特殊ConfigMap，特殊判断放行
	if attributes.GetKind().Kind == "ConfigMap" && IsSpecialConfigMap(attributes.GetObject().(KubeResource).GetName()) {
		return true
	}

	// 如果没有 ase.cloud.alipay.com/sub-cluster-name，代表资源不受ASE托管，放行
	subClusterName, ok := getSubClusterName(attributes.GetObject().(KubeResource))
	if !ok {
		return true
	}

	glog.V(4).Infof("ase admission controller (%s) processing resource %s", callerName, kubeResource.GetName())

	clusterName := getClusterNameFromMetadata(kubeResource.GetObjectMeta())
	managedCluster := ase.getManagedClusterInfo(clusterName)

	if managedCluster == nil {
		glog.V(4).Infof("managed cluster (%s) not found for %s, skipping", clusterName, kubeResource.GetName())
		return true
	}

	subClusterFound := false
	for _, managedSubCluster := range managedCluster.ManagedSubClusters {
		if managedSubCluster.SubClusterName == subClusterName {
			subClusterFound = true
		}
	}

	if !subClusterFound {
		glog.V(4).Infof("subCluster %s not found in managed cluster (%s) not found for %s, skipping", subClusterName, clusterName, kubeResource.GetName())
		return true
	}

	return false
}

func alreadyProcessedByAse(attributes admission.Attributes) bool {
	if attributes.GetObject().(KubeResource).GetAnnotations() == nil {
		return false
	}
	processed, ok := attributes.GetObject().(KubeResource).GetAnnotations()[AnnotationProcessed]
	if !ok || processed != "true" {
		return false
	}
	return true
}

func setProcessedByAse(attributes admission.Attributes) {
	if attributes.GetObject().(KubeResource).GetAnnotations() == nil {
		attributes.GetObject().(KubeResource).SetAnnotations(map[string]string{})
	}
	attributes.GetObject().(KubeResource).GetAnnotations()[AnnotationProcessed] = "true"
}

func getClusterNameFromMetadata(metadata metav1.Object) string {
	return metadata.GetLabels()[LabelCluster]
}

func (ase *Ase) GetSubClusterSchedulerConfig(clusterName string, subClusterName string) *SchedulerConfig {
	var schedulerConfig SchedulerConfig
	configMap := ase.GetConfigMapByName(clusterName, "ase-sub-cluster-scheduler-config-"+subClusterName)

	if configMap == nil {
		return nil
	}

	_ = json.Unmarshal([]byte(configMap.Data["config"]), &schedulerConfig)

	return &schedulerConfig
}

func (ase *Ase) getManagedClusterInfo(clusterName string) *ManagedClusterInfo {

	var managedCluster ManagedClusterInfo
	configMap := ase.GetConfigMapByName(clusterName, AseManagedClusterConfigMapName)

	if configMap == nil {
		return nil
	}

	err := json.Unmarshal([]byte(configMap.Data["config"]), &managedCluster)

	if err != nil {
		glog.V(4).Infof("failed to unmarshal managedCluster of %s", configMap.Name)
		return nil
	}

	return &managedCluster
}

var AdmitFuncAddAseAnnotationsAndLabels = func(ase *Ase, a admission.Attributes) error {

	kubeResource := a.GetObject().(KubeResource)

	glog.V(4).Infof("ase admission controller adding ase annotations and labels to %s", kubeResource.GetName())

	cluster := ase.getManagedClusterInfo(getClusterNameFromMetadata(kubeResource.GetObjectMeta()))

	labels := kubeResource.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[LabelTenant] = cluster.ClusterTenantName
	labels[LabelWorkspace] = cluster.ClusterWorkspaceIdentity
	labels[LabelRegionId] = cluster.ClusterRegionId
	labels[LabelCluster] = cluster.ClusterName

	kubeResource.SetLabels(labels)

	return nil
}

var AdmitFuncPodOverSchedule = func(ase *Ase, a admission.Attributes) error {
	kubeResource := a.GetObject().(KubeResource)
	annotations := kubeResource.GetAnnotations()

	subClusterName, _ := getSubClusterName(kubeResource)
	subClusterSchedulerConfig := ase.GetSubClusterSchedulerConfig(getClusterNameFromMetadata(kubeResource.GetObjectMeta()), subClusterName)
	if subClusterSchedulerConfig == nil {
		glog.V(4).Infof("ase admission controller no sub cluster scheduler config found")
		return nil
	}
	containers := a.GetObject().(*core.Pod).Spec.Containers
	for i := range containers {
		if !a.GetObject().(*core.Pod).Spec.Containers[i].Resources.Requests.Cpu().IsZero() {
			// 配置 cpu requests
			var ratio float64
			val, ok := annotations[AnnotationCpuOverScheduleRatio]
			if ok {
				var err error
				ratio, err = strconv.ParseFloat(val, 64)
				if err != nil {
					ok = false
				}
			}
			if !ok {
				ratio = subClusterSchedulerConfig.CpuOverScheduleRatio
			}
			newResourceList := make(core.ResourceList)
			for key, val := range a.GetObject().(*core.Pod).Spec.Containers[i].Resources.Requests {
				if key == "cpu" {
					scale := val.ScaledValue(-3)
					newResourceList[key] = *apiResource.NewScaledQuantity(int64(float64(scale)/ratio), -3)
				} else {
					newResourceList[key] = val.DeepCopy()
				}
			}
			a.GetObject().(*core.Pod).Spec.Containers[i].Resources.Requests = newResourceList

			// 配置 cpu limits
			var hard bool
			val, ok = annotations[AnnotationHardCpuOverSchedule]
			if ok {
				var err error
				hard, err = strconv.ParseBool(val)
				if err != nil {
					ok = false
				}
			}
			if !ok {
				hard = subClusterSchedulerConfig.HardCpuOverSchedule
			}

			if hard {
				newResourceList := make(core.ResourceList)
				for key, val := range a.GetObject().(*core.Pod).Spec.Containers[i].Resources.Limits {
					if key == "cpu" {
						newResourceList[key] = a.GetObject().(*core.Pod).Spec.Containers[i].Resources.Requests["cpu"]
					} else {
						newResourceList[key] = val.DeepCopy()
					}
				}
				a.GetObject().(*core.Pod).Spec.Containers[i].Resources.Limits = newResourceList
			}
		}
	}
	return nil
}

var ValidateFuncCheckImagePermission = func(ase *Ase, a admission.Attributes) error {
	kubeResource := a.GetObject().(KubeResource)
	labels := kubeResource.GetLabels()
	subClusterName, _ := getSubClusterName(kubeResource)

	clusterName := getClusterNameFromMetadata(kubeResource.GetObjectMeta())
	imageConfigConfigMap := ase.GetConfigMapByName(clusterName, "ase-sub-cluster-image-config-"+subClusterName)

	managedCluster := ase.getManagedClusterInfo(clusterName)

	if imageConfigConfigMap == nil {
		glog.V(4).Infof("ase admission controller no sub cluster image config found")
		return nil
	}

	var imageConfig ImageConfig

	err := json.Unmarshal([]byte(imageConfigConfigMap.Data["config"]), &imageConfig)

	if err != nil {
		glog.V(4).Infof("ase admission controller cannot unmarshal ImageConfig")
		return nil
	}

	checkImagePermissionUrl := imageConfig.CheckImagePermissionUrl
	if checkImagePermissionUrl == "" {
		glog.V(4).Infof("ase admission controller checkImagePermissionUrl is empty, skipping")
		return nil
	}

	containers := a.GetObject().(*core.Pod).Spec.Containers
	for _, container := range containers {

		payload := CheckImagePermissionPayload{
			ImageUrl:          container.Image,
			ContainerName:     container.Name,
			ClusterId:         managedCluster.ClusterId,
			ClusterTenantName: labels[LabelTenant],
			ClusterWorkspace:  labels[LabelWorkspace],
		}

		jsonStr, _ := json.Marshal(payload)

		req, err := http.NewRequest("POST", checkImagePermissionUrl, bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")
		client := makeHttpClient()
		resp, err := client.Do(req)

		glog.V(4).Infof("ase admission controller checking image permission %s %s", checkImagePermissionUrl, jsonStr)

		if err != nil {
			return admission.NewForbidden(a, errors.New("error checking image permission (internal error: "+err.Error()+")"))
		}

		defer resp.Body.Close()

		if strings.Index(resp.Status, "200") != 0 {
			return admission.NewForbidden(a, errors.New("check image permission failed"))
		}

	}

	return nil
}

var ValidateFuncCheckNodeAnnotations = func(ase *Ase, a admission.Attributes) error {
	kubeResource := a.GetObject().(KubeResource)
	labels := kubeResource.GetLabels()
	subClusterName, _ := getSubClusterName(kubeResource)

	// 如果是ASE托管的Node，就需要检查必要的Label、Annotation是否存在
	requiredLabels := []string{LabelNodeGroupName}

	for _, labelKey := range requiredLabels {
		_, ok := labels[labelKey]
		if !ok {
			return admission.NewForbidden(a, errors.New("missing required label for node: "+labelKey))
		}
	}

	clusterName, _ := labels[LabelCluster]
	nodeGroupName, _ := labels[LabelNodeGroupName]

	nodeGroupConfigMapName := "aks-sub-cluster-node-group-config-" + subClusterName + "-" + nodeGroupName
	configMap := ase.GetConfigMapByName(clusterName, nodeGroupConfigMapName)

	if configMap == nil {
		return admission.NewForbidden(a, errors.New("invalid node group "+nodeGroupName+", config map "+nodeGroupConfigMapName+" not found"))
	}

	return nil
}

var makeHttpClient = func() HttpClient {
	return &http.Client{}
}

func InjectHttpClient(factory func() HttpClient) {
	makeHttpClient = factory
}

func (ase *Ase) ShallowCopyWithTenant(tenant multitenancy.TenantInfo) interface{} {
	return ase
}

func getStackTrace() string {
	b := make([]byte, 65536)
	n := runtime.Stack(b, true)
	return string(b[:n])
}

func getNameOrGenerateNameFromPod(pod *core.Pod) string {
	if len(pod.GetGenerateName()) > 0 && len(pod.GetName()) == 0 {
		return pod.GetGenerateName() + "?????"
	}
	return pod.GetName()
}


/**
TODO: 		  9. 如果资源类型为Container，存在disk_quato配置：ase.cloud.alipay.com/disk-quota-in-bytes，复制到标准anno中（sigma3.1，未来）

1. ase admin control 初始化时，list/watch configmap：ase-managed-cluster
该configmap会存储ase托管的cluster信息（需list/watch所有集群，并缓存）：
{
    clusterId: "xxx",
    clusterName: "yyy",
	clusterTenantId: "1",
	clusterTenantName: "q",
	clusterWorkspaceId: "2",
	clusterWorkspaceIdentity: "q",
	clusterRegionId: "3",
    ...,
    subClusters: [{
      subClusterName: "zzz",
      ...
    }, ...]  // 托管的子集群数组
}

依托该configmap，admin control获取并被告知当前ase所维护的集群信息。

2. 通过aks clusterId Anno（alpha.cloud.alipay.com/tenant-id，alpha.cloud.alipay.com/cluster-id，实际是name）判断k8s resource是否是admin control托管资源。


3. 若不是，则just return，交给其它admin control去处理。


4. 若是，则进入下面逻辑处理。


5. 判断资源上是否存在Anno：ase.cloud.alipay.com/sub-cluster-name，不存在则admin control报错，如412 Bad Request。


6. 为资源append immutable anno：
ase.cloud.alipay.com/is-ase-resource=true
ase.cloud.alipay.com/cluster-id=<alpha.cloud.alipay.com/cluster-id>
ase.cloud.alipay.com/cluster-name=<ase-managed-clusters.managedClusters.clusterName>
ase.cloud.alipay.com/cluster-tenant-id=<ase-managed-clusters.managedClusters.clusterTenantId>
ase.cloud.alipay.com/cluster-tenant-name=<ase-managed-clusters.managedClusters.clusterTenantName>
ase.cloud.alipay.com/cluster-workspace-id=<ase-managed-clusters.managedClusters.clusterWorkspaceId>
ase.cloud.alipay.com/cluster-workspace-identity=<ase-managed-clusters.managedClusters.clusterWorkspaceIdentity>
ase.cloud.alipay.com/cluster-region-id=<ase-managed-clusters.managedClusters.clusterRegionId>


7. 为资源append (immutable) label：
ase.cloud.alipay.com/cluster-id=<anno: ase.cloud.alipay.com/cluster-id>
ase.cloud.alipay.com/sub-cluster-name=<anno: ase.cloud.alipay.com/sub-cluster-name>
ase.cloud.alipay.com/is-ase-resource=true


8. 如果Anno：ase.cloud.alipay.com/sub-cluster-name值不为空，且资源类型为NODE，则做如下检查：
node.ase.cloud.alipay.com/node-group-name不为空，且值合法，需要读取NodeGroup相应ConfigMap是否存在之类
node.ase.cloud.alipay.com/node-reserved-cpu不为空，且合法
node.ase.cloud.alipay.com/node-reserved-memory不为空，且合法
node.ase.cloud.alipay.com/node-reserved-storage不为空，且合法
上述任意检查不满足则返回412 Bad Request之类


8. 如果anno上sub-cluster-name不为空，且资源类型为POD，则append或override POD schedulerName为：
ase-sub-cluster-scheduler-<subClusterName>


8. 如果anno上sub-cluster-name不为空，且资源类型为Container，读取Container上Anno：scheduling.ase.cloud.alipay.com/cpu-over-schedule-ratio
（判断值的合法性，Double ...）
如果Container Anno为空，则读取 ase-sub-cluster-scheduler-config-<subClusterName> ConfigMap（考虑性能优化）中的 cpuOverScheduleRatio，
将Container中的requested除以这个值并更新


8. 如果anno上sub-cluster-name不为空，且资源类型为Container，读取Container上Anno：scheduling.ase.cloud.alipay.com/hard-cpu-over-schedule
如果Container Anno为空，则读取 ase-sub-cluster-scheduler-config-<subClusterName> ConfigMap（考虑性能优化）中的 hardCpuOverSchedule，
将Container中的limit值设置为requested并更新


9. 如果资源类型为Container，存在disk_quato配置：ase.cloud.alipay.com/disk-quota-in-bytes，复制到标准anno中（sigma3.1，未来）
未来将IOPS、NETWORK等等限额全部标准化到Sigma3.1中


10. 如果anno上sub-cluster-name不为空，且资源类型为Container，
则读取 ase-sub-cluster-image-config-<subClusterName> ConfigMap（考虑性能优化）中的 checkImagePermissionUrl，
如果不为空，则构造如下报文，调用该接口：

     * 调用http rest hook检测镜像权限。
     * POST [checkImagePermissionUrl] (中枢内网可访问)
     * BODY内容固定位：
     * {
     * imageUrl: "xxx",
     * containerId: "xxx",
     * clusterId: "xxx",  // aks cluster id
     * clusterTenantId: "xxx",
     * clusterWorkspaceId: "xxx",
     * actualTenantId: "xxx",
     * actualWorkspaceId: "xxx",
     * ...
     * }

*/
