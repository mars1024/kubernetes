package apiserver

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-aggregator/pkg/apis/apiregistration"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/multitenancy"
	"gitlab.alipay-inc.com/antcloud-aks/aks-k8s-api/pkg/kube-aggregator/pkg/controllers/autoregister"
)

type CommonAPIServicesGetter func(tenant multitenancy.TenantInfo) []*apiregistration.APIService

var _ autoregister.CommonAPIHandlerManager = &APIAggregator{}

func (s *APIAggregator) AddCommonAPIService(apiService *apiregistration.APIService) error {
	for _, commonAPIService := range s.globalCommonAPIServices {
		if commonAPIService.Spec.Group == apiService.Spec.Group && commonAPIService.Spec.Version == apiService.Spec.Version {
			return nil
		}
	}
	apiService = apiService.DeepCopy()
	s.globalCommonAPIServices = append(s.globalCommonAPIServices, apiService)
	apiregistration.SetAPIServiceCondition(apiService, apiregistration.APIServiceCondition{
		Type:               apiregistration.Available,
		Status:             apiregistration.ConditionTrue,
		LastTransitionTime: metav1.Now(),
	})
	return nil
}

func mergeDuplicatedAPIServices(services1, services2 []*apiregistration.APIService) []*apiregistration.APIService {
	for i, apisvc2 := range services2 {
		found := false
		for _, apisvc1 := range services1 {
			if apisvc1.Spec.Group == apisvc2.Spec.Group && apisvc1.Spec.Version == apisvc2.Spec.Version {
				found = true
			}
		}
		if !found {
			services1 = append(services1, services2[i])
		}
	}
	return services1
}
