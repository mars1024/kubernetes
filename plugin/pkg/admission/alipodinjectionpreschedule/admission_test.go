package alipodinjectionpreschedule

import (
	"encoding/json"
	"reflect"
	"testing"

	sigma2api "gitlab.alibaba-inc.com/sigma/sigma-api/sigma"
	sigmak8sapi "gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "k8s.io/kubernetes/pkg/apis/core"
)

func TestSetAppStageUnit(t *testing.T) {
	pod := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "jiuzhu-test",
			Labels: map[string]string{
				sigmak8sapi.LabelSite: "et2sqa",
				labelAppUnit:          "CENTER_UNIT.unsz",
				labelAppStage:         "PRE_PUBLISH",
			},
		},
		Spec: api.PodSpec{
			Affinity: &api.Affinity{
				NodeAffinity: &api.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &api.NodeSelector{
						NodeSelectorTerms: []api.NodeSelectorTerm{
							{
								MatchExpressions: []api.NodeSelectorRequirement{
									{
										Key:      sigmak8sapi.LabelResourcePool,
										Operator: api.NodeSelectorOpIn,
										Values:   []string{"sigma_public"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	podLabelUnit := pod.Labels[labelAppUnit]
	podLabelStage := pod.Labels[labelAppStage]

	var routeRules sigma2api.RouteRules
	if err := json.Unmarshal([]byte(routeRulesStr), &routeRules); err != nil {
		t.Fatalf("test aliPodInjectionPreSchedule unmarshal route-rules failed: %v", err)
	}

	var podRouteRule *sigma2api.RouteRuleDetail
	for _, r := range routeRules.Rules {
		if r.AppEnv == podLabelStage && r.AppUnit == podLabelUnit {
			t.Logf(">>> in rule: %+v", r)
			podRouteRule = &r
			break
		}
	}
	t.Logf(">>> get rule: %+v", podRouteRule)

	expectPod := pod.DeepCopy()
	expectPod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions = []api.NodeSelectorRequirement{
		{
			Key:      sigmak8sapi.LabelResourcePool,
			Operator: api.NodeSelectorOpIn,
			Values:   []string{"sigma_public"},
		},
		{
			Key:      labelAppUnit,
			Operator: api.NodeSelectorOpIn,
			Values:   []string{"CENTER_UNIT.unsz"},
		},
		{
			Key:      labelAppStage,
			Operator: api.NodeSelectorOpIn,
			Values:   []string{"PRE_PUBLISH"},
		},
	}

	setAppStageUnit(pod, podRouteRule, nil)

	if !reflect.DeepEqual(pod, expectPod) {
		t.Fatalf("failed to test setAppStageUnit, expect: %+v, actually: %+v",
			expectPod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0],
			pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0])
	}
}

var routeRulesStr = `{"UpdateTime":"2018-09-14 16:37:47","Rules":[{"AppUnit":"CENTER_UNIT.center","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.center","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_sg","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_id","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_id","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.eu_us","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.eu_us","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.us","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.us","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.rg_ru","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.rg_ru","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.eu_usdrc","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.eu_usdrc","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.zbyk","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.zbyk","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.paytm","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.paytm","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.zbyk","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.zbyk","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unsz","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.unsz","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.us","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.us","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unzbmix","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.unzbmix","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.rg_ru","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.rg_ru","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_sg","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.eu_us","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.eu_us","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_id","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_id","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_ph","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_ph","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_my","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_my","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_vn","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_vn","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_th","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_th","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.ding","AppEnv":"DAILY","PhyServerIdentity":"CENTER_UNIT.center","PhyServerEnv":"DAILY","IpLabel":""},{"AppUnit":"CENTER_UNIT.recycle","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.center","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.spas","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.center","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.region_zb","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.center","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.region_zb","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.center","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.spas","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.center","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_pk","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_pk","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_bd","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_mm","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_np","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_lk","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_lk","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_np","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_mm","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.lazada_bd","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.lazada_sg","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_vpc_bj","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_vpc_bj","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_vpc_bj","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_vpc_bj","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_vpc_sz","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_vpc_sz","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_vpc_sz","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_vpc_sz","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_vpc_my","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_vpc_my","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_vpc_my","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_vpc_my","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_vpc_id","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_vpc_id","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_vpc_id","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_vpc_id","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_region_vpc_us_west","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_region_vpc_us_west","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_region_vpc_us_west","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_region_vpc_us_west","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_region_vpc_shanghai","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_region_vpc_shanghai","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_region_vpc_shanghai","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_region_vpc_shanghai","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_vpc_de","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_vpc_de","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_vpc_de","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_vpc_de","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_vpc_hk","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_vpc_hk","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_vpc_hk","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_vpc_hk","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_region_vpc_ap_southeast_1","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_region_vpc_ap_southeast_1","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.aliyun_region_vpc_ap_southeast_1","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.aliyun_region_vpc_ap_southeast_1","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unsh","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.unsh","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unsh","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.unsh","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unzbmix","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.unzbmix","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unzbmix","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.unzbmix","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unzbmix25g","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.unzbmix25g","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unzbmix25g","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.unzbmix25g","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unszyun","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.unszyun","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unszyun","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.unszyun","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unshyun","AppEnv":"PRE_PUBLISH","PhyServerIdentity":"CENTER_UNIT.unshyun","PhyServerEnv":"PRE_PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unshyun","AppEnv":"PUBLISH","PhyServerIdentity":"CENTER_UNIT.unshyun","PhyServerEnv":"PUBLISH","IpLabel":""},{"AppUnit":"CENTER_UNIT.unszyun","AppEnv":"SMALLFLOW","PhyServerIdentity":"CENTER_UNIT.unszyun","PhyServerEnv":"SMALLFLOW","IpLabel":""},{"AppUnit":"CENTER_UNIT.unshyun","AppEnv":"SMALLFLOW","PhyServerIdentity":"CENTER_UNIT.unshyun","PhyServerEnv":"SMALLFLOW","IpLabel":""}]}`

func TestUpdatePodLabelsCompatible(t *testing.T) {

	pod := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "jiuzhu-test",
			Labels: map[string]string{
				sigmak8sapi.LabelSite: "et2sqa",
				labelAppUnit:          "CENTER_UNIT.unsz",
				labelAppStage:         "PRE_PUBLISH",
			},
		},
	}

	updatePodLabelsCompatible(pod, map[string]string{
		labelAppUnit:  "ali.AppUnit",
		labelAppStage: "ali.AppStage",
	})

	gotLabels := pod.Labels
	expectedLabels := map[string]string{
		sigmak8sapi.LabelSite: "et2sqa",
		labelAppUnit:          "CENTER_UNIT.unsz",
		labelAppStage:         "PRE_PUBLISH",
		"ali.AppUnit":         "CENTER_UNIT.unsz",
		"ali.AppStage":        "PRE_PUBLISH",
	}

	if !reflect.DeepEqual(gotLabels, expectedLabels) {
		t.Fatalf("expected labels: %#v, got labels: %#v", expectedLabels, gotLabels)
	}

}
