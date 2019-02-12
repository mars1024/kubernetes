package vipserverutil

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/kubernetes/test/e2e/framework"
)

var (
	client = &http.Client{Timeout: time.Second * 5}
)

type Backends struct {
	IPs []IPInfo `json:"ips"`
}
type IPInfo struct {
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	Weight  int    `json:"weight"`
	Cluster string `json:"cluster"`
}

func GetBackends(domain string) (Backends, error) {
	var backends Backends

	baseUrl := "http://jmenv.tbsite.net:8080/vipserver/serverlist"
	var data string
	var err error
	for i := 1; i <= 5; i++ {
		_, data, err = doRequest(baseUrl, "")
		if err == nil {
			break
		}
		framework.Logf("get server ips failed will retry in one second, err: %v", err)
		time.Sleep(time.Second)
	}
	if err != nil {
		return backends, err
	}
	serverIPs := strings.Split(strings.TrimSpace(string(data)), "\n")
	if serverIPs[0] == "" {
		return backends, fmt.Errorf("get vipserver ip failed")
	}

	for i := 1; i <= 5; i++ {
		serverIP := serverIPs[rand.Intn(len(serverIPs))]
		framework.Logf("get vipserver ip %v", serverIP)
		queryUrl := fmt.Sprintf("http://%v/vipserver/api/ip4Dom?dom=%v&redirect=0", serverIP, domain)
		_, data, err = doRequest(queryUrl, "")
		if err == nil {
			break
		}
		framework.Logf("get backend failed will retry in one second, err: %v", err)
		time.Sleep(time.Second)
	}
	if err != nil {
		return backends, err
	}

	if err := json.Unmarshal([]byte(data), &backends); err != nil {
		framework.Logf("get vipserver response %v", data)
		return backends, nil
	}
	return backends, nil
}

func RemoveDomain(domain string) {
	baseUrl := "http://jmenv.tbsite.net:8080/vipserver/serverlist"
	_, data, err := doRequest(baseUrl, "")
	if err != nil {
		return
	}
	serverIPs := strings.Split(strings.TrimSpace(string(data)), "\n")
	if serverIPs[0] == "" {
		return
	}
	deleteUrl := fmt.Sprintf("http://%v/vipserver/api/remvDom?dom=%v&token=8630s", serverIPs[0], domain)
	doRequest(deleteUrl, "")
}

func doRequest(url, reqBody string) (code int, respBody string, err error) {
	var req *http.Request
	if reqBody == "" {
		req, err = http.NewRequest("GET", url, nil)
	} else {
		req, err = http.NewRequest("POST", url, strings.NewReader(reqBody))
	}
	if err != nil {
		return 0, "", fmt.Errorf("create request failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("send http request failed: %v", err)
	}
	defer resp.Body.Close()

	bodyByte, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("read response failed: %v", err)
	}
	return resp.StatusCode, string(bodyByte), nil
}

func WaitUntilBackendsCorrect(pods *v1.PodList, domain string, checkPeriod, timeout time.Duration) error {
	return wait.PollImmediate(checkPeriod, timeout, checkBackends(pods, domain))
}

func checkBackends(pods *v1.PodList, domain string) wait.ConditionFunc {
	return func() (bool, error) {
		backends, err := GetBackends(domain)
		if err != nil {
			return false, err
		}

		if backendsIsCorrect(pods, backends) {
			return true, nil
		}
		return false, nil
	}
}

func backendsIsCorrect(podList *v1.PodList, backends Backends) bool {
	// filter out unready pods
	var pods []*v1.Pod
	if podList != nil {
		for _, pod := range podList.Items {
			if pod.Status.ContainerStatuses[0].Ready == true {
				pods = append(pods, &pod)
			}
		}
	}

	framework.Logf("len: %v/%v", len(pods), len(backends.IPs))
	if len(pods) != len(backends.IPs) {
		return false
	}

	ipInDomain := make(map[string]bool)
	for _, addr := range backends.IPs {
		ipInDomain[addr.IP] = true
	}
	framework.Logf("current backends: %v", ipInDomain)
	for _, pod := range pods {
		if !ipInDomain[pod.Status.PodIP] {
			return false
		}
	}
	return true
}
