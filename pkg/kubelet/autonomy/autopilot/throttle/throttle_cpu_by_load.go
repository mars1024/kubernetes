package throttle

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"sync/atomic"

	"github.com/golang/glog"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	v1Type "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/kubelet/apis/cri"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/json"
)

const throttleKey = "/throttle"

// VCPU96StabiltiyNodeLoad 96 VCPU stability load 120,120 is just guess value. currently no experience value for it.
var VCPU96StabiltiyNodeLoad int32 = 120

// VCPU64StabilityNodeLoad 64 VCPU stability load 80, 80 is experience value.
var VCPU64StabilityNodeLoad int32 = 80

// VCPU32StabilityNodeLoad 32 VCPU stability load 50, 50 is experience value.
var VCPU32StabilityNodeLoad int32 = 50

// VCPU24StabilityNodeLoad 24 VCPU stabitliy load 40, 40 is experience value.
var VCPU24StabilityNodeLoad int32 = 40

// DefaultCPULimit used only if container without setting request value from PodSpec.
var DefaultCPULimit int64 = 1024

// CPUByLoadThrottleManager logical as:
// dockerService--> ContainerManager { runtimeService,cgroupManager,cpuManager } --> CgroupManger -->
// ThrottleCPUByLoadInterface throttle cpu by load PR url : https://lark.alipay.com/sae/slave/throttle.
// ThrottleCPUByLoadInterface to throttle sub-cgroup's cfs quota to scale down expected node load.
// This feature default is not enable.
type CPUByLoadThrottleManager struct {
	containerManager  cri.ContainerManager
	recorder          record.EventRecorder
	calcPeriod        time.Duration
	startOnce         sync.Once
	vcpuNum           int
	stabilityNodeLoad int32
	// inputData is fully data dependency object.
	inputData InputData
	// if throttle working return true, else return false. Default false.
	runStatus         bool
	rwmutex           *sync.RWMutex
	recoverPriority   ContainerThrottlePriority
	throttledPriority ContainerThrottlePriority
	runningFlag       int32
}

// NewThrottleCPUByLoadManager new a throttleCPUByLoadManager.
func NewThrottleCPUByLoadManager(recorder record.EventRecorder,
	containerManager cri.ContainerManager, calcPeriod time.Duration, vcpuNum int,
	inputData InputData,
	recoverPriority ContainerThrottlePriority,
	throttlePriority ContainerThrottlePriority) *CPUByLoadThrottleManager {
	manager := &CPUByLoadThrottleManager{
		containerManager:  containerManager,
		recorder:          recorder,
		calcPeriod:        calcPeriod,
		vcpuNum:           vcpuNum,
		runStatus:         false,
		inputData:         inputData,
		recoverPriority:   recoverPriority,
		throttledPriority: throttlePriority,
		rwmutex:           new(sync.RWMutex),
		runningFlag:       0,
	}
	if manager.calcPeriod > 5*time.Minute {
		manager.calcPeriod = 5 * time.Minute
	}
	//TODO optimize the experience after online,but first we should have a value to start.
	if 96 >= manager.vcpuNum && manager.vcpuNum > 64 { // (64,96] vcpu, the experience stability nc load is 120.
		manager.stabilityNodeLoad = VCPU96StabiltiyNodeLoad
	} else if 64 >= manager.vcpuNum && manager.vcpuNum > 32 { // (32,64] vcpu, the experience stability nc load is 80.
		manager.stabilityNodeLoad = VCPU64StabilityNodeLoad
	} else if 24 < manager.vcpuNum && manager.vcpuNum <= 32 { // (24,32] vcpu, the experience stability nc load is 50.
		manager.stabilityNodeLoad = VCPU32StabilityNodeLoad
	} else { // <=24 vcpu, the experience stability nc load is 40.
		manager.stabilityNodeLoad = VCPU24StabilityNodeLoad
	}

	return manager
}

var _ Manager = new(CPUByLoadThrottleManager)

// Name returns autopilot controller name.
func (m *CPUByLoadThrottleManager) Name() string {
	return "ThrottleCPU"
}

// Recover syncs containers cgroups from master.
func (m *CPUByLoadThrottleManager) Recover() {
	return
}

// Operate runs the AutopilotService object.
func (m *CPUByLoadThrottleManager) Operate(annotations map[string]string) {
	m.CheckStart(annotations)
	return
}

// Start to throttle function.
func (m *CPUByLoadThrottleManager) Start(executionIntervalSeconds time.Duration) {
	m.startOnce.Do(func() {
		if executionIntervalSeconds <= 0 {
			glog.V(4).Info("throttle cpu by load disabled.")
			return
		}

		m.runStatus = true
		m.calcPeriod = executionIntervalSeconds
		glog.V(4).Infof("Starting throttle cpu by load every:%v for ever.", executionIntervalSeconds)
		// currently run period , an optimize method is auto-check under some conditions.
		// for example: load pressure too lower,then with a large period for next time check.
		go wait.Forever(func() { m.doPodCPUThrottleService() }, m.calcPeriod)
	})
	return
}

// Stop sets the runStatus to false.
func (m *CPUByLoadThrottleManager) Stop() {
	m.Disable()
	return
}

// IsRunning returns runStatus.
func (m *CPUByLoadThrottleManager) IsRunning() bool {
	return m.runStatus
}

// Exec is the executor for adjusting resource.
func (m *CPUByLoadThrottleManager) Exec() error {
	return nil
}

// CheckStart only for global switch controller the throttle service.
func (m *CPUByLoadThrottleManager) CheckStart(annotations map[string]string) {
	//throttle key = node.beta1.sigma.ali/autopilot/throttle
	if throttleJSONStr, exist := annotations[api.AnnotationAutopilot+throttleKey]; exist {
		var throttleConf SwitchConfig
		err := json.Unmarshal([]byte(throttleJSONStr), &throttleConf)
		if err != nil {
			glog.V(4).Infof("Autopilot throttle config json string format:%v", err)
			return
		}
		if throttleConf.SwitchOn {
			m.Enable()
			m.doPodCPUThrottleService()
		} else {
			m.Disable()
		}
	}
}

// ThrottleStatus GetStatus return throttle working status.
func (m *CPUByLoadThrottleManager) ThrottleStatus() bool {
	m.rwmutex.RLock()
	stats := m.runStatus
	m.rwmutex.RUnlock()
	return stats
}

// Enable Soft controller. Trigger start throttle stop working.
func (m *CPUByLoadThrottleManager) Enable() {
	m.rwmutex.Lock()
	m.runStatus = true
	m.rwmutex.Unlock()
	glog.V(4).Info("Stop throttleCPUByLoadManager...")
}

// Disable Soft controller. Trigger stop throttle stop working.
func (m *CPUByLoadThrottleManager) Disable() {
	m.rwmutex.Lock()
	m.runStatus = false
	m.rwmutex.Unlock()
	glog.V(4).Info("Stop throttleCPUByLoadManager...")
}

// doPodCPUThrottleService core flow for throttle.
// only throttle one container to spec quota or recover at one period time.
// currently only throttle cpu by load.
func (m *CPUByLoadThrottleManager) doPodCPUThrottleService() {
	//check this throttle work need to run or not.
	if !m.runStatus {
		return
	}
	if m.runningFlag != 0 {
		glog.V(4).Infof("Last time throttle has not finished yet! Pass out this currency.")
		return
	}
	atomic.AddInt32(&m.runningFlag, 1)

	existWarn := m.CheckLoadWarn()
	// node without load over-running,so to recover one container's cpu quota for throttled before.
	if !existWarn {
		cgroupName, err := m.SelectThrottledContainerToRecover(m.inputData, m.recoverPriority)
		if err != nil {
			glog.Errorf("select the already throttled container error:%v", err)
		}
		m.ThrottleCgroupToSpecQuota(cgroupName, 0)
		m.recorder.Eventf(m, v1Type.EventTypeNormal, "RecoverCPU", "cgroupName:%s throttleCPUByLoad happened", cgroupName)
		atomic.AddInt32(&m.runningFlag, -1)
		return
	}
	// node load over-running happened,so to throttle the biggest over-ration container 's cpu quota to spec value.
	cgroupName, specQuota, err := m.SelectTheContainerToThrottle(m.inputData, m.throttledPriority)
	if err != nil {
		glog.Errorf("select the biggest load container error:%v", err)
		atomic.AddInt32(&m.runningFlag, -1)
		return
	}
	m.ThrottleCgroupToSpecQuota(cgroupName, specQuota)
	m.recorder.Eventf(m, v1Type.EventTypeWarning, "ThrottleCPU", "cgroupName:%s throttleCPUByLoad happened", cgroupName)
	atomic.AddInt32(&m.runningFlag, -1)
}

// GetObjectKind just return emptyObjectKind.
func (m *CPUByLoadThrottleManager) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

// DeepCopyObject implement.
func (m *CPUByLoadThrottleManager) DeepCopyObject() runtime.Object {
	if c := m.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy may need to optimize.
func (m *CPUByLoadThrottleManager) DeepCopy() *CPUByLoadThrottleManager {
	if m != nil {
		dst := new(CPUByLoadThrottleManager)
		dst.runStatus = m.runStatus
		dst.vcpuNum = m.vcpuNum
		return dst
	}
	return nil
}

// SelectThrottledContainerToRecover recover already throttled container's cpu quota to zero.
// use case.
// 很多应用比如Java应用，GC的时候都会有很短的时间cpu上去，有可能碰到整机load高的话，永远被throttle了，也不好。
// 考虑例如整机load下来了 AND Pod本身的usage 也降下来了.
// 遍历
func (m *CPUByLoadThrottleManager) SelectThrottledContainerToRecover(inputData InputData, recoverPriority ContainerThrottlePriority) (string, error) {

	containerIDS, err := m.filterContainersWouldBeRecover()
	if err != nil {
		return "", err
	}
	//fmt.Printf("containerIDS :%v\n",containerIDS)
	var params []*ParamRef
	for _, ids := range containerIDS {
		params = append(params, &ParamRef{Name: ids})
	}
	result, err := recoverPriority.SelectCouldThrottleContainer(m.inputData, params)
	return result.Name, err
}

// filterPodsWouldRecover any container's cpu quota > 0 will be selected.
func (m *CPUByLoadThrottleManager) filterContainersWouldBeRecover() ([]string, error) {
	var containerIDS []string
	podStats, err := m.inputData.ListPodStats()
	if err != nil {
		return containerIDS, err
	}
	alreadyCheckContainers := make(map[string]bool)

	option := cadvisorapiv2.RequestOptions{
		IdType:    cadvisorapiv2.TypeName,
		Count:     2, // 2 samples are needed to compute "instantaneous" CPU
		Recursive: true,
	}

	for _, podState := range podStats {
		podName := podState.PodRef.Name
		podNameSpace := podState.PodRef.Namespace
		pod, exist := m.inputData.GetPodByName(podNameSpace, podName)
		if !exist {
			glog.Errorf("Can not find the pod:%s of namespace:%s ", podName, podNameSpace)
			continue
		}
		containerStatuses := pod.Status.ContainerStatuses

		for _, containerStatus := range containerStatuses {
			containerName := containerStatus.Name
			key := fmt.Sprintf("%s-%s", podName, containerName)
			if _, exist := alreadyCheckContainers[key]; exist {
				continue
			}

			containerSpec, err := m.inputData.ContainerSpec(containerName, option)
			if err != nil || len(containerSpec) == 0 {
				msg := fmt.Sprintf("container: %s not found in pod: %s", containerName, podName)
				return containerIDS, errors.New(msg)
			}

			for _, container := range containerSpec {
				// default there is not setting the cfs quota,
				// if only happened load over running,that means cgroup's cpu.quota
				// already be setted one value large than 0.
				alreadyCheckContainers[key] = true
				if container.Cpu.Quota > 0 {
					cid := &kubecontainer.ContainerID{}
					err := cid.ParseString(containerStatus.ContainerID)
					if err != nil {
						glog.V(4).Infof("Parse containerID  %v", err)
						continue
					}
					containerIDS = append(containerIDS, cid.ID)
				}
			}
		}
	}
	return containerIDS, nil
}

// SelectTheContainerToThrottle return the biggest load container's cgroup name.
// TODO currently just select the biggest over ration of Load of container.
// need to optimize with weight and priority and so on.
func (m *CPUByLoadThrottleManager) SelectTheContainerToThrottle(inputData InputData, throttlePriority ContainerThrottlePriority) (string, int64, error) {

	containerIDSParams, err := m.filterContainersWouldBeThrottled()
	if err != nil {
		return "", 0, err
	}
	if len(containerIDSParams) < 1 {
		return "", 0, nil
	}
	result, err := throttlePriority.SelectCouldThrottleContainer(m.inputData, containerIDSParams)
	return result.Name, result.CurrentValues.(int64), err
}

func (m *CPUByLoadThrottleManager) filterContainersWouldBeThrottled() ([]*ParamRef, error) {
	var params []*ParamRef
	var cpuSpec int64
	podStats, err := m.inputData.ListPodStats()
	if err != nil {
		return nil, err
	}

	// 遍历所有pod的所有container, 找出container avgload与cpu Spec request 比值最大的containerID.
	for _, podState := range podStats {
		podName := podState.PodRef.Name
		podNameSpace := podState.PodRef.Namespace
		pod, exist := m.inputData.GetPodByName(podNameSpace, podName)
		if !exist {
			glog.Errorf("Can not find the pod:%s of namespace:%s ", podName, podNameSpace)
			continue
		}

		containerStatuses := pod.Status.ContainerStatuses
		//fmt.Printf("containerStatuses:%v\n", containerStatuses)
		for _, containerStatus := range containerStatuses {
			containerName := containerStatus.Name
			stat, err := getContainerLastestStats(m.inputData, containerName, false)
			if err != nil {
				glog.Errorf("GetCgroupStatats of containerName:%s %v", containerName, err)
				break
			}
			if stat == nil {
				glog.Errorf("container:%s stat is nil", containerName)
				break
			}
			//cpuShare request value,'limit' is the cfs share.
			cpuSpec, exist = getContainerCPUSpec(containerName, pod)
			if !exist {
				cpuSpec = DefaultCPULimit
			}

			if cpuSpec == 0 {
				cpuSpec = DefaultCPULimit
			}

			loadValue := stat.Cpu.LoadAverage
			//compute the over ratio of this container.
			if float64(loadValue) > float64(cpuSpec) {
				paramRef := ParamRef{
					Name:          containerStatus.ContainerID,
					SpecValues:    cpuSpec,
					CurrentValues: loadValue,
				}
				params = append(params, &paramRef)
			}
		}
	}
	return params, nil
}

// ThrottleCgroupToSpecQuota update container's cpu quota under assumption: init container without set cfs quota.
// As current logical implement, just set the cgroup's cpu quota to the podSpec request.
func (m *CPUByLoadThrottleManager) ThrottleCgroupToSpecQuota(containerID string, specQuota int64) error {
	resources := &runtimeapi.LinuxContainerResources{
		CpuQuota: specQuota,
	}
	return m.containerManager.UpdateContainerResources(containerID, resources)
}

// CheckLoadWarn check node load whether too large or not.
// if more than stabilityLoad value,return true,else false.
// TODO make more effective，it should take care of history data.
func (m *CPUByLoadThrottleManager) CheckLoadWarn() bool {
	loadValue, err := m.GetNodeLoad()
	if err != nil {
		return false
	}
	return loadValue >= m.stabilityNodeLoad
}

// GetNodeLoad return node current load value.
// Currently load just define a float64 object without giving a common struct such as.
// type struct {
//    one float64
//    five float64
//    fifteen float64
//    timeStamp   time
// }
func (m *CPUByLoadThrottleManager) GetNodeLoad() (int32, error) {
	machineStats, err := getContainerInfoV2(m.inputData, "/", true)
	if err != nil {
		glog.V(4).Infof("Get machine stats by ContainerInfoV2(/,...) %v", err)
	}
	return machineStats.Stats[0].Cpu.LoadAverage, nil
}

// getContainerLastestStats returns the latest stats of the container having the
// specified containerName from cadvisor.
func getContainerLastestStats(input InputData, containerName string, updateStats bool) (*cadvisorapiv2.ContainerStats, error) {
	info, err := getContainerInfoV2(input, containerName, updateStats)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	stats, found := latestContainerStats(info)
	if !found {
		return nil, fmt.Errorf("failed to get latest stats from container info for %q", containerName)
	}
	return stats, nil
}

// latestContainerStats returns the latest container stats from cadvisor, or nil if none exist
func latestContainerStats(info *cadvisorapiv2.ContainerInfo) (*cadvisorapiv2.ContainerStats, bool) {
	stats := info.Stats
	if len(stats) < 1 {
		return nil, false
	}
	latest := stats[len(stats)-1]
	if latest == nil {
		return nil, false
	}
	return latest, true
}

// getContainerInfoV2 returns the information of the container with the specified
// containerName from cadvisor.
func getContainerInfoV2(input InputData, containerName string, updateStats bool) (*cadvisorapiv2.ContainerInfo, error) {
	var maxAge *time.Duration
	if updateStats {
		age := 0 * time.Second
		maxAge = &age
	}
	infoMap, err := input.ContainerInfoV2(containerName, cadvisorapiv2.RequestOptions{
		IdType:    cadvisorapiv2.TypeName,
		Count:     2, // 2 samples are needed to compute "instantaneous" CPU
		Recursive: false,
		MaxAge:    maxAge,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get container info for %q: %v", containerName, err)
	}

	if infoMap == nil {
		return nil, nil
	}

	if len(infoMap) != 1 {
		return nil, fmt.Errorf("unexpected number of containers: %v", len(infoMap))
	}

	info := infoMap[containerName]
	return &info, nil
}

// getCPUSpec return container cpu request of pod with containerName.
func getContainerCPUSpec(containerName string, pod *v1Type.Pod) (int64, bool) {
	for _, container := range pod.Spec.Containers {
		// return the define container name cpu request
		if container.Name == containerName && container.Resources.Requests != nil {
			return container.Resources.Requests.Cpu().AsInt64()
		}
	}
	return 0, false
}
