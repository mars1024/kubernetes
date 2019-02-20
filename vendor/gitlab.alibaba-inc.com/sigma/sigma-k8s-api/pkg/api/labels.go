/*
Copyright 2017 The Kubernetes Authors.

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

package api

const (
	// serial tag of pod, should be unique globally
	LabelPodSn = "sigma.ali/sn"

	// ip allocated for pod
	LabelPodIp = "sigma.ali/ip"

	// application name
	LabelAppName = "sigma.ali/app-name"

	// group of pod in cmdb like armory
	LabelInstanceGroup = "sigma.ali/instance-group"

	// application deploy unit, equal to app name + env in ant sigma.
	LabelDeployUnit = "sigma.ali/deploy-unit"

	// site of pod
	LabelSite = "sigma.ali/site"

	// Physical core as the topology key.
	// It is used to spread cpuset assignment across physical cores.
	TopologyKeyPhysicalCore = "sigma.ali/physical-core"

	// Logical core as the topology key.
	// It is used to unshare a single logical core in over-quota case.
	TopologyKeyLogicalCore = "sigma.ali/logical-core"

	// pod container mode, e.g. dockervm/pod/...
	LabelPodContainerModel = "sigma.ali/container-model"

	// qos of pod. label value is defined in qosclass
	LabelPodQOSClass = "sigma.ali/qos"

	// if true, user can modify /etc/hosts, /etc/resolv.conf and /etc/hostname in container .
	LabelHostDNS = "ali.host.dns"

	// the type of container.
	LabelServerType = "com.alipay.acs.container.server_type"
)
