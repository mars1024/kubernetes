package autoregister

import "k8s.io/kube-aggregator/pkg/apis/apiregistration"

type CommonAPIHandlerManager interface {
	AddCommonAPIService(apiService *apiregistration.APIService) error
}
