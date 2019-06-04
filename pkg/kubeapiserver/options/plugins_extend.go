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
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/plugin/pkg/admission/antitamper"
	"k8s.io/kubernetes/plugin/pkg/admission/ase"
	"k8s.io/kubernetes/plugin/pkg/admission/servicenetallocator"
	akspodpostschedule "k8s.io/kubernetes/plugin/pkg/admission/podpostschedule"
	aksprivatecloud "k8s.io/kubernetes/plugin/pkg/admission/privatecloud"
	monotype "k8s.io/kubernetes/plugin/pkg/admission/antcloud/monotype"
	capinjection "k8s.io/kubernetes/plugin/pkg/admission/antcloud"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/plugin/admission/clusterinjection"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/plugin/admission/objectmetareconcile"
)

var AllOrderedCafePlugins = []string{
	clusterinjection.PluginName,    // MinionClusterInjection
	akspodpostschedule.PluginName,  // Alipay AntCloud PodPostSchedule
	monotype.PluginName,            // Antcloud monotype mutating plugin
	capinjection.PluginName,        // Antcloud CapInjection mutating plugin
	antitamper.PluginName,          // Anti Tampering of Critical ConfigMaps/Labels/Annotations
	ase.PluginName,                 // ASE
	servicenetallocator.PluginName, // ServiceNetAllocator
	aksprivatecloud.PluginName,     // Private AntCloud
	objectmetareconcile.PluginName, // ObjectMetaReconcile
}

func RegisterCafeAdmissionPlugins(plugins *admission.Plugins) {
	clusterinjection.Register(plugins)
	akspodpostschedule.Register(plugins)
	monotype.Register(plugins)
	capinjection.Register(plugins)
	servicenetallocator.Register(plugins)
	antitamper.Register(plugins)
	ase.Register(plugins)
	// private antCloud
	aksprivatecloud.Register(plugins)
	objectmetareconcile.Register(plugins)
}
