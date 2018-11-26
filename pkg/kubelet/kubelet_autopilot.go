/*
Copyright 2018 The Kubernetes Authors.

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

package kubelet

import (
	"time"

	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/kubelet/apis/cri"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/autopilot"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
	kubepod "k8s.io/kubernetes/pkg/kubelet/pod"
	"k8s.io/kubernetes/pkg/kubelet/server/stats"
)

// StartAutopilotService Starts executing autopilotService.
func StartAutopilotService(podManager kubepod.Manager,
	summaryProvider stats.SummaryProvider,
	containerRuntime cri.RuntimeService,
	Node *v1.Node,
	heartbeatClient v1core.CoreV1Interface,
	UpdateInterval time.Duration,
	cadvisorClient cadvisor.Interface,
	recorder record.EventRecorder,
	numCores int,
	StatsProvider stats.StatsProvider,
) {
	if Node == nil {
		glog.Error("the node is nil when starting autopilot service")
		return
	}

	node := Node.GetName()

	annotationPara := autopilot.NewAnnotationPara(heartbeatClient, node)
	kubeParam := autopilot.NewKubeletParam(Node, UpdateInterval)
	autopilotControllers, err := autopilot.NewControllers(podManager, summaryProvider, containerRuntime, kubeParam, annotationPara, cadvisorClient, recorder, numCores, StatsProvider)
	if err != nil {
		glog.Errorf("create autopilot service controllers failed: %v", err)
		return
	}
	autopilotControllers.Start()
}
