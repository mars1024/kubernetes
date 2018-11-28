package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"crypto/hmac"
	"crypto/sha1"
	"net/http"

	"github.com/golang/glog"

	"io/ioutil"
)

const (
	// StaragentURL URL of SA
	StaragentURL = "http://inc.agent.alibaba-inc.com/api/task"
	// StaragentAPITimeout API timeout of SA
	StaragentAPITimeout = "30"
	// StaragentAPIKey API key of SA
	StaragentAPIKey = "0eb5fc49b702c8f4ac3d17d5950af8ec"
	// StaragentAPICode API code of SA
	StaragentAPICode = "3a9338cfab4e1462fe51c69f295102f0"
)

// TaskSyncResponse response of SA task
type TaskSyncResponse struct {
	UID       string `json:"UID"`
	SUCCESS   bool   `json:"SUCCESS"`
	ERRORMSG  string `json:"ERRORMSG"`
	JOBNAME   string `json:"JOBNAME"`
	JOBRESULT string `json:"JOBRESULT"`
	ERRORCODE string `json:"ERRORCODE"`
	ERRORTYPE string `json:"ERRORTYPE"`
	ERRORURL  string `json:"ERRORURL"`
	IP        string `json:"IP"`
}

// hamcSha1 sha1 encrypt the content by key
func hamcSha1(key string, content string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(content))
	ret := fmt.Sprintf("%x", mac.Sum(nil))
	return ret
}

// constructStarAgentTaskURL construct the SA URL
func constructStarAgentTaskURL(cmd, hostIP, hostSN string) string {
	var tmp string
	tmp = "exeurl" + cmd + "ip" + hostIP + "key" + StaragentAPIKey + "servicetag" + hostSN + "timeout" + StaragentAPITimeout + "timestamp" + strconv.FormatInt(time.Now().Unix(), 10)

	var taskURL string
	taskURL = StaragentURL + "?"
	taskURL = taskURL + "exeurl=" + url.QueryEscape(cmd)
	taskURL = taskURL + "&timeout=" + StaragentAPITimeout
	taskURL = taskURL + "&ip=" + hostIP
	taskURL = taskURL + "&key=" + StaragentAPIKey
	taskURL = taskURL + "&servicetag=" + hostSN
	taskURL = taskURL + "&sign=" + hamcSha1(StaragentAPICode, tmp)
	taskURL = taskURL + "&timestamp=" + strconv.FormatInt(time.Now().Unix(), 10)
	glog.Infof("stargent task url is: %s", taskURL)
	return taskURL
}

// ResponseFromStarAgentTask run the cmd on specified host and get response
func ResponseFromStarAgentTask(cmd, hostIP, hostSN string) (string, error) {
	// 构建check请求
	url := constructStarAgentTaskURL(cmd, hostIP, hostSN)
	// 发送staragent调用
	resp, err := http.Get(url)
	if err != nil {
		glog.Errorf("Get error from staragent api, error: %v", err)
		return "", errors.New("Get error when sending staragent api request")
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// 解析staragent调用是否成功返回
	return checkStarAgentTaskResponse(respBody)
}

func checkStarAgentTaskResponse(resp []byte) (string, error) {
	taskSyncResponse := TaskSyncResponse{}
	err := json.Unmarshal(resp, &taskSyncResponse)
	if err != nil {
		msg := fmt.Sprintf("parse string %s to json taskSyncResponse error, %v", string(resp), err)
		glog.Error(msg)
		return "", fmt.Errorf(msg)
	}
	if !taskSyncResponse.SUCCESS {
		return "", fmt.Errorf("Task sync response fails, ERRORTYPE[%s],ERRORCODE[%s],ERRORMSG[%s]",
			taskSyncResponse.ERRORTYPE, taskSyncResponse.ERRORCODE, taskSyncResponse.ERRORMSG)
	}
	glog.Infof("StarAgent api response of is: %s", taskSyncResponse.JOBRESULT)
	return taskSyncResponse.JOBRESULT, nil
}
