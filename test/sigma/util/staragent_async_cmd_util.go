/*
Copyright 2019 Alipay.

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

package util

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/util/flowcontrol"
)

type CmdResult struct {
	Output string
	Error  error
}

type StaragentResp struct {
	ErrorMsg  string `json:"ERRORMSG"`
	JobResult string `json:"JOBRESULT"`
	Success   bool   `json:"SUCCESS"`
	UID       string `json:"UID"`
	JobName   string `json:"JOBNAME"`
}

type StaragentAsyncResp struct {
	Tid    string                `json:"TID"`
	Tasks  []*StaragentAsyncTask `json:"TASKS"`
	ErrMsg string                `json:"ERRORMSG"`
}

type StaragentAsyncTask struct {
	Uid string `json:"UID"`
	IP  string `json:"IP"`
}

type StaragentAsyncResult struct {
	Uid       string `json:"uid"`
	Success   bool   `json:"SUCCESS"`
	Status    string `json:"STATUS"`
	JobResult string `json:"JOBRESULT"`
	ErrorMsg  string `json:"ERRORMSG"`
	ErrorUrl  string `json:"ERRORURL"`
}

const (
	maxRetry       = 10
	simpleCmdPath  = "echo"
	simpleCmdParam = "hello"
)

const (
	longThrottleLatency = 50 * time.Millisecond
)

var (
	throttle = flowcontrol.NewTokenBucketRateLimiter(100, 5)

	ErrStaragentServerNotAvailable = fmt.Errorf("Staragent Server Not Available After Retry %d", maxRetry)
)

func tryThrottle() {
	now := time.Now()
	throttle.Accept()

	if latency := time.Since(now); latency > longThrottleLatency {
		glog.V(4).Infof("Staragent throttling request took %v", latency)
	}
}

type SaClient struct {
	Key    string
	Sign   string
	Server string
}

func (s *SaClient) Cmd(host string, cmd string) (output string, err error) {
	glog.V(4).Infof("SaClient#cmd, host: %s, cmd: %s", host, cmd)
	batchResult := s.batchIpCmdWithRetry([]string{host}, cmd)

	ret := batchResult[host]

	return ret.Output, ret.Error
}

func (s *SaClient) BatchCmd(hosts []string, cmd string) map[string]*CmdResult {
	return s.batchIpCmdWithRetry(hosts, cmd)
}

func generateSign(sign string, query map[string]string) string {
	slist := []string{}
	for k := range query {
		slist = append(slist, k)
	}
	querystr := ""
	sort.Strings(slist)
	for _, k := range slist {
		querystr += k + query[k]
	}

	key := []byte(sign)
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(querystr))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func (s *SaClient) syncCmd(ip string, path string, param string) (ret string, err error) {
	tryThrottle()

	query := map[string]string{
		"exeurl":    fmt.Sprintf("cmd://%s(%s)", path, param),
		"key":       s.Key,
		"ip":        ip,
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		"sync":      "true",
		"timeout":   strconv.Itoa(10),
	}
	query["sign"] = generateSign(s.Sign, query)
	values := url.Values{}
	for k, v := range query {
		values.Set(k, v)
	}
	resp, err := http.PostForm(fmt.Sprintf("http://%s/api/task", s.Server), values)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if nil != err {
		return "", err
	}

	staragentResp := &StaragentResp{}

	if err = json.Unmarshal(body, staragentResp); nil != err {
		return "", err
	} else {
		if staragentResp.Success == true {
			return staragentResp.JobResult, nil
		} else {
			return "", fmt.Errorf("staragent cmd api failed. message : %s", staragentResp.ErrorMsg)
		}
	}
}

func (s *SaClient) agentAliveBySimpleCmd(ip string) (err error) {
	msg, err := s.syncCmd(ip, simpleCmdPath, simpleCmdParam)
	if nil != err {
		return err
	} else if strings.TrimSpace(msg) == simpleCmdParam {
		return nil
	} else {
		return fmt.Errorf("agent simple cmd alive check failed. message: %s", msg)
	}
}

func (s *SaClient) getBatchIpPostData(ips []string, cmd string) url.Values {
	query := map[string]string{
		"exeurl":    fmt.Sprintf("cmd://%s", cmd),
		"key":       s.Key,
		"ip":        strings.Join(ips, ","),
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		"sync":      "false",
		"timeout":   strconv.Itoa(60 * 10),
	}
	query["sign"] = generateSign(s.Sign, query)
	values := url.Values{}
	for k, v := range query {
		values.Set(k, v)
	}
	return values
}

func (s *SaClient) postCmdDataWithRetry(data url.Values, index int) (ret *StaragentAsyncResp, err error) {
	tryThrottle()

	resp, err := http.PostForm(fmt.Sprintf("http://%s/api/task", s.Server), data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		if index < maxRetry {
			index++
			return s.postCmdDataWithRetry(data, index)
		} else {
			return nil, ErrStaragentServerNotAvailable
		}
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if index < maxRetry {
			index++
			return s.postCmdDataWithRetry(data, index)
		} else {
			return nil, ErrStaragentServerNotAvailable
		}
	}

	staragentResp := StaragentAsyncResp{}
	if err := json.Unmarshal(body, &staragentResp); err != nil {
		return nil, errors.New(fmt.Sprintf("Call startagent error: %s", string(body)))
	}

	return &staragentResp, nil
}

func (s *SaClient) queryResult(uid string) (finished bool, msg string, err error) {
	query := map[string]string{
		"uid":       uid,
		"key":       s.Key,
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
	}
	query["sign"] = generateSign(s.Sign, query)
	values := url.Values{}
	for k, v := range query {
		values.Set(k, v)
	}
	resp, err := http.PostForm(fmt.Sprintf("http://%s/api/query", s.Server), values)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if nil != err {
		return false, "", err
	}

	results := make([]*StaragentAsyncResult, 0)

	err = json.Unmarshal(body, &results)

	if nil != err {
		return false, "", err
	}

	if 0 >= len(results) {
		return false, "", errors.New("task result length === 0")
	}

	result := results[0]

	switch result.Status {
	case "running":
		return false, result.JobResult, nil
	case "finish":
		if result.Success {
			return true, result.JobResult, nil
		} else {
			return true, result.JobResult, fmt.Errorf("task failed, stdout: %s, err msg: %s", result.JobResult, result.ErrorMsg)
		}
	case "notfound":
		return false, result.JobResult, errors.New("task not found")
	default:
		return false, result.JobResult, fmt.Errorf("status not found, status: %s, stdout: %s, err msg: %s", result.Status, result.JobResult, result.ErrorMsg)
	}
}

func (s *SaClient) queryBatchTaskOnce(uidIpMap map[string]string, finishedMap map[string]*CmdResult) {
	for uid, ip := range uidIpMap {
		finished, msg, err := s.queryResult(uid)

		if finished {
			finishedMap[ip] = &CmdResult{
				Output: msg,
				Error:  err,
			}
			delete(uidIpMap, uid)
		}
	}
}

func (s *SaClient) checkBatchTaskHostAlive(uidIpMap map[string]string, finishedMap map[string]*CmdResult) {
	for uid, ip := range uidIpMap {
		err := s.agentAliveBySimpleCmd(ip)

		if nil != err {
			finishedMap[ip] = &CmdResult{
				Output: "",
				Error:  err,
			}
			delete(uidIpMap, uid)
		}
	}
}

func (s *SaClient) pollBatchTask(uidIpMap map[string]string) map[string]*CmdResult {
	ret := make(map[string]*CmdResult, len(uidIpMap))

	s.queryBatchTaskOnce(uidIpMap, ret)

	if 0 == len(uidIpMap) {
		return ret
	}

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	tryCount := 0

	for {
		select {
		case <-ticker.C:
			tryCount++
			s.queryBatchTaskOnce(uidIpMap, ret)

			if 0 == len(uidIpMap) {
				return ret
			}

			if 0 == tryCount%10 {
				s.checkBatchTaskHostAlive(uidIpMap, ret)
				if 0 == len(uidIpMap) {
					return ret
				}
			}
		}
	}
}

func (s *SaClient) batchIpCmdWithRetry(ips []string, cmd string) map[string]*CmdResult {
	postData := s.getBatchIpPostData(ips, cmd)

	resp, err := s.postCmdDataWithRetry(postData, 0)

	if nil != err {
		ret := make(map[string]*CmdResult, len(ips))
		for _, ip := range ips {
			ret[ip] = &CmdResult{
				Error: err,
			}
		}

		return ret
	}

	uidIpMap := make(map[string]string, len(ips))
	for i := range resp.Tasks {
		t := resp.Tasks[i]

		uidIpMap[t.Uid] = t.IP
	}

	return s.pollBatchTask(uidIpMap)
}
