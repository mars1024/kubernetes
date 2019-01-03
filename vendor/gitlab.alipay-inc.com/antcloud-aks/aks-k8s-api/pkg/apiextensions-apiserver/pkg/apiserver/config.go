package apiserver

import (
	"k8s.io/apiextensions-apiserver/pkg/apiserver"
	"k8s.io/apimachinery/pkg/version"
	genericapiserver "k8s.io/apiserver/pkg/server"
	apiextensionsapiserver "k8s.io/apiextensions-apiserver/pkg/apiserver"
)

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func CompleteConfig(cfg *apiserver.Config) CompletedConfig {
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}

	c.GenericConfig.EnableDiscovery = false
	c.GenericConfig.Version = &version.Info{
		Major: "0",
		Minor: "1",
	}

	return CompletedConfig{&c}
}

func CreateAPIExtensionsServer(apiextensionsConfig *apiextensionsapiserver.Config, delegateAPIServer genericapiserver.DelegationTarget) (*CustomResourceDefinitions, error) {
	return CompleteConfig(apiextensionsConfig).New(delegateAPIServer)
}