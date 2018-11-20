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

package autopilot

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/kubelet/apis/cri"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/autopilot/adaptivescale"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/autopilot/throttle"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
	kubepod "k8s.io/kubernetes/pkg/kubelet/pod"
	"k8s.io/kubernetes/pkg/kubelet/server/stats"
	"k8s.io/kubernetes/pkg/kubelet/util"
)

const (
	// nodeAnnotationsFetchRetry specifies how many times kubelet retries when get node annotations failed.
	nodeAnnotationsFetchRetry = 5
)

// autopilotProvider is the provider serves autopilot service.
type autopilotProvider interface {
	KubeletParameter
}

// KubeletParameter is the interface provides kubelet parameter.
type KubeletParameter interface {
	GetNodeInfo() *v1.Node
	GetUpdateInterval() time.Duration
}

// KubeletParam is parameter autopilot service need.
type KubeletParam struct {
	Node           *v1.Node
	UpdateInterval time.Duration
}

// NewKubeletParam returns KubeletParam object.
func NewKubeletParam(Node *v1.Node, UpdateInterval time.Duration) *KubeletParam {
	return &KubeletParam{
		Node:           Node,
		UpdateInterval: UpdateInterval,
	}
}

// GetNodeInfo ruturns Node
func (sp *KubeletParam) GetNodeInfo() *v1.Node {
	return sp.Node
}

// GetUpdateInterval returns nodeStatusUpdateInterval.
func (sp *KubeletParam) GetUpdateInterval() time.Duration {
	return sp.UpdateInterval
}

// AnnotationPara is the parameters for fetching node annotations.
type AnnotationPara struct {
	heartbeatClient v1core.CoreV1Interface
	nodeName        string
}

// NewAnnotationPara returns AnnotationPara object.
func NewAnnotationPara(heartbeatClient v1core.CoreV1Interface, nodeName string) *AnnotationPara {
	return &AnnotationPara{
		heartbeatClient: heartbeatClient,
		nodeName:        nodeName,
	}
}

// AnnotationSyncer syncs the node annotation from master.
type AnnotationSyncer interface {
	Sync() (map[string]string, error)
}

// Sync returns node annotations.
func (ap *AnnotationPara) Sync() (map[string]string, error) {
	if ap.heartbeatClient == nil || ap.nodeName == "" {
		return map[string]string{}, nil
	}
	annotations, err := ap.fetchNodeAnnotations()
	if err != nil {
		return map[string]string{}, fmt.Errorf("Unable to fetch node annotations: %v", err)
	}
	return annotations, nil
}

// fetchNodeAnnotations fetch node annotations from master with retries.
func (ap *AnnotationPara) fetchNodeAnnotations() (map[string]string, error) {
	var err error
	for i := 0; i < nodeAnnotationsFetchRetry; i++ {
		if annotations, err := ap.tryFetchNodeAnnotations(i); err == nil {
			return annotations, nil
		}
		glog.Errorf("Error fetch node annotations, will retry: %v", err)
	}
	return map[string]string{}, fmt.Errorf("fetching node annotations times exceed retry count")
}

// tryFetchNodeAnnotations tries to fetch node annotations from master.
func (ap *AnnotationPara) tryFetchNodeAnnotations(tryNumber int) (map[string]string, error) {
	// In large clusters, GET and PUT operations on Node objects coming
	// from here are the majority of load on apiserver and etcd.
	// To reduce the load on etcd, we are serving GET operations from
	// apiserver cache (the data might be slightly delayed but it doesn't
	// seem to cause more conflict - the delays are pretty small).
	// If it results in a conflict, all retries are served directly from etcd.
	opts := metav1.GetOptions{}
	if tryNumber == 0 {
		util.FromApiserverCache(&opts)
	}
	node, err := ap.heartbeatClient.Nodes().Get(string(ap.nodeName), opts)
	if err != nil {
		return map[string]string{}, fmt.Errorf("error getting node %q: %v", ap.nodeName, err)
	}

	originalNode := node.DeepCopy()
	if originalNode == nil {
		return map[string]string{}, fmt.Errorf("nil %q node object", ap.nodeName)
	}

	if node.ObjectMeta.Annotations != nil {
		return originalNode.ObjectMeta.Annotations, nil
	}

	return map[string]string{}, nil
}

// Service is the autopilpot controller interface
type Service interface {
	// Name returns autopilot controller name.
	Name() string
	// Operate engines the autopilot controller.
	Operate(map[string]string)
	// Recover syncs containers' cgroups from master.
	Recover()
	// Start starts autopilot controller.
	Start(execInterval time.Duration)
	// Stop stops autopilot controller.
	Stop()
	// IsRunning returns autopilot controller status.
	IsRunning() bool
}

// Register registers autopilot controller
func (c *Controllers) Register(name string, service Service) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, ok := c.services[name]
	if ok {
		glog.V(0).Infof("controller %s has been registerd.", name)
	}

	c.services[name] = service
	return nil
}

// Start starts autopilot service.
func (c *Controllers) Start() {
	go wait.Until(func() {
		annotations, err := c.syncer.Sync()
		if err != nil {
			// get node annotation failed.
			glog.Errorf("autopilot service gets node annotation failed: %v", err)
		}

		for _, v := range c.services {
			// autopilot service includes serveral controllers; those controllers operate independently.
			// TODO: it should add a strategy scheduler to decide when to operate which controller.
			v.Operate(annotations)
		}
	}, c.nodeStatusUpdateFrequency, wait.NeverStop)
}

// Controllers provides autopilot service parameters.
type Controllers struct {
	syncer                    AnnotationSyncer
	lock                      sync.Mutex
	services                  map[string]Service
	nodeStatusUpdateFrequency time.Duration
}

// NewControllers returns controllers serve autopilot service.
func NewControllers(
	podManager kubepod.Manager,
	summaryProvider stats.SummaryProvider,
	containerRuntime cri.RuntimeService,
	autopilotProvider autopilotProvider,
	syncer AnnotationSyncer,
	cadvisorClient cadvisor.Interface,
	recorder record.EventRecorder,
	numCores int,
	StatsProvider stats.StatsProvider) (*Controllers, error) {
	node := autopilotProvider.GetNodeInfo()
	nodeUpdateInterval := autopilotProvider.GetUpdateInterval()

	// initialize Adaptivescale Controller
	adaptivescaleController := adaptivescale.NewController(node, podManager, summaryProvider, containerRuntime, false, 10*time.Second)

	// initialize Throttle Controller
	var recoverPriority throttle.ContainerThrottlePriority = new(throttle.DefaultThrottlePriority)
	var throttlePriority throttle.ContainerThrottlePriority = new(throttle.DefaultThrottlePriority)

	var defaultInputData = &throttle.DefaultThrottleInputData{
		StatsClient:    StatsProvider,
		CadvisorClient: cadvisorClient,
	}
	throttleCPUManage := throttle.NewThrottleCPUByLoadManager(
		recorder,
		containerRuntime,
		nodeUpdateInterval,
		numCores,
		defaultInputData,
		recoverPriority,
		throttlePriority,
	)

	c := &Controllers{
		services: make(map[string]Service),
		syncer:   syncer,
		nodeStatusUpdateFrequency: nodeUpdateInterval,
	}

	c.Register(adaptivescaleController.Name(), adaptivescaleController)
	c.Register(throttleCPUManage.Name(), throttleCPUManage)

	return c, nil
}
