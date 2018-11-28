package swarm

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"gitlab.alibaba-inc.com/sigma/sigma-api/sigma"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/json"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/util"
)

// 做一个cache 避免每次都去armory获取
var maps = &sync.Map{}

// 做一个cache 生命周期是每一次跑case
// 机房->真实可调度的机器列表
var schedulerInfoMap = &sync.Map{}

func GetHostPod(hostSn string) map[string]*sigma.AllocPlan {
	if hostSn == "" {
		return nil
	}
	site := ""
	if value, ok := maps.Load(hostSn); !ok {
		nsInfo, err := util.QueryArmory(fmt.Sprintf("sn=='%v'", hostSn))
		if err != nil {
			framework.Logf("[ERR]: QueryArmory return err: %s", err)
			return nil
		}
		if len(nsInfo) != 1 {
			framework.Logf("[ERR]: sn: %s not in armory", hostSn)
			return nil
		}
		maps.Store(hostSn, &nsInfo[0])
		site = strings.ToLower(nsInfo[0].Site)
	} else {
		site = strings.ToLower(value.(*util.ArmoryInfo).Site)
	}

	if schedulerInfo, err := getSchedulerInfo(site); err == nil {
		url := fmt.Sprintf("http://%s:%s/host/%s/pod", schedulerInfo.HostIp, schedulerInfo.Port, hostSn)
		res, err := http.Get(url)
		if err != nil {
			framework.Logf("GetHostPod url:%s, err:%v", url, err)
			return nil
		}

		bodys, _ := ioutil.ReadAll(res.Body)
		if len(bodys) == 0 {
			return nil
		}
		var allocPlans map[string]*sigma.AllocPlan
		err = json.Unmarshal(bodys, &allocPlans)
		framework.Logf("GetHostPod url:%s, hostSn:%s, podSize:%d, err:%v", url, hostSn, len(allocPlans), err)
		if len(allocPlans) > 0 {
			filterAllocPlans := map[string]*sigma.AllocPlan{}
			for key, allocPlan := range allocPlans {
				if allocPlan.CpuQuota > 0 {
					filterAllocPlans[key] = allocPlan
				}
			}
			return filterAllocPlans
		}
		return allocPlans
	} else {
		framework.Logf("query schedulerInfo by site:%s failed", site)
		return nil
	}
}

func getSchedulerInfo(site string) (*schedulerAddressInfo, error) {
	if schedulerInfo, ok := schedulerInfoMap.Load(site); ok {
		if checkHostIsLeader(schedulerInfo.(*schedulerAddressInfo)) {
			return schedulerInfo.(*schedulerAddressInfo), nil
		}
	}

	err := wait.Poll(2*time.Second, 20*time.Second, func() (done bool, err error) {
		schedulerInfos := getSchedulerInfos(site)
		if len(schedulerInfos) == 0 {
			return false, nil
		}
		if len(schedulerInfos) > 1 {
			for _, schedulerInfo := range schedulerInfos {
				if checkHostIsLeader(schedulerInfo) {
					schedulerInfoMap.Store(site, schedulerInfo)
				}
			}
		} else {
			schedulerInfoMap.Store(site, schedulerInfos[0])
		}
		if _, ok := schedulerInfoMap.Load(site); ok {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return nil, err
	}

	schedulerInfo, _ := schedulerInfoMap.Load(site)
	return schedulerInfo.(*schedulerAddressInfo), nil
}

func checkHostIsLeader(schedulerAddressInfo *schedulerAddressInfo) bool {
	c := &http.Client{
		Timeout: 5 * time.Second,
	}
	rep, err := c.Get(fmt.Sprintf("http://%s:%s/leader", schedulerAddressInfo.HostIp, schedulerAddressInfo.Port))
	if err != nil {
		return false
	}
	bytes, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		return false
	}

	return string(bytes) == "true"
}
