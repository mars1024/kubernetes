package util

import (
	"encoding/json"
	"io/ioutil"

	"k8s.io/api/core/v1"
)

// LoadConfigMapFromFile create a configmap object from file
func LoadConfigMapFromFile(file string) (*v1.ConfigMap, error) {
	fileContent, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var configmap *v1.ConfigMap
	err = json.Unmarshal(fileContent, &configmap)
	if err != nil {
		return nil, err
	}
	return configmap, nil
}
