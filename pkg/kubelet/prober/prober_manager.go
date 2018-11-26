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

package prober

import (
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/prober/results"
	"k8s.io/kubernetes/pkg/kubelet/status"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
)

// ProberResults stores the results of a probe as prometheus metrics.
var ProberResults = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: "prober",
		Name:      "probe_result",
		Help:      "The result of a liveness or readiness probe for a container.",
	},
	[]string{"probe_type", "container_name", "pod_name", "namespace", "pod_uid"},
)

// Manager manages pod probing. It creates a probe "worker" for every container that specifies a
// probe (AddPod). The worker periodically probes its assigned container and caches the results. The
// manager use the cached probe results to set the appropriate Ready state in the PodStatus when
// requested (UpdatePodStatus). Updating probe parameters is not currently supported.
// TODO: Move liveness probing out of the runtime, to here.
type Manager interface {
	// AddPod creates new probe workers for every container probe. This should be called for every
	// pod created.
	AddPod(pod *v1.Pod)

	// UpdatePod creates new probe workers and stops the old probe workers if continer's probe is changed.
	UpdatePod(pod *v1.Pod)

	// RemovePod handles cleaning up the removed pod state, including terminating probe workers and
	// deleting cached results.
	RemovePod(pod *v1.Pod)

	// CleanupPods handles cleaning up pods which should no longer be running.
	// It takes a list of "active pods" which should not be cleaned up.
	CleanupPods(activePods []*v1.Pod)

	// UpdatePodStatus modifies the given PodStatus with the appropriate Ready state for each
	// container based on container running status, cached probe results and worker states.
	UpdatePodStatus(types.UID, *v1.PodStatus)

	// Start starts the Manager sync loops.
	Start()
}

type manager struct {
	// Map of active workers for probes
	workers map[probeKey]*worker
	// Lock for accessing & mutating workers
	workerLock sync.RWMutex

	// The statusManager cache provides pod IP and container IDs for probing.
	statusManager status.Manager

	// readinessManager manages the results of readiness probes
	readinessManager results.Manager

	// livenessManager manages the results of liveness probes
	livenessManager results.Manager

	// prober executes the probe actions.
	prober *prober

	// updateReadinessCache will cache the last readiness result when readiness prober is updating.
	updateReadinessCache map[probeKey]results.Result
}

func NewManager(
	statusManager status.Manager,
	livenessManager results.Manager,
	runner kubecontainer.ContainerCommandRunner,
	refManager *kubecontainer.RefManager,
	recorder record.EventRecorder) Manager {

	prober := newProber(runner, refManager, recorder)
	readinessManager := results.NewManager()
	return &manager{
		statusManager:        statusManager,
		prober:               prober,
		readinessManager:     readinessManager,
		livenessManager:      livenessManager,
		workers:              make(map[probeKey]*worker),
		updateReadinessCache: map[probeKey]results.Result{},
	}
}

// Start syncing probe status. This should only be called once.
func (m *manager) Start() {
	// Start syncing readiness.
	go wait.Forever(m.updateReadiness, 0)
}

// Key uniquely identifying container probes
type probeKey struct {
	podUID        types.UID
	containerName string
	probeType     probeType
}

// Type of probe (readiness or liveness)
type probeType int

const (
	liveness  probeType = iota
	readiness
)

// For debugging.
func (t probeType) String() string {
	switch t {
	case readiness:
		return "Readiness"
	case liveness:
		return "Liveness"
	default:
		return "UNKNOWN"
	}
}

func (m *manager) AddPod(pod *v1.Pod) {
	m.workerLock.Lock()
	defer m.workerLock.Unlock()

	key := probeKey{podUID: pod.UID}
	for _, c := range pod.Spec.Containers {
		key.containerName = c.Name

		if c.ReadinessProbe != nil {
			key.probeType = readiness
			if _, ok := m.workers[key]; ok {
				glog.Errorf("Readiness probe already exists! %v - %v",
					format.Pod(pod), c.Name)
				return
			}
			w := newWorker(m, readiness, pod, c)
			m.workers[key] = w
			go w.run()
		}

		if c.LivenessProbe != nil {
			key.probeType = liveness
			if _, ok := m.workers[key]; ok {
				glog.Errorf("Liveness probe already exists! %v - %v",
					format.Pod(pod), c.Name)
				return
			}
			w := newWorker(m, liveness, pod, c)
			m.workers[key] = w
			go w.run()
		}
	}
}

func (m *manager) updateContainer(pod *v1.Pod, container *v1.Container, probe probeType) {
	key := probeKey{podUID: pod.UID, containerName: container.Name}
	probeTypeName := ""

	// Get container's probe information according to probe type.
	containerProbe := &v1.Probe{}
	if probe == liveness {
		containerProbe = container.LivenessProbe
		key.probeType = liveness
		probeTypeName = "liveness prober"
	} else if probe == readiness {
		containerProbe = container.ReadinessProbe
		key.probeType = readiness
		probeTypeName = "readiness prober"
	} else {
		glog.Errorf("Ignore to update container's probe worker because of unknown probe type: %v", probe)
		return
	}

	if containerProbe != nil {
		worker, workerExists := m.workers[key]
		// Case 1: Update worker if probe changes.
		if workerExists && !reflect.DeepEqual(containerProbe, worker.spec) {
			glog.V(0).Infof("Update %s for container %v-%v with new probe: %v",
				probeTypeName, format.Pod(pod), container.Name, containerProbe)
			worker.isCacheResult = true
			worker.stop()

			var defaultTimeoutSeconds int32 = 5

			// Wait for worker to exit.
			// Manager shouldn't be locked by updateContainer or UpdatePod because worker should be removed during wating time.
			timeout := worker.spec.TimeoutSeconds
			if timeout == 0 {
				timeout = defaultTimeoutSeconds
			}
			timerTimeout := time.NewTicker(time.Duration(3*timeout) * time.Second)
			// The TimeoutSeconds such as 20s is too long for checking whether worker is exited or not.
			// The max interval is 5 second.
			interval := worker.spec.TimeoutSeconds
			if interval > defaultTimeoutSeconds || interval == 0 {
				interval = defaultTimeoutSeconds
			}
			timerInterval := time.NewTicker(time.Duration(interval) * time.Second)
		DONE:
			for {
				select {
				case <-timerInterval.C:
					if _, exists := m.workers[key]; !exists {
						break DONE
					}
				case <-timerTimeout.C:
					glog.Errorf("Failed to update %s because of timeout when waiting probe worker to exit: %v",
						probeTypeName, worker)
					return
				}
			}

			w := newWorker(m, probe, pod, *container)
			// Lock manager because we need to add a worker.
			m.workerLock.Lock()
			defer m.workerLock.Unlock()
			m.workers[key] = w
			go w.run()
		}
		// Case 2: Add worker if worker doesn't exists.
		if !workerExists {
			// Lock manager because we need to add a worker.
			m.workerLock.Lock()
			defer m.workerLock.Unlock()
			glog.V(0).Infof("Add %s for container %v-%v with new probe: %v",
				probeTypeName, format.Pod(pod), container.Name, containerProbe)
			w := newWorker(m, probe, pod, *container)
			m.workers[key] = w
			go w.run()
		}
		// Case 3: Keep worker because nothing changes.
		return
	}

	// Case 4: Delete worker if probe doesn't exist in spec.
	if worker, ok := m.workers[key]; ok {
		glog.V(0).Infof("Remove %s for container %v-%v because new probe is nil",
			probeTypeName, format.Pod(pod), container.Name)
		worker.stop()
		return
	}

	// Case 5: There is no probe in spec and workers.
	// Make sure container is not in updateReadinessCache.
	// Case: old worker is deleted, new worker is not created(timeout), user delete probe in spec
	// TODO: Support liveness
	m.workerLock.Lock()
	defer m.workerLock.Unlock()
	switch probe {
	case readiness:
		delete(m.updateReadinessCache, key)
	}
	return
}

func (m *manager) UpdatePod(pod *v1.Pod) {
	if pod.DeletionTimestamp != nil {
		return
	}
	for _, c := range pod.Spec.Containers {
		m.updateContainer(pod, &c, liveness)
		m.updateContainer(pod, &c, readiness)
	}
}

func (m *manager) RemovePod(pod *v1.Pod) {
	m.workerLock.RLock()
	defer m.workerLock.RUnlock()

	key := probeKey{podUID: pod.UID}
	for _, c := range pod.Spec.Containers {
		key.containerName = c.Name
		for _, probeType := range [...]probeType{readiness, liveness} {
			key.probeType = probeType
			if worker, ok := m.workers[key]; ok {
				worker.stop()
			}
		}
	}
}

func (m *manager) CleanupPods(activePods []*v1.Pod) {
	desiredPods := make(map[types.UID]sets.Empty)
	for _, pod := range activePods {
		desiredPods[pod.UID] = sets.Empty{}
	}

	m.workerLock.Lock()
	defer m.workerLock.Unlock()

	for key, worker := range m.workers {
		if _, ok := desiredPods[key.podUID]; !ok {
			worker.stop()
		}
	}

	for key := range m.updateReadinessCache {
		if _, ok := desiredPods[key.podUID]; !ok {
			delete(m.updateReadinessCache, key)
		}
	}
}

func (m *manager) UpdatePodStatus(podUID types.UID, podStatus *v1.PodStatus) {
	for i, c := range podStatus.ContainerStatuses {
		var ready bool
		if c.State.Running == nil {
			ready = false
		} else if result, ok := m.readinessManager.Get(kubecontainer.ParseContainerID(c.ContainerID)); ok {
			ready = result == results.Success
		} else if result, ok := m.getReadinessResultCache(podUID, c.Name); ok {
			ready = result == results.Success
		}else {
			// The check whether there is a probe which hasn't run yet.
			_, exists := m.getWorker(podUID, c.Name, readiness)
			ready = !exists
		}
		podStatus.ContainerStatuses[i].Ready = ready
	}
	// init containers are ready if they have exited with success or if a readiness probe has
	// succeeded.
	for i, c := range podStatus.InitContainerStatuses {
		var ready bool
		if c.State.Terminated != nil && c.State.Terminated.ExitCode == 0 {
			ready = true
		}
		podStatus.InitContainerStatuses[i].Ready = ready
	}
}

func (m *manager) getWorker(podUID types.UID, containerName string, probeType probeType) (*worker, bool) {
	m.workerLock.RLock()
	defer m.workerLock.RUnlock()
	worker, ok := m.workers[probeKey{podUID, containerName, probeType}]
	return worker, ok
}

// Called by the worker after exiting.
func (m *manager) removeWorker(podUID types.UID, containerName string, probeType probeType) {
	m.workerLock.Lock()
	defer m.workerLock.Unlock()
	delete(m.workers, probeKey{podUID, containerName, probeType})
}

// workerCount returns the total number of probe workers. For testing.
func (m *manager) workerCount() int {
	m.workerLock.RLock()
	defer m.workerLock.RUnlock()
	return len(m.workers)
}

func (m *manager) updateReadiness() {
	update := <-m.readinessManager.Updates()

	ready := update.Result == results.Success
	m.statusManager.SetContainerReadiness(update.PodUID, update.ContainerID, ready)
}

func (m *manager) addReadinessResultCache(podUID types.UID, containerName string, result results.Result) {
	m.workerLock.Lock()
	defer m.workerLock.Unlock()
	key := probeKey{podUID, containerName, readiness}
	m.updateReadinessCache[key] = result
}

func (m *manager) removeReadinessResultCache(podUID types.UID, containerName string) {
	m.workerLock.Lock()
	defer m.workerLock.Unlock()
	key := probeKey{podUID, containerName, readiness}
	delete(m.updateReadinessCache, key)
}

func (m *manager) getReadinessResultCache(podUID types.UID, containerName string) (results.Result, bool) {
	m.workerLock.RLock()
	defer m.workerLock.RUnlock()
	key := probeKey{podUID, containerName, readiness}
	if result, exists := m.updateReadinessCache[key]; exists {
		return result, true
	}
	return results.Failure, false
}
