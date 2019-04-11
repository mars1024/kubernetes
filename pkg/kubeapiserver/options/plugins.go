/*
Copyright 2014 The Kubernetes Authors.

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

// This file exists to force the desired plugin implementations to be linked.
// This should probably be part of some configuration fed into the build for a
// given binary target.
import (
	// Cloud providers
	_ "k8s.io/kubernetes/pkg/cloudprovider/providers"

	// Admission policies
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/plugin/initialization"
	"k8s.io/apiserver/pkg/admission/plugin/namespace/lifecycle"
	mutatingwebhook "k8s.io/apiserver/pkg/admission/plugin/webhook/mutating"
	validatingwebhook "k8s.io/apiserver/pkg/admission/plugin/webhook/validating"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/plugin/pkg/admission/admit"
	"k8s.io/kubernetes/plugin/pkg/admission/alipodinjectionpostschedule"
	"k8s.io/kubernetes/plugin/pkg/admission/alipodinjectionpreschedule"
	"k8s.io/kubernetes/plugin/pkg/admission/alipodlifecyclehook"
	"k8s.io/kubernetes/plugin/pkg/admission/alwayspullimages"
	"k8s.io/kubernetes/plugin/pkg/admission/antiaffinity"
	"k8s.io/kubernetes/plugin/pkg/admission/armory"
	"k8s.io/kubernetes/plugin/pkg/admission/ase"
	"k8s.io/kubernetes/plugin/pkg/admission/auditdelete"
	"k8s.io/kubernetes/plugin/pkg/admission/containerstate"
	"k8s.io/kubernetes/plugin/pkg/admission/defaulttolerationseconds"
	"k8s.io/kubernetes/plugin/pkg/admission/deny"
	"k8s.io/kubernetes/plugin/pkg/admission/eventratelimit"
	"k8s.io/kubernetes/plugin/pkg/admission/exec"
	"k8s.io/kubernetes/plugin/pkg/admission/extendedresourcetoleration"
	"k8s.io/kubernetes/plugin/pkg/admission/gc"
	"k8s.io/kubernetes/plugin/pkg/admission/imagepolicy"
	"k8s.io/kubernetes/plugin/pkg/admission/limitranger"
	"k8s.io/kubernetes/plugin/pkg/admission/namespace/autoprovision"
	"k8s.io/kubernetes/plugin/pkg/admission/namespace/exists"
	"k8s.io/kubernetes/plugin/pkg/admission/networkstatus"
	"k8s.io/kubernetes/plugin/pkg/admission/newalipodinjectionpreschedule"
	"k8s.io/kubernetes/plugin/pkg/admission/noderestriction"
	"k8s.io/kubernetes/plugin/pkg/admission/nodetaint"
	"k8s.io/kubernetes/plugin/pkg/admission/poddeletionflowcontrol"
	"k8s.io/kubernetes/plugin/pkg/admission/podnodeselector"
	"k8s.io/kubernetes/plugin/pkg/admission/podpreset"
	"k8s.io/kubernetes/plugin/pkg/admission/podtolerationrestriction"
	podpriority "k8s.io/kubernetes/plugin/pkg/admission/priority"
	"k8s.io/kubernetes/plugin/pkg/admission/resourcequota"
	"k8s.io/kubernetes/plugin/pkg/admission/security/podsecuritypolicy"
	"k8s.io/kubernetes/plugin/pkg/admission/securitycontext/scdeny"
	"k8s.io/kubernetes/plugin/pkg/admission/serviceaccount"
	"k8s.io/kubernetes/plugin/pkg/admission/sigmascheduling"
	"k8s.io/kubernetes/plugin/pkg/admission/storage/persistentvolume/label"
	"k8s.io/kubernetes/plugin/pkg/admission/storage/persistentvolume/resize"
	"k8s.io/kubernetes/plugin/pkg/admission/storage/storageclass/setdefault"
	"k8s.io/kubernetes/plugin/pkg/admission/storage/storageobjectinuseprotection"

	alipayappcert "k8s.io/kubernetes/plugin/pkg/admission/alipay/appcert"
	alipaycmdb "k8s.io/kubernetes/plugin/pkg/admission/alipay/cmdb"
	alipaycpushare "k8s.io/kubernetes/plugin/pkg/admission/alipay/cpushare"
	alipayimagepullsecret "k8s.io/kubernetes/plugin/pkg/admission/alipay/imagepullsecret"
	alipayinclusterkube "k8s.io/kubernetes/plugin/pkg/admission/alipay/inclusterkube"
	alipaypodlocation "k8s.io/kubernetes/plugin/pkg/admission/alipay/podlocation"
	alipaypodpreset "k8s.io/kubernetes/plugin/pkg/admission/alipay/podpreset"
	alipayreadinessgate "k8s.io/kubernetes/plugin/pkg/admission/alipay/readinessgate"
	alipayresource "k8s.io/kubernetes/plugin/pkg/admission/alipay/resource"
	alipayresourcemutationbe "k8s.io/kubernetes/plugin/pkg/admission/alipay/resourcemutationbe"
	alipayresourcemutationqos "k8s.io/kubernetes/plugin/pkg/admission/alipay/resourcemutationqos"
	alipayrouter "k8s.io/kubernetes/plugin/pkg/admission/alipay/router"
	alipaysetdefault "k8s.io/kubernetes/plugin/pkg/admission/alipay/setdefault"
	alipaysetdefaulthostconfig "k8s.io/kubernetes/plugin/pkg/admission/alipay/setdefaulthostconfig"
	alipaysidecar "k8s.io/kubernetes/plugin/pkg/admission/alipay/sidecar"
	alipayzappinfo "k8s.io/kubernetes/plugin/pkg/admission/alipay/zappinfo"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/plugin/admission/clusterinjection"
	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/plugin/admission/objectmetareconcile"
	akspodpostschedule "k8s.io/kubernetes/plugin/pkg/admission/podpostschedule"
	"k8s.io/kubernetes/plugin/pkg/admission/servicenetallocator"
)

// AllOrderedPlugins is the list of all the plugins in order.
var AllOrderedPlugins = []string{
	admit.PluginName,                        // AlwaysAdmit
	autoprovision.PluginName,                // NamespaceAutoProvision
	lifecycle.PluginName,                    // NamespaceLifecycle
	exists.PluginName,                       // NamespaceExists
	scdeny.PluginName,                       // SecurityContextDeny
	antiaffinity.PluginName,                 // LimitPodHardAntiAffinityTopology
	podpreset.PluginName,                    // PodPreset
	limitranger.PluginName,                  // LimitRanger
	serviceaccount.PluginName,               // ServiceAccount
	noderestriction.PluginName,              // NodeRestriction
	nodetaint.PluginName,                    // TaintNodesByCondition
	alwayspullimages.PluginName,             // AlwaysPullImages
	imagepolicy.PluginName,                  // ImagePolicyWebhook
	podsecuritypolicy.PluginName,            // PodSecurityPolicy
	podnodeselector.PluginName,              // PodNodeSelector
	podpriority.PluginName,                  // Priority
	defaulttolerationseconds.PluginName,     // DefaultTolerationSeconds
	podtolerationrestriction.PluginName,     // PodTolerationRestriction
	exec.DenyEscalatingExec,                 // DenyEscalatingExec
	exec.DenyExecOnPrivileged,               // DenyExecOnPrivileged
	eventratelimit.PluginName,               // EventRateLimit
	extendedresourcetoleration.PluginName,   // ExtendedResourceToleration
	label.PluginName,                        // PersistentVolumeLabel
	setdefault.PluginName,                   // DefaultStorageClass
	storageobjectinuseprotection.PluginName, // StorageObjectInUseProtection
	gc.PluginName,                           // OwnerReferencesPermissionEnforcement
	resize.PluginName,                       // PersistentVolumeClaimResize
	mutatingwebhook.PluginName,              // MutatingAdmissionWebhook
	initialization.PluginName,               // Initializers
	validatingwebhook.PluginName,            // ValidatingAdmissionWebhook
	resourcequota.PluginName,                // ResourceQuota
	deny.PluginName,                         // AlwaysDeny
	auditdelete.PluginName,                  // NamespaceDelete

	alipayappcert.PluginName,                 // Alipay app cert
	alipaysidecar.PluginName,                 // Alipay Sidecar
	armory.PluginName,                        // Armory
	containerstate.PluginName,                // ContainerState
	networkstatus.PluginName,                 // NetworkStatus
	alipodlifecyclehook.PluginName,           // AliPodLifeTimeHook
	alipodinjectionpreschedule.PluginName,    // AliPodInjectionPreSchedule
	newalipodinjectionpreschedule.PluginName, // AliPodInjectionPreSchedule
	// sigmascheduling must admit after alipodinjectionpreschedule
	sigmascheduling.PluginName,             // SigmaScheduling
	alipodinjectionpostschedule.PluginName, // AliPodInjectionPostSchedule
	poddeletionflowcontrol.PluginName,      // PodDeletionFlowControl

	alipayresourcemutationqos.PluginName,  // Alipay resource mutation admission for burstable
	alipaysetdefault.PluginName,           // Alipay SetDefault
	alipaycmdb.PluginName,                 // Alipay CMDB
	alipayinclusterkube.PluginName,        // Alipay in-cluster kubernetes service
	alipaypodlocation.PluginName,          // Alipay PodLocation
	alipaypodpreset.PluginName,            // Alipay PodPreset
	alipayzappinfo.PluginName,             // Alipay ZAppInfo
	alipayresource.PluginName,             // Alipay resource validation admission
	alipaycpushare.PluginName,             // Alipay cpushare injection admission
	alipayimagepullsecret.PluginName,      // Alipay image pull secret injection admission
	alipaysetdefaulthostconfig.PluginName, // Alipay SetDefaultHostConfig
	alipayresourcemutationbe.PluginName,   // Alipay resource mutation admission for best effort
	alipayrouter.PluginName,               // Alipay router mutation admission for container router configuration.
	alipayreadinessgate.PluginName,        // Alipay readiness fate mutation admission for pod condition.

	clusterinjection.PluginName,    // MinionClusterInjection
	akspodpostschedule.PluginName,  // Alipay AntCloud PodPostSchedule
	ase.PluginName,                 // ASE
	servicenetallocator.PluginName, // ServiceNetAllocator
	objectmetareconcile.PluginName, // ObjectMetaReconcile
}

// RegisterAllAdmissionPlugins registers all admission plugins and
// sets the recommended plugins order.
func RegisterAllAdmissionPlugins(plugins *admission.Plugins) {
	admit.Register(plugins) // DEPRECATED as no real meaning
	alwayspullimages.Register(plugins)
	antiaffinity.Register(plugins)
	defaulttolerationseconds.Register(plugins)
	deny.Register(plugins) // DEPRECATED as no real meaning
	eventratelimit.Register(plugins)
	exec.Register(plugins)
	extendedresourcetoleration.Register(plugins)
	gc.Register(plugins)
	imagepolicy.Register(plugins)
	limitranger.Register(plugins)
	autoprovision.Register(plugins)
	exists.Register(plugins)
	noderestriction.Register(plugins)
	nodetaint.Register(plugins)
	label.Register(plugins) // DEPRECATED in favor of NewPersistentVolumeLabelController in CCM
	podnodeselector.Register(plugins)
	podpreset.Register(plugins)
	podtolerationrestriction.Register(plugins)
	resourcequota.Register(plugins)
	podsecuritypolicy.Register(plugins)
	podpriority.Register(plugins)
	scdeny.Register(plugins)
	serviceaccount.Register(plugins)
	setdefault.Register(plugins)
	resize.Register(plugins)
	storageobjectinuseprotection.Register(plugins)

	auditdelete.Register(plugins)

	armory.Register(plugins)
	containerstate.Register(plugins)
	networkstatus.Register(plugins)
	alipodlifecyclehook.Register(plugins)
	alipodinjectionpreschedule.Register(plugins)
	newalipodinjectionpreschedule.Register(plugins)
	sigmascheduling.Register(plugins)
	alipodinjectionpostschedule.Register(plugins)
	poddeletionflowcontrol.Register(plugins)

	alipaysidecar.Register(plugins)
	alipayresourcemutationqos.Register(plugins)
	alipaysetdefault.Register(plugins)
	alipaycmdb.Register(plugins)
	alipayinclusterkube.Register(plugins)
	alipaypodlocation.Register(plugins)
	alipaypodpreset.Register(plugins)
	alipayzappinfo.Register(plugins)
	alipayimagepullsecret.Register(plugins)
	alipayresource.Register(plugins)
	alipaycpushare.Register(plugins)
	alipayappcert.Register(plugins)
	alipaysetdefaulthostconfig.Register(plugins)
	alipayresourcemutationbe.Register(plugins)
	alipayrouter.Register(plugins)
	alipayreadinessgate.Register(plugins)

	clusterinjection.Register(plugins)
	akspodpostschedule.Register(plugins)
	ase.Register(plugins)
	// alwasy keep it the last one in the list
	objectmetareconcile.Register(plugins)
}

// DefaultOffAdmissionPlugins get admission plugins off by default for kube-apiserver.
func DefaultOffAdmissionPlugins() sets.String {
	defaultOnPlugins := sets.NewString(
		lifecycle.PluginName,                //NamespaceLifecycle
		limitranger.PluginName,              //LimitRanger
		serviceaccount.PluginName,           //ServiceAccount
		setdefault.PluginName,               //DefaultStorageClass
		resize.PluginName,                   //PersistentVolumeClaimResize
		defaulttolerationseconds.PluginName, //DefaultTolerationSeconds
		mutatingwebhook.PluginName,          //MutatingAdmissionWebhook
		validatingwebhook.PluginName,        //ValidatingAdmissionWebhook
		resourcequota.PluginName,            //ResourceQuota
	)

	if utilfeature.DefaultFeatureGate.Enabled(features.PodPriority) {
		defaultOnPlugins.Insert(podpriority.PluginName) //PodPriority
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.TaintNodesByCondition) {
		defaultOnPlugins.Insert(nodetaint.PluginName) //TaintNodesByCondition
	}

	return sets.NewString(AllOrderedPlugins...).Difference(defaultOnPlugins)
}
