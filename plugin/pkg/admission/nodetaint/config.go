package nodetaint

import (
	"io"

	"k8s.io/apimachinery/pkg/util/yaml"
	api "k8s.io/kubernetes/pkg/apis/core"
)

// AdmissionConfig holds config data for admission controllers
type AdmissionConfig struct {
	Taints []api.Taint
}

// LoadConfiguration loads the provided configuration.
func loadConfiguration(config io.Reader) (*AdmissionConfig, error) {
	// if no config is provided, return a default configuration
	var admissionConfig AdmissionConfig
	if config == nil {
		return &admissionConfig, nil
	}

	// we have a config so parse it.
	d := yaml.NewYAMLOrJSONDecoder(config, 4096)
	if err := d.Decode(&admissionConfig); err != nil {
		return nil, err
	}

	return &admissionConfig, nil
}
