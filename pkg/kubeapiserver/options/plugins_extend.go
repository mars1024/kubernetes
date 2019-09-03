/*
Copyright 2019 The Kubernetes Authors.

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

package options

import (
	"gitlab.alipay-inc.com/antcloud-aks/cafe-cluster-operator/plugin/admission/clusterresourcequota"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/plugin/admission/clusterinjection"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/plugin/admission/objectmetareconcile"
	"k8s.io/apiserver/pkg/admission"
	monotype "k8s.io/kubernetes/plugin/pkg/admission/antcloud/monotype"
	"k8s.io/kubernetes/plugin/pkg/admission/antitamper"
	akspodpostschedule "k8s.io/kubernetes/plugin/pkg/admission/podpostschedule"
	aksprivatecloud "k8s.io/kubernetes/plugin/pkg/admission/privatecloud"
	"k8s.io/kubernetes/plugin/pkg/admission/servicenetallocator"
)

var AllOrderedCafePlugins = []string{
	clusterinjection.PluginName,     // MinionClusterInjection
	akspodpostschedule.PluginName,   // Alipay AntCloud PodPostSchedule
	monotype.PluginName,             // Antcloud monotype mutating plugin
	antitamper.PluginName,           // Anti Tampering of Critical ConfigMaps/Labels/Annotations
	servicenetallocator.PluginName,  // ServiceNetAllocator
	aksprivatecloud.PluginName,      // Private AntCloud
	clusterresourcequota.PluginName, // ClusterResourceQuota
	//imagepullsecret.PluginName,      // CafeImageSecret
	objectmetareconcile.PluginName,  // ObjectMetaReconcile
}

func RegisterCafeAdmissionPlugins(plugins *admission.Plugins) {
	clusterinjection.Register(plugins)
	akspodpostschedule.Register(plugins)
	monotype.Register(plugins)
	servicenetallocator.Register(plugins)
	antitamper.Register(plugins)
	// private antCloud
	aksprivatecloud.Register(plugins)
	clusterresourcequota.Register(plugins)
	//imagepullsecret.Register(plugins)
	objectmetareconcile.Register(plugins)
}
