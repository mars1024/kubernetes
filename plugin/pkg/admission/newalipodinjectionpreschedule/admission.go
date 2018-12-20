package newalipodinjectionpreschedule

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/template"

	"github.com/golang/glog"
	sigma2api "gitlab.alibaba-inc.com/sigma/sigma-api/sigma"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
    "k8s.io/apimachinery/pkg/selection"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	settingslisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
	"k8s.io/kubernetes/pkg/util/slice"
)

const (
	PluginName                   = "NewAliPodInjectionPreSchedule"
	globalConfigName             = "sigma-alipodglobalrules-config"
	//appConfigName                = "sigma-alipodapprules-config"
	globalLabelMappingConfigName = "sigma2-sigma3-label-mapping"

	labelPodIpLabel = "sigma.alibaba-inc.com/ip-label"
	labelAppUnit    = "sigma.alibaba-inc.com/app-unit"
	labelAppStage   = "sigma.alibaba-inc.com/app-stage"

	//return punish if match failed(eg."su18" == na62)
	labelMatchPunish = -10000

	grayScaleApp = "sigma-injectionappgrayscale-config"

)

var matchLabelKey = map[string]int{sigmak8sapi.LabelInstanceGroup:1000, labelAppStage:100,
	labelAppUnit:10, sigmak8sapi.LabelSite:1}

//check
// aliPodInjectionPreSchedule is an implementation of admission.Interface.
type newaliPodInjectionPreSchedule struct {
	*admission.Handler
	client          internalclientset.Interface
	configMapLister settingslisters.ConfigMapLister
}

//check
var _ admission.MutationInterface = &newaliPodInjectionPreSchedule{}
var _ = kubeapiserveradmission.WantsInternalKubeInformerFactory(&newaliPodInjectionPreSchedule{})
var _ = kubeapiserveradmission.WantsInternalKubeClientSet(&newaliPodInjectionPreSchedule{})

//check
// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewPlugin(), nil
	})
}

//check
// NewPlugin creates a new aliPodInjectionPreSchedule plugin.
func NewPlugin() *newaliPodInjectionPreSchedule {
	return &newaliPodInjectionPreSchedule{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}
}

//check
func (plugin *newaliPodInjectionPreSchedule) ValidateInitialization() error {
	if plugin.client == nil {
		return fmt.Errorf("[newinjection]%s requires a client", PluginName)
	}
	return nil
}

//check
func (c *newaliPodInjectionPreSchedule) SetInternalKubeClientSet(client internalclientset.Interface) {
	c.client = client
}

//check
func (a *newaliPodInjectionPreSchedule) SetInternalKubeInformerFactory(f informers.SharedInformerFactory) {
	configMapInformer := f.Core().InternalVersion().ConfigMaps()
	a.configMapLister = configMapInformer.Lister()
	a.SetReadyFunc(func() bool { return configMapInformer.Informer().HasSynced() })
}

// Admit injects a pod with the specific fields for each pod preset it matches.
func (c *newaliPodInjectionPreSchedule) Admit(a admission.Attributes) error {

	if a.GetResource().GroupResource() != api.Resource("pods") {
		return nil
	}

	//glog.V(3).Infof("aliPodInjectionPreSchedule preStart to admit %s, operation: %v, subresource: %v, pod: %v", key, a.GetOperation(), a.GetSubresource(), dumpJson(pod))
	//the resource of the admission is subresource of pod(eg./pods/foo/status),and operation is Create
	if len(a.GetSubresource()) != 0 || a.GetOperation() != admission.Create {
		return nil
	}

	pod, ok := a.GetObject().(*api.Pod)
	if !ok {
		return errors.NewBadRequest("[newinjection]Resource was marked with kind Pod but was unable to be converted by newaliPodInjectionPreSchedule")
	}
	key := pod.Namespace + "/" + pod.Name
	//glog.V(3).Infof("[newinjection]new aliPodInjectionPreSchedule start to admit %s, operation: %v, subresource: %v, pod: %v", key, a.GetOperation(), a.GetSubresource(), dumpJson(pod))

	grayScale, err := c.configMapLister.ConfigMaps("kube-system").Get(grayScaleApp)
	if err != nil {
		if errors.IsNotFound(err) {
			glog.V(3).Infof("[newinjection]new aliPodInjectionPreSchedule not found grayScale config map for %s, exit", key)
		} else {
			glog.V(3).Infof("[newinjection]new aliPodInjectionPreSchedule load configmap grascaleApp error %b, exit", err)
		}
		return nil
	}

	grayScaleData := grayScale.Data
	if !isGray(pod.Labels[sigmak8sapi.LabelAppName], grayScaleData) {
		return nil
	}

	glog.V(3).Infof("[newinjection] new aliPodInjectionPreSchedule %s is gray, enter", pod.Labels[sigmak8sapi.LabelAppName])

	if pod.Labels[sigmak8sapi.LabelPodContainerModel] != "dockervm" {
		glog.V(3).Infof("[newinjection] sigmak8sapi.LabelPodContainerModel] != dockervm, %s", pod.Labels[sigmak8sapi.LabelPodContainerModel])
		return nil
	}

	podLabelUnit := pod.Labels[labelAppUnit]
	podLabelStage := pod.Labels[labelAppStage]

	if getMainContainer(pod) == nil {
		errors.NewBadRequest("Not found main container in pod")
	}
	//glog.V(3).Infof("[newinjection] start load global rules")
	// global配置
	globalConfigMap, err := c.configMapLister.ConfigMaps("kube-system").Get(globalConfigName)
	if errors.IsNotFound(err) {
		glog.V(5).Infof("[newinjection]new aliPodInjectionPreSchedule not found global config map for %s", key)
	} else if err != nil {
		glog.Warningf("[newinjection]new aliPodInjectionPreSchedule find app config map for %s failed: %v", key, err)
	}

	//glog.V(3).Infof("[newinjection] global rules is  %+v", globalConfigMap)

	var globalConfigMapData map[string]string
	if globalConfigMap != nil {
		globalConfigMapData = globalConfigMap.Data
	}


	//glog.V(3).Infof("[newinjection] start load labelconfig")

	// 标签映射配置
	labelConfigMap, err := c.configMapLister.ConfigMaps("kube-system").Get(globalLabelMappingConfigName)
	if errors.IsNotFound(err) {
		glog.V(5).Infof("[newinjection]new aliPodInjectionPreSchedule not found global label mapping config map for %s", key)
	} else if err != nil {
		glog.Warningf("[newinjection]new aliPodInjectionPreSchedule find global label mapping config map for %s failed: %v", key, err)
	}

	// 先做各种解析数据的事情

	var routeRules sigma2api.RouteRules
	if data, ok := globalConfigMapData["route-rules"]; ok {
		if err := json.Unmarshal([]byte(data), &routeRules); err != nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal route-rules failed: %v", err)
		}
	}
	var podRouteRule *sigma2api.RouteRuleDetail
	for _, r := range routeRules.Rules {
		if r.AppEnv == podLabelStage && r.AppUnit == podLabelUnit {
			podRouteRule = &r
			break
		}
	}

	var appUnitStageConstraint appUnitStageConstraint
	if data, ok := globalConfigMapData["app-unitstage-constraint"]; ok {
		if err := json.Unmarshal([]byte(data), &appUnitStageConstraint); err != nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal app-unitstage-constraint failed: %v", err)
		}
	}

	//FIXME
	//resourcePoolMapping := make(map[string]string)
	//if data, ok := globalConfigMapData["resourcepool-mapping"]; ok {
	//	if err := json.Unmarshal([]byte(data), &resourcePoolMapping); err != nil {
	//		glog.Warningf("aliPodInjectionPreSchedule unmarshal resourcepool-mapping failed: %v", err)
	//	}
	//}

	//remove since iplabel is removed from iplabel
	//var dynamicStrategy dynamicStrategy
	//if data, ok := globalConfigMapData["dynamic-schedulerules"]; ok {
	//	if err := json.Unmarshal([]byte(data), &dynamicStrategy); err != nil {
	//		glog.Warningf("aliPodInjectionPreSchedule unmarshal dynamic-schedulerules failed: %v", err)
	//	}
	//}

	oldTaintLabels := make([]string, 0)
	if data, ok := globalConfigMapData["taint-labels"]; ok {
		if err := json.Unmarshal([]byte(data), &oldTaintLabels); err != nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal taint-labels failed: %v", err)
		}
	}
	taintLabels := make([]string, 0)
	for _, tl := range oldTaintLabels {
		taintLabels = append(taintLabels, sigma2ToSigma3Label(labelConfigMap, tl))
	}

	var globalScheduleRules sigma2api.GlobalRules
	if data, ok := globalConfigMapData["global-schedulerules"]; ok {
		if err := json.Unmarshal([]byte(data), &globalScheduleRules); err != nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal global-schedulerules failed: %v", err)
		}
	}

	//p0m0NodegroupMap := make(map[string]string)
	//if data, ok := globalConfigMapData["p0m0-nodegroup-map"]; ok {
	//	if err := json.Unmarshal([]byte(data), &p0m0NodegroupMap); err != nil {
	//		glog.Warningf("aliPodInjectionPreSchedule unmarshal p0m0-nodegroup-map failed: %v", err)
	//	}
	//}

	labelsCompatible := make(map[string]string)
	if data, ok := globalConfigMapData["label-compatible"]; ok {
		if err := json.Unmarshal([]byte(data), &labelsCompatible); err != nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal label-compatible failed: %v", err)
		}
	}

	// app配置
    req1, _ := labels.NewRequirement("sigma.alibaba-inc.com/appconfig-beta1", selection.Equals, []string{"true"})
	appConfigMapAll, err := c.configMapLister.ConfigMaps(pod.Namespace).List(labels.NewSelector().Add(*req1))
	if errors.IsNotFound(err) {
		glog.V(5).Infof("[newinjection]new aliPodInjectionPreSchedule not found app config map for %s", key)
	} else if err != nil {
		glog.Warningf("[newinjection]new aliPodInjectionPreSchedule find global config map for %s failed: %v", key, err)
	}

	//glog.V(3).Infof("[newinjection] app configmap is %+v", appConfigMapAll)

	//match configmap according to tetrad
	appConfigMap := matchConfigMapWithSelector(appConfigMapAll, pod)

	//glog.V(3).Infof("[newinjection]new aliPodInjectionPreSchedule matched configmap %+v", appConfigMap)

	var appConfigMapData map[string]string
	if appConfigMap != nil {
		appConfigMapData = appConfigMap.Data
	}

	var constraints constraints
	if data, ok := appConfigMapData["constraints"]; ok {
		if err := json.Unmarshal([]byte(data), &constraints); err !=nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal constraints for %s failed: %v", key, err)
		}
	}

	var spread spread
	if data, ok := appConfigMapData["spread"]; ok {
		if err := json.Unmarshal([]byte(data), &spread); err !=nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal spread for %s failed: %v", key, err)
		}
	}

	//var dependency dependency
	//if data, ok := appConfigMapData["dependency"]; ok {
	//	if err := json.Unmarshal([]byte(data), &dependency); err !=nil {
	//		glog.Warningf("newaliPodInjectionPreSchedule unmarshal dependency for %s failed: %v", key, err)
	//	}
	//}

	var allocSpec allocSpec
	if data, ok := appConfigMapData["allocSpec"]; ok {
		if err := json.Unmarshal([]byte(data), &allocSpec); err !=nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal allocSpec for %s failed: %v", key, err)
		}
	}

	var hostConfig hostConfig
	if data, ok := appConfigMapData["hostConfig"]; ok {
		if err := json.Unmarshal([]byte(data), &hostConfig); err !=nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal hostconfig for %s failed: %v", key, err)
		}
	}

	var extConfig extConfig
	if data, ok := appConfigMapData["extConfig"]; ok {
		if err := json.Unmarshal([]byte(data), &extConfig); err !=nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal extConfig for %s failed: %v", key, err)
		}
	}

	var prohibit prohibit
	if data, ok := appConfigMapData["prohibit"]; ok {
		if err := json.Unmarshal([]byte(data), &prohibit); err !=nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal prohibit for %s failed: %v", key, err)
		}
	}

	var monopolize monopolize
	if data, ok := appConfigMapData["monopolize"]; ok {
		if err := json.Unmarshal([]byte(data), &monopolize); err !=nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal monopolize for %s failed: %v", key, err)
		}
	}

	/*
	var appMetaInfo appMetaInfo
	if data, ok := appConfigMapData["metainfos"]; ok {
		if err := json.Unmarshal([]byte(data), &appMetaInfo); err != nil {
			glog.Warningf("aliPodInjectionPreSchedule unmarshal metainfos for %s failed: %v", key, err)
		}
	}

	var staticStrategy sigma2api.AdvancedStrategy
	if data, ok := appConfigMapData["static-schedulerules-"+pod.Labels[sigmak8sapi.LabelInstanceGroup]]; ok {
		if err := json.Unmarshal([]byte(data), &staticStrategy); err != nil {
			glog.Warningf("aliPodInjectionPreSchedule unmarshal staticStrategy nodegroup for %s failed: %v", key, err)
		}
	} else if data, ok := appConfigMapData["static-schedulerules"]; ok {
		if err := json.Unmarshal([]byte(data), &staticStrategy); err != nil {
			glog.Warningf("aliPodInjectionPreSchedule unmarshal staticStrategy app for %s failed: %v", key, err)
		}
	}

	var cpuSetModeAdvConfig cpuSetModeAdvConfig
	if data, ok := appConfigMapData["cpuset-mode-adv-rule"]; ok {
		if err := json.Unmarshal([]byte(data), &cpuSetModeAdvConfig); err != nil {
			glog.Warningf("aliPodInjectionPreSchedule unmarshal cpuSetModeAdvConfig for %s failed: %v", key, err)
		}
	}

	var appNamingMockRules appNamingMockRules
	if data, ok := appConfigMapData["mock-stageunit-rules"]; ok {
		if err := json.Unmarshal([]byte(data), &appNamingMockRules); err != nil {
			glog.Warningf("aliPodInjectionPreSchedule unmarshal mock-stageunit-rules for %s failed: %v", key, err)
		}
	}
	var podNamingMockRule *appNamingMockRuleDetail
	for _, r := range appNamingMockRules.Rules {
		if r.AppEnv == podLabelStage && r.AppUnit == podLabelUnit {
			podNamingMockRule = &r
		}
	}
	*/

	var podAllocSpec sigmak8sapi.AllocSpec
	if data, ok := pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec]; ok {
		if err := json.Unmarshal([]byte(data), &podAllocSpec); err != nil {
			glog.Warningf("[newinjection]new aliPodInjectionPreSchedule unmarshal exists pod alloc-spec for %s failed: %v", key, err)
		}
	}
	defer func() {
		if !reflect.DeepEqual(podAllocSpec, sigmak8sapi.AllocSpec{}) {
			pod.Annotations[sigmak8sapi.AnnotationPodAllocSpec] = dumpJson(podAllocSpec)
		}
	}()

	// 集团3.1已经不用deploy-unit概念了，兼容蚂蚁
	// 是否需要保留？
	if pod.Labels[sigmak8sapi.LabelDeployUnit] == "" && pod.Labels[sigmak8sapi.LabelInstanceGroup] != "" {
		pod.Labels[sigmak8sapi.LabelDeployUnit] = pod.Labels[sigmak8sapi.LabelInstanceGroup]
	}

	//check EnableOverQuota
	checkEnableOverQuota()

	// 1. 动态规则，现在只算iplabel了,是否还需要? appMetaInfo?
	// is to remove from sigma 3.1
	//dynamicStrategyMap := loadDynamicStrategyMap(pod, &appMetaInfo, &dynamicStrategy)

	// 2. 确定IpLabel, extConfig
	// dynamicStrategyMap is to remove from sigma 3.1 , will not need it
	setIpLabel(pod, podRouteRule, &extConfig)

	// 3. 设置调度规则里的单元、用途, check, 确认逻辑是否正确，与sigma2.0有出入
	setAppStageUnit(pod, podRouteRule, &appUnitStageConstraint)

	// 设置调度规则里的资源池，废弃
	//setResourcePool(pod, resourcePoolMapping)

	// 4. 设置其他默认调度规则
	setPodScheduleRulesCommon(pod)

	// 5. 设置NetPriority,
	setNetPriority(pod, &hostConfig)

	// 6. 设置Binds
	//FIXME 判断是都是sigma2.0 pod 迁移到sigma3.0，
	// 部分规则以sigma2.0设置为主，不进行merge，需要确认哪些规则需要判断
	if !isRebuildPod(pod) {
		setBindsToVolumeAndMounts(pod, &hostConfig)
	}

	// 7. 设置allocSpec里的HostConfigInfo
	setHostConfigInfo(pod, &podAllocSpec, &hostConfig)

	// FIXME why need isRebuild？
	// 8. 设置其他一些staticStrategy规则，如调度标签，privileged，host网络
	if !isRebuildPod(pod) {
		setPrivileged(pod, &hostConfig )
		setNetworkMode(pod, &hostConfig, &extConfig)
		setConstrains(pod, &constraints, taintLabels)
	}

	// 9. 设置应用互斥（包括同应用最大实例数）
	setPodAllocSpecAntiAffinityRules(pod, &podAllocSpec, &prohibit, &globalScheduleRules)

	// 10. cpu set/share相关配置
	if !isRebuildPod(pod) {
		setPodCPUConfigs(pod, &podAllocSpec, &extConfig.CpuSetModeAdvConfig, &globalScheduleRules)
	}

	// 11. 为强制标加tolerate
	setPodTolerateForMandatoryLabels(pod, taintLabels)

	// 12. 一些环境变量
	setPodEnvCommon(pod)

	// 13. 配一些挂载目录
	setPodMountsCommon(pod)

	// 14. 兼容2.0一些label
	updatePodLabelsCompatible(pod, labelsCompatible)

	// 15. 一些mock规则，用于测试
	//setMockRules(pod, podNamingMockRule)

	//16. 这是独占规则
	setMonopolize(pod, &podAllocSpec, &monopolize)

	//glog.V(3).Infof("[newinjection]new aliPodInjectionPreSchedule finish to admit %s, operation: %v, pod: %v", key, a.GetOperation(), dumpJson(pod))

	return nil
}

func matchConfigMapWithSelector(appConfigMapAll []*api.ConfigMap, pod *api.Pod) *api.ConfigMap {
	var target *api.ConfigMap = nil
	var maxScore = -10000
	//podLabel is the arr of tetrad, order is group, stage, unit, site
	for _, appconfig := range appConfigMapAll {
		if score := scoring(pod, appconfig); score > maxScore {
			maxScore = score
			target = appconfig
		}
	}
	if target == nil {
		glog.Errorf("[newinjection][matchConfigMapWithSelector] match tetrad error, the tetrad ")
	}
	return target
}

//this func is used to mark score for configmap
//the default order is group, stage, unit, site
//and corresponding weight is 1000, 100, 10, 1
//socore =  punish or sum(element_n_is_matched  weight_n) for n =1,2,3,4
func scoring(pod *api.Pod, configMap *api.ConfigMap) (score int) {
	score = 0
	//matchLabelKey is a map {(name of element in tetrad) : weight}
	for key, weight := range matchLabelKey{
		//here, we define that if an element in tetrad is unset,
		// the label will not exit (rather than key:"")
		if configMapLabel, ok := configMap.Labels[key]; ok {
			//label matched add score
			if 0 == strings.Compare(configMapLabel, pod.Labels[key]) {
				score += weight
			} else {
				//match failed
				return labelMatchPunish
			}
		}
		//label not exit in configmap, go on
	}
	return score
}

func loadDynamicStrategyMap(pod *api.Pod, appMetaInfo *appMetaInfo, dynamicStrategy *dynamicStrategy) map[string]string {
	if dynamicStrategy == nil {
		return nil
	}

	extInfo := make(map[string]string)
	if appMetaInfo != nil && appMetaInfo.ExtInfo != nil {
		extInfo = appMetaInfo.ExtInfo
	}

	extInfo["ali.AppStage"] = pod.Labels[labelAppStage]
	extInfo["ali.AppUnit"] = pod.Labels[labelAppUnit]
	extInfo["ali.Site"] = pod.Labels[sigmak8sapi.LabelSite]
	extInfo["ali.AppName"] = pod.Labels[sigmak8sapi.LabelAppName]

	var err error
	var ipLabelDynamic string

	// 动态加载iplabel
	if iplabel, _ := dynamicStrategy.ExtRules["IpLabel"]; iplabel != nil {
		var tmpl *template.Template
		if goTemplate, _ := iplabel["GoTemplate"]; goTemplate != "" {
			if tmpl, err = template.New("ipLabel template").Parse(strings.Trim(goTemplate, "\ufeff")); err != nil {
				//log.Errorf("[loadDynamicStrategy] ip label goTemplate parse failed, error:%v, appreq:%v", err, appreq.RequirementId)
			}
		}
		if tmpl != nil {
			var buffer bytes.Buffer
			if err = tmpl.Execute(&buffer, extInfo); err != nil {
				//log.Errorf("[loadDynamicStrategy] goTemplate exec failed, template name:%v, error:%v, appreq:%v", tmpl.Name(), err, appreq.RequirementId)
			} else {
				ipLabelDynamic = buffer.String()
				//log.Errorf("[loadDynamicStrategy] goTemplate exec success, template name:%v, ipLabelDynamic:%s, appreq:%v", tmpl.Name(), ipLabelDynamic, appreq.RequirementId)
			}
		}
	}

	// 动态加载resource pool
	//if customlabel := dynamicStrategy.Constraints["CustomLabels"]; customlabel != nil {
	//	if resourcepool := customlabel["ResourcePool"]; resourcepool != nil {
	//		var tmpl *template.Template
	//		if goTemplate := resourcepool["GoTemplate"]; goTemplate != "" {
	//			//log.Infof("[loadDynamicStrategy] get the resource pool template:%s", goTemplate)
	//			if tmpl, err = template.New("resource pool template").Parse(strings.Trim(goTemplate, "\ufeff")); err != nil {
	//				//log.Errorf("[loadDynamicStrategy] resource pool goTemplate parse failed, error:%v, appreq:%v", err, appreq.RequirementId)
	//				//return nil, err
	//			}
	//		}
	//		if tmpl != nil {
	//			var buffer bytes.Buffer
	//			if err = tmpl.Execute(&buffer, appMetaInfo.ExtInfo); err != nil {
	//				//log.Errorf("[loadDynamicStrategy] goTemplate exec failed, template name:%v, error:%v, apprea:%v", tmpl.Name(), err, appreq.RequirementId)
	//				//return nil, err
	//			} else {
	//				resourcePoolDynamic = buffer.String()
	//				//log.Errorf("[loadDynamicStrategy] goTemplate exec failed, template name:%v, resourcePoolDynamic:%s, apprea:%v", tmpl.Name(), resourcePoolDynamic, appreq.RequirementId)
	//			}
	//		}
	//	}
	//}

	dynStrategyMap := map[string]string{
		"IpLabel": ipLabelDynamic,
		//"ResourcePool": resourcePoolDynamic,
	}
	//glog.V(5).Infof("[newinjection]new loadDynamicStrategyMap for %s/%s get %v", pod.Namespace, pod.Name, dynStrategyMap)
	return dynStrategyMap
}

//check, confirm how to get dynamicStrategyMap,
//func setIpLabel(pod *api.Pod, podRouteRule *sigma2api.RouteRuleDetail, extConfig *extConfig, dynamicStrategyMap map[string]string) {
func setIpLabel(pod *api.Pod, podRouteRule *sigma2api.RouteRuleDetail, extConfig *extConfig) {
	// IpLabel有3个来源，优先级从高往低是：
	// 1. pod labels里直接传入
	// 2. 应用应用规则配置，extconfig配置里的Iplabel
	// 3. 中间件去标，global配置里的route-rules

	if _, ok := pod.Labels[labelPodIpLabel]; ok {
		return
	}

	if extConfig != nil && extConfig.IpLabel !=""{
		pod.Labels[labelPodIpLabel] = extConfig.IpLabel
		return

	}

	if podRouteRule != nil {
		pod.Labels[labelPodIpLabel] = podRouteRule.IpLabel
		return
	}
	// removed from sigma 3.1
	//if ipLabel, ok := dynamicStrategyMap["IpLabel"]; ok {
	//	pod.Labels[labelPodIpLabel] = ipLabel
	//}
}

func setAppStageUnit(pod *api.Pod, podRouteRule *sigma2api.RouteRuleDetail, appUnitStageConstraint *appUnitStageConstraint) {
	// 确定调度匹配的单元、用途标，优先级从高往低：
	// 1. podRouteRule 中间件去标规则中的物理机单元、用途
	// 2. appUnitStageConstraint 配置的用途单元映射
	// 3. label中的单元用途

	// 3. label中的单元用途
	podScheduleUnit := pod.Labels[labelAppUnit]
	podScheduleStage := pod.Labels[labelAppStage]
	// 1. podRouteRule 中间件去标规则中的物理机单元、用途
	if podRouteRule != nil {
		podScheduleStage = podRouteRule.PhyServerEnv
		podScheduleUnit = podRouteRule.PhyServerIdentity

	// 2. appUnitStageConstraint 配置的用途单元映射
	// different from logic in sigma2.0,
	// just a sub-set to run or may not need other rules
	} else if appUnitStageConstraint != nil {
		if podScheduleStage == "DAILY" && slice.ContainsString(appUnitStageConstraint.UnitToCenterForDaily, podScheduleUnit, nil) {
			podScheduleUnit = "CENTER_UNIT.center"
		} else if slice.ContainsString(appUnitStageConstraint.StageToDefault, podScheduleStage, nil) {
			site := pod.Labels[sigmak8sapi.LabelSite]
			if strings.HasSuffix(site, "sqa") || site == "zth" {
				podScheduleStage = "DAILY"
			} else {
				podScheduleStage = "PUBLISH"
			}
		}
	}

	addKVIntoNodeSelectorWithOverwrite(pod, labelAppUnit, podScheduleUnit)
	addKVIntoNodeSelectorWithOverwrite(pod, labelAppStage, podScheduleStage)
}

func setPodScheduleRulesCommon(pod *api.Pod) {
	if site, ok := pod.Labels[sigmak8sapi.LabelSite]; ok {
		addKVIntoNodeSelectorWithOverwrite(pod, sigmak8sapi.LabelSite, site)
	}
}

//func setResourcePool(pod *api.Pod, resourcePoolMapping map[string]string) {
//	affinityRequireNodeSelector := getAffinityRequiredNodeSelector(pod)
//	for i := 0; i < len(affinityRequireNodeSelector.NodeSelectorTerms); i++ {
//		nsTerm := &affinityRequireNodeSelector.NodeSelectorTerms[i]
//		var exists bool
//		var resourcePool string
//		for _, req := range nsTerm.MatchExpressions {
//			if req.Key == labelResourcePool {
//				// 已经传入了资源池，就不做处理
//				exists = true
//				break
//			}
//
//			if rp, ok := resourcePoolMapping[req.Key]; ok {
//				resourcePool = rp
//			}
//
//			if req.Key == "server.owner" {
//				for _, v := range req.Values {
//					if strings.HasPrefix(v, "zeus_lark_") {
//						resourcePool = "lark"
//					}
//				}
//			}
//		}
//
//		if !exists && resourcePool != "" {
//			nsTerm.MatchExpressions = append(nsTerm.MatchExpressions, api.NodeSelectorRequirement{
//				Key:      labelResourcePool,
//				Operator: api.NodeSelectorOpIn,
//				Values:   []string{resourcePool},
//			})
//		}
//	}
//	setAffinityRequiredNodeSelector(pod, affinityRequireNodeSelector)
//}

//check, same to sigma2.0
func setNetPriority(pod *api.Pod, hostConfig *hostConfig) {
	if _, ok := pod.Annotations[sigmak8sapi.AnnotationNetPriority]; ok {
		return
	}

	var netPriority string

	//netPriority的计算
	// http://docs.alibaba-inc.com/pages/viewpage.action?pageId=479572415
	// http://docs.alibaba-inc.com/pages/viewpage.action?pageId=671351156
	if hostConfig != nil && len(hostConfig.NetPriority) > 0 {
		//网络金银铜 {"DEFAULT":"010010", "sigmabosshost":"010001" } 表示sigmabosshost分组用银牌，其它用金牌。
		// 010010 (金牌3) 010001 (银牌5) 010000 (在线铜牌7) 010000 (离线铜牌)
		netPriorityStr := ""
		// first set NetPriority as default
		if appNetPriority, ok := hostConfig.NetPriority["DEFAULT"]; ok {
			netPriorityStr = appNetPriority
		}
		// if the NetPriority of LabelInstanceGroup is seted,
		// set pod NetPriority according to LabelInstanceGroup
		if appNetPriority, ok := hostConfig.NetPriority[pod.Labels[sigmak8sapi.LabelInstanceGroup]]; ok {
			netPriorityStr = appNetPriority
		}
		//在线默认是铜牌
		if netPriorityStr == "010010" {
			netPriority = "3"
		} else if netPriorityStr == "010001" {
			netPriority = "5"
		} else if netPriorityStr == "010000" {
			netPriority = "7"
		} else if netPriorityStr == "010000" {
			//FIXME 这块后面再完善，目前暂时不大好改。。。
		}
	}

	// 3. 优先级的分配： 保留:0-2， 在线业务: 3-7, 离线业务： 8-15
	// http://docs.alibaba-inc.com/pages/viewpage.action?pageId=479572415
	// http://docs.alibaba-inc.com/pages/viewpage.action?pageId=671351156
	if netPriority == "" {
		netPriority = "5"
	}

	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string, 1)
	}
	pod.Annotations[sigmak8sapi.AnnotationNetPriority] = netPriority
}

func setBindsToVolumeAndMounts(pod *api.Pod, hostConfig *hostConfig) {
	if hostConfig == nil || len(hostConfig.Binds) == 0 {
		return
	}

	for _, bind := range hostConfig.Binds {
		words := strings.Split(bind, ":")
		hostPath := words[0]
		containerPath := words[1]
		var readOnly bool
		if len(words) == 3 && words[2] == "ro" {
			readOnly = true
		}

		if findPathInHostVolumes(pod, hostPath) || findPathInContainerMounts(pod, containerPath) {
			continue
		}

		volumeName := "static-strategy-" + hash(bind)[:8]
		pod.Spec.Volumes = append(pod.Spec.Volumes, api.Volume{
			Name: volumeName,
			VolumeSource: api.VolumeSource{
				HostPath: &api.HostPathVolumeSource{
					Path: hostPath,
				},
			},
		})

		newContainers := make([]api.Container, 0, len(pod.Spec.Containers))
		for _, c := range pod.Spec.Containers {
			c.VolumeMounts = append(c.VolumeMounts, api.VolumeMount{
				Name:      volumeName,
				MountPath: containerPath,
				ReadOnly:  readOnly,
			})
			newContainers = append(newContainers, c)
		}
		pod.Spec.Containers = newContainers
	}
}

func hash(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func findPathInHostVolumes(pod *api.Pod, path string) bool {
	for _, v := range pod.Spec.Volumes {
		if v.HostPath != nil && v.HostPath.Path == path {
			return true
		}
	}
	return false
}

func findPathInContainerMounts(pod *api.Pod, path string) bool {
	for _, c := range pod.Spec.Containers {
		for _, m := range c.VolumeMounts {
			if m.MountPath == path {
				return true
			}
		}
	}
	return false
}

func setHostConfigInfo(pod *api.Pod, allocSpec *sigmak8sapi.AllocSpec, hostConfig *hostConfig) {

	mainContainer := getMainContainer(pod)
	allocSpecContainer := getAllocSpecContainer(allocSpec, mainContainer.Name)
	hostConfigInfo := &allocSpecContainer.HostConfig

	if hostConfig != nil {
		// 混部场景下应用的memoryWaterMarkRation
		if hostConfig.MemoryWmarkRatio != 0 {
			hostConfigInfo.MemoryWmarkRatio = hostConfig.MemoryWmarkRatio
		}

		//FIXME sigma3.1 hostConfigInfo do not have Runtime attribution
		//// 容器runtime
		//if hostConfig.Runtime != "" {
		//	hostConfigInfo.Runtime = hostConfig.Runtime
		//	glog.Infof("[mergeHostAndExtConfig] set container runtime to %v", hostConfig.Runtime)
		//}

		//FIXME set GpuSpec

	}

	// 加载intel三级缓存策略
	hostConfigInfo.IntelRdtMba = ""
	hostConfigInfo.IntelRdtGroup = "sigma"



	setAllocSpecContainer(allocSpec, allocSpecContainer)
}

//func setOtherStaticStrategy(pod *api.Pod, labelConfigMap *api.ConfigMap, staticStrategy *sigma2api.AdvancedStrategy) {
func setPrivileged(pod *api.Pod,  hostConfig *hostConfig) {
	if hostConfig == nil {
		return
	}

	mainContainer := getMainContainer(pod)

	//advancedParserConfig := staticStrategy.AdvancedParserConfig
	if hostConfig != nil && hostConfig.Privileged== "true" && mainContainer != nil {
		privileged := true
		if mainContainer.SecurityContext == nil {
			mainContainer.SecurityContext = &api.SecurityContext{
				Privileged: &privileged,
			}
		} else {
			mainContainer.SecurityContext.Privileged = &privileged
		}
	}

	// FIXME: 暂不支持UserDevices，目前似乎也没有应用配置这个规则
	// 挂载的Devices, 挂载的devices必须是严格按照pathOnHost:pathInContainer:Mode的格式
	// 如果上层已经传入devices且不为空，则以上层传入为准，否则以高级策略为准
	//if len(advancedParserConfig.UserDevices) > 0 {
	//
	//}
}

func setConstrains(pod *api.Pod,  constraints *constraints, taintLabels []string) {
	if constraints != nil {
		// mergeConstraintsPlan
		for key, value := range *constraints {
			addKVIntoNodeSelectorNoOverwrite(pod, sigma2ToSigma3Label(nil, key) ,value)
		}
	}

}

func setNetworkMode(pod *api.Pod,  hostConfig *hostConfig, extConfig *extConfig) {
	if hostConfig == nil && extConfig == nil {
		return
	}

	if hostConfig != nil && hostConfig.NetworkMode == "host" {
		if pod.Spec.SecurityContext == nil {
			pod.Spec.SecurityContext = &api.PodSecurityContext{HostNetwork: true}
		} else {
			pod.Spec.SecurityContext.HostNetwork = true
		}
	}
	if extConfig != nil && extConfig.IsHost == "true" {
		addKVIntoNodeSelectorWithOverwrite(pod, "IsHost", "true")

		if hostConfig.NetworkMode != "container" {
			if pod.Spec.SecurityContext == nil {
				pod.Spec.SecurityContext = &api.PodSecurityContext{HostNetwork: true}
			} else {
				pod.Spec.SecurityContext.HostNetwork = true
			}
		}

		//// FIXME: 这个逻辑咋搞？
		//if extConfig.ContainerAsHost {
		//	//glog.Infof("[loadContainerAsHost] get containerAsHost, reqId: %s, containerAsHost: %s", appreq.RequirementId, containerAsHost)
		//	pod.Labels["ali.RegisterContainerAsHost"] = "true"
		//}
	}
}

func setPodAllocSpecAntiAffinityRules(pod *api.Pod, podAllocSpec *sigmak8sapi.AllocSpec, prohibit *prohibit,
	globalScheduleRules *sigma2api.GlobalRules,) {
	podAntiAffinity := getAllocSpecPodAntiAffinity(podAllocSpec)
	nodegroup := pod.Labels[sigmak8sapi.LabelInstanceGroup]

	//// P0M0 后续不再支持
	//if nodeGroupType, ok := p0m0NodegroupMap[nodegroup]; ok {
	//	nodeGroups, maxCount := getP0M0Limit(p0m0NodegroupMap, nodeGroupType)
	//	addPodAppAntiAffinityMatchExpressions(podAntiAffinity, sigmak8sapi.LabelInstanceGroup, nodeGroups, "kubernetes.io/hostname", maxCount, true, 0)
	//	nodeGroups, maxCount = getP0M0Limit(p0m0NodegroupMap, "p0+m0")
	//	addPodAppAntiAffinityMatchExpressions(podAntiAffinity, sigmak8sapi.LabelInstanceGroup, nodeGroups, "kubernetes.io/hostname", maxCount, true, 0)
	//}

	// 多个地方可能有最大单机实例数限制，取最小的值
	var maxInstancePerHost, MaxInstancePerPhyHost = 0, 0

	//// 按照CandidatePlans的顺序，最后一个candidate作为required，其余的作为preferred，且Weight按CandidatePlans的顺序从大到小
	//for i := 0; i < len(staticStrategy.CandidatePlans); i++ {
	//	cp := staticStrategy.CandidatePlans[i]
	//	if cp == nil {
	//		continue
	//	}

		//if i+1 == len(staticStrategy.CandidatePlans) {
			// mergeProhibitPlan
			// 最后一个作为required
			//FIXME 取消candidateplan后，constrains 只有一组， 设为required
			for appName, maxCount := range prohibit.AppConstraints {
				//addPodAppAntiAffinityMatchLabels(podAntiAffinity, sigmak8sapi.LabelAppName, appName, "kubernetes.io/hostname", maxCount, true, 0)
				addPodAppAntiAffinityMatchLabels(podAntiAffinity, sigmak8sapi.LabelAppName, appName, "kubernetes.io/hostname", maxCount, true, 0)
			}

			// mergeSpreadPlan
			// 如果请求已经带了分组最大单机实例数，以请求里的为准
			//if len(nodegroup) > 0 {
			//	if spread.AliMaxInstancePerHost > 0 {
			//		maxInstancePerHost = spread.AliMaxInstancePerHost
			//	}
			//	if spread.AliMaxInstancePerPhyHost > 0{
			//		MaxInstancePerPhyHost = spread.AliMaxInstancePerPhyHost
			//	}

				//maxInstancePerHost, MaxInstancePerPhyHost = spread.AliMaxInstancePerHost, spread.AliMaxInstancePerPhyHost
				//if v, ok := cp.Constraints.CustomLabels["ali.MaxAllocatePercent"]; ok {
				//	tmpMax, err := strconv.Atoi(v)
				//	if err == nil && tmpMax < maxInstancePerHost {
				//		maxInstancePerHost = tmpMax
				//	}
				//}

				//FIXME need MaxInstancePerRack, MaxInstancePerAsw ?
				//if cp.Spread.MaxInstancePerRack > 0 {
				//	addPodAppAntiAffinityMatchLabels(podAntiAffinity, sigmak8sapi.LabelInstanceGroup, nodegroup, "sigma.ali/rack", cp.Spread.MaxInstancePerRack, true, 0)
				//}
				//if cp.Spread.MaxInstancePerAsw > 0 {
				//	addPodAppAntiAffinityMatchLabels(podAntiAffinity, sigmak8sapi.LabelInstanceGroup, nodegroup, "sigma.ali/asw", cp.Spread.MaxInstancePerAsw, true, 0)
				//}
				// FIXME: MaxInstancePerPhyHost and MaxInstancePerFrame
			//}
		//} else {
		//
		//	// 前面的都是preferred
		//	for appName, maxCount := range prohibit.AppConstraints {
		//		// 先YY一下，按照candidate顺序，从80递减至10, 只有一个设为100
		//		weight := 100
		//		//if weight < 10 {
		//		//	weight = 10
		//		//}
		//		addPodAppAntiAffinityMatchLabels(podAntiAffinity, sigmak8sapi.LabelAppName, appName, "kubernetes.io/hostname", maxCount, false, weight)
		//	}
		//}
	//}

	// 公网下沉机器默认单机最大实例数为5
	if pubNetReq := findAffinityRequiredNodeSelectorRequirement(pod, sigmak8sapi.LabelIsPubNetServer); pubNetReq != nil && slice.ContainsString(pubNetReq.Values, "true", nil) {
		if maxInstancePerHost == 0 || maxInstancePerHost > 5 {
			maxInstancePerHost = 5
		}
		if MaxInstancePerPhyHost == 0 || MaxInstancePerPhyHost > 5 {
			MaxInstancePerPhyHost = 5
		}
	}

	// 如果没有最大实例数，且符合这些规则，就设置默认的单机最大实例数规则
	if maxInstancePerHost == 0 && pod.Labels["sigma.ali/upstream-component"] != "smoking" {
		resourcePoolReq := findAffinityRequiredNodeSelectorRequirement(pod, sigmak8sapi.LabelResourcePool)
		appStageReq := findAffinityRequiredNodeSelectorRequirement(pod, labelAppStage)

		// 不在独占列表里
		if !slice.ContainsString(globalScheduleRules.Monopolize.AppConstraints, pod.Labels[sigmak8sapi.LabelAppName], nil) &&
			!slice.ContainsString(globalScheduleRules.Monopolize.DUConstraints, pod.Labels[sigmak8sapi.LabelInstanceGroup], nil) &&
			resourcePoolReq != nil && slice.ContainsString(resourcePoolReq.Values, "sigma_public", nil) &&
			appStageReq != nil && slice.ContainsString(appStageReq.Values, "PUBLISH", nil) {
			maxInstancePerHost = 2
		}
	}

    if maxInstancePerHost > 0 {
        addPodAppAntiAffinityMatchLabels(podAntiAffinity, sigmak8sapi.LabelInstanceGroup, nodegroup, "kubernetes.io/hostname", maxInstancePerHost, true, 0)
        addPodAppAntiAffinityMatchLabels(podAntiAffinity, sigmak8sapi.LabelDeployUnit, nodegroup, "kubernetes.io/hostname", maxInstancePerHost, true, 0)
    }

	setAllocSpecPodAntiAffinity(podAllocSpec, podAntiAffinity)
}

// FIXME : hard-coded
func setPodCPUConfigs(pod *api.Pod, podAllocSpec *sigmak8sapi.AllocSpec, cpuSetModeAdvConfig *cpuSetModeAdvConfig, globalScheduleRules *sigma2api.GlobalRules) {

	mainContainer := getMainContainer(pod)
	cpuCnt := mainContainer.Resources.Requests.Cpu().Value()
	cpuCntStr := fmt.Sprint(cpuCnt)
	allocSpecContainer := getAllocSpecContainer(podAllocSpec, mainContainer.Name)

	// setAppReqCpuSetMode

	if site, ok := pod.Labels[sigmak8sapi.LabelSite]; ok &&
		(site == "zth" || site == "et2" || site == "eu13" || site == "et15" || site == "su18" || site == "na61" || site == "na62") {
		addPodSpecHostFileVolume(pod, "libsysconf-alibaba", "/lib/libsysconf-alibaba.so", api.HostPathFile, []string{"/lib/libsysconf-alibaba.so"}, false)
	}

	addContainerEnvWithOverwrite(mainContainer, "SYSCONF_COMM", "java,uwsgi,processor,getconf,celery,vipsrv-dns,xagent")
	addContainerEnvWithOverwrite(mainContainer, "SIGMA_MAX_PROCESSORS_LIMIT", cpuCntStr)
	addContainerEnvWithOverwrite(mainContainer, "LEGACY_CONTAINER_SIZE_CPU_COUNT", cpuCntStr)
	addContainerEnvWithOverwrite(mainContainer, "AJDK_MAX_PROCESSORS_LIMIT", cpuCntStr)

	addContainerEnvWithOverwrite(mainContainer, "SIGMA_MAX_CPU_QUOTA", fmt.Sprint(cpuCnt*100))
	addContainerEnvWithOverwrite(mainContainer, "SIGMA_CPU_REQUEST", fmt.Sprint(cpuCnt*1000))
	addContainerEnvWithOverwrite(mainContainer, "SIGMA_CPU_LIMIT", fmt.Sprint(cpuCnt*1000))

	cpusetMode := getCpusetMode(pod, cpuSetModeAdvConfig)
	if cpusetMode != "share" {
		allocSpecContainer.Resource.CPU.CPUSet = &sigmak8sapi.CPUSetSpec{
			SpreadStrategy: sigmak8sapi.SpreadStrategySpread,
		}

		//临时方案 对于tair的申请 的这个分组，全部按照samecore的方式玩
		if req := findAffinityRequiredNodeSelectorRequirement(pod, "server.owner"); req != nil &&
			slice.ContainsString(req.Values, "zeus_lark_tair_overlay_7u2", nil) {
			allocSpecContainer.Resource.CPU.CPUSet.SpreadStrategy = sigmak8sapi.SpreadStrategySameCoreFirst
		}
	} else {
		addContainerEnvWithOverwrite(mainContainer, "SIGMA_CPUSHARE", "true")
		addContainerEnvWithOverwrite(mainContainer, "GOMAXPROCS", cpuCntStr)
		addContainerEnvWithOverwrite(mainContainer, "LD_PRELOAD", "/lib/libsysconf-alibaba.so")
		addContainerEnvWithOverwrite(mainContainer, "OPEN_NGINX_CONF_REWRITE", "true")
	}

	// cpu互斥
	// FIXME: 找hongchao确认下
	appName := pod.Labels[sigmak8sapi.LabelAppName]
	//nodeGroup := pod.Labels[sigmak8sapi.LabelInstanceGroup]
	// FIXME: DU级别的互斥没办法写，除非这里能查到nodegroup对应的应用名，才能配置namspace列表
	if globalScheduleRules != nil && slice.ContainsString(globalScheduleRules.CpuSetMutex.AppConstraints, appName, nil) {
		// 加入cpu互斥规则
		cpuAntiAffinity := getAllocSpecCPUAntiAffinity(podAllocSpec)
		cpuAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(cpuAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution, v1.WeightedPodAffinityTerm{
			Weight: 100,
			PodAffinityTerm: v1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      sigmak8sapi.LabelAppName,
							Operator: metav1.LabelSelectorOpIn,
							Values:   globalScheduleRules.CpuSetMutex.AppConstraints,
						},
					},
				},
				Namespaces:  appNamesToNamespaces(globalScheduleRules.CpuSetMutex.AppConstraints),
				TopologyKey: "sigma.ali/physical-core",
			},
		})

		setAllocSpecCPUAntiAffinity(podAllocSpec, cpuAntiAffinity)
	}

	setAllocSpecContainer(podAllocSpec, allocSpecContainer)
}

func setPodTolerateForMandatoryLabels(pod *api.Pod, taintLabels []string) {
	keysNeedApplyTolerate := make(map[string]struct{}, 0)

	requiredNodeSelector := getAffinityRequiredNodeSelector(pod)
	for _, term := range requiredNodeSelector.NodeSelectorTerms {
		for _, req := range term.MatchExpressions {
			if slice.ContainsString(taintLabels, req.Key, nil) {
				keysNeedApplyTolerate[req.Key] = struct{}{}
			}
		}
	}

	preferredTerms := getAffinityPreferredSchedulingTerms(pod)
	for _, term := range preferredTerms {
		for _, req := range term.Preference.MatchExpressions {
			if slice.ContainsString(taintLabels, req.Key, nil) {
				keysNeedApplyTolerate[req.Key] = struct{}{}
			}
		}
	}

	for key := range keysNeedApplyTolerate {
		pod.Spec.Tolerations = append(pod.Spec.Tolerations, api.Toleration{
			Key:      key,
			Operator: api.TolerationOpExists,
			Effect:   api.TaintEffectNoSchedule,
		})
	}
}

func setPodMountsCommon(pod *api.Pod) {
	// 默认关闭serviceaccount
	automountServiceAccountToken := false
	if pod.Spec.AutomountServiceAccountToken == nil {
		pod.Spec.AutomountServiceAccountToken = &automountServiceAccountToken
	}

	addPodSpecEmptyVolume(pod, "vol-sigmalogs", []string{"/var/log/sigma"})
	if !isRebuildPod(pod) {
		addPodSpecHostFileVolume(pod, "route-tmpl", "/opt/ali-iaas/env_create/route.tmpl", api.HostPathFile, []string{"/etc/route.tmpl"}, true)
		addPodSpecEmptyVolume(pod, "cai-alivmcommon", []string{"/home/admin/cai/alivmcommon"})
		addPodSpecEmptyVolume(pod, "tms", []string{"/home/admin/tms"})
		addPodSpecEmptyVolume(pod, "staragent-plugins", []string{"/home/staragent/plugins"})
		addPodSpecEmptyVolume(pod, "snapshots-diamond", []string{"/home/admin/snapshots/diamond"})
		addPodSpecEmptyVolume(pod, "localdatas", []string{"/home/admin/localDatas"})
		addPodSpecEmptyVolume(pod, "vmcommon", []string{"/home/admin/cai/top_foot_vm", "/home/admin/vmcommon"})
	}
}

func addPodSpecHostFileVolume(pod *api.Pod, volumeName, hostPath string, pType api.HostPathType, containerPaths []string, readOnly bool) {
	pod.Spec.Volumes = append(pod.Spec.Volumes, api.Volume{
		Name: volumeName,
		VolumeSource: api.VolumeSource{
			HostPath: &api.HostPathVolumeSource{
				Path: hostPath,
				Type: &pType,
			},
		},
	})
	for i := 0; i < len(pod.Spec.Containers); i++ {
		c := &pod.Spec.Containers[i]
		for _, path := range containerPaths {
			c.VolumeMounts = append(c.VolumeMounts, api.VolumeMount{
				Name:      volumeName,
				MountPath: path,
				ReadOnly:  readOnly,
			})
		}
	}
}

func addPodSpecEmptyVolume(pod *api.Pod, volumeName string, containerPaths []string) {
	pod.Spec.Volumes = append(pod.Spec.Volumes, api.Volume{
		Name: volumeName,
		VolumeSource: api.VolumeSource{
			EmptyDir: &api.EmptyDirVolumeSource{},
		},
	})
	for i := 0; i < len(pod.Spec.Containers); i++ {
		c := &pod.Spec.Containers[i]
		for _, path := range containerPaths {
			c.VolumeMounts = append(c.VolumeMounts, api.VolumeMount{
				Name:      volumeName,
				MountPath: path,
			})
		}
	}
}

func setPodEnvCommon(pod *api.Pod) {
	mainContainer := getMainContainer(pod)
	if mainContainer == nil {
		return
	}

	if sn, ok := pod.Labels[sigmak8sapi.LabelPodSn]; ok {
		addContainerEnvWithOverwrite(mainContainer, "SN", sn)
	}
	addContainerEnvWithOverwrite(mainContainer, "ali_run_mode", "common_vm")
	addContainerEnvNoOverwrite(mainContainer, "ali_admin_uid", "0")
}

func updatePodLabelsCompatible(pod *api.Pod, labelsCompatible map[string]string) {
	for newLabel, oldLabel := range labelsCompatible {
		if v, ok := pod.Labels[newLabel]; ok {
			pod.Labels[oldLabel] = v
		}
	}
}

func setMockRules(pod *api.Pod, appNamingMockRules *appNamingMockRuleDetail) {
	if appNamingMockRules == nil {
		return
	}

	addKVIntoNodeSelectorWithOverwrite(pod, labelAppUnit, appNamingMockRules.PhyServerIdentity)
	addKVIntoNodeSelectorWithOverwrite(pod, labelAppStage, appNamingMockRules.PhyServerEnv)

	if appNamingMockRules.NamingUnit != "" {
		pod.Labels["mock.sigma.alibaba-inc.com/app-unit"] = appNamingMockRules.NamingUnit
	}
	if appNamingMockRules.NamingEnv != "" {
		pod.Labels["mock.sigma.alibaba-inc.com/app-stage"] = appNamingMockRules.NamingEnv
	}
}

//FIXME
// this function is reserve for set EnableOverQuota
//
func checkEnableOverQuota(){
	//check EnableOverQuota is set by normandy

	//check EnableOverQuota is set by atom

	//if not exist, forbid create pod

}


//FIXME implement later
/*
 * 加载亲和策略
 * 亲和策略使用方式： 在sigmaBoss申请新标签： sigmaLabel.ScheduleAffinityLabelsConfigKey，类型是：
 */
//func setAffinityLabels(pod *api.Pod, podAllocSpec *sigmak8sapi.AllocSpec,  extConfig *extConfig) {
//	if extConfig == nil {
//		return
//	}
//	labelsStr := extConfig.AliAcheduleAffinity
//	if labelsStr == "" {
//		return
//	}
//	labels := make(map[string]string)
//	if err := json.Unmarshal([]byte(labelsStr), &labels); err != nil {
//		glog.Errorf("[loadAffinityLabels] the app affinity labels unmarshal error, labels is:%v, errorDetail : %v",
//			labelsStr, err)
//		return
//	}
//
//	if requirement.Affinity.NodeLabels == nil {
//		requirement.Affinity.NodeLabels = make(map[string]string)
//	}
//	for k, v := range labels {
//		requirement.Affinity.NodeLabels[k] = v
//	}
//
//}

func setMonopolize(pod *api.Pod, podAllocSpec  *sigmak8sapi.AllocSpec, monopolize *monopolize){
	if monopolize == nil {
		return
	}
	value := []string{pod.Labels[sigmak8sapi.LabelAppName]}

	cpuAntiAffinity := getAllocSpecCPUAntiAffinity(podAllocSpec)
	podAppAntiAffinity := getAllocSpecPodAntiAffinity(podAllocSpec)

	if monopolize.AppMonopolize == "true" {
		podAppAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(podAppAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, sigmak8sapi.PodAffinityTerm{
			PodAffinityTerm : v1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      sigmak8sapi.LabelAppName,
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   value,
						},
					},
				},
				//Namespaces:  appNamesToNamespaces(globalScheduleRules.CpuSetMutex.AppConstraints),
				TopologyKey: "kubernetes.io/hostname",
			},
		})
		setAllocSpecPodAntiAffinity(podAllocSpec, podAppAntiAffinity)
	}

	if monopolize.CpuSetMonopolize == "true" {
		cpuAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(cpuAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution, v1.WeightedPodAffinityTerm{
			Weight: 1,
			PodAffinityTerm: v1.PodAffinityTerm{
                // cpu anti need only TopologKey
				//LabelSelector: &metav1.LabelSelector{
				//	MatchExpressions: []metav1.LabelSelectorRequirement{
				//		{
				//			Key:      sigmak8sapi.LabelAppName,
				//			Operator: metav1.LabelSelectorOpIn,
				//			Values:   globalScheduleRules.CpuSetMutex.AppConstraints,
				//		},
				//	},
				//},
				//Namespaces:  appNamesToNamespaces(globalScheduleRules.CpuSetMutex.AppConstraints),
				TopologyKey: "sigma.ali/logical-core",
			},
		})
		setAllocSpecCPUAntiAffinity(podAllocSpec, cpuAntiAffinity)
	}
}

func isGray(appName string, grayscaleData map[string]string) bool {
	mode, ok := grayscaleData["mode"]
	if !ok {
		glog.Warningf("[preinjection]new aliPodInjectionPreSchedule can not find mode gray apps in configmap")
	}

	appList, ok := grayscaleData["apps"]
	if !ok {
		glog.Warningf("[preinjection]new aliPodInjectionPreSchedule can not find data gray apps in configmap")
	}

	if mode == "open" {
		return true
	} else if mode == "close" {
		return false
	}

	apps := make([]string, 0)
	json.Unmarshal([]byte(appList), &apps)

	if mode == "include" {
		if slice.ContainsString(apps, appName, nil) {
			return true
		}
	} else if mode == "exclude" {
		if !slice.ContainsString(apps, appName, nil) {
			return true
		}
	} else {
		glog.Warningf("[preinjection]new aliPodInjectionPreSchedule unknow mode in gray configmap")
	}

	return false
}
