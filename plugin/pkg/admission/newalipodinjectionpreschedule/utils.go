package newalipodinjectionpreschedule

import (
	"encoding/json"
	"strings"

	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "k8s.io/kubernetes/pkg/apis/core"
)

func dumpJson(v interface{}) string {
	str, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

func isRebuildPod(pod *api.Pod) bool {
	if pod.Initializers == nil {
		return false
	}
	for _, initializer := range pod.Initializers.Pending {
		if initializer.Name == "pod-rebuild.sigma.ali" {
			return true
		}
	}
	return false
}

func appNameToNamespace(appName string) string {
	ns := strings.ToLower(appName)
	ns = strings.Replace(ns, "_", "-", -1)
	ns = strings.Replace(ns, ".", "-", -1)
	return ns
}

func appNamesToNamespaces(appNames []string) []string {
	namespaces := make([]string, 0, len(appNames))
	for _, name := range appNames {
		namespaces = append(namespaces, appNameToNamespace(name))
	}
	return namespaces
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

func addContainerEnvNoOverwrite(container *api.Container, key, value string) {
	if container == nil {
		return
	}
	for _, e := range container.Env {
		if e.Name == key {
			return
		}
	}
	container.Env = append(container.Env, api.EnvVar{
		Name:  key,
		Value: value,
	})
}

func addContainerEnvWithOverwrite(container *api.Container, key, value string) {
	if container == nil {
		return
	}
	for _, e := range container.Env {
		if e.Name == key {
			e.Value = value
			break
		}
	}
	container.Env = append(container.Env, api.EnvVar{
		Name:  key,
		Value: value,
	})
}

func getAffinityRequiredNodeSelector(pod *api.Pod) *api.NodeSelector {
	var affinityRequiredNodeSelector *api.NodeSelector
	if pod.Spec.Affinity != nil && pod.Spec.Affinity.NodeAffinity != nil {
		affinityRequiredNodeSelector = pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	}
	if affinityRequiredNodeSelector == nil {
		affinityRequiredNodeSelector = &api.NodeSelector{
			NodeSelectorTerms: []api.NodeSelectorTerm{{}},
		}
	//	if v is nil, len(v) is zero
	} else if len(affinityRequiredNodeSelector.NodeSelectorTerms) == 0 {
		affinityRequiredNodeSelector.NodeSelectorTerms = []api.NodeSelectorTerm{{}}
	}
	return affinityRequiredNodeSelector
}

func getAffinityPreferredSchedulingTerms(pod *api.Pod) []api.PreferredSchedulingTerm {
	if pod.Spec.Affinity != nil && pod.Spec.Affinity.NodeAffinity != nil {
		return pod.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
	}
	return nil
}

func setAffinityRequiredNodeSelector(pod *api.Pod, nodeSelector *api.NodeSelector) {
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &api.Affinity{}
	}
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &api.NodeAffinity{}
	}
	pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = nodeSelector
}

func findAffinityRequiredNodeSelectorRequirement(pod *api.Pod, key string) *api.NodeSelectorRequirement {
	nodeSelector := getAffinityRequiredNodeSelector(pod)
	if nodeSelector != nil {
		for _, term := range nodeSelector.NodeSelectorTerms {
			for _, req := range term.MatchExpressions {
				if req.Key == key {
					return &req
				}
			}
		}
	}
	return nil
}

func addKVIntoNodeSelectorNoOverwrite(pod *api.Pod, key, value string) {
	if pod == nil || len(key) == 0 || len(value) == 0 {
		return
	}
	nodeSelector := getAffinityRequiredNodeSelector(pod)
	for i := 0; i < len(nodeSelector.NodeSelectorTerms); i++ {
		term := &nodeSelector.NodeSelectorTerms[i]
		var found bool
		for _, req := range term.MatchExpressions {
			if req.Key == key {
				found = true
				break
			}
		}
		if !found {
			term.MatchExpressions = append(term.MatchExpressions, api.NodeSelectorRequirement{
				Key:      key,
				Operator: api.NodeSelectorOpIn,
				Values:   []string{value},
			})
		}
	}
	setAffinityRequiredNodeSelector(pod, nodeSelector)
}

func addKVIntoNodeSelectorWithOverwrite(pod *api.Pod, key, value string) {
	if key == "" || value == "" {
		return
	}
	nodeSelector := getAffinityRequiredNodeSelector(pod)
	for i := 0; i < len(nodeSelector.NodeSelectorTerms); i++ {
		term := &nodeSelector.NodeSelectorTerms[i]
		var found bool
		for j := 0; j < len(term.MatchExpressions); j++ {
			req := &term.MatchExpressions[j]
			if req.Key == key {
				found = true
				req.Operator = api.NodeSelectorOpIn
				req.Values = []string{value}
			}
		}
		if !found {
			term.MatchExpressions = append(term.MatchExpressions, api.NodeSelectorRequirement{
				Key:      key,
				Operator: api.NodeSelectorOpIn,
				Values:   []string{value},
			})
		}
	}
	setAffinityRequiredNodeSelector(pod, nodeSelector)
}

func getAllocSpecCPUAntiAffinity(podAllocSpec *sigmak8sapi.AllocSpec) *sigmak8sapi.CPUAntiAffinity {
	var cpuAntiAffinity *sigmak8sapi.CPUAntiAffinity
	if podAllocSpec != nil && podAllocSpec.Affinity != nil {
		cpuAntiAffinity = podAllocSpec.Affinity.CPUAntiAffinity
	}
	if cpuAntiAffinity == nil {
		cpuAntiAffinity = &sigmak8sapi.CPUAntiAffinity{}
	}
	return cpuAntiAffinity
}

func setAllocSpecCPUAntiAffinity(podAllocSpec *sigmak8sapi.AllocSpec, cpuAntiAffinity *sigmak8sapi.CPUAntiAffinity) {
	if podAllocSpec == nil {
		return
	}
	if podAllocSpec.Affinity == nil {
		podAllocSpec.Affinity = &sigmak8sapi.Affinity{}
	}
	podAllocSpec.Affinity.CPUAntiAffinity = cpuAntiAffinity
}

func getAllocSpecPodAntiAffinity(podAllocSpec *sigmak8sapi.AllocSpec) *sigmak8sapi.PodAntiAffinity {
	var podAntiAffinity *sigmak8sapi.PodAntiAffinity
	if podAllocSpec != nil && podAllocSpec.Affinity != nil {
		podAntiAffinity = podAllocSpec.Affinity.PodAntiAffinity
	}
	if podAntiAffinity == nil {
		podAntiAffinity = &sigmak8sapi.PodAntiAffinity{}
	}
	return podAntiAffinity
}

//func getAllocSpecNodeAffinity(podAllocSpec *sigmak8sapi.AllocSpec) *sigmak8sapi.PodAntiAffinity {
//	var podAntiAffinity *sigmak8sapi.PodAffinity
//	if podAllocSpec != nil && podAllocSpec.Affinity != nil {
//		podAntiAffinity = podAllocSpec.Affinity.PodAntiAffinity
//	}
//	if podAntiAffinity == nil {
//		podAntiAffinity = &sigmak8sapi.PodAntiAffinity{}
//	}
//	return podAntiAffinity
//}

func setAllocSpecPodAntiAffinity(podAllocSpec *sigmak8sapi.AllocSpec, podAntiAffinity *sigmak8sapi.PodAntiAffinity) {
	if podAllocSpec == nil {
		return
	}
	if podAllocSpec.Affinity == nil {
		podAllocSpec.Affinity = &sigmak8sapi.Affinity{}
	}
	podAllocSpec.Affinity.PodAntiAffinity = podAntiAffinity
}

func addPodAppAntiAffinityMatchLabels(podAntiAffinity *sigmak8sapi.PodAntiAffinity, key, value string, topologyKey string, maxCount int, isRequired bool, weight int) {
	if podAntiAffinity == nil || len(key) == 0 || len(value) == 0 || maxCount < 0 {
		return
	}

	// 先检查请求里有没有相同的规则，有的话以请求为准
	//hasSameRule := func(podAffinityTerm *v1.PodAffinityTerm) bool {
	//	if podAffinityTerm.LabelSelector != nil && podAffinityTerm.TopologyKey == topologyKey {
	//		for k, v := range podAffinityTerm.LabelSelector.MatchLabels {
	//			if k == key && v == value {
	//				return true
	//			}
	//		}
	//		for _, v := range podAffinityTerm.LabelSelector.MatchExpressions {
	//			if v.Key == key && slice.ContainsString(v.Values, value, nil) {
	//				return true
	//			}
	//		}
	//	}
	//	return false
	//}
	//if isRequired {
	//	for _, podAffinityTerm := range podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
	//		if hasSameRule(&podAffinityTerm.PodAffinityTerm) {
	//			return
	//		}
	//	}
	//} else {
	//	for _, weightedPodAffinityTerm := range podAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
	//		if hasSameRule(&weightedPodAffinityTerm.PodAffinityTerm) {
	//			return
	//		}
	//	}
	//}

	// 请求里没有同类规则，就把规则注入
	newTerm := v1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				key: value,
			},
		},
		TopologyKey: topologyKey,
	}
	if isRequired {
		podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, sigmak8sapi.PodAffinityTerm{
			PodAffinityTerm: newTerm,
			MaxCount:        int64(maxCount),
		})
	} else {
		podAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(podAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution, sigmak8sapi.WeightedPodAffinityTerm{
			WeightedPodAffinityTerm: v1.WeightedPodAffinityTerm{
				PodAffinityTerm: newTerm,
				Weight:          int32(weight),
			},
			MaxCount: int64(maxCount),
		})
	}
}

func addPodAppAntiAffinityMatchExpressions(podAntiAffinity *sigmak8sapi.PodAntiAffinity, key string, values []string, topologyKey string, maxCount int, isRequired bool, weight int) {
	if podAntiAffinity == nil || key == "" || len(values) == 0 {
		return
	}

	newTerm := v1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      key,
					Operator: metav1.LabelSelectorOpIn,
					Values:   values,
				},
			},
		},
		TopologyKey: topologyKey,
	}
	if isRequired {
		podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, sigmak8sapi.PodAffinityTerm{
			PodAffinityTerm: newTerm,
			MaxCount:        int64(maxCount),
		})
	} else {
		podAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(podAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution, sigmak8sapi.WeightedPodAffinityTerm{
			WeightedPodAffinityTerm: v1.WeightedPodAffinityTerm{
				PodAffinityTerm: newTerm,
				Weight:          int32(weight),
			},
			MaxCount: int64(maxCount),
		})
	}
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

func getCpusetMode(pod *api.Pod, cpuSetModeAdvConfig *cpuSetModeAdvConfig) (cpuMode string) {
	if cpuSetModeAdvConfig == nil {
		return
	}

	nodegroup := pod.Labels[sigmak8sapi.LabelInstanceGroup]
	site := pod.Labels[sigmak8sapi.LabelSite]
	unit := pod.Labels[labelAppUnit]

	//优先级排序：（1）单元+机房+分组，（2）机房+分组，（3）分组；
	if cpuSetModeAdvConfig.NodeGroupRules != nil && len(cpuSetModeAdvConfig.NodeGroupRules) > 0 {
		matchConditionsSize := 0
		for i := 0; i < len(cpuSetModeAdvConfig.NodeGroupRules); i++ {
			ruleTmp := cpuSetModeAdvConfig.NodeGroupRules[i]

			if matchConditionsSize < 3 &&
				ruleTmp.NodeGroup != "" && ruleTmp.NodeGroup == nodegroup &&
				ruleTmp.Cell != "" && ruleTmp.Cell == site &&
				ruleTmp.AppUnit != "" && ruleTmp.AppUnit == unit {
				cpuMode = ruleTmp.CpuSetMode
				matchConditionsSize = 3
				break
			}

			if matchConditionsSize < 2 &&
				ruleTmp.NodeGroup != "" && ruleTmp.NodeGroup == nodegroup &&
				ruleTmp.Cell != "" && ruleTmp.Cell == site {
				cpuMode = ruleTmp.CpuSetMode
				matchConditionsSize = 2
				continue
			}

			if matchConditionsSize < 1 &&
				ruleTmp.NodeGroup != "" && ruleTmp.NodeGroup == nodegroup {
				matchConditionsSize = 1
				cpuMode = ruleTmp.CpuSetMode
				continue
			}
		}
	}
	if cpuMode == "" { //看看跟分组的默认策略是否能对的上
		cpuMode = cpuSetModeAdvConfig.AppRule
	}

	if cpuMode == "" {
		cpuMode = "default"
	}

	return
}

func getP0M0Limit(p0m0NodegroupMap map[string]string, ruleType string) ([]string, int) {
	ret := map[string][]string{
		"p0": make([]string, 0),
		"m0": make([]string, 0),
	}
	for nodegroup, t := range p0m0NodegroupMap {
		ret[t] = append(ret[t], nodegroup)
	}

	// FIXME: 这里数字都用的A8机器的P0M0限制，先把su18上线搞定吧。。
	switch ruleType {
	case "p0":
		return ret["p0"], 4
	case "m0":
		return ret["m0"], 2
	case "p0+m0":
		return append(ret["p0"], ret["m0"]...), 4
	}

	return nil, 0
}
