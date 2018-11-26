// +build linux

/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cm

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	api "k8s.io/kubernetes/pkg/apis/core"
)

const (
	// configMapCNIServiceAddress config map data key
	configMapNameCustomCgroupParent = "custom-cgroup-parents"
	// configMapNameSpaceOfCNI config map nameSpace
	configMapNamespaceCustomCgroupParent = "kube-system"
	// configMapNameOfCNI config map name
	configMapKeyCustomCgroupParent = "custom-cgroup-parents"
)

// listWatchCustomCgroupParentConfigMap list and watch custom cgroup parents configmap through api server.
func (cm *containerManagerImpl) listWatchCustomCgroupParentConfigMap(c clientset.Interface) {
	glog.V(4).Info("Get custom cgroup parent config map")
	configMapFIFO := cache.NewFIFO(cache.MetaNamespaceKeyFunc)
	fieldSelector := fields.Set{api.ObjectNameField: configMapNameCustomCgroupParent}.AsSelector()

	configMapLW := cache.NewListWatchFromClient(c.CoreV1().RESTClient(), "configmaps",
		configMapNamespaceCustomCgroupParent, fieldSelector)
	r := cache.NewReflector(configMapLW, &v1.ConfigMap{}, configMapFIFO, 0)
	go r.Run(wait.NeverStop)

	popFunc := func() {
		_, err := configMapFIFO.Pop(func(obj interface{}) error {
			configMap, ok := obj.(*v1.ConfigMap)
			if !ok {
				return fmt.Errorf("Failed to convert to v1.ConfigMap: %v", obj)
			}
			// Get customCgroupParents from configmap
			customCgroupParents, err := getCustomCgroupParentsFromConfigmap(configMap)
			if err != nil {
				return err
			}

			glog.V(0).Infof("Update supported custom cgroup parents dir from %v to %v", cm.customCgroupParents, customCgroupParents)

			// Lock containerManagerImpl to update customCgroupParents field
			cm.Lock()
			defer cm.Unlock()

			cm.customCgroupParents = customCgroupParents
			return nil
		})
		if err != nil {
			glog.Errorf("Failed to update costomCgrupParents: %v", err)
		}
	}
	go wait.Forever(popFunc, 0)
}

// getCustomCgroupParentsFromConfigmap get supported custom cgroup parents from configmap.
func getCustomCgroupParentsFromConfigmap(configMap *v1.ConfigMap) ([]string, error) {
	if configMap == nil {
		return []string{}, fmt.Errorf("Invalid: configMap is nil")
	}
	customCgroupParentsStr, exists := configMap.Data[configMapKeyCustomCgroupParent]
	if !exists {
		return []string{}, fmt.Errorf("Failed to get custom cgroup parent from ConfigMap: %v", configMap)
	}
	splitChar := ";"
	customCgroupParents := strings.Split(customCgroupParentsStr, splitChar)
	return customCgroupParents, nil
}