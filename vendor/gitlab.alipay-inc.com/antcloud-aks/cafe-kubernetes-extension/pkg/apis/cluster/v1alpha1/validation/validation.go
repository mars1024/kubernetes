/*
Copyright 2018 The Alipay.com Inc Authors.

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

package validation

import (
	"net"
	"regexp"

	"k8s.io/apimachinery/pkg/util/validation/field"
	validationutil "k8s.io/apimachinery/pkg/util/validation"
	genericvalidation "k8s.io/apimachinery/pkg/api/validation"

	"gitlab.alipay-inc.com/antcloud-aks/cafe-kubernetes-extension/pkg/apis/cluster"
)

// ValidateMinionClusterName can be used to check whether the given MinionCluster
// name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
const (
	minionClusterNameMaxLen = 63
)

var (
	ValidateMinionClusterNameMsg   = "minion cluster name must consist of alphanumeric characters or '-'"
	ValidateMinionClusterNameRegex = regexp.MustCompile(validMinionClusterNameFmt)
	validMinionClusterNameFmt      = `^[a-zA-Z0-9\-]+$`
)

func ValidateMinionCluster(obj *cluster.MinionCluster) field.ErrorList {
	fldPath := field.NewPath("metadata")
	allErrs := genericvalidation.ValidateObjectMeta(&obj.ObjectMeta, false, ValidateMinionClusterName, fldPath)
	allErrs = append(allErrs, ValidateMinionClusterSpecificLabels(obj.ObjectMeta.Labels, fldPath.Child("labels"))...)
	allErrs = append(allErrs, ValidateMinionClusterSpec(&obj.Spec, field.NewPath("spec"))...)

	return allErrs
}

func ValidateMinionClusterName(name string, prefix bool) (allErrs []string) {
	if !ValidateMinionClusterNameRegex.MatchString(name) {
		allErrs = append(allErrs, validationutil.RegexError(ValidateMinionClusterNameMsg, validMinionClusterNameFmt, "example-com"))
	}
	if len(name) > minionClusterNameMaxLen {
		allErrs = append(allErrs, validationutil.MaxLenError(minionClusterNameMaxLen))
	}
	return allErrs
}

func ValidateMinionClusterSpecificLabels(labels map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if tenantName, ok := labels[cluster.LabelTenantName]; !ok || len(tenantName) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Key(cluster.LabelTenantName), ""))
	}
	if workspaceName, ok := labels[cluster.LabelWorkspaceName]; !ok || len(workspaceName) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Key(cluster.LabelWorkspaceName), ""))
	}
	if clusterName, ok := labels[cluster.LabelClusterName]; !ok || len(clusterName) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Key(cluster.LabelClusterName), ""))
	}

	return allErrs
}

func ValidateMinionClusterSpec(spec *cluster.MinionClusterSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateMinionClusterNetworking(spec.Networking, fldPath.Child("networking"))...)
	return allErrs
}

// Validates the networking configuration
func validateMinionClusterNetworking(networking *cluster.Networking, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	networkType := networking.NetworkType
	if networkType != "" && networkType != cluster.VPCNetwork && networkType != cluster.ClassicNetwork && networkType != cluster.VLANNetwork {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("networkType"), networking.NetworkType, "must be a valid NetworkType"))
	}
	// Validate service ClusterIP range
	if _, _, err := net.ParseCIDR(networking.ServiceClusterIPRange); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("serviceClusterIPRange"), networking.ServiceClusterIPRange, "must be a valid CIDR notation IP range"))
	}
	// Validate service NodePort range
	allErrs = append(allErrs, validateMinionClusterServiceNodePortRange(networking.ServiceNodePortRange, fldPath.Child("serviceNodePortRange"))...)
	// Validate pod IP range
	if networkType == cluster.VPCNetwork {
		if len(networking.PodIPRange) == 0 {
			allErrs = append(allErrs, field.Required(fldPath.Child("podIPRange"), "must specify pod IP range when networkType is VPC"))
		} else {
			for key, value := range networking.PodIPRange {
				if _, _, err := net.ParseCIDR(value); err != nil {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("podIPRange").Key(key), value, "must be a valid CIDR notation IP range"))
				}
			}
		}
	}
	// Validate DNS
	allErrs = append(allErrs, validateMinionClusterDNS(networking.DNS, fldPath.Child("dns"))...)
	// Validate MasterEndpointIP
	if ip := net.ParseIP(networking.MasterEndpointIP); ip == nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("masterEndpointIP"), networking.MasterEndpointIP, "must be a valid IP address"))
	}
	return allErrs
}

func validateMinionClusterServiceNodePortRange(serviceNodePortRange cluster.PortRange, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if serviceNodePortRange.Base < 0 || serviceNodePortRange.Base > 65535 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("base"), serviceNodePortRange.Base, "must be between 0 and 65535, inclusive"))
	}
	if serviceNodePortRange.Base+serviceNodePortRange.Size > 65535 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("size"), serviceNodePortRange.Size, "must be between 0 and 65535, inclusive"))
	}
	return allErrs
}

func validateMinionClusterDNS(dns *cluster.DNS, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if dns != nil {
		if dns.Local != nil && dns.External != nil {
			allErrs = append(allErrs, field.Invalid(fldPath, "resource", "local and external dns can not specified simultaneously"))
		}
		if dns.Local != nil {
			localDNS := dns.Local
			localDNSPath := fldPath.Child("local")
			if len(localDNS.Image) == 0 {
				allErrs = append(allErrs, field.Required(localDNSPath.Child("image"), ""))
			}
			if len(localDNS.ClusterIP) == 0 {
				allErrs = append(allErrs, field.Required(localDNSPath.Child("clusterIP"), ""))
			}
			if isIP := (net.ParseIP(localDNS.ClusterIP) != nil); !isIP {
				allErrs = append(allErrs, field.Invalid(localDNSPath.Child("clusterIP"), localDNS.ClusterIP, "must be a valid IP address"))
			}
		}
	}
	return allErrs
}
