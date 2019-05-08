/*
Copyright 2016 The Kubernetes Authors.

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

package core

import "k8s.io/api/core/v1"

//GetQuotaServiceType returns ServiceType if the service type is eligible to track against a quota, nor return ""
func GetQuotaServiceType(service *v1.Service) v1.ServiceType {
	switch service.Spec.Type {
	case v1.ServiceTypeNodePort:
		return v1.ServiceTypeNodePort
	case v1.ServiceTypeLoadBalancer:
		return v1.ServiceTypeLoadBalancer
	}
	return v1.ServiceType("")
}
