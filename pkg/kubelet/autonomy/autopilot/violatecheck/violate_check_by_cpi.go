package violatecheck

import (
	"encoding/json"
	"sync"
	"time"

	"strconv"
	"sync/atomic"

	"fmt"
	"github.com/golang/glog"
	"gitlab.alibaba-inc.com/sigma/sigma-k8s-api/pkg/api"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/kubelet/autonomy/autopilot/slo"
)

//DefaultViolateCheckManager define a violate check manger.
type DefaultViolateCheckManager struct {
	inputDataForViolate ViolateNeedData
	recorder            record.EventRecorder
	calcPeriod          time.Duration
	startOnce           sync.Once
	runningFlag         int32
	// if violate can work return true, else return false. Default false.
	runStatus bool
	rwmutex   *sync.RWMutex
}

var _ CheckManager = new(DefaultViolateCheckManager)

// NewDefaultViolateCheckManager create default violate check.
func NewDefaultViolateCheckManager(inputData ViolateNeedData, recorder record.EventRecorder,
	calcPeriod time.Duration) *DefaultViolateCheckManager {
	manager := &DefaultViolateCheckManager{
		inputDataForViolate: inputData,
		recorder:            recorder,
		calcPeriod:          calcPeriod,
		runStatus:           false,
		rwmutex:             new(sync.RWMutex),
		runningFlag:         0,
	}
	return manager
}

// ContainerViolateState the core violate check logical.
// if real qps > target_qps then violate report it.
// if real rt  > target_rt then violate report it.
// if real cpi > target_cpi then violate report it.
// together  with qps\rt\cpi and other aspect of violate information, should to be optimized step by step later.
func (cm *DefaultViolateCheckManager) ContainerViolateState(cpi float32, rt float32, qps float32, pod *v1.Pod) ([]slo.ContainerSLOType, error) {
	var vs []slo.ContainerSLOType
	if qps != slo.QPSNULL {
		if targetqps, exist := pod.Annotations[AnnotationQPS]; exist {
			tqps, _ := strconv.ParseFloat(targetqps, 32)
			if qps > float32(tqps) {
				glog.V(2).Infof("QPS overhead Pod name:%v", pod.Name)
				vs = append(vs, slo.QPSViolate)
			}
		}
	}

	if rt != slo.QPSNULL {
		if targetrt, exist := pod.Annotations[AnnotationRT]; exist {
			trt, _ := strconv.ParseFloat(targetrt, 32)
			if rt > float32(trt) {
				glog.V(2).Infof("RT overhead Pod name:%v", pod.Name)
				vs = append(vs, slo.RTViolate)
			}
		}
	}

	if cpi != slo.CPINULL {
		if targetcpi, exist := pod.Annotations[AnnotationCPI]; exist {
			tcpi, _ := strconv.ParseFloat(targetcpi, 32)
			if cpi > float32(tcpi) {
				glog.V(2).Infof("CPI overhead Pod name:%v", pod.Name)
				vs = append(vs, slo.CPIViolate)
			}
		}
	}
	return vs, nil
}

// ViolateCheck the core control logical of violate.
func (cm *DefaultViolateCheckManager) ViolateCheck(annotations map[string]string) ([]string, error) {
	//node.beta1.sigma.ali/autopilot/violate
	if violateJSONStr, exist := annotations[api.AnnotationAutopilot+ViolateKey]; exist {
		var violateConf ViolateConfig
		err := json.Unmarshal([]byte(violateJSONStr), &violateConf)
		if err != nil {
			glog.V(4).Infof("Autopilot throttle config json string format:%v", err)
			return nil, err
		}
		if violateConf.SwitchOn {
			cm.Enable()
			cm.doViolateCheck()
		} else {
			cm.Disable()
		}
	}
	return nil, nil
}

// AutoManager auto control when stop itself under some condition.
// TODO
func (cm *DefaultViolateCheckManager) AutoManager() {

}

// Enable set run status true.
func (cm *DefaultViolateCheckManager) Enable() {
	cm.rwmutex.Lock()
	cm.runStatus = true
	cm.rwmutex.Unlock()
	glog.V(4).Info("Stop violate check Manager...")
}

// Disable set run status false.
func (cm *DefaultViolateCheckManager) Disable() {
	cm.rwmutex.Lock()
	cm.runStatus = false
	cm.rwmutex.Unlock()
	glog.V(4).Info("Stop violate check manager...")
}

// ViolateCheck default impl violate check.
func (cm *DefaultViolateCheckManager) doViolateCheck() ([]string, error) {

	if !cm.runStatus {
		return nil, nil
	}

	if cm.runningFlag != 0 {
		glog.V(4).Infof("Last time throttle has not finished yet! Pass out this currency.")
		return nil, nil
	}
	atomic.AddInt32(&cm.runningFlag, 1)

	podStats, err := cm.inputDataForViolate.ListPodStats()
	if err != nil {
		glog.V(4).Infof("get pod stats failed. %v", err)
		atomic.AddInt32(&cm.runningFlag, -1)
		return nil, err
	}
	var results []string
	for _, podStat := range podStats {
		cs := podStat.Containers
		for _, container := range cs {
			cn := container.Name
			pns := podStat.PodRef.Namespace
			pn := podStat.PodRef.Name
			cpi, ecpi := cm.inputDataForViolate.GetContainerRuntimeSLOValue(slo.CPIViolate, pns, pn, cn)
			if ecpi != nil {
				glog.V(2).Infof("namespace:%s,podname:%s,containername:%s CPI failed %v", pns, pn, cn, ecpi)
				cpi = slo.CPINULL
			}

			rt, ert := cm.inputDataForViolate.GetContainerRuntimeSLOValue(slo.RTViolate, pns, pn, cn)
			if ert != nil {
				glog.V(2).Infof("namespace:%s,podname:%s,containername:%s RT failed %v", pns, pn, cn, ert)
				rt = slo.RTNULL
			}

			qps, eqps := cm.inputDataForViolate.GetContainerRuntimeSLOValue(slo.QPSViolate, pns, pn, cn)
			if eqps != nil {
				glog.V(2).Infof("namespace:%s,podname:%s,containername:%s QPS failed %v", pns, pn, cn, eqps)
				rt = slo.QPSNULL
			}

			pod, exist := cm.inputDataForViolate.GetPodByName(pns, pn)
			if !exist {
				glog.V(2).Infof("namespace:%s,podname:%s not exist %v", pns, pn, eqps)
				continue
			}

			vtype, err := cm.ContainerViolateState(cpi, rt, qps, pod)
			if err != nil {
				glog.V(2).Infof("container violate state computer failed %v", err)
				continue
			}
			if len(vtype) > 0 {
				results = append(results, fmt.Sprintf("%s,%s,%s", pns, pn, cn))
				cm.recorder.Eventf(pod, v1.EventTypeWarning, slo.ArrayToString(vtype), "NameSpace:%v,PodName:%s,ContainerName:%v exceed the security value.", pns, pn, cn)
			}
		}
	}
	atomic.AddInt32(&cm.runningFlag, -1)
	return results, nil
}

// StartDirectly background to check container realtime CPI whether violate the target value from PodSpec.
// This entry point is only used under DefaultViolateCheckManager itself control the work.
// If there is no value in PodSpec. Use the default experience value as the threshold value.
// If ContainerCPI > PodSpec.CPI, report to center.
func (cm *DefaultViolateCheckManager) StartDirectly() {
	cm.startOnce.Do(func() {
		if cm.calcPeriod <= 0 {
			glog.V(4).Infof("ViolateCheckManager disabled")
			return
		}
		// 5 minute just a experience value.
		if cm.calcPeriod > time.Minute*5 {
			cm.calcPeriod = time.Minute * 5
		}

		glog.V(4).Infof("Starting violate check every :%v for ever", cm.calcPeriod)
		go wait.Forever(func() { cm.doViolateCheck() }, cm.calcPeriod)
	})
}
