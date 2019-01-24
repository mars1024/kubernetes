package ant_sigma_bvt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/samalba/dockerclient"
	"k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/sigma/swarm"
	"k8s.io/kubernetes/test/sigma/util"
)

type ArmoryServer struct {
	User string
	Key  string
}

var s *AdapterServer
var a *ArmoryServer

//CheckAdapterParameters() check input parameters.
func CheckAdapterParameters() {
	if util.AlipayCertPath == "" || util.AlipayAdapterAddress == "" || util.ArmoryUser == "" || util.ArmoryKey == "" {
		panic("Load adapter bvt parameters failed, null value is not allowed.")
	}
	s = &AdapterServer{
		AlipayCeritficatePath: util.AlipayCertPath,
		AdapterAddress:        util.AlipayAdapterAddress,
	}
	a = &ArmoryServer{
		User: util.ArmoryUser,
		Key:  util.ArmoryKey,
	}
}

//LoadBaseCreateFile() get base create config for sigma-adapter.
func LoadBaseCreateFile(file string) (*dockerclient.ContainerConfig, error) {
	config := &dockerclient.ContainerConfig{
		HostConfig: dockerclient.HostConfig{},
	}
	content, err := ioutil.ReadFile(file)
	if err != nil {
		framework.Logf("Read sigma2.0 create config failed, path:%v, err: %+v", file, err)
		return nil, err
	}
	err = json.Unmarshal(content, config)
	if err != nil {
		framework.Logf("Unmarshal sigma2.0 content failed, %+v", err)
		return nil, err
	}
	return config, nil
}

//GetPodLists() list pods use label selector.
func GetPodLists(kubeClient clientset.Interface, key, value, ns string) ([]v1.Pod, error) {
	listOptions := metav1.ListOptions{
		LabelSelector:   labels.Set(map[string]string{key: value}).AsSelectorPreValidated().String(),
		ResourceVersion: "0",
	}
	podLists, err := kubeClient.CoreV1().Pods(ns).List(listOptions)
	if err != nil {
		return nil, err
	}
	return podLists.Items, nil
}

//checkPodDelete() check pod is deleted.
func checkPodDelete(kubeClient clientset.Interface, pod *v1.Pod) error {
	timeout := 1 * time.Minute
	t := time.Now()
	for {
		_, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if k8serr.IsNotFound(err) {
			return nil
		}
		if time.Since(t) >= timeout {
			return fmt.Errorf("Gave up waiting for pod %s is removed after %v seconds",
				pod.Name, time.Since(t).Seconds())
		}
		framework.Logf("Retrying to check whether pod %s is removed", pod.Name)
		time.Sleep(5 * time.Second)
	}
}

//GetCreateResultWithTimeOut() get adapter create result, return containerInfo if create succeed.
func GetCreateResultWithTimeOut(client clientset.Interface, requestId string, timeout time.Duration, ns string) (*swarm.AllocResult, error) {
	t := time.Now()
	for {
		task, body, err := s.GetAsyncJson(requestId)
		framework.Logf("Get Async request %v, task:%#v, body:%v, err:%v", requestId, DumpJson(task), body, err)
		if err != nil || body != "" || task == nil {
			return nil, fmt.Errorf("get container result failed.")
		}

		if task.State == "finish" {
			framework.Logf("finish to query sigma 2.0 request id[%s]", requestId)
			for _, ac := range task.Actions {
				framework.Logf("Actions:%#v", *ac)
				if ac.State != "success" {
					continue
				}
				result := &swarm.AllocResult{}
				framework.Logf("requestId: %v action result:%v", requestId, ac.Result)
				err = json.Unmarshal([]byte(ac.Result), result)
				if err != nil {
					return nil, fmt.Errorf("parse 2.0 request action error: %s", err.Error())
				}
				return result, nil
			}
			break
		}
		if time.Since(t) >= timeout {
			pods, err := GetPodLists(client, "ali.RequestId", requestId, ns)
			framework.Logf("Get RequestId:%v pods: %#v, err:%v", requestId, pods, err)
			return nil, fmt.Errorf("timeout for querying the request id[%s]", requestId)
		}
		framework.Logf("retrying to query sigma-adapter request id[%s]...", requestId)
		time.Sleep(10 * time.Second)
	}
	return nil, nil
}

//GetUpgradeResultWithTimeOut() get upgrade result.
func GetUpgradeResultWithTimeOut(requestId string, timeout time.Duration) (bool, error) {
	t := time.Now()
	for {
		task, body, err := s.GetAsyncJson(requestId)
		framework.Logf("Get Async request %v, task:%#v, body:%#v, err:%#v", requestId, DumpJson(task), body, err)
		if err != nil || body != "" || task == nil {
			return false, fmt.Errorf("get container result failed.")
		}

		if task.State == "finish" {
			framework.Logf("finish to query sigma 2.0 request id[%s]", requestId)

			for _, ac := range task.Actions {
				if ac.State != "success" {
					continue
				}
				result := map[string]string{}
				err := json.Unmarshal([]byte(ac.Result), &result)
				if err != nil {
					return false, fmt.Errorf("parse 2.0 request action error: %s", err.Error())
				}
				if result["ErrorMsg"] != "" {
					return false, fmt.Errorf("upgrade pod failed.")
				}
				return true, nil
			}
			break
		}
		if time.Since(t) >= timeout {
			return false, fmt.Errorf("timeout for querying the request id[%s]", requestId)
		}
		framework.Logf("retrying to query sigma-adapter request id[%s]...", requestId)
		time.Sleep(10 * time.Second)
	}
	return false, nil
}

//GetOptionsUseExec() get container exec result.
func GetOptionsUseExec(f *framework.Framework, pod *v1.Pod, cmd []string) (string, string, error) {
	return f.ExecWithOptions(framework.ExecOptions{
		Command:       cmd,
		Namespace:     pod.Namespace,
		PodName:       pod.Name,
		ContainerName: pod.Spec.Containers[0].Name,
		CaptureStdout: true,
		CaptureStderr: true,
	})
}

//CompareMemory() mem compare.
func CompareMemory(mem int64, stdout string) bool {
	var cMem int64
	fields := strings.Split(stdout, "\n")
	for _, field := range fields {
		if strings.Contains(field, "MemTotal") {
			seg := strings.Fields(field)
			if len(seg) != 3 {
				return false
			}
			mem, err := strconv.Atoi(seg[1])
			if err != nil {
				return false
			}
			cMem = int64(mem) * 1024
		}
	}
	if mem == cMem {
		return true
	}
	framework.Logf("Memory doesnot match, mem:%v, cmem:%v", mem, cMem)
	return false
}

//CompareCPU() cpu compare.
func CompareCPU(cpuCount int64, stdout string) bool {
	var cpu int64
	framework.Logf("CPUInfo:%v", stdout)
	fields := strings.Split(stdout, "\n")
	for _, field := range fields {
		if strings.Contains(field, "processor") {
			seg := strings.Fields(field)
			if len(seg) != 3 {
				return false
			}
			cpu += 1
		}
	}
	if cpuCount == cpu {
		return true
	}
	framework.Logf("CPU does not match. cpu:%v, cCpu:%v", cpuCount, cpu)
	return false
}

//CompareDisk() disk compare.
func CompareDisk(diskSize int64, stdout string) bool {
	var disk int64
	fields := strings.Split(stdout, "\n")
	for _, field := range fields {
		if strings.Contains(field, "/") {
			seg := strings.Fields(field)
			if len(seg) != 6 {
				return false
			}
			if seg[5] == "/" {
				disk = Quota2Byte(seg[1])
			}
		}
	}
	if diskSize == disk {
		return true
	}
	framework.Logf("disksize does not match. disk:%v, cdisk:%v", diskSize, disk)
	return false
}

//CompareENV() env compare.
func CompareENV(env []string, stdout string) bool {
	framework.Logf("Env:%v, %v", env, stdout)
	cEnv := strings.Split(stdout, "\n")
	envMap := map[string]string{}
	for _, e := range cEnv {
		envMap[e] = e
	}
	for _, line := range env {
		if _, ok := envMap[line]; !ok {
			return false
		}
	}
	return true
}

//CompareIpAddress() ip compare.
func CompareIpAddress(ip string, stdout string) bool {
	if strings.Contains(stdout, ip) {
		return true
	}
	return false
}

//NewUpgradeConfig() generate upgrade config for sigma2.0
func NewUpgradeConfig(env string) *dockerclient.ContainerConfig {
	upgradeConfig := &dockerclient.ContainerConfig{
		HostConfig: dockerclient.HostConfig{},
		Labels: map[string]string{
			"ali.Async": "true",
		},
		Env: []string{env},
	}
	return upgradeConfig
}
