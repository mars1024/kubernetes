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

package cluster

type NetworkType string

const (
	VPCNetwork     NetworkType = "VPC"
	ClassicNetwork NetworkType = "CLASSIC"
	VLANNetwork    NetworkType = "VLAN"
)

func (in NetworkType) DeepCopy() (out NetworkType) {
	return in
}

type MinionClusterPhase string

// These are the valid phases of a cluster.
const (
	// ClusterInitializing means the cluster is available for use in the system
	ClusterInitializing MinionClusterPhase = "Initializing"
	// ClusterActive means the cluster is available for use in the system
	ClusterActive MinionClusterPhase = "Active"
	// ClusterTerminating means the cluster is undergoing graceful termination
	ClusterTerminating MinionClusterPhase = "Terminating"
)

func (in MinionClusterPhase) DeepCopy() (out MinionClusterPhase) {
	return in
}

type MinionClusterConditionType string

func (in MinionClusterConditionType) DeepCopy() (out MinionClusterConditionType) {
	return in
}
